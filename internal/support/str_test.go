package support

import (
	"strings"
	"testing"
)

func TestSlug(t *testing.T) {
	cases := map[string]string{
		"Hello World":      "hello-world",
		"  Go_Web 开发 ":     "go-web",
		"Foo--Bar":         "foo-bar",
		"multiple   space": "multiple-space",
		"---":              "",
	}
	for in, want := range cases {
		if got := Slug(in); got != want {
			t.Errorf("Slug(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestStripTags(t *testing.T) {
	if got := StripTags("<p>你好 <b>world</b></p>"); got != "你好 world" {
		t.Errorf("StripTags = %q", got)
	}
}

func TestMakeExcerpt(t *testing.T) {
	got := MakeExcerpt("<p>第一行</p>\n<p>第二行</p>", 200)
	if got != "第一行 第二行" {
		t.Errorf("MakeExcerpt = %q", got)
	}

	long := strings.Repeat("字", 250)
	got = MakeExcerpt("<div>"+long+"</div>", 200)
	if !strings.HasSuffix(got, "...") {
		t.Errorf("超长摘要应以 ... 结尾，got %q", got)
	}
	if len([]rune(got)) != 203 {
		t.Errorf("摘要长度应为 200+3，got %d", len([]rune(got)))
	}
}

func TestStrLimit(t *testing.T) {
	if got := StrLimit("abcdef", 3, "..."); got != "abc..." {
		t.Errorf("StrLimit = %q", got)
	}
	if got := StrLimit("短文本", 10, "..."); got != "短文本" {
		t.Errorf("不超限时应原样返回，got %q", got)
	}
	// 按 rune 而非字节截断
	if got := StrLimit("一二三四五", 2, "..."); got != "一二..." {
		t.Errorf("中文截断错误，got %q", got)
	}
}
