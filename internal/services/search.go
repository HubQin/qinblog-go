package services

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/analysis/lang/cjk"
	"github.com/blevesearch/bleve/v2/mapping"
	"github.com/blevesearch/bleve/v2/search/query"

	"github.com/qin/qinblog/internal/config"
	"github.com/qin/qinblog/internal/database"
	"github.com/qin/qinblog/internal/models"
	"github.com/qin/qinblog/internal/support"
)

// Search 全文搜索服务（bleve 替代 TNTSearch，CJK 分词器处理中文）
type searchService struct {
	index bleve.Index
}

var Search = &searchService{}

type postDoc struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

func buildIndexMapping() mapping.IndexMapping {
	textField := bleve.NewTextFieldMapping()
	textField.Analyzer = cjk.AnalyzerName

	doc := bleve.NewDocumentMapping()
	doc.AddFieldMappingsAt("title", textField)
	doc.AddFieldMappingsAt("body", textField)

	m := bleve.NewIndexMapping()
	m.DefaultMapping = doc
	m.DefaultAnalyzer = cjk.AnalyzerName
	return m
}

// Init 打开（不存在则创建）索引
func (s *searchService) Init() error {
	path := config.C.BleveIndexPath
	idx, err := bleve.Open(path)
	if err == bleve.ErrorIndexPathDoesNotExist {
		if mkErr := os.MkdirAll(filepath.Dir(path), 0o755); mkErr != nil {
			return mkErr
		}
		idx, err = bleve.New(path, buildIndexMapping())
	}
	if err != nil {
		return err
	}
	s.index = idx
	return nil
}

// IndexPost 索引/更新一篇文章（Post saved 后调用，等价 Searchable）
func (s *searchService) IndexPost(p *models.Post) {
	if s.index == nil {
		return
	}
	_ = s.index.Index(strconv.FormatUint(uint64(p.ID), 10), postDoc{
		Title: p.Title,
		// body 存的是 markdown 原文，先渲染再去标签得到纯文本
		Body: support.StripTags(support.Parsedown(p.Body)),
	})
}

// DeletePost 从索引中移除文章
func (s *searchService) DeletePost(id uint) {
	if s.index == nil {
		return
	}
	_ = s.index.Delete(strconv.FormatUint(uint64(id), 10))
}

// RebuildAll 全量重建索引（cmd/indexer 使用），返回索引的文章数
func (s *searchService) RebuildAll() (int, error) {
	var posts []models.Post
	if err := database.DB.Find(&posts).Error; err != nil {
		return 0, err
	}
	batch := s.index.NewBatch()
	for i := range posts {
		if err := batch.Index(strconv.FormatUint(uint64(posts[i].ID), 10), postDoc{
			Title: posts[i].Title,
			Body:  support.StripTags(support.Parsedown(posts[i].Body)),
		}); err != nil {
			return 0, err
		}
	}
	if err := s.index.Batch(batch); err != nil {
		return 0, err
	}
	return len(posts), nil
}

// queryIDs 按相关度返回匹配的文章 ID
func (s *searchService) queryIDs(keyword string, limit int) []uint {
	if s.index == nil || strings.TrimSpace(keyword) == "" {
		return nil
	}
	titleQ := bleve.NewMatchQuery(keyword)
	titleQ.SetField("title")
	titleQ.SetBoost(2)
	bodyQ := bleve.NewMatchQuery(keyword)
	bodyQ.SetField("body")

	req := bleve.NewSearchRequestOptions(query.NewDisjunctionQuery([]query.Query{titleQ, bodyQ}), limit, 0, false)
	result, err := s.index.Search(req)
	if err != nil {
		return nil
	}
	ids := make([]uint, 0, len(result.Hits))
	for _, hit := range result.Hits {
		if id, err := strconv.ParseUint(hit.ID, 10, 64); err == nil {
			ids = append(ids, uint(id))
		}
	}
	return ids
}

// SearchPosts 搜索文章：bleve 命中后回 DB 取数据；非管理员仅显示 is_show=1（等价 PostsController::search）
func (s *searchService) SearchPosts(keyword string, isAdmin bool, page, perPage int) ([]models.Post, int64) {
	ids := s.queryIDs(keyword, 1000)
	if len(ids) == 0 {
		return nil, 0
	}

	q := database.DB.Model(&models.Post{}).Where("id IN ?", ids)
	if !isAdmin {
		q = q.Where("is_show = ?", 1)
	}
	var posts []models.Post
	if err := q.Preload("Category").Preload("User").Find(&posts).Error; err != nil {
		return nil, 0
	}

	// 按 bleve 相关度顺序重排
	order := make(map[uint]int, len(ids))
	for i, id := range ids {
		order[id] = i
	}
	sorted := make([]models.Post, len(posts))
	copy(sorted, posts)
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if order[sorted[j].ID] < order[sorted[i].ID] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	total := int64(len(sorted))
	if page < 1 {
		page = 1
	}
	start := (page - 1) * perPage
	if start >= len(sorted) {
		return nil, total
	}
	end := start + perPage
	if end > len(sorted) {
		end = len(sorted)
	}
	return sorted[start:end], total
}

// Highlight 高亮关键词（等价搜索页的 $highlighter->highlight）
func Highlight(text, keyword string) string {
	terms := strings.Fields(strings.TrimSpace(keyword))
	if len(terms) == 0 {
		return text
	}
	// 完整关键词优先高亮
	if !containsAny(terms, keyword) {
		terms = append([]string{keyword}, terms...)
	}
	for _, term := range terms {
		if term == "" {
			continue
		}
		re, err := regexp.Compile("(?i)" + regexp.QuoteMeta(term))
		if err != nil {
			continue
		}
		text = re.ReplaceAllString(text, `<span class="highlight">$0</span>`)
	}
	return text
}

func containsAny(terms []string, s string) bool {
	for _, t := range terms {
		if t == s {
			return true
		}
	}
	return false
}
