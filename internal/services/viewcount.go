package services

import (
	"fmt"
	"time"

	"github.com/qin/qinblog/internal/database"
)

// ViewCount 浏览量统计（对应 ViewCountHelper：Redis 按日 hash 计数 + 定时同步）
type viewCountService struct{}

var ViewCount = &viewCountService{}

const viewCountPrefix = "posts"

func viewCountHashKey(t time.Time) string {
	return fmt.Sprintf("%sview_count:%s", viewCountPrefix, t.Format("2006-01-02"))
}

// Incr 文章浏览量 +1，返回今日缓存中的浏览量
func (s *viewCountService) Incr(postID uint) int64 {
	hash := viewCountHashKey(time.Now())
	field := fmt.Sprintf("%d", postID)
	if err := database.Redis.HIncrBy(database.Ctx, hash, field, 1).Err(); err != nil {
		return 0
	}
	count, _ := database.Redis.HGet(database.Ctx, hash, field).Int64()
	return count
}

// SyncYesterday 将昨日 Redis 中的浏览量累加到 posts.view_count（等价 syncYesterdayViewCount）
func (s *viewCountService) SyncYesterday() error {
	hash := viewCountHashKey(time.Now().AddDate(0, 0, -1))
	data, err := database.Redis.HGetAll(database.Ctx, hash).Result()
	if err != nil {
		return err
	}
	for id, v := range data {
		database.DB.Exec("UPDATE posts SET view_count = view_count + ? WHERE id = ?", v, id)
	}
	return database.Redis.Del(database.Ctx, hash).Err()
}
