package admin

import (
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

func page(c *gin.Context) int {
	n, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if n < 1 {
		n = 1
	}
	return n
}

func paramID(c *gin.Context) uint {
	id, _ := strconv.Atoi(c.Param("id"))
	return uint(id)
}

// Dashboard 仪表盘（文章/评论/用户计数）
func Dashboard(c *gin.Context) {
	var postCount, commentCount, userCount int64
	database.DB.Model(&models.Post{}).Count(&postCount)
	database.DB.Model(&models.Comment{}).Count(&commentCount)
	database.DB.Model(&models.User{}).Count(&userCount)
	render.HTML(c, http.StatusOK, "admin/dashboard", gin.H{
		"postCount": postCount, "commentCount": commentCount, "userCount": userCount,
		"adminActive": "dashboard",
	})
}

// ---------- 文章管理 ----------

// PostsIndex 文章列表（筛选：关键词/是否显示）
func PostsIndex(c *gin.Context) {
	q := database.DB.Model(&models.Post{}).Preload("Category").Preload("User").Order("created_at DESC")
	kw := strings.TrimSpace(c.Query("keyword"))
	if kw != "" {
		q = q.Where("title LIKE ?", "%"+kw+"%")
	}
	isShow := c.Query("is_show")
	if isShow != "" {
		q = q.Where("is_show = ?", isShow)
	}
	var posts []models.Post
	p, _ := support.Paginate(q, page(c), 20, &posts, c.Request.URL.Path, c.Request.URL.Query())
	render.HTML(c, http.StatusOK, "admin/posts", gin.H{
		"posts": posts, "pagination": p, "adminActive": "posts",
		"keyword": kw, "isShow": isShow,
	})
}

// PostToggleShow 显示/隐藏切换
func PostToggleShow(c *gin.Context) {
	var post models.Post
	if err := database.DB.First(&post, paramID(c)).Error; err == nil {
		newVal := int8(1)
		if post.IsShow == 1 {
			newVal = 0
		}
		database.DB.Model(&post).UpdateColumn("is_show", newVal)
		post.IsShow = newVal
		services.Posts.RefreshCounts(&post)
		services.Sidebar.ForgetPostRelated()
	}
	c.Redirect(http.StatusFound, back(c, "/admin/posts"))
}

// PostDestroy 删除文章
func PostDestroy(c *gin.Context) {
	var post models.Post
	if err := database.DB.First(&post, paramID(c)).Error; err == nil {
		_ = services.Posts.Delete(&post)
		middleware.Flash(c, "success", "文章已删除")
	}
	c.Redirect(http.StatusFound, back(c, "/admin/posts"))
}

// ---------- 评论管理 ----------

// CommentsIndex 评论列表（可筛选待审核）
func CommentsIndex(c *gin.Context) {
	q := database.DB.Model(&models.Comment{}).Preload("User").Preload("Post").Order("created_at DESC")
	pending := c.Query("pending")
	if pending == "1" {
		q = q.Where("approved = ?", 0)
	}
	var comments []models.Comment
	p, _ := support.Paginate(q, page(c), 20, &comments, c.Request.URL.Path, c.Request.URL.Query())
	render.HTML(c, http.StatusOK, "admin/comments", gin.H{
		"comments": comments, "pagination": p, "adminActive": "comments", "pending": pending,
	})
}

// CommentsReview 批量审核（等价 laravel-admin 的 comments/review）
func CommentsReview(c *gin.Context) {
	ids := c.PostFormArray("ids[]")
	if len(ids) == 0 {
		ids = c.PostFormArray("ids")
	}
	if len(ids) > 0 {
		database.DB.Model(&models.Comment{}).Where("id IN ?", ids).UpdateColumn("approved", 1)
		middleware.Flash(c, "success", "审核通过")
	}
	c.Redirect(http.StatusFound, back(c, "/admin/comments"))
}

// CommentDestroy 删除评论
func CommentDestroy(c *gin.Context) {
	var comment models.Comment
	if err := database.DB.First(&comment, paramID(c)).Error; err == nil {
		_ = services.Comments.Delete(&comment)
		middleware.Flash(c, "success", "评论已删除")
	}
	c.Redirect(http.StatusFound, back(c, "/admin/comments"))
}

// ---------- 分类 ----------

func CategoriesIndex(c *gin.Context) {
	var items []models.Category
	database.DB.Find(&items)
	render.HTML(c, http.StatusOK, "admin/categories", gin.H{"items": items, "adminActive": "categories"})
}

func CategorySave(c *gin.Context) {
	item := models.Category{
		Name:        strings.TrimSpace(c.PostForm("name")),
		Icon:        strings.TrimSpace(c.PostForm("icon")),
		Description: strings.TrimSpace(c.PostForm("description")),
	}
	if item.Name == "" {
		middleware.Flash(c, "danger", "名称不能为空")
		c.Redirect(http.StatusFound, "/admin/categories")
		return
	}
	if id := paramID(c); id > 0 {
		database.DB.Model(&models.Category{}).Where("id = ?", id).
			Updates(map[string]interface{}{"name": item.Name, "icon": item.Icon, "description": item.Description})
	} else {
		database.DB.Create(&item)
	}
	services.Sidebar.ForgetPostRelated()
	middleware.Flash(c, "success", "保存成功")
	c.Redirect(http.StatusFound, "/admin/categories")
}

func CategoryDestroy(c *gin.Context) {
	database.DB.Delete(&models.Category{}, paramID(c))
	services.Sidebar.ForgetPostRelated()
	middleware.Flash(c, "success", "已删除")
	c.Redirect(http.StatusFound, "/admin/categories")
}

// ---------- 标签 ----------

func TagsIndex(c *gin.Context) {
	var items []models.Tag
	database.DB.Order("post_count DESC").Find(&items)
	render.HTML(c, http.StatusOK, "admin/tags", gin.H{"items": items, "adminActive": "tags"})
}

func TagSave(c *gin.Context) {
	name := strings.TrimSpace(c.PostForm("name"))
	if name == "" {
		middleware.Flash(c, "danger", "名称不能为空")
		c.Redirect(http.StatusFound, "/admin/tags")
		return
	}
	if id := paramID(c); id > 0 {
		database.DB.Model(&models.Tag{}).Where("id = ?", id).Updates(map[string]interface{}{
			"name": name, "description": c.PostForm("description"),
		})
	} else {
		database.DB.Create(&models.Tag{Name: name, Description: c.PostForm("description")})
	}
	services.Sidebar.ForgetPostRelated()
	middleware.Flash(c, "success", "保存成功")
	c.Redirect(http.StatusFound, "/admin/tags")
}

func TagDestroy(c *gin.Context) {
	id := paramID(c)
	database.DB.Exec("DELETE FROM post_tag WHERE tag_id = ?", id)
	database.DB.Delete(&models.Tag{}, id)
	services.Sidebar.ForgetPostRelated()
	middleware.Flash(c, "success", "已删除")
	c.Redirect(http.StatusFound, "/admin/tags")
}

// ---------- 专题 ----------

func TopicsIndex(c *gin.Context) {
	var items []models.Topic
	database.DB.Find(&items)
	render.HTML(c, http.StatusOK, "admin/topics", gin.H{"items": items, "adminActive": "topics"})
}

func TopicSave(c *gin.Context) {
	name := strings.TrimSpace(c.PostForm("name"))
	if name == "" {
		middleware.Flash(c, "danger", "名称不能为空")
		c.Redirect(http.StatusFound, "/admin/topics")
		return
	}
	if id := paramID(c); id > 0 {
		database.DB.Model(&models.Topic{}).Where("id = ?", id).Updates(map[string]interface{}{
			"name": name, "description": c.PostForm("description"),
		})
	} else {
		database.DB.Create(&models.Topic{Name: name, Description: c.PostForm("description")})
	}
	middleware.Flash(c, "success", "保存成功")
	c.Redirect(http.StatusFound, "/admin/topics")
}

func TopicDestroy(c *gin.Context) {
	database.DB.Delete(&models.Topic{}, paramID(c))
	middleware.Flash(c, "success", "已删除")
	c.Redirect(http.StatusFound, "/admin/topics")
}

// ---------- 栏目 ----------

func ColumnsIndex(c *gin.Context) {
	var items []models.Column
	database.DB.Find(&items)
	render.HTML(c, http.StatusOK, "admin/columns", gin.H{"items": items, "adminActive": "columns"})
}

func ColumnSave(c *gin.Context) {
	name := strings.TrimSpace(c.PostForm("name"))
	if name == "" {
		middleware.Flash(c, "danger", "名称不能为空")
		c.Redirect(http.StatusFound, "/admin/columns")
		return
	}
	if id := paramID(c); id > 0 {
		database.DB.Model(&models.Column{}).Where("id = ?", id).Updates(map[string]interface{}{
			"name": name, "link": c.PostForm("link"), "description": c.PostForm("description"),
		})
	} else {
		database.DB.Create(&models.Column{Name: name, Link: c.PostForm("link"), Description: c.PostForm("description")})
	}
	services.Sidebar.ForgetColumns()
	middleware.Flash(c, "success", "保存成功")
	c.Redirect(http.StatusFound, "/admin/columns")
}

func ColumnDestroy(c *gin.Context) {
	database.DB.Delete(&models.Column{}, paramID(c))
	services.Sidebar.ForgetColumns()
	middleware.Flash(c, "success", "已删除")
	c.Redirect(http.StatusFound, "/admin/columns")
}

// ---------- 友链 ----------

func LinksIndex(c *gin.Context) {
	var items []models.LinkModel
	database.DB.Order("sort").Find(&items)
	render.HTML(c, http.StatusOK, "admin/links", gin.H{"items": items, "adminActive": "links"})
}

func LinkSave(c *gin.Context) {
	name := strings.TrimSpace(c.PostForm("name"))
	if name == "" {
		middleware.Flash(c, "danger", "名称不能为空")
		c.Redirect(http.StatusFound, "/admin/links")
		return
	}
	sort, _ := strconv.Atoi(c.DefaultPostForm("sort", "0"))
	status, _ := strconv.Atoi(c.DefaultPostForm("status", "1"))
	if id := paramID(c); id > 0 {
		database.DB.Model(&models.LinkModel{}).Where("id = ?", id).Updates(map[string]interface{}{
			"name": name, "url": c.PostForm("url"), "logo": c.PostForm("logo"),
			"sort": sort, "status": status,
		})
	} else {
		database.DB.Create(&models.LinkModel{
			Name: name, URL: c.PostForm("url"), Logo: c.PostForm("logo"),
			Sort: sort, Status: int8(status),
		})
	}
	services.Sidebar.ForgetLinks()
	middleware.Flash(c, "success", "保存成功")
	c.Redirect(http.StatusFound, "/admin/links")
}

func LinkDestroy(c *gin.Context) {
	database.DB.Delete(&models.LinkModel{}, paramID(c))
	services.Sidebar.ForgetLinks()
	middleware.Flash(c, "success", "已删除")
	c.Redirect(http.StatusFound, "/admin/links")
}

// ---------- 用户 ----------

func UsersIndex(c *gin.Context) {
	q := database.DB.Model(&models.User{}).Order("created_at DESC")
	var users []models.User
	p, _ := support.Paginate(q, page(c), 20, &users, c.Request.URL.Path, c.Request.URL.Query())
	render.HTML(c, http.StatusOK, "admin/users", gin.H{"users": users, "pagination": p, "adminActive": "users"})
}

// ---------- 站点设置 ----------

func SettingsShow(c *gin.Context) {
	render.HTML(c, http.StatusOK, "admin/settings", gin.H{"settings": services.Settings.All(), "adminActive": "settings"})
}

func SettingsSave(c *gin.Context) {
	values := map[string]string{}
	for _, key := range services.SettingKeys {
		values[key] = c.PostForm(key)
	}
	if err := services.Settings.Save(values); err != nil {
		middleware.Flash(c, "danger", "保存失败："+err.Error())
	} else {
		middleware.Flash(c, "success", "设置已保存")
	}
	c.Redirect(http.StatusFound, "/admin/settings")
}

func back(c *gin.Context, fallback string) string {
	if ref := c.Request.Referer(); ref != "" {
		return ref
	}
	return fallback
}
