package main

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/qin/qinblog/internal/config"
	"github.com/qin/qinblog/internal/database"
	"github.com/qin/qinblog/internal/models"
	"github.com/qin/qinblog/internal/services"
	"github.com/qin/qinblog/internal/support"
)

// hexo 博客文章迁移命令：
// 1. 解析 D:\hexo-blog\source\_posts 下的 Markdown 文章（front matter: title/date/tags）
// 2. 将正文引用的本地图片复制到 UPLOAD_PATH 下，并把引用重写为 /storage/... 绝对路径
// 3. 复用 services.Posts.Store 创建文章（标签关联/计数维护/搜索索引/缓存失效自动完成）
// 4. 回写 created_at/updated_at 保留 hexo 原始发布时间

const (
	hexoPostsDir    = `D:\hexo-blog\source\_posts`
	defaultCategory = 10 // Other
	adminUserID     = 1
)

var imgRe = regexp.MustCompile(`!\[[^\]]*\]\(([^)\s]+)\)`)

type hexoPost struct {
	Title string
	Date  time.Time
	Tags  []string
	Body  string
	Slug  string
	File  string
}

func main() {
	cfg := config.Load()
	if err := database.Init(cfg); err != nil {
		log.Fatalf("database init: %v", err)
	}
	// 注意：不初始化搜索索引——server 运行时持有 bleve 文件锁，bleve.Open 会阻塞等待；
	// Store 的 afterSaved 中 IndexPost 在索引为空时静默跳过，迁移后统一用 cmd/indexer 重建

	ensureTagNameCapacity()

	posts := parseHexoPosts()
	if len(posts) == 0 {
		log.Fatal("未发现可迁移的文章")
	}

	for _, p := range posts {
		body := migrateImages(&p)
		tagIDs := resolveTagIDs(p.Tags)


		post, err := services.Posts.Store(adminUserID, services.PostInput{
			Title:      p.Title,
			Body:       body,
			CategoryID: defaultCategory,
			TagIDs:     tagIDs,
			IsShow:     1,
			Slug:       p.Slug,
		})
		if err != nil {
			log.Fatalf("创建文章 %q 失败: %v", p.Title, err)
		}

		// 保留 hexo 原始发布时间（Store 会写入当前时间）
		if err := database.DB.Model(&models.Post{}).Where("id = ?", post.ID).
			Updates(map[string]interface{}{"created_at": p.Date, "updated_at": p.Date}).Error; err != nil {
			log.Fatalf("回写发布时间失败 %q: %v", p.Title, err)
		}

		log.Printf("[ok] #%d %q (slug=%s, tags=%v, date=%s)", post.ID, p.Title, post.Slug, p.Tags, p.Date.Format("2006-01-02"))
	}

	log.Printf("迁移完成，共 %d 篇文章", len(posts))
}

// ensureTagNameCapacity tags.name 原为 varchar(10)，无法容纳 golangci-lint(12字符)，需扩容
func ensureTagNameCapacity() {
	var colType string
	database.DB.Raw("SELECT COLUMN_TYPE FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'tags' AND COLUMN_NAME = 'name'").Scan(&colType)
	if colType == "varchar(50)" {
		return
	}
	log.Printf("扩容 tags.name: %s -> varchar(50)", colType)
	if err := database.DB.Exec("ALTER TABLE tags MODIFY name varchar(50) NOT NULL").Error; err != nil {
		log.Fatalf("ALTER TABLE tags 失败: %v", err)
	}
}

// parseHexoPosts 解析 _posts 目录下的文章，排除 hello-world.md 示例文章
func parseHexoPosts() []hexoPost {
	entries, err := os.ReadDir(hexoPostsDir)
	if err != nil {
		log.Fatalf("读取 %s: %v", hexoPostsDir, err)
	}
	var posts []hexoPost
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") || e.Name() == "hello-world.md" {
			continue
		}
		file := filepath.Join(hexoPostsDir, e.Name())
		data, err := os.ReadFile(file)
		if err != nil {
			log.Fatalf("读取 %s: %v", file, err)
		}
		p := parseFrontMatter(string(data))
		p.File = file
		p.Slug = strings.TrimSuffix(e.Name(), ".md")
		if p.Title == "" {
			log.Fatalf("%s 缺少 title", file)
		}
		posts = append(posts, p)
	}
	sort.Slice(posts, func(i, j int) bool { return posts[i].Date.Before(posts[j].Date) })
	return posts
}

