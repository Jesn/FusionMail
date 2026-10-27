package service

import (
	"testing"
	"time"

	"fusionmail/internal/model"
	"fusionmail/pkg/crypto"
)

// credentialTestResolver 用真实加密服务构造解析器（与生产同一套加解密）。
func credentialTestResolver(t *testing.T) (*CredentialResolver, *crypto.Service) {
	t.Helper()

	svc, err := crypto.NewService(testEncryptionKey)
	if err != nil {
		t.Fatalf("构造 crypto service 失败: %v", err)
	}
	return NewCredentialResolver(svc, nil), svc
}

// encryptCredential 把明文加密成账号里存的形式。
func encryptCredential(t *testing.T, svc *crypto.Service, plaintext string) string {
	t.Helper()

	encrypted, err := svc.Encrypt([]byte(plaintext))
	if err != nil {
		t.Fatalf("加密凭据失败: %v", err)
	}
	return encrypted
}

func credentialTestAccount(adapterName, authType string) *model.EmailAccount {
	return &model.EmailAccount{
		UID:   "acct-cred",
		Email: "user@example.com",
		AdapterRef: &model.Adapter{
			Name:     adapterName,
			AuthType: authType,
		},
	}
}

// TestResolveRequiresAccount 覆盖入参校验：nil 账号必须报错而不是 panic。
func TestResolveRequiresAccount(t *testing.T) {
	resolver, _ := credentialTestResolver(t)

	if _, err := resolver.Resolve(nil); err == nil {
		t.Fatal("nil 账号应返回 error")
	}
}

// TestResolveWithoutDecryptor 覆盖依赖缺失：没有解密器时必须显式报错。
func TestResolveWithoutDecryptor(t *testing.T) {
	resolver := NewCredentialResolver(nil, nil)
	account := credentialTestAccount("imap", "password")

	if _, err := resolver.Resolve(account); err == nil {
		t.Fatal("缺少解密器时应返回 error")
	}
}

// TestResolvePasswordCredentials 覆盖默认（密码）认证：
// 密文解密后直接作为密码，且不改动其他字段。
func TestResolvePasswordCredentials(t *testing.T) {
	resolver, svc := credentialTestResolver(t)
	account := credentialTestAccount("smtp", "password")
	account.EncryptedCredentials = encryptCredential(t, svc, "s3cret-password")

	credentials, err := resolver.Resolve(account)
	if err != nil {
		t.Fatalf("解析凭据失败: %v", err)
	}
	if credentials.Password != "s3cret-password" {
		t.Errorf("Password = %q，期望明文密码", credentials.Password)
	}
	if credentials.Email != "user@example.com" {
		t.Errorf("Email = %q", credentials.Email)
	}
	if credentials.AuthType != "password" {
		t.Errorf("AuthType = %q，期望 password", credentials.AuthType)
	}
}

// TestResolveQuickCredentials 覆盖 quick 认证：
// refresh_token / client_id 从 JSON 里取，client_secret 必须清空（quick 不用密钥）。
func TestResolveQuickCredentials(t *testing.T) {
	resolver, svc := credentialTestResolver(t)
	account := credentialTestAccount("linuxdo", "quick")
	account.EncryptedCredentials = encryptCredential(t, svc,
		`{"refresh_token":"rt-1","client_id":"cid-1","client_secret":"should-be-cleared"}`)

	credentials, err := resolver.Resolve(account)
	if err != nil {
		t.Fatalf("解析 quick 凭据失败: %v", err)
	}
	if credentials.RefreshToken != "rt-1" || credentials.ClientID != "cid-1" {
		t.Errorf("quick 凭据解析错误: %+v", credentials)
	}
	if credentials.ClientSecret != "" {
		t.Errorf("quick 认证不应带 client_secret，实际 %q", credentials.ClientSecret)
	}
}

// TestResolveQuickCredentialsMalformedJSON 覆盖脏数据：
// quick 账号的密文不是合法 JSON 时必须报错，不能静默用空凭据去连。
func TestResolveQuickCredentialsMalformedJSON(t *testing.T) {
	resolver, svc := credentialTestResolver(t)
	account := credentialTestAccount("linuxdo", "quick")
	account.EncryptedCredentials = encryptCredential(t, svc, "not-json")

	if _, err := resolver.Resolve(account); err == nil {
		t.Fatal("quick 凭据非法 JSON 时应返回 error")
	}
}

// TestResolveAuthTypeFromCredentialsOverridesAccount 覆盖凭据内 auth_type 优先：
// 账号元数据与密文不一致时以密文为准（换认证方式后旧元数据可能没刷新）。
func TestResolveAuthTypeFromCredentialsOverridesAccount(t *testing.T) {
	resolver, svc := credentialTestResolver(t)
	account := credentialTestAccount("smtp", "password")
	account.EncryptedCredentials = encryptCredential(t, svc,
		`{"auth_type":"quick","refresh_token":"rt-2","client_id":"cid-2"}`)

	credentials, err := resolver.Resolve(account)
	if err != nil {
		t.Fatalf("解析凭据失败: %v", err)
	}
	if credentials.AuthType != "quick" {
		t.Errorf("AuthType = %q，期望以凭据内的 quick 为准", credentials.AuthType)
	}
	if credentials.RefreshToken != "rt-2" {
		t.Errorf("RefreshToken = %q，期望 rt-2", credentials.RefreshToken)
	}
}

