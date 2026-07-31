package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"

	"github.com/qin/qinblog/internal/config"
	"github.com/qin/qinblog/internal/database"
	"github.com/qin/qinblog/internal/middleware"
	"github.com/qin/qinblog/internal/render"
	"github.com/qin/qinblog/internal/router"
	"github.com/qin/qinblog/internal/services"
)

func main() {
	cfg := config.Load()

	if err := database.Init(cfg); err != nil {
		log.Fatalf("database init: %v", err)
	}
	if err := services.Settings.Init(); err != nil {
		log.Fatalf("settings init: %v", err)
	}
	if err := services.Search.Init(); err != nil {
		log.Printf("[warn] search index init: %v（搜索功能不可用，可执行 go run ./cmd/indexer 重建）", err)
	}
	if err := render.Init("web/templates"); err != nil {
		log.Fatalf("templates init: %v", err)
	}

	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.Default()
	router.Setup(r)

	// 定时任务：每日 00:05 同步昨日浏览量（等价 schedule 的 syncYesterdayViewCount）
	c := cron.New()
	if _, err := c.AddFunc("5 0 * * *", func() {
		if err := services.ViewCount.SyncYesterday(); err != nil {
			log.Printf("[warn] sync view count: %v", err)
		}
	}); err != nil {
		log.Printf("[warn] cron: %v", err)
	}
	c.Start()
	defer c.Stop()

	// 包一层方法伪造（表单 _method=PUT/DELETE）
	addr := fmt.Sprintf(":%d", cfg.AppPort)
	log.Printf("QinBlog listening on %s", addr)
	if err := http.ListenAndServe(addr, middleware.MethodOverride(r)); err != nil {
		log.Fatal(err)
	}
}
