package web

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
	oauthgithub "golang.org/x/oauth2/github"

	"github.com/qin/qinblog/internal/config"
	"github.com/qin/qinblog/internal/database"
	"github.com/qin/qinblog/internal/middleware"
	"github.com/qin/qinblog/internal/models"
	"github.com/qin/qinblog/internal/render"
	"github.com/qin/qinblog/internal/services"
	"github.com/qin/qinblog/internal/support"
)

// ShowLogin 登录页（GET /login）
func ShowLogin(c *gin.Context) {
	render.HTML(c, http.StatusOK, "auth/login", nil)
}

// Login 登录（POST /login）
func Login(c *gin.Context) {
	email := strings.TrimSpace(c.PostForm("email"))
	password := c.PostForm("password")
	remember := c.PostForm("remember") != ""

	var user models.User
	if err := database.DB.Where("email = ?", email).First(&user).Error; err != nil ||
		!support.CheckPassword(user.Password, password) {
		middleware.FlashErrors(c, map[string]string{"email": "用户名和密码不匹配"})
		middleware.FlashOld(c, c.Request.PostForm)
		c.Redirect(http.StatusFound, "/login")
		return
	}

	middleware.Login(c, &user, remember)
	c.Redirect(http.StatusFound, "/")
}

// Logout 登出（POST /logout）
func Logout(c *gin.Context) {
	middleware.Logout(c)
	c.Redirect(http.StatusFound, "/")
}

// ShowRegister 注册页（GET /register）
func ShowRegister(c *gin.Context) {
	render.HTML(c, http.StatusOK, "auth/register", nil)
}

// Register 注册（POST /register）
func Register(c *gin.Context) {
	name := strings.TrimSpace(c.PostForm("name"))
	email := strings.TrimSpace(c.PostForm("email"))
	password := c.PostForm("password")
	confirm := c.PostForm("password_confirmation")

	errs := map[string]string{}
	if l := len([]rune(name)); l < 2 || l > 25 {
		errs["name"] = "用户名必须介于 2 - 25 个字符之间"
	}
	if email == "" || !strings.Contains(email, "@") {
		errs["email"] = "请填写正确的邮箱地址"
	}
	if len(password) < 6 {
		errs["password"] = "密码至少 6 个字符"
	} else if password != confirm {
		errs["password"] = "两次输入的密码不一致"
	}
	if len(errs) == 0 {
		var count int64
		database.DB.Model(&models.User{}).Where("email = ?", email).Count(&count)
		if count > 0 {
			errs["email"] = "邮箱已被注册"
		}
	}
	if len(errs) > 0 {
		middleware.FlashErrors(c, errs)
		middleware.FlashOld(c, c.Request.PostForm)
		c.Redirect(http.StatusFound, "/register")
		return
	}

	hash, err := support.HashPassword(password)
	if err != nil {
		middleware.Flash(c, "danger", "注册失败，请重试")
		c.Redirect(http.StatusFound, "/register")
		return
	}
	user := models.User{Name: name, Email: email, Password: hash}
	if err := database.DB.Create(&user).Error; err != nil {
		middleware.Flash(c, "danger", "注册失败，请重试")
		c.Redirect(http.StatusFound, "/register")
		return
	}

	// 发送验证邮件并登录（等价 MustVerifyEmail 注册流程）
	go func() { _ = services.Mail.SendEmailVerification(&user) }()
	middleware.Login(c, &user, false)
	c.Redirect(http.StatusFound, "/email/verify")
}

// EmailVerifyNotice 验证邮箱提示页（GET /email/verify）
func EmailVerifyNotice(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user.HasVerifiedEmail() {
		c.Redirect(http.StatusFound, "/")
		return
	}
	render.HTML(c, http.StatusOK, "auth/verify", nil)
}

// EmailVerify 验证邮箱签名链接（GET /email/verify/:id）
func EmailVerify(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var user models.User
	if err := database.DB.First(&user, id).Error; err != nil {
		c.String(http.StatusNotFound, "用户不存在")
		return
	}
	if !services.Mail.VerifyEmailSignature(&user, c.Query("expires"), c.Query("signature")) {
		c.String(http.StatusForbidden, "验证链接无效或已过期")
		return
	}
	if !user.HasVerifiedEmail() {
		database.DB.Model(&user).UpdateColumn("email_verified_at", time.Now())
	}
	middleware.Flash(c, "success", "邮箱验证成功")
	c.Redirect(http.StatusFound, "/")
}

// EmailResend 重发验证邮件（POST /email/resend）
func EmailResend(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if !user.HasVerifiedEmail() {
		go func() { _ = services.Mail.SendEmailVerification(user) }()
	}
	middleware.Flash(c, "success", "验证邮件已重新发送")
	c.Redirect(http.StatusFound, "/email/verify")
}

// ShowForgotPassword 找回密码页（GET /password/reset）
func ShowForgotPassword(c *gin.Context) {
	render.HTML(c, http.StatusOK, "auth/passwords/email", nil)
}