// TestResolveOAuth2CredentialsInline 覆盖 OAuth2 内联凭据：
// 密文自带 client_id/client_secret 时不需要回查服务商配置。
func TestResolveOAuth2CredentialsInline(t *testing.T) {
	resolver, svc := credentialTestResolver(t)
	account := credentialTestAccount("gmail", "oauth2")
	expiry := time.Now().Add(time.Hour).UTC().Truncate(time.Second)
	account.EncryptedCredentials = encryptCredential(t, svc,
		`{"access_token":"at-1","refresh_token":"rt-1","client_id":"cid-1","client_secret":"cs-1","token_expiry":"`+
			expiry.Format(time.RFC3339)+`"}`)

	credentials, err := resolver.Resolve(account)
	if err != nil {
		t.Fatalf("解析 OAuth2 凭据失败: %v", err)
	}
	if credentials.AccessToken != "at-1" || credentials.RefreshToken != "rt-1" {
		t.Errorf("token 解析错误: %+v", credentials)
	}
	if credentials.ClientID != "cid-1" || credentials.ClientSecret != "cs-1" {
		t.Errorf("client 凭据解析错误: %+v", credentials)
	}
	if !credentials.TokenExpiry.Equal(expiry) {
		t.Errorf("TokenExpiry = %v，期望 %v", credentials.TokenExpiry, expiry)
	}
}

// TestResolveOAuth2WithoutProviderConfig 覆盖 OAuth2 缺少 client 凭据且未配置服务商：
// 当前实现返回空 client 凭据而不是报错（调用方靠后续 401 暴露问题）。
// 这里锁定行为，避免「静默返回空凭据」被误改成 panic。
func TestResolveOAuth2WithoutProviderConfig(t *testing.T) {
	resolver, svc := credentialTestResolver(t)
	account := credentialTestAccount("gmail", "oauth2")
	account.EncryptedCredentials = encryptCredential(t, svc, `{"access_token":"at-only"}`)

	credentials, err := resolver.Resolve(account)
	if err != nil {
		t.Fatalf("未配置服务商时不应报错: %v", err)
	}
	if credentials.ClientID != "" || credentials.ClientSecret != "" {
		t.Errorf("未配置服务商却拿到了 client 凭��: %+v", credentials)
	}
	if credentials.AccessToken != "at-only" {
		t.Errorf("AccessToken = %q，期望 at-only", credentials.AccessToken)
	}
}

// TestResolveOAuth2MalformedJSON 覆盖 OAuth2 密文非法 JSON。
func TestResolveOAuth2MalformedJSON(t *testing.T) {
	resolver, svc := credentialTestResolver(t)
	account := credentialTestAccount("gmail", "oauth2")
	account.EncryptedCredentials = encryptCredential(t, svc, "not-json")

	if _, err := resolver.Resolve(account); err == nil {
		t.Fatal("OAuth2 凭据非法 JSON 时应返回 error")
	}
}

// TestResolveDecryptFailureFailsClosed 覆盖解密失败：
// 密文损坏时必须报错，绝不能把密文当明文密码用出去。
func TestResolveDecryptFailureFailsClosed(t *testing.T) {
	resolver, _ := credentialTestResolver(t)
	account := credentialTestAccount("smtp", "password")
	account.EncryptedCredentials = "broken-ciphertext"

	credentials, err := resolver.Resolve(account)
	if err == nil {
		t.Fatalf("解密失败时应返回 error，实际拿到凭据 %+v", credentials)
	}
	if credentials != nil {
		t.Errorf("解密失败时不应返回凭据: %+v", credentials)
	}
}

// TestResolveIMAPServerConfig 覆盖 IMAP 服务端配置与加密方式映射。
func TestResolveIMAPServerConfig(t *testing.T) {
	cases := []struct {
		name       string
		encryption string
		wantTLS    bool
		wantStart  bool
	}{
		{name: "ssl", encryption: "ssl", wantTLS: true},
		{name: "空值默认 TLS", encryption: "", wantTLS: true},
		{name: "starttls", encryption: "starttls", wantStart: true},
		{name: "none 明文", encryption: "none"},
		{name: "未知值按 TLS 处理", encryption: "weird", wantTLS: true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			resolver, svc := credentialTestResolver(t)
			account := credentialTestAccount("imap", "password")
			account.EncryptedCredentials = encryptCredential(t, svc, "pw")
			account.ProviderRef = &model.Provider{
				Name:           "自建 IMAP",
				IMAPHost:       "imap.example.com",
				IMAPPort:       993,
				IMAPEncryption: c.encryption,
			}

			credentials, err := resolver.Resolve(account)
			if err != nil {
				t.Fatalf("解析 IMAP 凭据失败: %v", err)
			}
			if credentials.Host != "imap.example.com" || credentials.Port != 993 {
				t.Errorf("服务端配置错误: host=%q port=%d", credentials.Host, credentials.Port)
			}
			if credentials.TLS != c.wantTLS || credentials.StartTLS != c.wantStart {
				t.Errorf("加密方式映射错误: TLS=%v StartTLS=%v，期望 %v/%v",
					credentials.TLS, credentials.StartTLS, c.wantTLS, c.wantStart)
			}
		})
	}
}

