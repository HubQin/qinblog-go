package middleware

import (
	"encoding/gob"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	sredis "github.com/gin-contrib/sessions/redis"
	"github.com/gin-gonic/gin"

	"github.com/qin/qinblog/internal/config"
	"github.com/qin/qinblog/internal/database"
	"github.com/qin/qinblog/internal/models"
	"github.com/qin/qinblog/internal/render"
	"github.com/qin/qinblog/internal/services"
	"github.com/qin/qinblog/internal/support"
)

func init() {
	// session 中存放的复合类型需要注册 gob
	gob.Register(map[string]string{})
}

const (
	sessKeyUserID = "user_id"
	sessKeyToken  = "_token"
	rememberCookie = "qinblog_remember"
	ctxKeyUser     = "current_user"
)

// Sessions 会话中间件：优先 redis store，不可用时退化为 cookie store
func Sessions() gin.HandlerFunc {
	name := "qinblog_session"
	secret := []byte(config.C.AppKey)
	if database.Redis != nil && database.Redis.Ping(database.Ctx).Err() == nil {
		store, err := sredis.NewStore(10, "tcp", config.C.RedisAddr(), "", config.C.RedisPassword, secret)
		if err == nil {
			return sessions.Sessions(name, store)
		}
	}
	return sessions.Sessions(name, cookie.NewStore(secret))
}

// MethodOverride 支持表单 _method=PUT/DELETE 方法伪造（等价 Laravel @method）
func MethodOverride(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			if m := r.PostFormValue("_method"); m != "" {
				m = strings.ToUpper(m)
				if m == http.MethodPut || m == http.MethodPatch || m == http.MethodDelete {
					r.Method = m
				}
			}
		}
		h.ServeHTTP(w, r)
	})
}

// CurrentUser 当前登录用户（未登录返回 nil）
func CurrentUser(c *gin.Context) *models.User {
	if u, ok := c.Get(ctxKeyUser); ok {
		if user, ok := u.(*models.User); ok {
			return user
		}
	}
	return nil
}

// Login 登录：写 session，可选记住我（5 年 cookie）
func Login(c *gin.Context, user *models.User, remember bool) {
	sess := sessions.Default(c)
	sess.Set(sessKeyUserID, user.ID)
	_ = sess.Save()

	if remember {
		token := support.RandomString(60)
		database.DB.Model(&models.User{}).Where("id = ?", user.ID).UpdateColumn("remember_token", token)
		c.SetCookie(rememberCookie, strconv.FormatUint(uint64(user.ID), 10)+"|"+token,
			5*365*24*3600, "/", "", false, true)
	}
}

// Logout 登出：清 session 与记住我 cookie
func Logout(c *gin.Context) {
	sess := sessions.Default(c)
	sess.Delete(sessKeyUserID)
	_ = sess.Save()
	c.SetCookie(rememberCookie, "", -1, "/", "", false, true)
}

// resolveUser 从 session / 记住我 cookie 解析当前用户
func resolveUser(c *gin.Context) *models.User {
	sess := sessions.Default(c)
	if v := sess.Get(sessKeyUserID); v != nil {
		var id uint
		switch n := v.(type) {
		case uint:
			id = n
		case int:
			id = uint(n)
		case int64:
			id = uint(n)
		}
		if id > 0 {
			var user models.User
			if err := database.DB.First(&user, id).Error; err == nil {
				return &user
			}
		}
	}

	// 记住我 cookie：user_id|remember_token
	if raw, err := c.Cookie(rememberCookie); err == nil && raw != "" {
		parts := strings.SplitN(raw, "|", 2)
		if len(parts) == 2 && parts[1] != "" {
			var user models.User
			if err := database.DB.Where("id = ? AND remember_token = ?", parts[0], parts[1]).First(&user).Error; err == nil {
				sess.Set(sessKeyUserID, user.ID)
				_ = sess.Save()
				return &user
			}
		}
	}
	return nil
}

// Flash 写入一次性提示消息（等价 session()->flash）
func Flash(c *gin.Context, level, message string) {
	sess := sessions.Default(c)
	sess.Set("flash_"+level, message)
	_ = sess.Save()
}

// FlashErrors 写入表单验证错误
func FlashErrors(c *gin.Context, errs map[string]string) {
	sess := sessions.Default(c)
	sess.Set("errors", errs)
	_ = sess.Save()
}

