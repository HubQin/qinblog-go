package web

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/qin/qinblog/internal/database"
	"github.com/qin/qinblog/internal/middleware"
	"github.com/qin/qinblog/internal/models"
	"github.com/qin/qinblog/internal/render"
	"github.com/qin/qinblog/internal/services"
	"github.com/qin/qinblog/internal/support"
)

// CategoryShow 分类文章列表（GET /categories/:id）
func CategoryShow(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var category models.Category
	if err := database.DB.First(&category, id).Error; err != nil {
		c.String(http.StatusNotFound, "分类不存在")
		return
	}
	q := database.DB.Model(&models.Post{}).Preload("Category").
		Scopes(models.ScopePublished, models.ScopeRecently).
		Where("category_id = ?", category.ID)
	var posts []models.Post
	p, err := support.Paginate(q, page(c), 20, &posts, c.Request.URL.Path, c.Request.URL.Query())
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	render.HTML(c, http.StatusOK, "posts/index", gin.H{
		"posts": posts, "pagination": p, "pageTitle": category.Name,
	})
}

// TagShow 标签文章列表（GET /tags/:id）
func TagShow(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var tag models.Tag
	if err := database.DB.First(&tag, id).Error; err != nil {
		c.String(http.StatusNotFound, "标签不存在")
		return
	}
	q := database.DB.Model(&models.Post{}).Preload("Category").
		Scopes(models.ScopePublished, models.ScopeRecently).
		Where("id IN (?)", database.DB.Table("post_tag").Where("tag_id = ?", tag.ID).Select("post_id"))
	var posts []models.Post
	p, err := support.Paginate(q, page(c), 20, &posts, c.Request.URL.Path, c.Request.URL.Query())
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	render.HTML(c, http.StatusOK, "posts/index", gin.H{
		"posts": posts, "pagination": p, "pageTitle": tag.Name,
	})
}

// TopicsIndex 专题列表页（GET /topics，每个专题展示前 5 篇）
func TopicsIndex(c *gin.Context) {
	var topics []models.Topic
	// 仅展示有已发布文章的专题，文章删光后空专题不再停留在前台
	database.DB.Where("id IN (?)", database.DB.Model(&models.Post{}).
		Where("is_show = ?", 1).Select("topic_id")).Find(&topics)
	// Preload+Limit 无法按父级限数，逐个专题取前 5 篇
	for i := range topics {
		database.DB.Select("id", "title", "slug", "topic_id", "created_at").
			Where("topic_id = ? AND is_show = ?", topics[i].ID, 1).
			Order("sort").Limit(5).Find(&topics[i].Posts)
	}
	render.HTML(c, http.StatusOK, "topics/index", gin.H{"topics": topics})
}

// TopicShow 专题文章列表（GET /topics/:id）
func TopicShow(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var topic models.Topic
	if err := database.DB.First(&topic, id).Error; err != nil {
		c.String(http.StatusNotFound, "专题不存在")
		return
	}
	q := database.DB.Model(&models.Post{}).Preload("Category").
		Scopes(models.ScopePublished).
		Where("topic_id = ?", topic.ID).Order("sort")
	var posts []models.Post
	p, err := support.Paginate(q, page(c), 20, &posts, c.Request.URL.Path, c.Request.URL.Query())
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	render.HTML(c, http.StatusOK, "posts/index", gin.H{
		"posts": posts, "pagination": p, "pageTitle": topic.Name,
	})
}

// UsersShow 用户主页（GET /users/:id，发布的文章 / 评论两个 tab，分页 5）
func UsersShow(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var user models.User
	if err := database.DB.First(&user, id).Error; err != nil {
		c.String(http.StatusNotFound, "用户不存在")
		return
	}

	tab := c.DefaultQuery("tab", "posts")
	data := gin.H{"profileUser": &user, "tab": tab}

	if tab == "comments" {
		q := database.DB.Model(&models.Comment{}).Preload("Post").
			Where("user_id = ?", user.ID).Order("created_at DESC")
		var comments []models.Comment
		p, err := support.Paginate(q, page(c), 5, &comments, c.Request.URL.Path, c.Request.URL.Query())
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}
		data["userComments"] = comments
		data["pagination"] = p
	} else {
		q := database.DB.Model(&models.Post{}).Preload("Category").
			Scopes(models.ScopePublished, models.ScopeRecently).
			Where("user_id = ?", user.ID)
		var posts []models.Post
		p, err := support.Paginate(q, page(c), 5, &posts, c.Request.URL.Path, c.Request.URL.Query())
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}
		data["userPosts"] = posts
		data["pagination"] = p
	}
	render.HTML(c, http.StatusOK, "users/show", data)
}

// NotificationsIndex 通知列表（GET /notifications，读取后清零未读）
func NotificationsIndex(c *gin.Context) {
	user := middleware.CurrentUser(c)
	q := database.DB.Model(&models.Notification{}).
		Where("notifiable_type = ? AND notifiable_id = ?", models.NotifiableTypeUser, user.ID).
		Order("created_at DESC")
	var notifications []models.Notification
	p, err := support.Paginate(q, page(c), 20, &notifications, c.Request.URL.Path, c.Request.URL.Query())
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	// 标记全部已读（等价 markAsRead）
	services.Notify.MarkAllRead(user.ID)

	render.HTML(c, http.StatusOK, "notifications/index", gin.H{
		"notifications": notifications, "pagination": p,
	})
}

// About 关于页（GET /about，模板内用 markdown 函数渲染 settings 中的 about）
func About(c *gin.Context) {
	about := services.Settings.Get("about")
	render.HTML(c, http.StatusOK, "pages/about", gin.H{"about": about})
}
