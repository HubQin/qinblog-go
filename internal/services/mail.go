package services

import (
	"fmt"
	"log"
	"net/url"
	"time"

	"gopkg.in/gomail.v2"

	"github.com/qin/qinblog/internal/config"
	"github.com/qin/qinblog/internal/database"
	"github.com/qin/qinblog/internal/models"
	"github.com/qin/qinblog/internal/support"
)

// Mail 邮件服务（邮箱验证 / 密码重置，对应 Laravel VerifyEmail、ResetPassword 通知）
type mailService struct{}

var Mail = &mailService{}

func (s *mailService) send(to, subject, htmlBody string) error {
	if config.C.MailHost == "" {
		log.Printf("[warn] mail not configured, skip sending to %s: %s", to, subject)
		return nil
	}
	m := gomail.NewMessage()
	m.SetAddressHeader("From", config.C.MailFromAddress, config.C.MailFromName)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", htmlBody)
	d := gomail.NewDialer(config.C.MailHost, config.C.MailPort, config.C.MailUsername, config.C.MailPassword)
	return d.DialAndSend(m)
}

func mailLayout(title, intro, action, link string) string {
	return fmt.Sprintf(`
<div style="max-width:570px;margin:0 auto;padding:32px;font-family:Helvetica,Arial,sans-serif;color:#3d4852;">
  <h1 style="font-size:18px;">%s</h1>
  <p>%s</p>
  <p style="text-align:center;margin:30px 0;">
    <a href="%s" style="background:#3490dc;border-radius:4px;color:#fff;display:inline-block;padding:8px 18px;text-decoration:none;">%s</a>
  </p>
  <p style="font-size:12px;color:#b0adc5;">如果无法点击按钮，请将以下链接复制到浏览器打开：<br>%s</p>
</div>`, title, intro, link, action, link)
}

// verifySubject 生成邮箱验证签名的主体
func verifySubject(userID uint, email string) string {
	return fmt.Sprintf("verify|%d|%s", userID, email)
}

// SendEmailVerification 发送验证邮件（签名链接 60 分钟有效，等价 VerifyEmail 通知）
func (s *mailService) SendEmailVerification(user *models.User) error {
	expires, sig := support.SignedPayload(config.C.AppKey, verifySubject(user.ID, user.Email), 60*time.Minute)
	link := fmt.Sprintf("%s/email/verify/%d?expires=%d&signature=%s", config.C.AppURL, user.ID, expires, sig)
	body := mailLayout("验证邮箱地址", "请点击下面的按钮验证您的邮箱地址。如果您并未注册账号，请忽略此邮件。", "验证邮箱地址", link)
	return s.send(user.Email, "验证邮箱地址", body)
}

// VerifyEmailSignature 校验邮箱验证链接签名
func (s *mailService) VerifyEmailSignature(user *models.User, expires, signature string) bool {
	return support.VerifySignedPayload(config.C.AppKey, verifySubject(user.ID, user.Email), expires, signature)
}

// SendPasswordReset 生成重置 token 存入 password_resets 表并发送邮件（等价 ResetPassword 通知）
func (s *mailService) SendPasswordReset(user *models.User) error {
	token := support.RandomString(64)
	database.DB.Exec("DELETE FROM password_resets WHERE email = ?", user.Email)
	if err := database.DB.Exec(
		"INSERT INTO password_resets (email, token, created_at) VALUES (?, ?, ?)",
		user.Email, support.Sha256Hex(token), time.Now(),
	).Error; err != nil {
		return err
	}
	link := fmt.Sprintf("%s/password/reset/%s?email=%s", config.C.AppURL, token, url.QueryEscape(user.Email))
	body := mailLayout("重置密码", "您收到此邮件是因为我们收到了您账号的密码重置请求。重置链接 60 分钟内有效，如果您并未申请重置密码，请忽略此邮件。", "重置密码", link)
	return s.send(user.Email, "重置密码", body)
}

// VerifyPasswordResetToken 校验密码重置 token（60 分钟有效）
func (s *mailService) VerifyPasswordResetToken(email, token string) bool {
	var row struct {
		Token     string
		CreatedAt time.Time
	}
	err := database.DB.Raw("SELECT token, created_at FROM password_resets WHERE email = ?", email).Scan(&row).Error
	if err != nil || row.Token == "" {
		return false
	}
	if time.Since(row.CreatedAt) > 60*time.Minute {
		return false
	}
	return row.Token == support.Sha256Hex(token)
}

// DeletePasswordResetToken 重置成功后删除 token
func (s *mailService) DeletePasswordResetToken(email string) {
	database.DB.Exec("DELETE FROM password_resets WHERE email = ?", email)
}
