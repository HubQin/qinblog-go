package services

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/qin/qinblog/internal/database"
	"github.com/qin/qinblog/internal/models"
	"github.com/qin/qinblog/internal/support"
)

// Notify 数据库通知服务（对应 PostCommented / CommentReplied 通知 + User::commentNotify）
type notifyService struct{}

var Notify = &notifyService{}

// commentNotify 写入通知并累加未读数；通知对象是评论者本人时跳过（等价 User::commentNotify）
func (s *notifyService) commentNotify(targetUserID, currentUserID uint, ntype string, data models.NotificationData) {
	if targetUserID == 0 || targetUserID == currentUserID {
		return
	}
	raw, _ := json.Marshal(data)
	database.DB.Create(&models.Notification{
		ID:             uuid.NewString(),
		Type:           ntype,
		NotifiableType: models.NotifiableTypeUser,
		NotifiableID:   targetUserID,
		Data:           string(raw),
	})
	database.DB.Model(&models.User{}).Where("id = ?", targetUserID).
		UpdateColumn("notification_count", gorm.Expr("notification_count + 1"))
}

// PostCommented 通知文章作者（等价 CommentObserver::created 中第一段）
func (s *notifyService) PostCommented(comment *models.Comment, post *models.Post, commenter *models.User) {
	s.commentNotify(post.UserID, commenter.ID, models.NotificationTypePostCommented, models.NotificationData{
		CommentID:      comment.ID,
		CommentContent: comment.Content,
		UserID:         commenter.ID,
		UserName:       commenter.Name,
		UserAvatar:     commenter.AvatarURL(),
		PostLink:       post.Link(fmt.Sprintf("#comment%d", comment.ID)),
		PostID:         post.ID,
		PostTitle:      post.Title,
	})
}

// CommentReplied 通知上级评论作者。文章作者与上级评论作者相同时不重复通知（等价 CommentObserver::created）
func (s *notifyService) CommentReplied(reply *models.Comment, parent *models.Comment, post *models.Post, replier *models.User) {
	if parent.UserID == post.UserID {
		return
	}
	s.commentNotify(parent.UserID, replier.ID, models.NotificationTypeCommentReplied, models.NotificationData{
		CommentID:            reply.ID,
		CommentContent:       reply.Content,
		UserID:               replier.ID,
		UserName:             replier.Name,
		UserAvatar:           replier.AvatarURL(),
		PostLink:             post.Link(fmt.Sprintf("#comment%d", reply.ID)),
		PostID:               post.ID,
		ParentCommentContent: support.MakeExcerpt(parent.Content, 30) + "...",
	})
}

// MarkAllRead 全部标记已读并清零未读数（等价 NotificationsController::index 的 markAsRead）
func (s *notifyService) MarkAllRead(userID uint) {
	database.DB.Model(&models.User{}).Where("id = ?", userID).
		UpdateColumn("notification_count", 0)
	database.DB.Model(&models.Notification{}).
		Where("notifiable_type = ? AND notifiable_id = ? AND read_at IS NULL", models.NotifiableTypeUser, userID).
		UpdateColumn("read_at", gorm.Expr("NOW()"))
}
