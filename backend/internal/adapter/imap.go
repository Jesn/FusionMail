package adapter

import (
	"fmt"
	"time"

	"github.com/emersion/go-imap/v2/imapclient"
)

// IMAPAdapter IMAP 协议适配器
type IMAPAdapter struct {
	config *Config
	client *imapclient.Client
}

// NewIMAPAdapter 创建 IMAP 适配器实例
func NewIMAPAdapter(config *Config) (*IMAPAdapter, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}

	if config.Credentials == nil {
		return nil, fmt.Errorf("credentials is required")
	}

	// 验证必需的配置
	if config.Credentials.Host == "" {
		return nil, fmt.Errorf("IMAP host is required")
	}

	if config.Credentials.Port == 0 {
		config.Credentials.Port = 993 // 默认 IMAP SSL 端口
	}

	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second // 默认超时 30 秒
	}

	return &IMAPAdapter{
		config: config,
	}, nil
}

// GetProviderType 获取提供商类型
func (a *IMAPAdapter) GetProviderType() string {
	return a.config.Provider
}

// GetProtocol 获取协议类型
func (a *IMAPAdapter) GetProtocol() string {
	return "imap"
}

// UIDSyncState UID 同步状态
type UIDSyncState struct {
	UIDValidity uint32 // IMAP UIDVALIDITY 值
	LastUID     uint32 // 上次同步的最大 UID
	MaxUID      uint32 // 当前邮箱的最大 UID
}
