package services

import (
	"encoding/json"
	"math/rand"
	"strings"
	"time"

	"github.com/qin/qinblog/internal/database"
	"github.com/qin/qinblog/internal/models"
)

// Sidebar 侧边栏与导航数据（对应 SidebarViewComposer + ColumnObserver 等缓存逻辑）
type sidebarService struct{}

var Sidebar = &sidebarService{}

// ArchiveItem 归档条目
type ArchiveItem struct {
	YearMonth string `json:"year_month"` // 形如 2020-05
	Count     int    `json:"count"`
}

const (
	cacheKeyCategories = "cache:categories"
	cacheKeyTags       = "cache:tags"
	cacheKeyArchives   = "cache:archives"
	cacheKeyLinks      = "cache:links"
	cacheKeyColumns    = "cache:columns"
)

var tagColors = []string{"primary", "secondary", "success", "danger", "info"}

// Categories 有已发布文章的分类
func (s *sidebarService) Categories() []models.Category {
	var categories []models.Category
	if cacheGet(cacheKeyCategories, &categories) {
		return categories
	}
	database.DB.
		Where("id IN (?)", database.DB.Model(&models.Post{}).Where("is_show = ?", 1).Select("category_id")).
		Find(&categories)
	cachePut(cacheKeyCategories, categories)
	return categories
}

// Tags 全部标签（渲染时补充随机颜色）
func (s *sidebarService) Tags() []models.Tag {
	var tags []models.Tag
	if !cacheGet(cacheKeyTags, &tags) {
		database.DB.Find(&tags)
		cachePut(cacheKeyTags, tags)
	}
	for i := range tags {
		tags[i].Color = tagColors[rand.Intn(len(tagColors))]
	}
	return tags
}

// Archives 按年月归档（等价 Post::archiveList 的 groupBy）
func (s *sidebarService) Archives() []ArchiveItem {
	var archives []ArchiveItem
	if cacheGet(cacheKeyArchives, &archives) {
		return archives
	}
	database.DB.Model(&models.Post{}).
		Select("DATE_FORMAT(created_at, '%Y-%m') AS year_month, COUNT(*) AS count").
		Where("is_show = ?", 1).
		Group("year_month").
		Order("year_month DESC").
		Scan(&archives)
	cachePut(cacheKeyArchives, archives)
	return archives
}

// Links 显示中的友链
func (s *sidebarService) Links() []models.LinkModel {
	var links []models.LinkModel
	if cacheGet(cacheKeyLinks, &links) {
		return links
	}
	database.DB.Where("status = ?", 1).Order("sort").Find(&links)
	cachePut(cacheKeyLinks, links)
	return links
}

// Columns 导航栏目
func (s *sidebarService) Columns() []models.Column {
	var columns []models.Column
	if cacheGet(cacheKeyColumns, &columns) {
		return normalizeColumnLinks(columns)
	}
	database.DB.Find(&columns)
	cachePut(cacheKeyColumns, columns)
	return normalizeColumnLinks(columns)
}

// normalizeColumnLinks 站内相对路径统一补前导 /（外链、锚点、绝对路径不动）。
// 否则在 /posts/:id/:slug 等多级 URL 下，相对链接会解析到错误地址被 301 拉回。
func normalizeColumnLinks(columns []models.Column) []models.Column {
	for i := range columns {
		link := columns[i].Link
		if link == "" || strings.HasPrefix(link, "/") || strings.HasPrefix(link, "#") || strings.Contains(link, ":") {
			continue
		}
		columns[i].Link = "/" + link
	}
	return columns
}

// ForgetPostRelated 文章变化后清除相关缓存（等价 PostObserver::forgetPostRelatedCache）
func (s *sidebarService) ForgetPostRelated() {
	database.Redis.Del(database.Ctx, cacheKeyCategories, cacheKeyTags, cacheKeyArchives)
}

// ForgetLinks 友链缓存清除（等价 LinkObserver）
func (s *sidebarService) ForgetLinks() {
	database.Redis.Del(database.Ctx, cacheKeyLinks)
}

// ForgetColumns 栏目缓存清除（等价 ColumnObserver）
func (s *sidebarService) ForgetColumns() {
	database.Redis.Del(database.Ctx, cacheKeyColumns)
}

func cacheGet(key string, dest interface{}) bool {
	data, err := database.Redis.Get(database.Ctx, key).Bytes()
	if err != nil {
		return false
	}
	return json.Unmarshal(data, dest) == nil
}

func cachePut(key string, value interface{}) {
	data, err := json.Marshal(value)
	if err != nil {
		return
	}
	// 与 rememberForever 类似，这里给一个较长的过期时间兜底
	database.Redis.Set(database.Ctx, key, data, 30*24*time.Hour)
}
