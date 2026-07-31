package render

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/qin/qinblog/internal/config"
	"github.com/qin/qinblog/internal/support"
)

// Engine 模板渲染引擎：layouts/partials + 每页面独立模板集
type Engine struct {
	templates map[string]*template.Template
	dir       string
}

var Default *Engine

// viteManifest 解析后的 Vite manifest（生产模式）
var viteManifest map[string]struct {
	File string   `json:"file"`
	CSS  []string `json:"css"`
	Src  string   `json:"src"`
}

// Init 加载全部模板与 Vite manifest
func Init(templateDir string) error {
	loadViteManifest()
	e, err := New(templateDir)
	if err != nil {
		return err
	}
	Default = e
	return nil
}

// New 构建模板引擎：每个页面模板与 layouts/、partials/ 组成独立模板集
func New(dir string) (*Engine, error) {
	e := &Engine{templates: map[string]*template.Template{}, dir: dir}

	var shared []string // layouts + partials
	var pages []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".html") {
			return err
		}
		rel, _ := filepath.Rel(dir, path)
		rel = filepath.ToSlash(rel)
		if strings.HasPrefix(rel, "layouts/") || strings.HasPrefix(rel, "partials/") {
			shared = append(shared, path)
		} else {
			pages = append(pages, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	for _, page := range pages {
		rel, _ := filepath.Rel(dir, page)
		name := strings.TrimSuffix(filepath.ToSlash(rel), ".html")
		files := append(append([]string{}, shared...), page)
		t, err := template.New(filepath.Base(page)).Funcs(FuncMap()).ParseFiles(files...)
		if err != nil {
			return nil, fmt.Errorf("parse template %s: %w", name, err)
		}
		e.templates[name] = t
	}
	return e, nil
}

// HTML 渲染页面（name 如 "posts/index"），data 会与 gin context 中注入的全局数据合并
func HTML(c *gin.Context, status int, name string, data gin.H) {
	if data == nil {
		data = gin.H{}
	}
	// 中间件注入的全局数据（siteConfigs/currentUser/sidebar/flash/csrf 等）
	if globals, ok := c.Get("view_globals"); ok {
		for k, v := range globals.(gin.H) {
			if _, exists := data[k]; !exists {
				data[k] = v
			}
		}
	}

	t, ok := Default.templates[name]
	if !ok {
		c.String(http.StatusInternalServerError, "template not found: %s", name)
		return
	}
	c.Status(status)
	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := t.ExecuteTemplate(c.Writer, "layouts/app", data); err != nil {
		_ = c.Error(err)
	}
}

// FuncMap 模板函数（对齐 Blade 中用到的 helper）
func FuncMap() template.FuncMap {
	return template.FuncMap{
		"safe":      func(s string) template.HTML { return template.HTML(s) },
		"safeattr":  func(s string) template.HTMLAttr { return template.HTMLAttr(s) },
		"safejs":    func(s string) template.JS { return template.JS(s) },
		"safeurl":   func(s string) template.URL { return template.URL(s) },
		"safecss":   func(s string) template.CSS { return template.CSS(s) },
		"json":      jsonEncode,
		"asset":     Asset,
		"vite":      ViteTags,
		"str_limit": func(s string, n int) string { return support.StrLimit(s, n, "...") },
		"excerpt":   func(s string, n int) string { return support.MakeExcerpt(s, n) },
		"date": func(t time.Time, layout string) string {
			if t.IsZero() {
				return ""
			}
			return t.Format(layout)
		},
		"timeago":  TimeAgo,
		"markdown": func(s string) template.HTML { return template.HTML(support.Parsedown(s)) },
		"add":      func(a, b int) int { return a + b },
		"sub":      func(a, b int) int { return a - b },
		"until":    func(n int) []int { r := make([]int, n); for i := range r { r[i] = i }; return r },
		"dict":     dict,
		"now":      time.Now,
	}
}

func jsonEncode(v interface{}) template.JS {
	b, err := json.Marshal(v)
	if err != nil {
		return "null"
	}
	return template.JS(b)
}

// dict 供 partial 传多参：{{template "x" dict "a" 1 "b" 2}}
func dict(pairs ...interface{}) (map[string]interface{}, error) {
	if len(pairs)%2 != 0 {
		return nil, fmt.Errorf("dict 参数必须成对")
	}
	m := make(map[string]interface{}, len(pairs)/2)
	for i := 0; i < len(pairs); i += 2 {
		key, ok := pairs[i].(string)
		if !ok {
			return nil, fmt.Errorf("dict 键必须是字符串")
		}
		m[key] = pairs[i+1]
	}
	return m, nil
}

// TimeAgo 中文相对时间（等价 diffForHumans）
func TimeAgo(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "刚刚"
	case d < time.Hour:
		return fmt.Sprintf("%d分钟前", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%d小时前", int(d.Hours()))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%d天前", int(d.Hours()/24))
	case d < 365*24*time.Hour:
		return fmt.Sprintf("%d个月前", int(d.Hours()/24/30))
	default:
		return fmt.Sprintf("%d年前", int(d.Hours()/24/365))
	}
}

// loadViteManifest 读取 Vite 构建产物 manifest
func loadViteManifest() {
	viteManifest = nil
	for _, p := range []string{
		"web/public/build/.vite/manifest.json",
		"web/public/build/manifest.json",
	} {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		if json.Unmarshal(data, &viteManifest) == nil {
			return
		}
	}
}

// Asset 静态资源路径：Vite 构建产物走 manifest，其余原样返回
func Asset(path string) string {
	path = strings.TrimPrefix(path, "/")
	if config.C != nil && config.C.ViteDev {
		return config.C.ViteDevURL + "/" + path
	}
	if entry, ok := viteManifest[path]; ok {
		return "/build/" + entry.File
	}
	return "/" + path
}

// ViteTags 输出入口 js/css 标签（等价 Blade 里的 mix() 引入）
func ViteTags(entry string) template.HTML {
	if config.C != nil && config.C.ViteDev {
		return template.HTML(fmt.Sprintf(
			`<script type="module" src="%s/@vite/client"></script>`+"\n"+
				`<script type="module" src="%s/%s"></script>`,
			config.C.ViteDevURL, config.C.ViteDevURL, entry))
	}
	m, ok := viteManifest[entry]
	if !ok {
		return ""
	}
	var sb strings.Builder
	for _, css := range m.CSS {
		sb.WriteString(fmt.Sprintf(`<link rel="stylesheet" href="/build/%s">`+"\n", css))
	}
	sb.WriteString(fmt.Sprintf(`<script type="module" src="/build/%s"></script>`, m.File))
	return template.HTML(sb.String())
}
