package service

import (
	"context"
	"strings"
	"testing"
	"time"
)

// TestParseCredentialsOAuth2 覆盖 OAuth2 凭据解密与字段映射。
func TestParseCredentialsOAuth2(t *testing.T) {
	env := newSendTestEnv(t, testSendAccount())

	expiry := time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC)
	account := testSendAccount()
	account.EncryptedCredentials = encryptJSON(t, env, `{
		"auth_type": "oauth2",
		"access_token": "access-token",
		"refresh_token": "refresh-token",
		"client_id": "client-id",
		"token_expiry": "2026-10-27T12:00:00Z"
	}`)

	creds, err := env.svc.parseCredentials(context.Background(), account)
	if err != nil {
		t.Fatalf("解析 OAuth2 凭据失败: %v", err)
	}
	if creds == nil {
		t.Fatal("OAuth2 凭据解析结果不应为 nil")
	}
	if creds.AccessToken != "access-token" || creds.RefreshToken != "refresh-token" {
		t.Errorf("token 解析错误: %+v", creds)
	}
	if creds.ClientID != "client-id" {
		t.Errorf("ClientID = %q，期望 client-id", creds.ClientID)
	}
	if !creds.TokenExpiry.Equal(expiry) {
		t.Errorf("TokenExpiry = %v，期望 %v", creds.TokenExpiry, expiry)
	}
}

// TestParseCredentialsQuick 覆盖 quick 导入凭据（只有 refresh_token + client_id）。
func TestParseCredentialsQuick(t *testing.T) {
	env := newSendTestEnv(t, testSendAccount())

	account := testSendAccount()
	account.EncryptedCredentials = encryptJSON(t, env, `{
		"auth_type": "quick",
		"refresh_token": "quick-refresh",
		"client_id": "quick-client"
	}`)

	creds, err := env.svc.parseCredentials(context.Background(), account)
	if err != nil {
		t.Fatalf("解析 quick 凭据失败: %v", err)
	}
	if creds == nil {
		t.Fatal("quick 凭据解析结果不应为 nil")
	}
	if creds.RefreshToken != "quick-refresh" || creds.ClientID != "quick-client" {
		t.Errorf("quick 凭据字段错误: %+v", creds)
	}
	if creds.AccessToken != "" {
		t.Errorf("quick 凭据不应带 access_token，实际 %q", creds.AccessToken)
	}
}

// TestParseCredentialsEmpty 覆盖无凭证账户（SMTP 走账户配置，返回 nil 而非报错）。
func TestParseCredentialsEmpty(t *testing.T) {
	env := newSendTestEnv(t, testSendAccount())

	creds, err := env.svc.parseCredentials(context.Background(), testSendAccount())
	if err != nil {
		t.Fatalf("空凭证不应报错: %v", err)
	}
	if creds != nil {
		t.Errorf("空凭证应返回 nil，实际 %+v", creds)
	}
}

// TestParseCredentialsCorruptedCiphertext 覆盖密文损坏：必须报错，不能静默降级
// 成「无凭证」（否则会用空 token 去发信，失败原因会被误判成服务商问题）。
func TestParseCredentialsCorruptedCiphertext(t *testing.T) {
	env := newSendTestEnv(t, testSendAccount())

	account := testSendAccount()
	account.EncryptedCredentials = "this-is-not-a-valid-ciphertext"

	creds, err := env.svc.parseCredentials(context.Background(), account)
	if err == nil {
		t.Fatal("损坏密文应返回 error")
	}
	if creds != nil {
		t.Errorf("解析失败时不应返回凭据: %+v", creds)
	}
	if !strings.Contains(err.Error(), "decrypt") {
		t.Errorf("error = %q，期望包含 decrypt", err.Error())
	}
}

// TestParseCredentialsMalformedJSON 覆盖「能解密但内容不是合法凭据」：
// 必须报解析错误，而不是退化成密码认证继续发信。
func TestParseCredentialsMalformedJSON(t *testing.T) {
	env := newSendTestEnv(t, testSendAccount())

	account := testSendAccount()
	account.EncryptedCredentials = encryptJSON(t, env, `{"auth_type":"quick","refresh_token":123}`)

	creds, err := env.svc.parseCredentials(context.Background(), account)
	if err == nil {
		t.Fatal("非法 quick 凭据应返回 error")
	}
	if creds != nil {
		t.Errorf("解析失败时不应返回凭据: %+v", creds)
	}
	if !strings.Contains(err.Error(), "quick") {
		t.Errorf("error = %q，期望点名 quick 凭据解析失败", err.Error())
	}
}

func encryptJSON(t *testing.T, env *sendTestEnv, raw string) string {
	t.Helper()

	ciphertext, err := env.crypto.Encrypt([]byte(raw))
	if err != nil {
		t.Fatalf("加密测试凭据失败: %v", err)
	}
	return ciphertext
}
