package web

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/qin/qinblog/internal/database"
	"github.com/qin/qinblog/internal/middleware"
	"github.com/qin/qinblog/internal/models"
	"github.com/qin/qinblog/internal/render"
	"github.com/qin/qinblog/internal/services"
	"github.com/qin/qinblog/internal/support"
)

// page 读取 ?page= 参数
func page(c *gin.Context) int {
	n, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if n < 1 {
		n = 1
	}
	return n
}

// canManagePost own 鉴权（等价 PostPolicy::own）
func canManagePost(c *gin.Context, post *models.Post) bool {
	user := middleware.CurrentUser(c)
	return user != nil && (user.ID == post.UserID || user.IsAdmin)
}

// PostsIndex 首页文章列表（GET /）
func PostsIndex(c *gin.Context) {
	q := database.DB.Model(&models.Post{}).Preload("Category").
		Scopes(models.ScopePublished, models.ScopeRecently)
	var posts []models.Post
	p, err := support.Paginate(q, page(c), 20, &posts, c.Request.URL.Path, c.Request.URL.Query())
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	render.HTML(c, http.StatusOK, "posts/index", gin.H{"posts": posts, "pagination": p})
}

// PostsShow 文章详情（GET /posts/:id/*slug）
func PostsShow(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var post models.Post
	if err := database.DB.Preload("Category").Preload("User").Preload("Tags").First(&post, id).Error; err != nil {
		c.String(http.StatusNotFound, "文章不存在")
		return
	}

	// slug 不符时 301 到规范链接
	slug := strings.Trim(strings.TrimPrefix(c.Param("slug"), "/"), "/")
	if post.Slug != "" && slug != post.Slug {
		c.Redirect(http.StatusMovedPermanently, post.Link())
		return
	}

	// 专题内文章列表
	var topicPosts []models.Post
	if post.TopicID > 0 {
		database.DB.Select("id", "title", "slug").Where("topic_id = ?", post.TopicID).
			Order("sort").Find(&topicPosts)
	}

	viewCountToday := services.ViewCount.Incr(post.ID)
	comments := services.Comments.ApprovedTree(post.ID)

	render.HTML(c, http.StatusOK, "posts/show", gin.H{
		"post":           &post,
		"topicPosts":     topicPosts,
		"viewCountToday": viewCountToday,
		"totalViews":     post.ViewCount + int(viewCountToday),
		"comments":       comments,
		"canManage":      canManagePost(c, &post),
	})
}

// selectOption 下拉选项（小写 json 键供 Vue 组件使用）
type selectOption struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
	Icon string `json:"icon,omitempty"`
}

// postFormOptions 发布/编辑页的下拉数据
func postFormOptions() gin.H {
	var categories, topics, tags []selectOption
	database.DB.Model(&models.Category{}).Select("id", "name", "icon").Find(&categories)
	database.DB.Model(&models.Topic{}).Select("id", "name").Find(&topics)
	database.DB.Model(&models.Tag{}).Select("id", "name").Find(&tags)
	return gin.H{"formCategories": categories, "formTopics": topics, "formTags": tags}
}

// PostsCreate 发布页（GET /posts/create）
func PostsCreate(c *gin.Context) {
	data := postFormOptions()
	data["post"] = &models.Post{IsShow: 1}
	data["postTagIDs"] = []uint{}
	render.HTML(c, http.StatusOK, "posts/create_and_edit", data)
}

// bindPostInput 解析表单输入
func bindPostInput(c *gin.Context) (services.PostInput, map[string]string) {
	categoryID, _ := strconv.Atoi(c.PostForm("category_id"))
	isShow, _ := strconv.Atoi(c.DefaultPostForm("is_show", "1"))
	sort, _ := strconv.Atoi(c.PostForm("sort"))
	order, _ := strconv.Atoi(c.PostForm("order"))
	in := services.PostInput{
		Title:      strings.TrimSpace(c.PostForm("title")),
		Body:       c.PostForm("body"),
		CategoryID: uint(categoryID),
		TopicID:    c.PostForm("topic_id"),
		Sort:       sort,
		TagIDs:     c.PostForm("tag_ids"),
		IsShow:     int8(isShow),
		Slug:       strings.TrimSpace(c.PostForm("slug")),
		Order:      order,
	}

	// 等价 PostRequest 验证规则
	errs := map[string]string{}
	if l := len([]rune(in.Title)); l < 2 || l > 100 {
		errs["title"] = "标题必须介于 2 - 100 个字符之间"
	}
	if len([]rune(in.Body)) < 6 {
		errs["body"] = "文章内容至少 6 个字符"
	}
	if in.CategoryID == 0 {
		errs["category_id"] = "请选择分类"
	}
	return in, errs
}

// PostsStore 发布文章（POST /posts）
func PostsStore(c *gin.Context) {
	user := middleware.CurrentUser(c)
	in, errs := bindPostInput(c)
	if len(errs) > 0 {
		middleware.FlashErrors(c, errs)
		middleware.FlashOld(c, c.Request.PostForm)
		c.Redirect(http.StatusFound, "/posts/create")
		return
	}
	post, err := services.Posts.Store(user.ID, in)
	if err != nil {
		middleware.FlashErrors(c, map[string]string{"error": err.Error()})
		c.Redirect(http.StatusFound, "/posts/create")
		return
	}
	middleware.Flash(c, "success", "发布成功")
	c.Redirect(http.StatusFound, post.Link())
}

