package support

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword bcrypt 加密（cost 10，与 Laravel 默认一致）
func HashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(b), err
}

// CheckPassword 校验密码。Laravel 生成的哈希前缀为 $2y$，
// Go 的 bcrypt 库使用 $2a$，两者算法相同，替换前缀后即可兼容存量用户。
func CheckPassword(hash, password string) bool {
	if strings.HasPrefix(hash, "$2y$") {
		hash = "$2a$" + hash[4:]
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// Sign 用 APP_KEY 对内容做 HMAC-SHA256 签名
func Sign(key, payload string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifySign 校验签名
func VerifySign(key, payload, signature string) bool {
	expected := Sign(key, payload)
	return hmac.Equal([]byte(expected), []byte(signature))
}

// SignedPayload 生成带过期时间的签名载荷（用于邮箱验证/密码重置链接）
// 返回 expires 与 signature
func SignedPayload(key, subject string, ttl time.Duration) (int64, string) {
	expires := time.Now().Add(ttl).Unix()
	sig := Sign(key, fmt.Sprintf("%s|%d", subject, expires))
	return expires, sig
}

// VerifySignedPayload 校验带过期时间的签名
func VerifySignedPayload(key, subject, expiresStr, signature string) bool {
	expires, err := strconv.ParseInt(expiresStr, 10, 64)
	if err != nil || time.Now().Unix() > expires {
		return false
	}
	return VerifySign(key, fmt.Sprintf("%s|%d", subject, expires), signature)
}

// Sha256Hex 计算 sha256 十六进制串（密码重置 token 存库用）
func Sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}