// parseFrontMatter 解析 YAML front matter 的 title/date/tags（支持内联 [a, b] 与列表 - a 两种写法）
func parseFrontMatter(md string) hexoPost {
	var p hexoPost
	rest := md
	if strings.HasPrefix(strings.TrimSpace(md), "---") {
		lines := strings.Split(md, "\n")
		inFM, inTagList := false, false
		var bodyLines []string
		for i, line := range lines {
			trimmed := strings.TrimSpace(line)
			if i == 0 && trimmed == "---" {
				inFM = true
				continue
			}
			if inFM && trimmed == "---" {
				inFM = false
				continue
			}
			if !inFM {
				bodyLines = append(bodyLines, line)
				continue
			}
			if trimmed == "" || strings.HasPrefix(trimmed, "#") {
				continue
			}
			if strings.HasPrefix(trimmed, "- ") {
				if inTagList {
					p.Tags = append(p.Tags, strings.TrimSpace(strings.TrimPrefix(trimmed, "- ")))
				}
				continue
			}
			inTagList = false
			if strings.HasPrefix(trimmed, "title:") {
				p.Title = strings.Trim(strings.TrimPrefix(trimmed, "title:"), `"' `)
			} else if strings.HasPrefix(trimmed, "date:") {
				p.Date, _ = time.ParseInLocation("2006-01-02 15:04:05", strings.TrimSpace(strings.TrimPrefix(trimmed, "date:")), time.Local)
			} else if strings.HasPrefix(trimmed, "tags:") {
				v := strings.TrimSpace(strings.TrimPrefix(trimmed, "tags:"))
				if strings.HasPrefix(v, "[") && strings.HasSuffix(v, "]") {
					for _, t := range strings.Split(strings.Trim(v, "[]"), ",") {
						if t = strings.TrimSpace(t); t != "" {
							p.Tags = append(p.Tags, t)
						}
					}
				} else if v == "" {
					inTagList = true
				}
			}
		}
		rest = strings.Join(bodyLines, "\n")
	}
	p.Body = strings.TrimSpace(rest)
	return p
}

// migrateImages 将正文中的本地相对路径图片复制到 UPLOAD_PATH/images/posts/{YYYYMM}/{slug}/，
// 并将引用重写为 /storage/... 绝对路径（供 {{markdown}} 渲染）。返回改写后的正文。
func migrateImages(p *hexoPost) string {
	monthDir := p.Date.Format("200601")
	destDir := filepath.Join(config.C.UploadPath, "images", "posts", monthDir, p.Slug)
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		log.Fatalf("创建图片目录 %s: %v", destDir, err)
	}

	count := 0
	body := imgRe.ReplaceAllStringFunc(p.Body, func(m string) string {
		sub := imgRe.FindStringSubmatch(m)
		src := sub[1]
		if strings.HasPrefix(src, "/") || strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://") {
			return m // 已是绝对路径，跳过
		}
		srcPath := filepath.Join(filepath.Dir(p.File), filepath.FromSlash(src))
		data, err := os.ReadFile(srcPath)
		if err != nil {
			log.Printf("[warn] %s 引用的图片不存在: %s", p.File, src)
			return m
		}
		dest := filepath.Join(destDir, filepath.Base(filepath.FromSlash(src)))
		if err := os.WriteFile(dest, data, 0o644); err != nil {
			log.Fatalf("复制图片 %s -> %s: %v", srcPath, dest, err)
		}
		count++
		rel := filepath.ToSlash(filepath.Join("images", "posts", monthDir, p.Slug, filepath.Base(filepath.FromSlash(src))))
		return strings.Replace(m, src, "/storage/"+rel, 1)
	})
	if count > 0 {
		log.Printf("[img] %s: 迁移 %d 张图片 -> %s", p.Title, count, destDir)
	}
	return body
}

// resolveTagIDs 大小写不敏感复用已有标签（utf8mb4_unicode_ci），否则以 name~random 交给 Store 新建。
// 注意必须用 json.Marshal 生成合法 JSON 数组（字符串元素带引号），否则 handleTag 解析失败会静默丢标签
func resolveTagIDs(tags []string) string {
	items := make([]interface{}, 0, len(tags))
	for _, name := range tags {
		var tag models.Tag
		err := database.DB.Where("name = ?", name).First(&tag).Error
		if err == nil {
			items = append(items, tag.ID)
			continue
		}
		items = append(items, name+"~"+support.RandomString(6))
	}
	b, err := json.Marshal(items)
	if err != nil {
		log.Fatalf("序列化 tag_ids 失败: %v", err)
	}
	return string(b)
}
