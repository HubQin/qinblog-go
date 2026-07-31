package models

import (
	"database/sql"
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// User 用户（映射 users 表，兼容 Laravel 存量数据）
type User struct {
	ID                uint         `gorm:"primaryKey"`
	Name              string       `gorm:"column:name"`
	Avatar            string       `gorm:"column:avatar"`
	Email             string       `gorm:"column:email"`
	EmailVerifiedAt   sql.NullTime `gorm:"column:email_verified_at"`
	Password          string       `gorm:"column:password"`
	RememberToken     string       `gorm:"column:remember_token"`
	Openid            string       `gorm:"column:openid"`
	Type              string       `gorm:"column:type"`
	IsAdmin           bool         `gorm:"column:is_admin"`
	NotificationCount int          `gorm:"column:notification_count"`
	CreatedAt         time.Time    `gorm:"column:created_at"`
	UpdatedAt         time.Time    `gorm:"column:updated_at"`
}

func (User) TableName() string { return "users" }

// AvatarURL 头像，无头像时使用默认头像
func (u User) AvatarURL() string {
	if u.Avatar != "" {
		return u.Avatar
	}
	return "/images/default_avartar.jpg"
}

// HasVerifiedEmail 邮箱是否已验证
func (u User) HasVerifiedEmail() bool {
	return u.EmailVerifiedAt.Valid
}

// Comment 评论（映射 comments 表，含软删除与多态关联）
type Comment struct {
	ID              uint           `gorm:"primaryKey"`
	UserID          uint           `gorm:"column:user_id"`
	Content         string         `gorm:"column:content"`
	ParentID        *uint          `gorm:"column:parent_id"`
	Approved        bool           `gorm:"column:approved"` // 是否审核通过
	CommentableID   uint           `gorm:"column:commentable_id"`
	CommentableType string         `gorm:"column:commentable_type"`
	CreatedAt       time.Time      `gorm:"column:created_at"`
	UpdatedAt       time.Time      `gorm:"column:updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"column:deleted_at"`

	User    *User     `gorm:"foreignKey:UserID"`
	Replies []Comment `gorm:"foreignKey:ParentID"`
	Post    *Post     `gorm:"foreignKey:CommentableID;references:ID"` // commentable 仅有 App\Post 一种类型
}

func (Comment) TableName() string { return "comments" }

// CommentableTypePost 与 Laravel 多态存量数据保持一致的类型值
const CommentableTypePost = "App\\Post"

// Notification 数据库通知（映射 Laravel notifications 表）
type Notification struct {
	ID             string       `gorm:"column:id;primaryKey;size:36"` // uuid
	Type           string       `gorm:"column:type"`
	NotifiableType string       `gorm:"column:notifiable_type"`
	NotifiableID   uint         `gorm:"column:notifiable_id"`
	Data           string       `gorm:"column:data"`
	ReadAt         sql.NullTime `gorm:"column:read_at"`
	CreatedAt      time.Time    `gorm:"column:created_at"`
	UpdatedAt      time.Time    `gorm:"column:updated_at"`
}

func (Notification) TableName() string { return "notifications" }

// 与 Laravel 存量数据兼容的通知类型
const (
	NotificationTypePostCommented  = "App\\Notifications\\PostCommented"
	NotificationTypeCommentReplied = "App\\Notifications\\CommentReplied"
	NotifiableTypeUser             = "App\\User"
)

// NotificationData 通知 data 字段的 JSON 结构
type NotificationData struct {
	CommentID            uint   `json:"comment_id"`
	CommentContent       string `json:"comment_content"`
	UserID               uint   `json:"user_id"`
	UserName             string `json:"user_name"`
	UserAvatar           string `json:"user_avatar"`
	PostLink             string `json:"post_link"`
	PostID               uint   `json:"post_id"`
	PostTitle            string `json:"post_title,omitempty"`
	ParentCommentContent string `json:"parent_comment_content,omitempty"`
}

// ParsedData 解析通知数据
func (n Notification) ParsedData() NotificationData {
	var d NotificationData
	_ = json.Unmarshal([]byte(n.Data), &d)
	return d
}

// IsPostCommented 是否为文章被评论通知
func (n Notification) IsPostCommented() bool {
	return n.Type == NotificationTypePostCommented
}
