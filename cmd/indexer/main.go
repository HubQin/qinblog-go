package main

import (
	"log"

	"github.com/qin/qinblog/internal/config"
	"github.com/qin/qinblog/internal/database"
	"github.com/qin/qinblog/internal/services"
)

// bleve 全量索引重建命令（首次上线或索引损坏时执行）
func main() {
	cfg := config.Load()
	if err := database.Init(cfg); err != nil {
		log.Fatalf("database init: %v", err)
	}
	if err := services.Search.Init(); err != nil {
		log.Fatalf("search index init: %v", err)
	}
	n, err := services.Search.RebuildAll()
	if err != nil {
		log.Fatalf("rebuild index: %v", err)
	}
	log.Printf("索引重建完成，共 %d 篇文章", n)
}
