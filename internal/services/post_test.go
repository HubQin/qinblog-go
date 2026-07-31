package services

import (
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/qin/qinblog/internal/database"
	"github.com/qin/qinblog/internal/models"
)

func newMockDB(t *testing.T) (sqlmock.Sqlmock, *gorm.DB) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	gdb, err := gorm.Open(mysql.New(mysql.Config{Conn: db, SkipInitializeWithVersion: true}),
		&gorm.Config{SkipDefaultTransaction: true})
	if err != nil {
		t.Fatalf("gorm open: %v", err)
	}
	return mock, gdb
}

// 唯一 slug：RLIKE 查重，重名时追加 -N（等价 PostService::creatUniqueSlug）
func TestCreateUniqueSlug(t *testing.T) {
	mock, gdb := newMockDB(t)
	old := database.DB
	database.DB = gdb
	t.Cleanup(func() { database.DB = old })

	// 无重名 → 原样
	mock.ExpectQuery(`SELECT count\(\*\) FROM .posts.`).
		WithArgs("^hello-world(-[0-9]+)?$", 0).
		WillReturnRows(sqlmock.NewRows([]string{"count(*)"}).AddRow(0))
	post := &models.Post{}
	Posts.CreateUniqueSlug(post, "Hello World")
	if post.Slug != "hello-world" {
		t.Errorf("slug = %q, want hello-world", post.Slug)
	}

	// 已有 2 个同名/带序号 → hello-world-2
	mock.ExpectQuery(`SELECT count\(\*\) FROM .posts.`).
		WithArgs("^hello-world(-[0-9]+)?$", 0).
		WillReturnRows(sqlmock.NewRows([]string{"count(*)"}).AddRow(2))
	post2 := &models.Post{}
	Posts.CreateUniqueSlug(post2, "Hello World")
	if post2.Slug != "hello-world-2" {
		t.Errorf("slug = %q, want hello-world-2", post2.Slug)
	}

	// 纯中文标题 slug 为空 → 不设置（等转异步翻译）
	post3 := &models.Post{}
	Posts.CreateUniqueSlug(post3, "你好世界")
	if post3.Slug != "" {
		t.Errorf("中文标题不应直接生成 slug，got %q", post3.Slug)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("SQL 预期未满足: %v", err)
	}
}

// tag_ids JSON 解析：数字为已有标签，"name~random" 新建（等价 PostService::handleTag）
func TestHandleTag(t *testing.T) {
	mock, gdb := newMockDB(t)

	// 已有标签：数字与数字字符串混合
	ids, err := Posts.handleTag(gdb, `[1,"2",3]`)
	if err != nil {
		t.Fatalf("handleTag: %v", err)
	}
	if len(ids) != 3 || ids[0] != 1 || ids[1] != 2 || ids[2] != 3 {
		t.Errorf("ids = %v, want [1 2 3]", ids)
	}

	// 新标签："golang~xxx" → INSERT tags，返回新 id
	mock.ExpectExec("INSERT INTO .tags.").
		WillReturnResult(sqlmock.NewResult(9, 1))
	ids, err = Posts.handleTag(gdb, `[5,"golang~abc123"]`)
	if err != nil {
		t.Fatalf("handleTag: %v", err)
	}
	if len(ids) != 2 || ids[0] != 5 || ids[1] != 9 {
		t.Errorf("ids = %v, want [5 9]", ids)
	}

	// 空串与非法 JSON → 空
	if ids, _ := Posts.handleTag(gdb, ""); ids != nil {
		t.Errorf("空串应返回 nil，got %v", ids)
	}
	if ids, _ := Posts.handleTag(gdb, "not-json"); ids != nil {
		t.Errorf("非法 JSON 应返回 nil，got %v", ids)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("SQL 预期未满足: %v", err)
	}
}

// 专题处理：空/数字/新建（等价 PostService::handleTopic）
func TestHandleTopic(t *testing.T) {
	mock, gdb := newMockDB(t)

	// 空 → 置 0
	post := &models.Post{TopicID: 5}
	if err := Posts.handleTopic(gdb, PostInput{TopicID: ""}, post); err != nil {
		t.Fatalf("handleTopic: %v", err)
	}
	if post.TopicID != 0 {
		t.Errorf("TopicID = %d, want 0", post.TopicID)
	}

	// 已有专题 id
	post = &models.Post{}
	if err := Posts.handleTopic(gdb, PostInput{TopicID: "3", Sort: 2}, post); err != nil {
		t.Fatalf("handleTopic: %v", err)
	}
	if post.TopicID != 3 || post.Sort != 2 {
		t.Errorf("TopicID/Sort = %d/%d, want 3/2", post.TopicID, post.Sort)
	}

	// "名称~随机" → 新建专题，sort 缺省 1
	mock.ExpectExec("INSERT INTO .topics.").
		WillReturnResult(sqlmock.NewResult(7, 1))
	post = &models.Post{}
	if err := Posts.handleTopic(gdb, PostInput{TopicID: "Go 进阶~xyz"}, post); err != nil {
		t.Fatalf("handleTopic: %v", err)
	}
	if post.TopicID != 7 || post.Sort != 1 {
		t.Errorf("TopicID/Sort = %d/%d, want 7/1", post.TopicID, post.Sort)
	}

	// 非法 id → 报错
	if err := Posts.handleTopic(gdb, PostInput{TopicID: "abc"}, &models.Post{}); err == nil {
		t.Error("非法专题 id 应返回错误")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("SQL 预期未满足: %v", err)
	}
}

func TestUniqueUints(t *testing.T) {
	got := uniqueUints([]uint{1, 2, 2, 3, 1})
	if len(got) != 3 || got[0] != 1 || got[1] != 2 || got[2] != 3 {
		t.Errorf("uniqueUints = %v, want [1 2 3]", got)
	}
}
