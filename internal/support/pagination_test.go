package support

import (
	"net/url"
	"testing"
)

// Pagination 基本计算
func TestPaginationBasic(t *testing.T) {
	p := NewPagination(1, 20, 100, "/posts", url.Values{})
	if p.LastPage() != 5 {
		t.Errorf("LastPage = %d, want 5", p.LastPage())
	}
	if !p.OnFirstPage() {
		t.Error("应在第一页")
	}
	if !p.HasMore() {
		t.Error("应有更多页")
	}
	if !p.HasPages() {
		t.Error("应显示分页")
	}
}

// Pagination 单页
func TestPaginationSinglePage(t *testing.T) {
	p := NewPagination(1, 20, 10, "/posts", url.Values{})
	if p.LastPage() != 1 {
		t.Errorf("LastPage = %d, want 1", p.LastPage())
	}
	if p.HasMore() {
		t.Error("不应有更多页")
	}
	if p.HasPages() {
		t.Error("不应显示分页")
	}
}

// Pagination 链接生成
func TestPaginationURL(t *testing.T) {
	q := url.Values{}
	q.Set("keyword", "go")
	p := NewPagination(2, 10, 50, "/posts", q)
	got := p.URL(3)
	if got == "" {
		t.Error("URL 不应为空")
	}
	// 应保留 keyword 参数
	if !containsStr(got, "keyword=go") {
		t.Errorf("URL 应保留 keyword 参数: %s", got)
	}
	if !containsStr(got, "page=3") {
		t.Errorf("URL 应包含 page=3: %s", got)
	}
}

// Pagination Links 生成
func TestPaginationLinks(t *testing.T) {
	// 少量页码：全部展示
	p := NewPagination(1, 10, 50, "/posts", url.Values{})
	links := p.Links()
	if len(links) != 5 {
		t.Errorf("5 页应有 5 个链接，got %d", len(links))
	}
	// 第一页应 active
	if !links[0].Active {
		t.Error("第一页应为 active")
	}

	// 大量页码：部分展示+省略号
	p2 := NewPagination(10, 10, 500, "/posts", url.Values{})
	links2 := p2.Links()
	hasDisabled := false
	for _, l := range links2 {
		if l.Disabled {
			hasDisabled = true
			break
		}
	}
	if !hasDisabled {
		t.Error("大量页码应有省略号")
	}
}

// Pagination PrevURL/NextURL
func TestPaginationPrevNext(t *testing.T) {
	p := NewPagination(3, 10, 100, "/posts", url.Values{})
	if !containsStr(p.PrevURL(), "page=2") {
		t.Errorf("PrevURL 应包含 page=2: %s", p.PrevURL())
	}
	if !containsStr(p.NextURL(), "page=4") {
		t.Errorf("NextURL 应包含 page=4: %s", p.NextURL())
	}
}

// page < 1 时修正为 1
func TestPaginationPageCorrection(t *testing.T) {
	p := NewPagination(-1, 10, 50, "/posts", url.Values{})
	if p.Page != 1 {
		t.Errorf("负数页应修正为 1，got %d", p.Page)
	}
	p2 := NewPagination(0, 10, 50, "/posts", url.Values{})
	if p2.Page != 1 {
		t.Errorf("零页应修正为 1，got %d", p2.Page)
	}
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
