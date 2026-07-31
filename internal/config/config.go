package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config 应用配置，从 .env 加载（键名与 Laravel 项目保持一致）
type Config struct {
	AppName    string
	AppEnv     string
	AppKey     string
	AppURL     string
	AppPort    int
	ViteDev    bool
	ViteDevURL string

	DBHost     string
	DBPort     int
	DBDatabase string
	DBUsername string
	DBPassword string

	RedisHost     string
	RedisPort     int
	RedisPassword string
	RedisDB       int

	MailHost        string
	MailPort        int
	MailUsername    string
	MailPassword    string
	MailFromAddress string
	MailFromName    string

	GithubClientID     string
	GithubClientSecret string
	GithubRedirectURL  string

	BaiduTranslateAppID string
	BaiduTranslateKey   string

	BleveIndexPath string
	UploadPath     string
}

var C *Config

// Load 加载 .env（不存在时仅使用系统环境变量）
func Load(envFiles ...string) *Config {
	_ = godotenv.Load(envFiles...)

	C = &Config{
		AppName:    env("APP_NAME", "QinBlog"),
		AppEnv:     env("APP_ENV", "production"),
		AppKey:     env("APP_KEY", "qinblog-insecure-key"),
		AppURL:     env("APP_URL", "http://localhost:8080"),
		AppPort:    envInt("APP_PORT", 8080),
		ViteDev:    envBool("VITE_DEV", false),
		ViteDevURL: env("VITE_DEV_URL", "http://localhost:5173"),

		DBHost:     env("DB_HOST", "127.0.0.1"),
		DBPort:     envInt("DB_PORT", 3306),
		DBDatabase: env("DB_DATABASE", "qinblog"),
		DBUsername: env("DB_USERNAME", "root"),
		DBPassword: env("DB_PASSWORD", ""),

		RedisHost:     env("REDIS_HOST", "127.0.0.1"),
		RedisPort:     envInt("REDIS_PORT", 6379),
		RedisPassword: env("REDIS_PASSWORD", ""),
		RedisDB:       envInt("REDIS_DB", 0),

		MailHost:        env("MAIL_HOST", ""),
		MailPort:        envInt("MAIL_PORT", 25),
		MailUsername:    env("MAIL_USERNAME", ""),
		MailPassword:    env("MAIL_PASSWORD", ""),
		MailFromAddress: env("MAIL_FROM_ADDRESS", "hello@example.com"),
		MailFromName:    env("MAIL_FROM_NAME", "QinBlog"),

		GithubClientID:     env("GITHUB_CLIENT_ID", ""),
		GithubClientSecret: env("GITHUB_CLIENT_SECRET", ""),
		GithubRedirectURL:  env("GITHUB_REDIRECT_URL", ""),

		BaiduTranslateAppID: env("BAIDU_TRANSLATE_APPID", ""),
		BaiduTranslateKey:   env("BAIDU_TRANSLATE_KEY", ""),

		BleveIndexPath: env("BLEVE_INDEX_PATH", "storage/bleve/posts.bleve"),
		UploadPath:     env("UPLOAD_PATH", "storage/app/public"),
	}
	return C
}

// DSN MySQL 连接串
func (c *Config) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		c.DBUsername, c.DBPassword, c.DBHost, c.DBPort, c.DBDatabase)
}

// RedisAddr Redis 地址
func (c *Config) RedisAddr() string {
	return fmt.Sprintf("%s:%d", c.RedisHost, c.RedisPort)
}

func env(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func envBool(key string, def bool) bool {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return def
}
