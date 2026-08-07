package main

import (
	"flag"
	"log"
	"strings"

	"github.com/qin/qinblog/internal/config"
	"github.com/qin/qinblog/internal/database"
	"github.com/qin/qinblog/internal/models"
	"github.com/qin/qinblog/internal/support"
)

// 一次性迁移命令：把存量 posts.body 中的 HTML 用 HTMLToMarkdown 回填为 markdown 原文。
// 用法：
//
//	go run ./cmd/migrate_body            # 正式执行
//	go run ./cmd/migrate_body --dry-run   # 只统计，不写库
func main() {
	dryRun := flag.Bool("dry-run", false, "只统计待迁移的文章数，不写库")
	flag.Parse()

	cfg := config.Load()
	if err := database.Init(cfg); err != nil {
		log.Fatalf("database init: %v", err)
	}

	var posts []models.Post
	if err := database.DB.Select("id", "title", "body").Find(&posts).Error; err != nil {
		log.Fatalf("load posts: %v", err)
	}

	converted, skipped, failed := 0, 0, 0
	for _, p := range posts {
		if !looksLikeHTML(p.Body) {
			skipped++ // 已是 markdown 原文（或空），跳过
			continue
		}
		md := support.HTMLToMarkdown(p.Body)
		if md == p.Body {
			// 转换失败时 HTMLToMarkdown 会原样返回，视为失败避免数据损坏
			failed++
			log.Printf("[warn] id=%d 《%s》转换失败，跳过", p.ID, p.Title)
			continue
		}
		converted++
		if *dryRun {
			continue
		}
		if err := database.DB.Model(&models.Post{}).Where("id = ?", p.ID).Update("body", md).Error; err != nil {
			failed++
			log.Printf("[error] id=%d 更新失败: %v", p.ID, err)
		}
	}

	action := "已迁移"
	if *dryRun {
		action = "待迁移(dry-run)"
	}
	log.Printf("迁移完成：%s %d 篇，跳过(已是markdown) %d 篇，失败 %d 篇", action, converted, skipped, failed)
	if !*dryRun && converted > 0 {
		log.Printf("提示：body 已变更，建议执行 go run ./cmd/indexer 重建搜索索引")
	}
}

// looksLikeHTML 粗略判断 body 是否为渲染后的 HTML（以块级标签开头或包含常见块级标签）
func looksLikeHTML(body string) bool {
	s := strings.TrimSpace(body)
	if s == "" {
		return false
	}
	if strings.HasPrefix(s, "<") {
		return true
	}
	for _, tag := range []string{"<p>", "<div>", "<h1>", "<h2>", "<h3>", "<h4>", "<pre>", "<ul>", "<ol>", "<table>", "<blockquote>"} {
		if strings.Contains(s, tag) {
			return true
		}
	}
	return false
}
