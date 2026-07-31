package services

import (
	"testing"

	"github.com/blevesearch/bleve/v2"

	"github.com/qin/qinblog/internal/models"
)

func newMemSearch(t *testing.T) *searchService {
	t.Helper()
	idx, err := bleve.NewMemOnly(buildIndexMapping())
	if err != nil {
		t.Fatalf("创建内存索引失败: %v", err)
	}
	return &searchService{index: idx}
}

// bleve 中文（CJK 分词）索引与检索
func TestBleveChineseSearch(t *testing.T) {
	s := newMemSearch(t)
	s.IndexPost(&models.Post{ID: 1, Title: "Go 语言入门教程", Body: "<p>这是一篇关于并发编程的文章</p>"})
	s.IndexPost(&models.Post{ID: 2, Title: "PHP 从入门到放弃", Body: "<p>Laravel 框架实战</p>"})

	ids := s.queryIDs("并发编程", 10)
	if len(ids) != 1 || ids[0] != 1 {
		t.Errorf("中文检索『并发编程』应命中文章 1，got %v", ids)
	}

	ids = s.queryIDs("Laravel", 10)
	if len(ids) != 1 || ids[0] != 2 {
		t.Errorf("英文检索『Laravel』应命中文章 2，got %v", ids)
	}

	// 标题匹配（title boost）
	ids = s.queryIDs("入门", 10)
	if len(ids) != 2 {
		t.Errorf("『入门』应命中两篇文章，got %v", ids)
	}

	// 空关键词
	if got := s.queryIDs("   ", 10); got != nil {
		t.Errorf("空关键词应返回 nil，got %v", got)
	}
}

// 更新与删除后索引应同步
func TestBleveIndexUpdateDelete(t *testing.T) {
	s := newMemSearch(t)
	s.IndexPost(&models.Post{ID: 1, Title: "golang 并发", Body: "旧内容"})
	s.IndexPost(&models.Post{ID: 1, Title: "python 教程", Body: "新内容"}) // 同 ID 覆盖

	if ids := s.queryIDs("golang", 10); len(ids) != 0 {
		t.Errorf("覆盖后旧标题不应命中，got %v", ids)
	}
	if ids := s.queryIDs("python", 10); len(ids) != 1 {
		t.Errorf("新标题应命中，got %v", ids)
	}

	s.DeletePost(1)
	if ids := s.queryIDs("python", 10); len(ids) != 0 {
		t.Errorf("删除后不应命中，got %v", ids)
	}
}

// 高亮：关键词包裹 span.highlight（等价搜索页高亮）
func TestHighlight(t *testing.T) {
	got := Highlight("Go 并发编程实践", "并发编程")
	want := `Go <span class="highlight">并发编程</span>实践`
	if got != want {
		t.Errorf("Highlight = %q, want %q", got, want)
	}
	// 大小写不敏感
	got = Highlight("Laravel and laravel", "laravel")
	if got != `<span class="highlight">Laravel</span> and <span class="highlight">laravel</span>` {
		t.Errorf("大小写不敏感高亮失败: %q", got)
	}
	// 空关键词原样返回
	if got := Highlight("text", "  "); got != "text" {
		t.Errorf("空关键词应原样返回，got %q", got)
	}
}
