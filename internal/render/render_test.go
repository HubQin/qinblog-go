package render

import (
	"database/sql"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/qin/qinblog/internal/models"
	"github.com/qin/qinblog/internal/support"
)

// TestRenderAllTemplates 对全部页面模板执行渲染，暴露字段名/函数/管道错误
func TestRenderAllTemplates(t *testing.T) {
	e, err := New("../../web/templates")
	if err != nil {
		t.Fatalf("解析模板失败: %v", err)
	}

	now := time.Now()
	user := &models.User{ID: 1, Name: "admin", Email: "a@b.c", IsAdmin: true,
		NotificationCount: 2, CreatedAt: now,
		EmailVerifiedAt: sql.NullTime{Time: now, Valid: true}}
	category := models.Category{ID: 1, Name: "Go", Icon: "fas fa-code", PostCount: 3}
	topic := &models.Topic{ID: 1, Name: "Go 入门", Description: "系列", PostCount: 2,
		Posts: []models.Post{{ID: 2, Title: "第二篇", Slug: "part-2"}}}
	post := &models.Post{ID: 1, Title: "测试文章", Body: "<p>正文</p>", Excerpt: "摘要",
		UserID: 1, CategoryID: 1, TopicID: 1, CommentCount: 1, ViewCount: 10, IsShow: 1,
		Slug: "test-post", CreatedAt: now, UpdatedAt: now,
		User: user, Category: category, Topic: topic,
		Tags: []models.Tag{{ID: 1, Name: "go", PostCount: 2}}}
	posts := []models.Post{*post}
	parentID := uint(1)
	comments := []models.Comment{{
		ID: 1, UserID: 1, Content: "<p>不错</p>", Approved: true, CreatedAt: now,
		User: user, Post: post,
		Replies: []models.Comment{{ID: 2, UserID: 1, Content: "<p>回复</p>",
			ParentID: &parentID, CreatedAt: now, User: user}},
	}}
	pagination := support.NewPagination(1, 20, 40, "/", url.Values{})

	globals := gin.H{
		"siteConfigs": map[string]string{"name": "QinBlog", "slogan": "记录",
			"main_color": "rgba(54,137,218,0.8)", "footer": "footer", "notice": "公告"},
		"currentUser": user,
		"csrfToken":   "token123",
		"flash":       map[string]string{"success": "ok"},
		"errors":      map[string]string{},
		"old":         map[string]string{},
		"routeClass":  "posts-index",
		"currentURL":  "/",
		"categories":  []models.Category{category},
		"tags":        []models.Tag{{ID: 1, Name: "go", PostCount: 2, Color: "info"}},
		"archives": []struct {
			YearMonth string
			Count     int
		}{{"2024-01", 3}},
		"links":   []models.LinkModel{{ID: 1, Name: "友链", URL: "https://example.com"}},
		"columns": []models.Column{{ID: 1, Name: "栏目", Link: "/categories/1"}},
	}

	type searchItem struct {
		models.Post
		HighlightTitle   string
		HighlightExcerpt string
	}

	pageData := map[string]gin.H{
		"posts/index": {"posts": posts, "pagination": pagination, "pageTitle": "Go"},
		"posts/show": {"post": post, "topicPosts": topic.Posts, "viewCountToday": int64(5),
			"totalViews": 15, "comments": comments, "canManage": true},
		"posts/create_and_edit": {"post": post, "postTagIDs": []uint{1},
			"formCategories": []models.Category{category},
			"formTopics":     []models.Topic{*topic},
			"formTags":       []models.Tag{{ID: 1, Name: "go"}}},
		"posts/search": {"items": []searchItem{{Post: *post,
			HighlightTitle: "高亮", HighlightExcerpt: "摘要"}},
			"pagination": pagination, "keyword": "go", "total": int64(1)},
		"topics/index":         {"topics": []models.Topic{*topic}},
		"users/show":           {"profileUser": user, "tab": "posts", "userPosts": posts, "pagination": pagination},
		"notifications/index":  {"notifications": []models.Notification{{ID: "uuid", Type: models.NotificationTypePostCommented, Data: `{"user_name":"tom","post_title":"标题","post_link":"/posts/1"}`, CreatedAt: now}}, "pagination": pagination},
		"pages/about":          {"about": "<p>关于</p>"},
		"errors/403":           {},
		"auth/login":           {},
		"auth/register":        {},
		"auth/verify":          {},
		"auth/passwords/email": {},
		"auth/passwords/reset": {"token": "tk", "email": "a@b.c"},
		"admin/dashboard":      {"postCount": int64(1), "commentCount": int64(2), "userCount": int64(3), "adminActive": "dashboard"},
		"admin/posts":          {"posts": posts, "pagination": pagination, "adminActive": "posts", "keyword": "", "isShow": ""},
		"admin/comments":       {"comments": comments, "pagination": pagination, "adminActive": "comments", "pending": ""},
		"admin/categories":     {"items": []models.Category{category}, "adminActive": "categories"},
		"admin/tags":           {"items": []models.Tag{{ID: 1, Name: "go", PostCount: 2}}, "adminActive": "tags"},
		"admin/topics":         {"items": []models.Topic{*topic}, "adminActive": "topics"},
		"admin/columns":        {"items": []models.Column{{ID: 1, Name: "栏目"}}, "adminActive": "columns"},
		"admin/links":          {"items": []models.LinkModel{{ID: 1, Name: "友链", URL: "https://example.com", Status: 1}}, "adminActive": "links"},
		"admin/users":          {"users": []models.User{*user}, "pagination": pagination, "adminActive": "users"},
		"admin/settings":       {"settings": map[string]string{"name": "QinBlog"}, "adminActive": "settings"},
	}

	// users/show 的评论 tab 单独跑一次
	pageData["users/show#comments"] = gin.H{"profileUser": user, "tab": "comments",
		"userComments": comments, "pagination": pagination}

	for name, data := range pageData {
		tplName := strings.SplitN(name, "#", 2)[0]
		tpl, ok := e.templates[tplName]
		if !ok {
			t.Errorf("缺少模板: %s", tplName)
			continue
		}
		merged := gin.H{}
		for k, v := range globals {
			merged[k] = v
		}
		for k, v := range data {
			merged[k] = v
		}
		var sb strings.Builder
		if err := tpl.ExecuteTemplate(&sb, "layouts/app", merged); err != nil {
			t.Errorf("渲染 %s 失败: %v", name, err)
		}
	}

	// 确认没有遗漏未测试的页面模板
	for name := range e.templates {
		if _, ok := pageData[name]; !ok {
			t.Errorf("页面模板 %s 未纳入渲染测试", name)
		}
	}
}