// PostsEdit 编辑页（GET /posts/:id/edit）
func PostsEdit(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var post models.Post
	if err := database.DB.First(&post, id).Error; err != nil {
		c.String(http.StatusNotFound, "文章不存在")
		return
	}
	if !canManagePost(c, &post) {
		c.String(http.StatusForbidden, "无权操作")
		return
	}
	// 编辑时把存储的 HTML 转回 markdown（等价 html_to_markdown）
	post.Body = support.HTMLToMarkdown(post.Body)

	data := postFormOptions()
	data["post"] = &post
	data["postTagIDs"] = services.Posts.TagIDs(post.ID)
	render.HTML(c, http.StatusOK, "posts/create_and_edit", data)
}

// PostsUpdate 更新文章（PUT /posts/:id）
func PostsUpdate(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var post models.Post
	if err := database.DB.First(&post, id).Error; err != nil {
		c.String(http.StatusNotFound, "文章不存在")
		return
	}
	if !canManagePost(c, &post) {
		c.String(http.StatusForbidden, "无权操作")
		return
	}
	in, errs := bindPostInput(c)
	if len(errs) > 0 {
		middleware.FlashErrors(c, errs)
		middleware.FlashOld(c, c.Request.PostForm)
		c.Redirect(http.StatusFound, fmt.Sprintf("/posts/%d/edit", post.ID))
		return
	}
	// slug 未修改时不重新生成
	if in.Slug == post.Slug {
		in.Slug = ""
	}
	updated, err := services.Posts.Update(&post, in)
	if err != nil {
		middleware.FlashErrors(c, map[string]string{"error": err.Error()})
		c.Redirect(http.StatusFound, fmt.Sprintf("/posts/%d/edit", post.ID))
		return
	}
	middleware.Flash(c, "success", "编辑成功")
	c.Redirect(http.StatusFound, updated.Link())
}

// PostsDestroy 删除文章（DELETE /posts/:id）
func PostsDestroy(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var post models.Post
	if err := database.DB.First(&post, id).Error; err != nil {
		c.String(http.StatusNotFound, "文章不存在")
		return
	}
	if !canManagePost(c, &post) {
		c.String(http.StatusForbidden, "无权操作")
		return
	}
	if err := services.Posts.Delete(&post); err != nil {
		middleware.Flash(c, "danger", err.Error())
	} else {
		middleware.Flash(c, "success", "删除成功！")
	}
	c.Redirect(http.StatusFound, "/")
}

// ArchiveShow 归档页（GET /archives/:year_month）
func ArchiveShow(c *gin.Context) {
	ym := c.Param("year_month")
	parts := strings.SplitN(ym, "-", 2)
	if len(parts) != 2 {
		c.String(http.StatusNotFound, "not found")
		return
	}
	q := database.DB.Model(&models.Post{}).Preload("Category").
		Scopes(models.ScopePublished, models.ScopeRecently).
		Where("YEAR(created_at) = ? AND MONTH(created_at) = ?", parts[0], parts[1])
	var posts []models.Post
	p, err := support.Paginate(q, page(c), 20, &posts, c.Request.URL.Path, c.Request.URL.Query())
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	render.HTML(c, http.StatusOK, "posts/index", gin.H{
		"posts": posts, "pagination": p,
		"pageTitle": fmt.Sprintf("%s年%s月", parts[0], parts[1]),
	})
}

// PostsSearch 全文搜索（GET /posts/search，需登录，等价 Laravel 中间件行为）
func PostsSearch(c *gin.Context) {
	keyword := support.StripTags(c.Query("query"))
	user := middleware.CurrentUser(c)
	isAdmin := user != nil && user.IsAdmin

	posts, total := services.Search.SearchPosts(keyword, isAdmin, page(c), 20)
	p := support.NewPagination(page(c), 20, total, c.Request.URL.Path, c.Request.URL.Query())

	// 高亮标题与摘要
	type searchItem struct {
		models.Post
		HighlightTitle   string
		HighlightExcerpt string
	}
	items := make([]searchItem, 0, len(posts))
	for _, post := range posts {
		items = append(items, searchItem{
			Post:             post,
			HighlightTitle:   services.Highlight(post.Title, keyword),
			HighlightExcerpt: services.Highlight(post.Excerpt, keyword),
		})
	}
	render.HTML(c, http.StatusOK, "posts/search", gin.H{
		"items": items, "pagination": p, "keyword": keyword, "total": total,
	})
}

// UploadPostImage 编辑器图片上传（POST /posts/upload_post_image）
func UploadPostImage(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"error": "上传失败", "filename": ""})
		return
	}
	path, err := services.Upload.SaveImage(file, "posts", 1024)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"error": "上传失败", "filename": ""})
		return
	}
	c.JSON(http.StatusOK, gin.H{"error": "上传成功", "filename": path})
}
