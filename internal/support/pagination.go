package support

import (
	"fmt"
	"net/url"

	"gorm.io/gorm"
)

// Pagination 分页数据，供模板渲染（对齐 Laravel 分页器行为）
type Pagination struct {
	Page    int
	PerPage int
	Total   int64
	Path    string     // 当前路径
	Query   url.Values // 需要保留的查询参数（不含 page）
}

// PageLink 分页链接项
type PageLink struct {
	Page     int
	URL      string
	Active   bool
	Disabled bool
	Label    string
}

// LastPage 总页数
func (p *Pagination) LastPage() int {
	if p.PerPage <= 0 {
		return 1
	}
	last := int((p.Total + int64(p.PerPage) - 1) / int64(p.PerPage))
	if last < 1 {
		last = 1
	}
	return last
}

// HasPages 是否需要显示分页
func (p *Pagination) HasPages() bool { return p.LastPage() > 1 }

// HasMore 是否有下一页
func (p *Pagination) HasMore() bool { return p.Page < p.LastPage() }

// OnFirstPage 是否第一页
func (p *Pagination) OnFirstPage() bool { return p.Page <= 1 }

// URL 生成某页链接，保留既有查询参数
func (p *Pagination) URL(page int) string {
	q := url.Values{}
	for k, vs := range p.Query {
		if k == "page" {
			continue
		}
		for _, v := range vs {
			q.Add(k, v)
		}
	}
	q.Set("page", fmt.Sprintf("%d", page))
	return p.Path + "?" + q.Encode()
}

// PrevURL 上一页链接
func (p *Pagination) PrevURL() string { return p.URL(p.Page - 1) }

// NextURL 下一页链接
func (p *Pagination) NextURL() string { return p.URL(p.Page + 1) }

// Links 生成分页窗口（首尾各 2 页 + 当前页前后 3 页，多余部分折叠为省略号）
func (p *Pagination) Links() []PageLink {
	last := p.LastPage()
	var links []PageLink

	appendPage := func(i int) {
		links = append(links, PageLink{
			Page:   i,
			URL:    p.URL(i),
			Active: i == p.Page,
			Label:  fmt.Sprintf("%d", i),
		})
	}

	if last <= 13 {
		for i := 1; i <= last; i++ {
			appendPage(i)
		}
		return links
	}

	window := 3
	start := p.Page - window
	end := p.Page + window
	if start < 1 {
		start = 1
	}
	if end > last {
		end = last
	}

	if start > 1 {
		appendPage(1)
		appendPage(2)
		if start > 3 {
			links = append(links, PageLink{Disabled: true, Label: "..."})
		}
	}
	for i := start; i <= end; i++ {
		if i == 2 && start > 1 || i == 1 {
			continue
		}
		appendPage(i)
	}
	if end < last {
		if end < last-2 {
			links = append(links, PageLink{Disabled: true, Label: "..."})
		}
		appendPage(last - 1)
		appendPage(last)
	}
	return links
}

// NewPagination 构建分页对象
func NewPagination(page, perPage int, total int64, path string, query url.Values) *Pagination {
	if page < 1 {
		page = 1
	}
	return &Pagination{Page: page, PerPage: perPage, Total: total, Path: path, Query: query}
}

// Paginate 对 GORM 查询执行 count + 分页查询，结果写入 dest
func Paginate(query *gorm.DB, page, perPage int, dest interface{}, path string, urlQuery url.Values) (*Pagination, error) {
	if page < 1 {
		page = 1
	}
	var total int64
	countQuery := query.Session(&gorm.Session{})
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, err
	}
	if err := query.Offset((page - 1) * perPage).Limit(perPage).Find(dest).Error; err != nil {
		return nil, err
	}
	return NewPagination(page, perPage, total, path, urlQuery), nil
}
