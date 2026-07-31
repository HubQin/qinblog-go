package support

import (
	"strconv"
	"strings"
	"testing"
	"time"
)

// Laravel 存量哈希前缀为 $2y$，须能直接校验
func TestCheckPasswordLaravelCompat(t *testing.T) {
	hash, err := HashPassword("secret123")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if !CheckPassword(hash, "secret123") {
		t.Error("原生 $2a$ 哈希校验失败")
	}

	laravelHash := "$2y$" + strings.TrimPrefix(hash, "$2a$")
	if !CheckPassword(laravelHash, "secret123") {
		t.Error("$2y$ 前缀（Laravel）哈希应校验通过")
	}
	if CheckPassword(laravelHash, "wrong-password") {
		t.Error("错误密码不应通过校验")
	}

	// Laravel bcrypt('password') 的真实产物
	real := "$2y$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi"
	if !CheckPassword(real, "password") {
		t.Error("Laravel 真实哈希应校验通过")
	}
}

func TestSignedPayload(t *testing.T) {
	key := "test-app-key"
	expires, sig := SignedPayload(key, "verify|1|a@b.c", time.Hour)
	expiresStr := strconv.FormatInt(expires, 10)

	if !VerifySignedPayload(key, "verify|1|a@b.c", expiresStr, sig) {
		t.Error("有效期内签名应校验通过")
	}
	if VerifySignedPayload(key, "verify|2|a@b.c", expiresStr, sig) {
		t.Error("篡改载荷应校验失败")
	}
	if VerifySignedPayload("other-key", "verify|1|a@b.c", expiresStr, sig) {
		t.Error("密钥不同应校验失败")
	}
	// 已过期
	pastExpires, pastSig := SignedPayload(key, "verify|1|a@b.c", -time.Hour)
	if VerifySignedPayload(key, "verify|1|a@b.c", strconv.FormatInt(pastExpires, 10), pastSig) {
		t.Error("过期签名应校验失败")
	}
}
