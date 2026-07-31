package support

import (
	"bytes"
	"strings"

	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/base"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/commonmark"
	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
)

var (
	md = goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithRendererOptions(html.WithUnsafe()),
	)

	// 对应 mews/purifier 的 post_body 规则：允许常见富文本与代码块标签
	sanitizePolicy = buildPolicy()

	htmlToMd = converter.NewConverter(
		converter.WithPlugins(base.NewBasePlugin(), commonmark.NewCommonmarkPlugin()),
	)
)

func buildPolicy() *bluemonday.Policy {
	p := bluemonday.UGCPolicy()
	p.AllowAttrs("class").OnElements("code", "pre", "span", "div", "p")
	p.AllowAttrs("align").OnElements("p", "div", "img")
	p.AllowImages()
	p.AllowTables()
	return p
}

// Parsedown Markdown 转 HTML，等价 helpers.php 的 parsedown()
func Parsedown(markdown string) string {
	var buf bytes.Buffer
	if err := md.Convert([]byte(markdown), &buf); err != nil {
		return markdown
	}
	return buf.String()
}

// Clean HTML 净化，等价 clean($html, 'post_body')
func Clean(rawHTML string) string {
	return sanitizePolicy.Sanitize(rawHTML)
}

// HTMLToMarkdown HTML 转 Markdown，编辑文章时使用
func HTMLToMarkdown(rawHTML string) string {
	out, err := htmlToMd.ConvertString(rawHTML)
	if err != nil {
		return rawHTML
	}
	return strings.TrimSpace(out)
}
