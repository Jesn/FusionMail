package storage

import (
	"fmt"
)

// NewProvider 创建存储提供者
func NewProvider(config *Config) (Provider, error) {
	switch config.Type {
	case "local", "":
		return NewLocalProvider(config.LocalPath, config.BaseURL)
	case "s3":
		// TODO: 实现 S3 存储
		return nil, fmt.Errorf("S3 storage not implemented yet")
	case "oss":
		// TODO: 实现 OSS 存储
		return nil, fmt.Errorf("OSS storage not implemented yet")
	default:
		return nil, fmt.Errorf("unsupported storage type: %s", config.Type)
	}
}
