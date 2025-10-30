package storage

import (
	"context"
	"io"
)

// Provider 存储提供者接口
type Provider interface {
	// Upload 上传文件
	// path: 文件路径（相对路径）
	// reader: 文件内容
	// contentType: 文件 MIME 类型
	// 返回：文件 URL 和错误
	Upload(ctx context.Context, path string, reader io.Reader, contentType string) (string, error)

	// Download 下载文件
	// path: 文件路径
	// 返回：文件内容和错误
	Download(ctx context.Context, path string) (io.ReadCloser, error)

	// Delete 删除文件
	// path: 文件路径
	Delete(ctx context.Context, path string) error

	// GetURL 获取文件访问 URL
	// path: 文件路径
	// 返回：文件 URL
	GetURL(ctx context.Context, path string) (string, error)

	// Exists 检查文件是否存在
	// path: 文件路径
	Exists(ctx context.Context, path string) (bool, error)
}

// Config 存储配置
type Config struct {
	Type      string // local, s3, oss
	LocalPath string // 本地存储路径
	BaseURL   string // 基础 URL（用于生成访问链接）

	// S3 配置
	S3Region    string
	S3Bucket    string
	S3AccessKey string
	S3SecretKey string

	// OSS 配置
	OSSEndpoint  string
	OSSBucket    string
	OSSAccessKey string
	OSSSecretKey string
}
