package services

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/mozillazg/go-pinyin"

	"github.com/qin/qinblog/internal/config"
)

// Translate 标题翻译服务（对应 TranslateService：百度翻译，未配置时回退拼音）
type translateService struct{}

var Translate = &translateService{}

type baiduTranslateResp struct {
	TransResult []struct {
		Src string `json:"src"`
		Dst string `json:"dst"`
	} `json:"trans_result"`
}

// Translate 中文翻译为英文，失败回退拼音
func (s *translateService) Translate(text string) string {
	appid := config.C.BaiduTranslateAppID
	key := config.C.BaiduTranslateKey

	// 如果没有配置百度翻译，自动使用兼容的拼音方案
	if appid == "" || key == "" {
		return s.Pinyin(text)
	}

	salt := fmt.Sprintf("%d", time.Now().Unix())
	sum := md5.Sum([]byte(appid + text + salt + key))
	sign := hex.EncodeToString(sum[:])

	q := url.Values{}
	q.Set("q", text)
	q.Set("from", "zh")
	q.Set("to", "en")
	q.Set("appid", appid)
	q.Set("salt", salt)
	q.Set("sign", sign)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get("http://api.fanyi.baidu.com/api/trans/vip/translate?" + q.Encode())
	if err != nil {
		return s.Pinyin(text)
	}
	defer resp.Body.Close()

	var result baiduTranslateResp
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil || len(result.TransResult) == 0 {
		// 如果百度翻译没有结果，使用拼音作为后备计划
		return s.Pinyin(text)
	}
	return result.TransResult[0].Dst
}

// Pinyin 中文转拼音 permalink（等价 overtrue/pinyin 的 permalink()）
func (s *translateService) Pinyin(text string) string {
	args := pinyin.NewArgs()
	// 非中文字符原样保留
	args.Fallback = func(r rune, a pinyin.Args) []string {
		return []string{string(r)}
	}
	var parts []string
	for _, segment := range pinyin.LazyPinyin(text, args) {
		segment = strings.TrimSpace(segment)
		if segment != "" {
			parts = append(parts, segment)
		}
	}
	return strings.Join(parts, "-")
}
