package services

import (
	"testing"
	"time"

	"github.com/qin/qinblog/internal/models"
	"github.com/qin/qinblog/internal/render"
)

// TimeAgo 中文相对时间格式化
func TestTimeAgo(t *testing.T) {
	cases := []struct {
		name string
		t    time.Time
		want string
	}{
		{"零值", time.Time{}, ""},
		{"刚刚", time.Now().Add(-10 * time.Second), "刚刚"},
		{"分钟前", time.Now().Add(-5 * time.Minute), "5分钟前"},
		{"小时前", time.Now().Add(-3 * time.Hour), "3小时前"},
		{"天前", time.Now().Add(-48 * time.Hour), "2天前"},
		{"个月前", time.Now().Add(-60 * 24 * time.Hour), "2个月前"},
		{"年前", time.Now().Add(-400 * 24 * time.Hour), "1年前"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := render.TimeAgo(c.t)
			if got != c.want {
				t.Errorf("TimeAgo(%v) = %q, want %q", c.t, got, c.want)
			}
		})
	}
}

// dict 模板函数：成对参数转 map
func TestDictFunc(t *testing.T) {
	// 通过 FuncMap 获取 dict 函数
	fm := render.FuncMap()
	dictFn, ok := fm["dict"]
	if !ok {
		t.Fatal("FuncMap 中缺少 dict 函数")
	}
	fn := dictFn.(func(...interface{}) (map[string]interface{}, error))

	// 正常成对
	m, err := fn("a", 1, "b", "hello")
	if err != nil {
		t.Fatalf("dict: %v", err)
	}
	if m["a"] != 1 || m["b"] != "hello" {
		t.Errorf("dict = %v, want {a:1 b:hello}", m)
	}

	// 奇数参数应报错
	_, err = fn("a", 1, "b")
	if err == nil {
		t.Error("奇数参数应返回错误")
	}
}

// Post.Link 链接生成：有/无 slug
func TestPostLink(t *testing.T) {
	p := models.Post{ID: 42, Slug: "hello-world"}
	got := p.Link()
	if got != "/posts/42/hello-world" {
		t.Errorf("Link() = %q, want /posts/42/hello-world", got)
	}

	// 带锚点
	got = p.Link("#comments")
	if got != "/posts/42/hello-world#comments" {
		t.Errorf("Link(#comments) = %q", got)
	}

	// 无 slug
	p2 := models.Post{ID: 7}
	got = p2.Link()
	if got != "/posts/7" {
		t.Errorf("无 slug Link() = %q, want /posts/7", got)
	}
}

// User.AvatarURL 有/无头像
func TestUserAvatarURL(t *testing.T) {
	u := models.User{Avatar: "https://example.com/pic.jpg"}
	if got := u.AvatarURL(); got != "https://example.com/pic.jpg" {
		t.Errorf("AvatarURL = %q", got)
	}
	u2 := models.User{}
	if got := u2.AvatarURL(); got != "/images/default_avartar.jpg" {
		t.Errorf("默认头像 = %q", got)
	}
}

// User.HasVerifiedEmail
func TestUserHasVerifiedEmail(t *testing.T) {
	u := models.User{}
	if u.HasVerifiedEmail() {
		t.Error("未设置验证时间应返回 false")
	}
	u.EmailVerifiedAt.Time = time.Now()
	u.EmailVerifiedAt.Valid = true
	if !u.HasVerifiedEmail() {
		t.Error("已设置验证时间应返回 true")
	}
}

// Comment 多态类型常量
func TestCommentableType(t *testing.T) {
	if models.CommentableTypePost != "App\\Post" {
		t.Errorf("CommentableTypePost = %q, want App\\Post", models.CommentableTypePost)
	}
}

// Notification 类型常量
func TestNotificationTypes(t *testing.T) {
	if models.NotificationTypePostCommented != "App\\Notifications\\PostCommented" {
		t.Errorf("类型常量不匹配")
	}
	if models.NotificationTypeCommentReplied != "App\\Notifications\\CommentReplied" {
		t.Errorf("类型常量不匹配")
	}
}

// Notification.ParsedData 解析 JSON
func TestNotificationParsedData(t *testing.T) {
	n := models.Notification{
		Data: `{"comment_id":5,"user_name":"Alice","post_id":10,"post_title":"Test"}`,
	}
	d := n.ParsedData()
	if d.CommentID != 5 || d.UserName != "Alice" || d.PostID != 10 || d.PostTitle != "Test" {
		t.Errorf("ParsedData = %+v", d)
	}
}

// Notification.IsPostCommented
func TestNotificationIsPostCommented(t *testing.T) {
	n := models.Notification{Type: models.NotificationTypePostCommented}
	if !n.IsPostCommented() {
		t.Error("应返回 true")
	}
	n2 := models.Notification{Type: models.NotificationTypeCommentReplied}
	if n2.IsPostCommented() {
		t.Error("应返回 false")
	}
}