// TestResolveIMAPRequiresServerConfig 覆盖 IMAP 缺服务端配置：
// 没有 host/port 时必须报错，否则会去连空地址。
func TestResolveIMAPRequiresServerConfig(t *testing.T) {
	resolver, svc := credentialTestResolver(t)
	account := credentialTestAccount("imap", "password")
	account.EncryptedCredentials = encryptCredential(t, svc, "pw")

	if _, err := resolver.Resolve(account); err == nil {
		t.Fatal("IMAP 缺少 host/port 时应返回 error")
	}
}

// TestResolvePOP3ServerConfig 覆盖 POP3 服务端配置。
func TestResolvePOP3ServerConfig(t *testing.T) {
	resolver, svc := credentialTestResolver(t)
	account := credentialTestAccount("pop3", "password")
	account.EncryptedCredentials = encryptCredential(t, svc, "pw")
	account.ProviderRef = &model.Provider{
		Name:           "自建 POP3",
		POP3Host:       "pop3.example.com",
		POP3Port:       995,
		POP3Encryption: "starttls",
	}

	credentials, err := resolver.Resolve(account)
	if err != nil {
		t.Fatalf("解析 POP3 凭据失败: %v", err)
	}
	if credentials.Host != "pop3.example.com" || credentials.Port != 995 {
		t.Errorf("POP3 服务端配置错误: host=%q port=%d", credentials.Host, credentials.Port)
	}
	if !credentials.StartTLS {
		t.Error("POP3 starttls 未生效")
	}
}

// TestResolveRewritesLegacyLinuxDoHost 覆盖历史域名修正：
// mail.linuxdo.org 已迁移到 mail.linux.do，解析时必须改写。
func TestResolveRewritesLegacyLinuxDoHost(t *testing.T) {
	resolver, svc := credentialTestResolver(t)
	account := credentialTestAccount("imap", "password")
	account.EncryptedCredentials = encryptCredential(t, svc, "pw")
	account.ProviderRef = &model.Provider{
		Name:           "linuxdo",
		IMAPHost:       "mail.linuxdo.org",
		IMAPPort:       993,
		IMAPEncryption: "ssl",
	}

	credentials, err := resolver.Resolve(account)
	if err != nil {
		t.Fatalf("解析凭据失败: %v", err)
	}
	if credentials.Host != "mail.linux.do" {
		t.Errorf("Host = %q，期望 mail.linux.do", credentials.Host)
	}
}

// TestResolveWithEncryptorFallback 覆盖只用 Encryptor 的构造方式（不持有 crypto.Service）。
func TestResolveWithEncryptorFallback(t *testing.T) {
	resolver := NewCredentialResolverWithEncryptor(stubEncryptor{}, nil)
	account := credentialTestAccount("imap", "password")
	account.EncryptedCredentials = "enc:pw"
	account.ProviderRef = &model.Provider{
		Name:           "自建 IMAP",
		IMAPHost:       "imap.example.com",
		IMAPPort:       993,
		IMAPEncryption: "ssl",
	}

	credentials, err := resolver.Resolve(account)
	if err != nil {
		t.Fatalf("解析凭据失败: %v", err)
	}
	if credentials.Password != "pw" {
		t.Errorf("Password = %q，期望 pw（stubEncryptor 去掉 enc: 前缀）", credentials.Password)
	}
}

// TestParseTokenExpiry 覆盖过期时间解析：只接受 RFC3339 字符串，其余一律零值。
func TestParseTokenExpiry(t *testing.T) {
	valid := "2026-10-27T12:00:00Z"

	cases := []struct {
		name  string
		value any
		want  time.Time
	}{
		{name: "合法 RFC3339", value: valid, want: time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC)},
		{name: "空字符串", value: "", want: time.Time{}},
		{name: "非法字符串", value: "not-a-time", want: time.Time{}},
		{name: "数字时间戳", value: float64(1700000000), want: time.Time{}},
		{name: "nil", value: nil, want: time.Time{}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := parseTokenExpiry(c.value); !got.Equal(c.want) {
				t.Errorf("parseTokenExpiry(%v) = %v，期望 %v", c.value, got, c.want)
			}
		})
	}
}
