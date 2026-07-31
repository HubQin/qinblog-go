package support

import (
	"crypto/rand"
	"math/big"
	"regexp"
	"strings"
	"unicode/utf8"
)

var (
	slugInvalidRe  = regexp.MustCompile(`[^a-z0-9\-_]+`)
	slugDashRe     = regexp.MustCompile(`-{2,}`)
	excerptSpaceRe = regexp.MustCompile(`\r\n|\r|\n+`)
	stripTagRe     = regexp.MustCompile(`(?s)<[^>]*>`)
)

// Slug 等价 Laravel 的 Str::slug：小写、空白与非法字符转 '-'
func Slug(text string) string {
	s := strings.ToLower(strings.TrimSpace(text))
	s = strings.ReplaceAll(s, "_", "-")
	s = strings.Join(strings.Fields(s), "-")
	s = slugInvalidRe.ReplaceAllString(s, "-")
	s = slugDashRe.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

// StripTags 去除 HTML 标签
func StripTags(html string) string {
	return stripTagRe.ReplaceAllString(html, "")
}

// MakeExcerpt 等价 helpers.php 的 make_excerpt：去标签、合并换行、按字符数截断
func MakeExcerpt(value string, length int) string {
	if length <= 0 {
		length = 200
	}
	excerpt := strings.TrimSpace(excerptSpaceRe.ReplaceAllString(StripTags(value), " "))
	return StrLimit(excerpt, length, "...")
}

// StrLimit 等价 Str::limit，按 rune 截断
func StrLimit(s string, limit int, end string) string {
	if utf8.RuneCountInString(s) <= limit {
		return s
	}
	runes := []rune(s)
	return strings.TrimSpace(string(runes[:limit])) + end
}

const randomChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// RandomString 生成随机字符串
func RandomString(n int) string {
	b := make([]byte, n)
	max := big.NewInt(int64(len(randomChars)))
	for i := range b {
		idx, _ := rand.Int(rand.Reader, max)
		b[i] = randomChars[idx.Int64()]
	}
	return string(b)
}
