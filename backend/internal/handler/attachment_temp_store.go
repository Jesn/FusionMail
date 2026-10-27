package handler

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// AttachmentTempStore 管理待发送附件的临时文件生命周期。
//
// 流程：UploadAttachment 把上传文件落盘到 baseDir → 前端在发送请求里
// 回传 temp_path → Send/Reply/Forward 从 temp_path 读回内容映射为
// adapter.OutgoingAttachment → 发送结束（无论成败）删除临时文件。
//
// 安全约束：temp_path 来自客户端 JSON，Read 必须做路径穿越防护
// （拒绝逃出 baseDir 的任何路径），并限制单文件大小。
type AttachmentTempStore struct {
	baseDir     string
	maxFileSize int64
}

// NewAttachmentTempStore 创建临时附件存储，baseDir 不存在时自动创建。
func NewAttachmentTempStore(baseDir string, maxFileSize int64) (*AttachmentTempStore, error) {
	if err := os.MkdirAll(baseDir, 0o700); err != nil {
		return nil, fmt.Errorf("创建附件临时目录失败: %w", err)
	}
	return &AttachmentTempStore{baseDir: baseDir, maxFileSize: maxFileSize}, nil
}

// Save 落盘附件内容，返回仅供本进程使用的 temp_path。
// 文件名不落盘（磁盘名用随机名），避免客户端文件名带来的任何路径问题。
func (s *AttachmentTempStore) Save(content []byte) (string, error) {
	if int64(len(content)) > s.maxFileSize {
		return "", fmt.Errorf("附件大小 %d 超过单文件限制 %d", len(content), s.maxFileSize)
	}
	name, err := randomTempName()
	if err != nil {
		return "", err
	}
	tempPath := filepath.Join(s.baseDir, name)
	if err := os.WriteFile(tempPath, content, 0o600); err != nil {
		return "", fmt.Errorf("写入附件临时文件失败: %w", err)
	}
	return tempPath, nil
}

// Read 读取临时附件内容。tempPath 必须位于 baseDir 内（路径穿越防护），
// 且大小不超过 maxFileSize。
func (s *AttachmentTempStore) Read(tempPath string) ([]byte, error) {
	clean, err := s.resolveWithinBase(tempPath)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(clean)
	if err != nil {
		return nil, fmt.Errorf("附件临时文件不存在: %w", err)
	}
	if info.Size() > s.maxFileSize {
		return nil, fmt.Errorf("附件大小 %d 超过单文件限制 %d", info.Size(), s.maxFileSize)
	}
	content, err := os.ReadFile(clean)
	if err != nil {
		return nil, fmt.Errorf("读取附件临时文件失败: %w", err)
	}
	return content, nil
}

// Remove 删除临时附件（尽力而为，失败不阻断发送流程）。
func (s *AttachmentTempStore) Remove(tempPath string) {
	clean, err := s.resolveWithinBase(tempPath)
	if err != nil {
		return
	}
	_ = os.Remove(clean)
}

// RemoveExpired 清理超过 maxAge 的孤儿临时文件（上传后从未发送的场景）。
// 由 UploadAttachment 顺带触发，避免引入后台任务；失败静默。
func (s *AttachmentTempStore) RemoveExpired(maxAge time.Duration) {
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return
	}
	cutoff := time.Now().Add(-maxAge)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil || info.ModTime().After(cutoff) {
			continue
		}
		_ = os.Remove(filepath.Join(s.baseDir, entry.Name()))
	}
}

// resolveWithinBase 校验 tempPath 解析后确实位于 baseDir 内，返回绝对路径。
func (s *AttachmentTempStore) resolveWithinBase(tempPath string) (string, error) {
	if tempPath == "" {
		return "", fmt.Errorf("temp_path 为空")
	}
	absBase, err := filepath.Abs(s.baseDir)
	if err != nil {
		return "", err
	}
	clean := filepath.Clean(tempPath)
	if !filepath.IsAbs(clean) {
		clean = filepath.Join(absBase, clean)
	}
	abs, err := filepath.Abs(clean)
	if err != nil {
		return "", err
	}
	// 必须严格位于 baseDir 内（而非 baseDir 自身或同名前缀目录）
	if abs != absBase && !strings.HasPrefix(abs, absBase+string(filepath.Separator)) {
		return "", fmt.Errorf("temp_path 越界：%s", tempPath)
	}
	return abs, nil
}

// randomTempName 生成 32 字符随机文件名。
func randomTempName() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("生成随机文件名失败: %w", err)
	}
	return hex.EncodeToString(buf) + ".att", nil
}
