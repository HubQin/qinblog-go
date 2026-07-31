package models

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// Post 文章（映射 posts 表）
type Post struct {
	ID           uint      `gorm:"primaryKey"`
	Title        string    `gorm:"column:title"`
	Body         string    `gorm:"column:body"`
	UserID       uint      `gorm:"column:user_id"`
	CategoryID   uint      `gorm:"column:category_id"`
	TopicID      uint      `gorm:"column:topic_id"`      // 所属专题,0代表无专题
	Sort         int       `gorm:"column:sort"`          // 用于专题排序
	CommentCount int       `gorm:"column:comment_count"` // 评论数量
	ViewCount    int       `gorm:"column:view_count"`    // 查看总数
	Order        int       `gorm:"column:order"`         // 排序
	IsShow       int8      `gorm:"column:is_show"`       // 是否显示
	Excerpt      string    `gorm:"column:excerpt"`       // 文章摘要，SEO 优化时使用
	Slug         string    `gorm:"column:slug"`          // SEO 友好的 URI
	CreatedAt    time.Time `gorm:"column:created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at"`

	User     *User    `gorm:"foreignKey:UserID"`
	Category Category `gorm:"foreignKey:CategoryID"`
	Topic    *Topic   `gorm:"foreignKey:TopicID"`
	Tags     []Tag    `gorm:"many2many:post_tag;joinForeignKey:post_id;joinReferences:tag_id"`
}

func (Post) TableName() string { return "posts" }

// Link 文章详情页链接，等价 Laravel 的 $post->link()（值接收者，便于模板中 range 元素调用）
func (p Post) Link(anchor ...string) string {
	link := fmt.Sprintf("/posts/%d", p.ID)
	if p.Slug != "" {
		link += "/" + p.Slug
	}
	if len(anchor) > 0 {
		link += anchor[0]
	}
	return link
}

// ScopePublished 已发布的文章
func ScopePublished(db *gorm.DB) *gorm.DB {
	return db.Where("is_show = ?", 1)
}

// ScopeRecently 按创建时间倒序
func ScopeRecently(db *gorm.DB) *gorm.DB {
	return db.Order("created_at DESC")
}

// Tag 标签
type Tag struct {
	ID          uint      `gorm:"primaryKey"`
	Name        string    `gorm:"column:name"`
	Description string    `gorm:"column:description"`
	PostCount   int       `gorm:"column:post_count"` // 标签下的文章总数
	CreatedAt   time.Time `gorm:"column:created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at"`

	// 侧边栏展示用的随机颜色，不入库
	Color string `gorm:"-"`
}

func (Tag) TableName() string { return "tags" }

// Topic 专题
type Topic struct {
	ID          uint      `gorm:"primaryKey"`
	Name        string    `gorm:"column:name"`
	Description string    `gorm:"column:description"`
	PostCount   int       `gorm:"column:post_count"`
	CreatedAt   time.Time `gorm:"column:created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at"`

	Posts []Post `gorm:"foreignKey:TopicID"`
}

func (Topic) TableName() string { return "topics" }

// Category 分类
type Category struct {
	ID          uint   `gorm:"primaryKey"`
	Name        string `gorm:"column:name"`
	Icon        string `gorm:"column:icon"`
	Description string `gorm:"column:description"`
	PostCount   int    `gorm:"column:post_count"` // 分类下的文章总数
}

func (Category) TableName() string { return "categories" }

// Column 导航栏目
type Column struct {
	ID          uint   `gorm:"primaryKey"`
	Name        string `gorm:"column:name"`
	Link        string `gorm:"column:link"`
	Description string `gorm:"column:description"`
}

func (Column) TableName() string { return "columns" }

// LinkModel 友情链接（links 表）
type LinkModel struct {
	ID        uint      `gorm:"primaryKey"`
	Name      string    `gorm:"column:name"`
	URL       string    `gorm:"column:url"`
	Logo      string    `gorm:"column:logo"`
	Sort      int       `gorm:"column:sort"`
	Status    int8      `gorm:"column:status"` // 是否显示
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (LinkModel) TableName() string { return "links" }

// Setting 站点设置（key-value，替代 laravel-admin 写 config/site.php 的方案）
type Setting struct {
	Key   string `gorm:"column:key;primaryKey;size:64"`
	Value string `gorm:"column:value;type:text"`
}

func (Setting) TableName() string { return "settings" }
