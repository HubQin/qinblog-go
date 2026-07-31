package services

import (
	"time"

	"gorm.io/gorm"

	"github.com/qin/qinblog/internal/database"
	"github.com/qin/qinblog/internal/models"
	"github.com/qin/qinblog/internal/support"
)

// Comments 评论服务（对应 CommentsController + CommentObserver）
type commentService struct{}

var Comments = &commentService{}

// Create 创建评论/回复，返回评论与所属文章
func (s *commentService) Create(user *models.User, postID uint, content string, parentID *uint) (*models.Comment, *models.Post, error) {
	var post models.Post
	if err := database.DB.First(&post, postID).Error; err != nil {
		return nil, nil, err
	}

	// creating：markdown 渲染 + 净化（等价 CommentObserver::creating）
	html := support.Clean(support.Parsedown(content))

	c := &models.Comment{
		UserID:          user.ID,
		Content:         html,
		ParentID:        parentID,
		Approved:        user.ID == 1, // 站长评论自动审核通过（等价 CommentsController::store）
		CommentableID:   post.ID,
		CommentableType: models.CommentableTypePost,
	}
	if err := database.DB.Create(c).Error; err != nil {
		return nil, nil, err
	}

	// created：刷新文章评论数 + 通知（等价 CommentObserver::created）
	s.updatePostCommentCount(post.ID)
	Notify.PostCommented(c, &post, user)
	if parentID != nil {
		var parent models.Comment
		if err := database.DB.First(&parent, *parentID).Error; err == nil {
			Notify.CommentReplied(c, &parent, &post, user)
		}
	}
	return c, &post, nil
}

// Delete 软删除评论，并递归软删所有下级回复（等价 CommentObserver::deleted）
func (s *commentService) Delete(comment *models.Comment) error {
	if err := database.DB.Delete(comment).Error; err != nil {
		return err
	}

	// 收集所有下级评论 id 并标记删除
	ids := s.childIDs([]uint{comment.ID})
	if len(ids) > 0 {
		database.DB.Model(&models.Comment{}).Where("id IN ?", ids).
			UpdateColumn("deleted_at", time.Now())
	}

	s.updatePostCommentCount(comment.CommentableID)
	return nil
}

// childIDs 递归获取下级评论 id
func (s *commentService) childIDs(parentIDs []uint) []uint {
	var all []uint
	current := parentIDs
	for len(current) > 0 {
		var ids []uint
		database.DB.Model(&models.Comment{}).Where("parent_id IN ?", current).Pluck("id", &ids)
		if len(ids) == 0 {
			break
		}
		all = append(all, ids...)
		current = ids
	}
	return all
}

// updatePostCommentCount 重新统计文章的有效评论数
func (s *commentService) updatePostCommentCount(postID uint) {
	database.DB.Model(&models.Post{}).Where("id = ?", postID).UpdateColumn("comment_count",
		gorm.Expr("(SELECT COUNT(*) FROM comments WHERE commentable_id = ? AND commentable_type = ? AND deleted_at IS NULL)",
			postID, models.CommentableTypePost))
}

// ApprovedTree 加载文章的评论树（顶级评论 + 递归回复，等价视图的 $post->comments）
func (s *commentService) ApprovedTree(postID uint) []models.Comment {
	var all []models.Comment
	database.DB.Preload("User").
		Where("commentable_id = ? AND commentable_type = ?", postID, models.CommentableTypePost).
		Order("created_at ASC").
		Find(&all)

	byParent := make(map[uint][]models.Comment)
	var roots []models.Comment
	for _, c := range all {
		if c.ParentID == nil {
			roots = append(roots, c)
		} else {
			byParent[*c.ParentID] = append(byParent[*c.ParentID], c)
		}
	}
	var attach func(list []models.Comment) []models.Comment
	attach = func(list []models.Comment) []models.Comment {
		for i := range list {
			list[i].Replies = attach(byParent[list[i].ID])
		}
		return list
	}
	return attach(roots)
}
