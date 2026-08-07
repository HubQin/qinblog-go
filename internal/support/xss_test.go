package support

import (
	"strings"
	"testing"
)

// XSS 防护：script 标签应被净化移除
func TestXSSScriptTag(t *testing.T) {
	inputs := []string{
		"<script>alert('xss')</script>",
		"<script src='evil.js'></script>",
		"<img src=x onerror=alert(1)>",
		"<div onclick=alert(1)>click</div>",
		"<a href='javascript:alert(1)'>link</a>",
		"<iframe src='evil.html'></iframe>",
	}
	for _, input := range inputs {
		got := Clean(Parsedown(input))
		if strings.Contains(got, "<script") {
			t.Errorf("script 标签未被净化: input=%q, got=%q", input, got)
		}
		if strings.Contains(got, "onerror=") || strings.Contains(got, "onclick=") {
			t.Errorf("事件处理器未被移除: input=%q, got=%q", input, got)
		}
		if strings.Contains(got, "javascript:") {
			t.Errorf("javascript: 协议未被移除: input=%q, got=%q", input, got)
		}
		if strings.Contains(got, "<iframe") {
			t.Errorf("iframe 标签未被净化: input=%q, got=%q", input, got)
		}
	}
}

// 安全 HTML 标签应被保留
func TestCleanPreservesSafeTags(t *testing.T) {
	safe := []struct {
		input string
		must  string
	}{
		{"<p>段落</p>", "<p>"},
		{"<strong>粗体</strong>", "<strong>"},
		{"<em>斜体</em>", "<em>"},
		{"<code>代码</code>", "<code>"},
		{`<a href="https://example.com">链接</a>`, "<a"},
		{"<img src='photo.jpg' alt='图片'>", "<img"},
	}
	for _, c := range safe {
		got := Clean(c.input)
		if !strings.Contains(got, c.must) {
			t.Errorf("安全标签 %q 应被保留，got=%q", c.must, got)
		}
	}
}

// 代码块 class 属性应被保留（语法高亮需要）
func TestCleanPreservesCodeClass(t *testing.T) {
	input := `<pre><code class="language-go">fmt.Println("hi")</code></pre>`
	got := Clean(input)
	if !strings.Contains(got, "language-go") {
		t.Errorf("code class 应被保留: got=%q", got)
	}
}

// 表格标签应被保留
func TestCleanPreservesTable(t *testing.T) {
	input := `<table><tr><td>数据</td></tr></table>`
	got := Clean(input)
	if !strings.Contains(got, "<table>") || !strings.Contains(got, "<td>") {
		t.Errorf("表格标签应被保留: got=%q", got)
	}
}

// Markdown 渲染后净化：完整流程
func TestMarkdownXSSPipeline(t *testing.T) {
	// 模拟文章正文渲染流程：Parsedown → Clean
	markdown := "# 标题\n\n正常内容\n\n<script>alert('xss')</script>\n\n**粗体**"
	html := Clean(Parsedown(markdown))
	if strings.Contains(html, "<script") {
		t.Errorf("Markdown XSS 管道中 script 未被净化: %q", html)
	}
	if !strings.Contains(html, "<h1") {
		t.Errorf("标题应被渲染: %q", html)
	}
	if !strings.Contains(html, "<strong>") {
		t.Errorf("粗体应被渲染: %q", html)
	}
}

// 评论渲染流程：Markdown → Clean（评论内容入库时处理）
func TestCommentRenderPipeline(t *testing.T) {
	content := "这是一条**好**评论\n\n<script>bad()</script>"
	html := Clean(Parsedown(content))
	if strings.Contains(html, "<script") {
		t.Errorf("评论 XSS 未被净化: %q", html)
	}
	if !strings.Contains(html, "<strong>") {
		t.Errorf("评论 Markdown 应被渲染: %q", html)
	}
}
