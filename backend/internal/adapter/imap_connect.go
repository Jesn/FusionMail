package adapter

import (
	"context"
	"crypto/tls"
	"fmt"
	"strings"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/emersion/go-sasl"
)

// Connect 连接到 IMAP 服务器
func (a *IMAPAdapter) Connect(ctx context.Context) error {
	addr := fmt.Sprintf("%s:%d", a.config.Credentials.Host, a.config.Credentials.Port)

	// 配置 TLS
	tlsConfig := &tls.Config{
		ServerName: a.config.Credentials.Host,
	}

	// 针对特定邮箱服务商的 TLS 配置优化
	host := strings.ToLower(a.config.Credentials.Host)

	// 139 邮箱（中国移动）需要更宽松的 TLS 配置
	if strings.Contains(host, "139.com") {
		fmt.Printf("[IMAP] Detected 139 Mail (China Mobile), using relaxed TLS config\n")
		tlsConfig.InsecureSkipVerify = true     // 跳过证书验证
		tlsConfig.MinVersion = tls.VersionTLS10 // 支持较旧的 TLS 版本
		tlsConfig.MaxVersion = tls.VersionTLS12 // 限制最高版本为 TLS 1.2
		// 关键：指定兼容的加密套件
		tlsConfig.CipherSuites = []uint16{
			tls.TLS_RSA_WITH_AES_128_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_AES_128_CBC_SHA256,
			tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		}
	}

	// QQ 邮箱
	if strings.Contains(host, "qq.com") {
		fmt.Printf("[IMAP] Detected QQ mail server, using relaxed TLS config\n")
		tlsConfig.MinVersion = tls.VersionTLS10
	}

	// 163/126 邮箱（网易）
	if strings.Contains(host, "163.com") || strings.Contains(host, "126.com") {
		fmt.Printf("[IMAP] Detected NetEase mail server, using relaxed TLS config\n")
		tlsConfig.MinVersion = tls.VersionTLS10
	}

	// 189 邮箱（中国电信）
	if strings.Contains(host, "189.cn") {
		fmt.Printf("[IMAP] Detected 189 Mail (China Telecom), using relaxed TLS config\n")
		tlsConfig.MinVersion = tls.VersionTLS10
	}

	// 创建 IMAP 客户端选项
	options := &imapclient.Options{
		TLSConfig: tlsConfig,
	}

	// 连接到服务器
	client, err := imapclient.DialTLS(addr, options)
	if err != nil {
		return fmt.Errorf("failed to connect to IMAP server: %w", err)
	}

	a.client = client

	// 登录
	if err := a.login(ctx); err != nil {
		a.client.Close()
		return err
	}

	return nil
}

// login 登录到 IMAP 服务器
func (a *IMAPAdapter) login(ctx context.Context) error {
	email := a.config.Credentials.Email
	password := a.config.Credentials.Password

	if email == "" || password == "" {
		return fmt.Errorf("email and password are required")
	}

	// 发送 IMAP ID 信息（某些服务器如 163 需要这个来识别客户端）
	clientID := &imap.IDData{
		Name:       "FusionMail",
		Version:    "1.0.0",
		Vendor:     "FusionMail",
		SupportURL: "https://fusionmail.com",
	}

	fmt.Printf("[IMAP] Sending ID command with client info...\n")
	_, err := a.client.ID(clientID).Wait()
	if err != nil {
		// ID 命令失败不应该阻止登录，只记录警告
		fmt.Printf("[IMAP] Warning: ID command failed: %v\n", err)
	} else {
		fmt.Printf("[IMAP] ID command sent successfully\n")
	}

	// 登录
	fmt.Printf("[IMAP] Logging in as %s...\n", email)

	// 优先尝试 SASL PLAIN 认证（更标准，Outlook 等现代服务更喜欢）
	// 尝试方式 1: 仅提供 authcid (username)，authzid (identity) 为空
	// 这是最常见的方式
	err = a.client.Authenticate(sasl.NewPlainClient("", email, password))
	if err != nil {
		fmt.Printf("[IMAP] SASL PLAIN (no identity) failed: %v. Retrying with identity...\n", err)

		// 尝试方式 2: authzid 和 authcid 都设置为 email
		// 某些服务器可能需要显式指定 identity
		err2 := a.client.Authenticate(sasl.NewPlainClient(email, email, password))
		if err2 != nil {
			fmt.Printf("[IMAP] SASL PLAIN (with identity) failed: %v. Retrying with LOGIN command...\n", err2)

			// 回退到 LOGIN 命令
			if loginErr := a.client.Login(email, password).Wait(); loginErr != nil {
				// 如果两者都失败，返回 LOGIN 的错误（通常更具描述性）
				// 但也要包含 SASL 的错误信息以便调试
				errMsg := fmt.Sprintf("failed to login (SASL: %v, LOGIN: %v)", err, loginErr)

				// 如果是 Outlook 或 Gmail，且错误提示登录失败，建议使用应用专用密码
				if (a.config.Provider == "outlook" || a.config.Provider == "gmail") &&
					(strings.Contains(loginErr.Error(), "NO") || strings.Contains(loginErr.Error(), "failed")) {
					errMsg += ". Hint: For Outlook/Gmail, you may need to use an App Password if 2FA is enabled, or use OAuth2."
				}

				return fmt.Errorf("%s", errMsg)
			}
		}
	}

	fmt.Printf("[IMAP] Login successful\n")

	return nil
}

// Disconnect 断开连接
func (a *IMAPAdapter) Disconnect() error {
	if a.client != nil {
		return a.client.Close()
	}
	return nil
}

// TestConnection 测试连接
func (a *IMAPAdapter) TestConnection(ctx context.Context) error {
	// 尝试连接
	if err := a.Connect(ctx); err != nil {
		return err
	}

	// 尝试列出邮箱
	listCmd := a.client.List("", "*", nil)
	for {
		mbox := listCmd.Next()
		if mbox == nil {
			break
		}
	}

	if err := listCmd.Close(); err != nil {
		return fmt.Errorf("failed to list mailboxes: %w", err)
	}

	// 断开连接
	return a.Disconnect()
}
