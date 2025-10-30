package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// LocalProvider 本地存储提供者
type LocalProvider struct {
	basePath string // 存储根目录
	baseURL  string // 基础 URL
}

// NewLocalProvider 创建本地存储提供者
func NewLocalProvider(basePath, baseURL string) (*LocalProvider, error) {
	// 确保存储目录存在
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	return &LocalProvider{
		basePath: basePath,
		baseURL:  baseURL,
	}, nil
}

// Upload 上传文件
func (p *LocalProvider) Upload(ctx context.Context, path string, reader io.Reader, contentType string) (string, error) {
	// 构建完整路径
	fullPath := filepath.Join(p.basePath, path)

	// 确保目录存在
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	// 创建文件
	file, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// 写入内容
	if _, err := io.Copy(file, reader); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	// 返回 URL
	url, err := p.GetURL(ctx, path)
	if err != nil {
		return "", err
	}

	return url, nil
}

// Download 下载文件
func (p *LocalProvider) Download(ctx context.Context, path string) (io.ReadCloser, error) {
	fullPath := filepath.Join(p.basePath, path)

	// 打开文件
	file, err := os.Open(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s", path)
		}
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	return file, nil
}

// Delete 删除文件
func (p *LocalProvider) Delete(ctx context.Context, path string) error {
	fullPath := filepath.Join(p.basePath, path)

	// 删除文件
	if err := os.Remove(fullPath); err != nil {
		if os.IsNotExist(err) {
			return nil // 文件不存在，视为删除成功
		}
		return fmt.Errorf("failed to delete file: %w", err)
	}

	// 尝试删除空目录
	dir := filepath.Dir(fullPath)
	_ = os.Remove(dir) // 忽略错误，因为目录可能不为空

	return nil
}

// GetURL 获取文件访问 URL
func (p *LocalProvider) GetURL(ctx context.Context, path string) (string, error) {
	// 如果配置了 baseURL，返回完整 URL
	if p.baseURL != "" {
		return fmt.Sprintf("%s/attachments/%s", p.baseURL, path), nil
	}

	// 否则返回相对路径
	return fmt.Sprintf("/attachments/%s", path), nil
}

// Exists 检查文件是否存在
func (p *LocalProvider) Exists(ctx context.Context, path string) (bool, error) {
	fullPath := filepath.Join(p.basePath, path)

	_, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}