// FlashOld 写入旧输入（表单回填）
func FlashOld(c *gin.Context, form url.Values) {
	old := make(map[string]string, len(form))
	for k, v := range form {
		if k == "password" || k == "password_confirmation" || k == "_token" {
			continue
		}
		if len(v) > 0 {
			old[k] = v[0]
		}
	}
	sess := sessions.Default(c)
	sess.Set("old", old)
	_ = sess.Save()
}

// pullFlash 读取并清除一次性 session 数据
func pullFlash(c *gin.Context) (map[string]string, map[string]string, map[string]string) {
	sess := sessions.Default(c)
	flash := map[string]string{}
	for _, level := range []string{"success", "info", "warning", "danger"} {
		if v := sess.Get("flash_" + level); v != nil {
			if msg, ok := v.(string); ok {
				flash[level] = msg
			}
			sess.Delete("flash_" + level)
		}
	}
	errs := map[string]string{}
	if v := sess.Get("errors"); v != nil {
		if m, ok := v.(map[string]string); ok {
			errs = m
		}
		sess.Delete("errors")
	}
	old := map[string]string{}
	if v := sess.Get("old"); v != nil {
		if m, ok := v.(map[string]string); ok {
			old = m
		}
		sess.Delete("old")
	}
	_ = sess.Save()
	return flash, errs, old
}

// csrfToken 取（或生成）session 中的 CSRF token
func csrfToken(c *gin.Context) string {
	sess := sessions.Default(c)
	if v := sess.Get(sessKeyToken); v != nil {
		if token, ok := v.(string); ok && token != "" {
			return token
		}
	}
	token := support.RandomString(40)
	sess.Set(sessKeyToken, token)
	_ = sess.Save()
	return token
}

// VerifyCSRF 校验非安全请求的 CSRF token（表单 _token 或 X-CSRF-TOKEN 头）
func VerifyCSRF() gin.HandlerFunc {
	return func(c *gin.Context) {
		switch c.Request.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			c.Next()
			return
		}
		token := c.PostForm("_token")
		if token == "" {
			token = c.GetHeader("X-CSRF-TOKEN")
		}
		if token == "" || token != csrfToken(c) {
			// 419：与 Laravel 的 Page Expired 保持一致
			c.String(419, "CSRF token mismatch")
			c.Abort()
			return
		}
		c.Next()
	}
}

// Globals 注入全局视图数据：当前用户、站点设置、侧边栏、flash、CSRF（等价 view composer）
func Globals() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := resolveUser(c)
		if user != nil {
			c.Set(ctxKeyUser, user)
		}

		flash, errs, old := pullFlash(c)

		routeClass := "index-page"
		if p := strings.Trim(c.Request.URL.Path, "/"); p != "" {
			routeClass = strings.ReplaceAll(strings.ReplaceAll(p, "/", "-"), ".", "-") + "-page"
		}

		globals := gin.H{
			"siteConfigs": services.Settings.All(),
			"currentUser": user,
			"csrfToken":   csrfToken(c),
			"flash":       flash,
			"errors":      errs,
			"old":         old,
			"routeClass":  routeClass,
			"currentURL":  c.Request.URL.String(),
			"categories":  services.Sidebar.Categories(),
			"tags":        services.Sidebar.Tags(),
			"archives":    services.Sidebar.Archives(),
			"links":       services.Sidebar.Links(),
			"columns":     services.Sidebar.Columns(),
		}
		c.Set("view_globals", globals)
		c.Next()
	}
}

// Auth 登录校验，未登录跳转登录页
func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if CurrentUser(c) == nil {
			if c.GetHeader("X-Requested-With") == "XMLHttpRequest" || strings.Contains(c.GetHeader("Accept"), "application/json") {
				c.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthenticated."})
			} else {
				c.Redirect(http.StatusFound, "/login")
			}
			c.Abort()
			return
		}
		c.Next()
	}
}

// Guest 仅允许未登录用户（登录/注册页）
func Guest() gin.HandlerFunc {
	return func(c *gin.Context) {
		if CurrentUser(c) != nil {
			c.Redirect(http.StatusFound, "/")
			c.Abort()
			return
		}
		c.Next()
	}
}

// Admin 仅允许管理员
func Admin() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := CurrentUser(c)
		if user == nil {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}
		if !user.IsAdmin {
			render.HTML(c, http.StatusForbidden, "errors/403", gin.H{"message": "无权访问"})
			c.Abort()
			return
		}
		c.Next()
	}
}