// SendResetLink 发送重置邮件（POST /password/email）
func SendResetLink(c *gin.Context) {
	email := strings.TrimSpace(c.PostForm("email"))
	var user models.User
	if err := database.DB.Where("email = ?", email).First(&user).Error; err != nil {
		middleware.FlashErrors(c, map[string]string{"email": "找不到该邮箱对应的用户"})
		c.Redirect(http.StatusFound, "/password/reset")
		return
	}
	if err := services.Mail.SendPasswordReset(&user); err != nil {
		middleware.Flash(c, "danger", "邮件发送失败，请稍后重试")
		c.Redirect(http.StatusFound, "/password/reset")
		return
	}
	middleware.Flash(c, "success", "重置密码邮件已发送，请查收")
	c.Redirect(http.StatusFound, "/password/reset")
}

// ShowResetPassword 重置密码页（GET /password/reset/:token）
func ShowResetPassword(c *gin.Context) {
	render.HTML(c, http.StatusOK, "auth/passwords/reset", gin.H{
		"token": c.Param("token"),
		"email": c.Query("email"),
	})
}

// ResetPassword 重置密码（POST /password/reset）
func ResetPassword(c *gin.Context) {
	email := strings.TrimSpace(c.PostForm("email"))
	token := c.PostForm("token")
	password := c.PostForm("password")
	confirm := c.PostForm("password_confirmation")

	if len(password) < 6 || password != confirm {
		middleware.FlashErrors(c, map[string]string{"password": "密码至少 6 个字符且两次输入一致"})
		c.Redirect(http.StatusFound, "/password/reset/"+token+"?email="+email)
		return
	}
	if !services.Mail.VerifyPasswordResetToken(email, token) {
		middleware.FlashErrors(c, map[string]string{"email": "重置链接无效或已过期"})
		c.Redirect(http.StatusFound, "/password/reset")
		return
	}

	var user models.User
	if err := database.DB.Where("email = ?", email).First(&user).Error; err != nil {
		middleware.FlashErrors(c, map[string]string{"email": "找不到该邮箱对应的用户"})
		c.Redirect(http.StatusFound, "/password/reset")
		return
	}
	hash, err := support.HashPassword(password)
	if err != nil {
		middleware.Flash(c, "danger", "重置失败，请重试")
		c.Redirect(http.StatusFound, "/password/reset")
		return
	}
	database.DB.Model(&user).UpdateColumn("password", hash)
	services.Mail.DeletePasswordResetToken(email)

	middleware.Login(c, &user, false)
	middleware.Flash(c, "success", "密码重置成功")
	c.Redirect(http.StatusFound, "/")
}

// githubOAuthConfig GitHub OAuth 配置
func githubOAuthConfig() *oauth2.Config {
	redirect := config.C.GithubRedirectURL
	if redirect == "" {
		redirect = config.C.AppURL + "/socials/github/callback"
	}
	return &oauth2.Config{
		ClientID:     config.C.GithubClientID,
		ClientSecret: config.C.GithubClientSecret,
		RedirectURL:  redirect,
		Scopes:       []string{"read:user"},
		Endpoint:     oauthgithub.Endpoint,
	}
}

// SocialRedirect 跳转 GitHub 授权（GET /socials/github/redirect）
func SocialRedirect(c *gin.Context) {
	if c.Param("social_type") != "github" {
		c.String(http.StatusNotFound, "不支持的第三方登录")
		return
	}
	c.Redirect(http.StatusFound, githubOAuthConfig().AuthCodeURL("state"))
}

// SocialCallback GitHub 回调（GET /socials/github/callback，等价 AuthorizationsController）
func SocialCallback(c *gin.Context) {
	if c.Param("social_type") != "github" {
		c.String(http.StatusNotFound, "不支持的第三方登录")
		return
	}
	code := c.Query("code")
	if code == "" {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	conf := githubOAuthConfig()
	token, err := conf.Exchange(ctx, code)
	if err != nil {
		middleware.Flash(c, "danger", "GitHub 登录失败")
		c.Redirect(http.StatusFound, "/login")
		return
	}

	// 获取 GitHub 用户信息
	client := conf.Client(ctx, token)
	resp, err := client.Get("https://api.github.com/user")
	if err != nil {
		middleware.Flash(c, "danger", "GitHub 登录失败")
		c.Redirect(http.StatusFound, "/login")
		return
	}
	defer resp.Body.Close()

	var ghUser struct {
		ID        int64  `json:"id"`
		Login     string `json:"login"`
		Name      string `json:"name"`
		AvatarURL string `json:"avatar_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&ghUser); err != nil || ghUser.ID == 0 {
		middleware.Flash(c, "danger", "GitHub 登录失败")
		c.Redirect(http.StatusFound, "/login")
		return
	}

	openid := strconv.FormatInt(ghUser.ID, 10)
	name := ghUser.Name
	if name == "" {
		name = ghUser.Login
	}

	// openid + type 查用户，无则创建（等价 AuthorizationsController）
	var user models.User
	err = database.DB.Where("openid = ? AND type = ?", openid, "github").First(&user).Error
	if err != nil {
		user = models.User{Name: name, Avatar: ghUser.AvatarURL, Openid: openid, Type: "github"}
		if err := database.DB.Create(&user).Error; err != nil {
			middleware.Flash(c, "danger", "GitHub 登录失败")
			c.Redirect(http.StatusFound, "/login")
			return
		}
	}
	middleware.Login(c, &user, true)
	c.Redirect(http.StatusFound, "/")
}
