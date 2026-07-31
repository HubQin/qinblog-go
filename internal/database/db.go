package database

import (
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"

	"github.com/qin/qinblog/internal/config"
)

var (
	DB    *gorm.DB
	Redis *redis.Client
	Ctx   = context.Background()
)

// Init 初始化 MySQL 与 Redis 连接
func Init(cfg *config.Config) error {
	logLevel := logger.Warn
	if cfg.AppEnv == "local" {
		logLevel = logger.Info
	}

	db, err := gorm.Open(mysql.Open(cfg.DSN()), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
		NamingStrategy: schema.NamingStrategy{
			SingularTable: false, // 与 Laravel 一致使用复数表名
		},
	})
	if err != nil {
		return err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	sqlDB.SetMaxOpenConns(50)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(time.Hour)

	DB = db

	Redis = redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr(),
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	if err := Redis.Ping(Ctx).Err(); err != nil {
		// Redis 不可用时仅告警：浏览量与缓存功能会降级，站点仍可运行
		log.Printf("[warn] redis unavailable: %v", err)
	}

	return nil
}
