package services

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/qin/qinblog/internal/database"
	"github.com/qin/qinblog/internal/models"
)

// Settings 站点设置服务：DB(settings 表) + 内存缓存
type settingsService struct {
	mu    sync.RWMutex
	cache map[string]string
}

var Settings = &settingsService{}

// 站点设置的全部键（对照 app/Admin/Forms/Setting.php）
var SettingKeys = []string{
	"name", "logo", "slogan", "seo_keyword", "seo_description", "iconfont_url",
	"beian", "main_color", "email", "notice", "footer", "about",
	"qr_wechat_office", "qr_weapp",
}

var settingDefaults = map[string]string{
	"name":       "QinBlog",
	"main_color": "rgba(54,137,218,0.8)",
	"footer":     "由Qin设计和编码",
}

// Init 建表（仅 settings 一张新表）并载入缓存
func (s *settingsService) Init() error {
	if err := database.DB.AutoMigrate(&models.Setting{}); err != nil {
		return err
	}
	// 写入缺省值（仅当键不存在时）
	for k, v := range settingDefaults {
		var count int64
		database.DB.Model(&models.Setting{}).Where("`key` = ?", k).Count(&count)
		if count == 0 {
			database.DB.Create(&models.Setting{Key: k, Value: v})
		}
	}
	return s.Reload()
}

// Reload 重新加载缓存
func (s *settingsService) Reload() error {
	var rows []models.Setting
	if err := database.DB.Find(&rows).Error; err != nil {
		return err
	}
	m := make(map[string]string, len(rows))
	for _, r := range rows {
		m[r.Key] = r.Value
	}
	s.mu.Lock()
	s.cache = m
	s.mu.Unlock()
	return nil
}

// Get 读取设置
func (s *settingsService) Get(key string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cache[key]
}

// All 全部设置（模板中作为 siteConfigs 使用）
func (s *settingsService) All() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]string, len(s.cache))
	for k, v := range s.cache {
		out[k] = v
	}
	return out
}

// Save 批量保存并刷新缓存
func (s *settingsService) Save(values map[string]string) error {
	for k, v := range values {
		var count int64
		database.DB.Model(&models.Setting{}).Where("`key` = ?", k).Count(&count)
		if count == 0 {
			if err := database.DB.Create(&models.Setting{Key: k, Value: v}).Error; err != nil {
				return err
			}
		} else {
			if err := database.DB.Model(&models.Setting{}).Where("`key` = ?", k).Update("value", v).Error; err != nil {
				return err
			}
		}
	}
	return s.Reload()
}

// ImportFromJSON 便于把老项目 config/site.php 的值一次性导入
func (s *settingsService) ImportFromJSON(data []byte) error {
	var values map[string]string
	if err := json.Unmarshal(data, &values); err != nil {
		return err
	}
	if err := s.Save(values); err != nil {
		return err
	}
	log.Printf("imported %d settings", len(values))
	return nil
}
