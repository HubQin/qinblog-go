package services

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"

	"gorm.io/gorm"

	"github.com/qin/qinblog/internal/database"
	"github.com/qin/qinblog/internal/models"
	"github.com/qin/qinblog/internal/support"
)

// Posts 文章服务（对应 PostService + PostObserver + TranslateSlug Job）
type postService struct{}

var Posts = &postService{}

// PostInput 创建/更新文章的表单输入（对照 PostRequest）
type PostInput struct {
	Title      string
	Body       string // markdown 原文
	CategoryID uint
	TopicID    string // 可能是数字 id 或 "新专题~随机串"
	Sort       int
	TagIDs     string // JSON 数组，元素为数字 id 或 "新标签~随机串"
	IsShow     int8
	Slug       string
	Order      int
}

// Store 创建文章
func (s *postService) Store(userID uint, in PostInput) (*models.Post, error) {
	post := &models.Post{UserID: userID}
	return s.save(post, in, true)
}

// Update 更新文章
func (s *postService) Update(post *models.Post, in PostInput) (*models.Post, error) {
	return s.save(post, in, false)
}

func (s *postService) save(post *models.Post, in PostInput, isNew bool) (*models.Post, error) {
	oldCategoryID := post.CategoryID

	// 等价 PostsController：body = parsedown()；PostObserver::saving：clean + make_excerpt
	post.Title = in.Title
	post.Body = support.Clean(support.Parsedown(in.Body))
	post.Excerpt = support.MakeExcerpt(post.Body, 200)
	post.CategoryID = in.CategoryID
	post.IsShow = in.IsShow
	post.Order = in.Order
	if in.Slug != "" {
		s.CreateUniqueSlug(post, in.Slug)
	}

	// 更新时记录旧标签，供 sync/detach 后重新统计
	var oldTagIDs []uint
	if !isNew && post.ID != 0 {
		database.DB.Table("post_tag").Where("post_id = ?", post.ID).Pluck("tag_id", &oldTagIDs)
	}

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		// 处理标签（新建 name~random 标签）
		tagIDs, err := s.handleTag(tx, in.TagIDs)
		if err != nil {
			return err
		}

		// 处理专题（新建 name~random 专题）
		if err := s.handleTopic(tx, in, post); err != nil {
			return err
		}

		if err := tx.Save(post).Error; err != nil {
			return err
		}

		// 同步文章标签（等价 attach/sync/detach）
		if !isNew {
			if err := tx.Exec("DELETE FROM post_tag WHERE post_id = ?", post.ID).Error; err != nil {
				return err
			}
		}
		for _, tagID := range tagIDs {
			if err := tx.Exec("INSERT INTO post_tag (post_id, tag_id) VALUES (?, ?)", post.ID, tagID).Error; err != nil {
				return err
			}
		}

		// 更新标签文章数量（新旧标签合并统计，删除无文章的标签）
		s.updateTagsPostCount(tx, uniqueUints(append(tagIDs, oldTagIDs...)))
		return nil
	})
	if err != nil {
		return nil, err
	}

	// 等价 PostObserver::saved
	s.afterSaved(post, oldCategoryID)
	return post, nil
}

// afterSaved saved 后置处理：空 slug 异步翻译、更新分类计数、清缓存、同步搜索索引
func (s *postService) afterSaved(post *models.Post, oldCategoryID uint) {
	if post.Slug == "" {
		go s.translateSlug(post.ID, post.Title)
	}
	s.updateCategoryPostCount(post.CategoryID)
	if oldCategoryID != 0 && oldCategoryID != post.CategoryID {
		s.updateCategoryPostCount(oldCategoryID)
	}
	Sidebar.ForgetPostRelated()
	Search.IndexPost(post)
}

// Delete 删除文章（等价 PostObserver::deleting/deleted）
func (s *postService) Delete(post *models.Post) error {
	var tagIDs []uint
	database.DB.Table("post_tag").Where("post_id = ?", post.ID).Pluck("tag_id", &tagIDs)

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(post).Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM post_tag WHERE post_id = ?", post.ID).Error; err != nil {
			return err
		}
		s.updateTagsPostCount(tx, tagIDs)
		return nil
	})
	if err != nil {
		return err
	}

	s.updateCategoryPostCount(post.CategoryID)
	Sidebar.ForgetPostRelated()
	Search.DeletePost(post.ID)
	return nil
}

