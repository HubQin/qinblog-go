package web

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/qin/qinblog/internal/database"
	"github.com/qin/qinblog/internal/middleware"
	"github.com/qin/qinblog/internal/models"
	"github.com/qin/qinblog/internal/services"
)

// CommentsStore 创建评论（POST /comments/store）
func CommentsStore(c *gin.Context) {
	user := middleware.CurrentUser(c)
	postID, _ := strconv.Atoi(c.PostForm("post_id"))
	content := strings.TrimSpace(c.PostForm("content"))
	if postID == 0 || len([]rune(content)) < 2 {
		middleware.Flash(c, "danger", "评论内容至少 2 个字符")
		c.Redirect(http.StatusFound, backURL(c))
		return
	}

	comment, post, err := services.Comments.Create(user, uint(postID), content, nil)
	if err != nil {
		middleware.Flash(c, "danger", "评论失败")
		c.Redirect(http.StatusFound, backURL(c))
		return
	}
	middleware.Flash(c, "success", "评论成功")
	c.Redirect(http.StatusFound, post.Link("#comment"+strconv.Itoa(int(comment.ID))))
}

// RepliesStore 创建回复（POST /replies/store）
func RepliesStore(c *gin.Context) {
	user := middleware.CurrentUser(c)
	postID, _ := strconv.Atoi(c.PostForm("post_id"))
	parentID, _ := strconv.Atoi(c.PostForm("parent_id"))
	content := strings.TrimSpace(c.PostForm("content"))
	if postID == 0 || parentID == 0 || len([]rune(content)) < 2 {
		middleware.Flash(c, "danger", "回复内容至少 2 个字符")
		c.Redirect(http.StatusFound, backURL(c))
		return
	}

	pid := uint(parentID)
	comment, post, err := services.Comments.Create(user, uint(postID), content, &pid)
	if err != nil {
		middleware.Flash(c, "danger", "回复失败")
		c.Redirect(http.StatusFound, backURL(c))
		return
	}
	middleware.Flash(c, "success", "回复成功")
	c.Redirect(http.StatusFound, post.Link("#comment"+strconv.Itoa(int(comment.ID))))
}

// CommentsDestroy 删除评论（DELETE /comments/:id，own 鉴权）
func CommentsDestroy(c *gin.Context) {
	user := middleware.CurrentUser(c)
	id, _ := strconv.Atoi(c.Param("id"))
	var comment models.Comment
	if err := database.DB.First(&comment, id).Error; err != nil {
		c.String(http.StatusNotFound, "评论不存在")
		return
	}
	if user.ID != comment.UserID && !user.IsAdmin {
		c.String(http.StatusForbidden, "无权操作")
		return
	}
	if err := services.Comments.Delete(&comment); err != nil {
		middleware.Flash(c, "danger", "删除失败")
	} else {
		middleware.Flash(c, "success", "评论已删除")
	}
	c.Redirect(http.StatusFound, backURL(c))
}

// backURL 返回来源页
func backURL(c *gin.Context) string {
	if ref := c.Request.Referer(); ref != "" {
		return ref
	}
	return "/"
}