// handleTag 解析 tag_ids JSON：数字为已有标签，"name~random" 创建新标签（等价 PostService::handleTag）
func (s *postService) handleTag(tx *gorm.DB, tagIDsJSON string) ([]uint, error) {
	if strings.TrimSpace(tagIDsJSON) == "" {
		return nil, nil
	}
	var raw []interface{}
	if err := json.Unmarshal([]byte(tagIDsJSON), &raw); err != nil {
		return nil, nil // 与 json_decode 失败返回空一致
	}
	var ids []uint
	for _, item := range raw {
		switch v := item.(type) {
		case float64:
			ids = append(ids, uint(v))
		case string:
			if n, err := strconv.ParseUint(v, 10, 64); err == nil {
				ids = append(ids, uint(n))
				continue
			}
			// 新标签：name~随机串
			name := strings.SplitN(v, "~", 2)[0]
			if name == "" {
				continue
			}
			tag := models.Tag{Name: name, PostCount: 1}
			if err := tx.Create(&tag).Error; err != nil {
				return nil, err
			}
			ids = append(ids, tag.ID)
		}
	}
	return ids, nil
}

// handleTopic 处理专题：含 "~" 时新建专题（等价 PostService::handleTopic）
func (s *postService) handleTopic(tx *gorm.DB, in PostInput, post *models.Post) error {
	if in.TopicID == "" || in.TopicID == "0" {
		post.TopicID = 0
		return nil
	}
	if strings.Contains(in.TopicID, "~") {
		topic := models.Topic{Name: strings.SplitN(in.TopicID, "~", 2)[0]}
		if err := tx.Create(&topic).Error; err != nil {
			return err
		}
		post.TopicID = topic.ID
	} else {
		n, err := strconv.ParseUint(in.TopicID, 10, 64)
		if err != nil {
			return fmt.Errorf("非法的专题 id: %s", in.TopicID)
		}
		post.TopicID = uint(n)
	}
	if in.Sort > 0 {
		post.Sort = in.Sort
	} else {
		post.Sort = 1
	}
	return nil
}

// CreateUniqueSlug 生成唯一 slug（等价 PostService::creatUniqueSlug 的 RLIKE 查重）
func (s *postService) CreateUniqueSlug(post *models.Post, text string) {
	slug := support.Slug(text)
	if slug == "" {
		return
	}
	var count int64
	database.DB.Model(&models.Post{}).
		Where("slug RLIKE ?", fmt.Sprintf("^%s(-[0-9]+)?$", slug)).
		Where("id <> ?", post.ID).
		Count(&count)
	if count > 0 {
		post.Slug = fmt.Sprintf("%s-%d", slug, count)
	} else {
		post.Slug = slug
	}
}

// updateTagsPostCount 重新统计标签文章数，删除无文章的标签（等价 PostService::updateTagsPostCount）
func (s *postService) updateTagsPostCount(tx *gorm.DB, tagIDs []uint) {
	if len(tagIDs) == 0 {
		return
	}
	type tagCount struct {
		TagID     uint
		PostCount int
	}
	var counts []tagCount
	tx.Table("post_tag").
		Select("tag_id, COUNT(tag_id) AS post_count").
		Where("tag_id IN ?", tagIDs).
		Group("tag_id").
		Scan(&counts)

	hasPost := make(map[uint]bool, len(counts))
	for _, c := range counts {
		hasPost[c.TagID] = true
		tx.Model(&models.Tag{}).Where("id = ?", c.TagID).UpdateColumn("post_count", c.PostCount)
	}
	var without []uint
	for _, id := range tagIDs {
		if !hasPost[id] {
			without = append(without, id)
		}
	}
	if len(without) > 0 {
		tx.Where("id IN ?", without).Delete(&models.Tag{})
	}
}

// updateCategoryPostCount 更新分类的已发布文章数（等价 PostObserver::dbUpdatePostCount）
func (s *postService) updateCategoryPostCount(categoryID uint) {
	if categoryID == 0 {
		return
	}
	database.DB.Model(&models.Category{}).Where("id = ?", categoryID).UpdateColumn("post_count",
		gorm.Expr("(SELECT COUNT(*) FROM posts WHERE category_id = ? AND is_show = 1)", categoryID))
}

// translateSlug 异步翻译标题生成 slug（等价 TranslateSlug 队列任务）
func (s *postService) translateSlug(postID uint, title string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[warn] translate slug panic: %v", r)
		}
	}()
	var post models.Post
	if err := database.DB.First(&post, postID).Error; err != nil {
		return
	}
	s.CreateUniqueSlug(&post, Translate.Translate(title))
	if post.Slug == "" {
		return
	}
	database.DB.Model(&models.Post{}).Where("id = ?", postID).UpdateColumn("slug", post.Slug)
}

// TagIDs 文章当前关联的标签 id
func (s *postService) TagIDs(postID uint) []uint {
	var ids []uint
	database.DB.Table("post_tag").Where("post_id = ?", postID).Pluck("tag_id", &ids)
	return ids
}

func uniqueUints(in []uint) []uint {
	seen := make(map[uint]bool, len(in))
	var out []uint
	for _, v := range in {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}
