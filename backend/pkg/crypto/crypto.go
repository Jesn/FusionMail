package crypto

import (
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// DefaultEncryptionKey 仅允许开发环境兜底使用。
//
// 必须恰好 32 字节：deriveKey 不再接受长度不符的密钥（见其注释）。
// 历史上这里是 31 字节的 "fusionmail-default-key-32-bytes"，被静默补零后使用。
const DefaultEncryptionKey = "fusionmail-dev-only-key-32bytes!"

// IsDefaultEncryptionKey 判断是否使用了不安全的默认加密密钥
func IsDefaultEncryptionKey(key string) bool {
	trimmed := strings.TrimSpace(key)
	return trimmed == "" || trimmed == DefaultEncryptionKey
}

// deriveKey 把配置里的密钥字符串规范化为 32 字节 AES-256 密钥。
//
// 拒绝静默填��/截断：补零后的短密钥（例如 ENCRYPTION_KEY=abc）与真正的
// 32 字节随机密钥在长度上无法区分，会让弱密钥被当成合规密钥长期使用。
// 长度不符一律报错，由调用方在启动阶段快速失败。
//
// 接受的形态：
//   - 恰好 32 字节的原始字符串
//   - 64 个十六进制字符（解码为 32 字节）
func deriveKey(keyStr string) ([]byte, error) {
	keyStr = strings.TrimSpace(keyStr)
	if keyStr == "" {
		return nil, errors.New("ENCRYPTION_KEY 为空")
	}

	if len(keyStr) == 64 {
		if decoded, err := hex.DecodeString(keyStr); err == nil {
			return decoded, nil
		}
	}

	if len(keyStr) != 32 {
		return nil, fmt.Errorf(
			"ENCRYPTION_KEY 必须恰好 32 字节或 64 位十六进制（当前 %d 字节）；"+
				"拒绝静默填充/截断，因为补零后的弱密钥与强密钥无法区分",
			len(keyStr))
	}

	return []byte(keyStr), nil
}

// Encryptor 加密器接口
type Encryptor interface {
	Encrypt(plaintext string) (string, error)
	Decrypt(ciphertext string) (string, error)
}

// aesEncryptor AES 加密器。
//
// 内部持有一个密钥环而不是单把密钥：密文带 key-id，支持 ENCRYPTION_KEY_PREVIOUS
// 过渡期解密历史密文，使密钥轮换成为可能。
type aesEncryptor struct {
	ring *KeyRing
}

// NewEncryptor 从环境变量创建加密器实例。
//
//	ENCRYPTION_KEY            当前密钥（加密用）
//	ENCRYPTION_KEY_PREVIOUS   历史密钥，逗号分隔（仅解密用，轮换过渡期需要）
func NewEncryptor() (Encryptor, error) {
	keyStr := strings.TrimSpace(os.Getenv("ENCRYPTION_KEY"))
	if keyStr == "" {
		// 开发环境使用默认密钥
		keyStr = DefaultEncryptionKey
	}

	ring, err := NewKeyRing(keyStr, splitPreviousKeys(os.Getenv("ENCRYPTION_KEY_PREVIOUS"))...)
	if err != nil {
		return nil, err
	}
	return &aesEncryptor{ring: ring}, nil
}

// splitPreviousKeys 解析逗号分隔的历史密钥列表。
func splitPreviousKeys(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

// NewEncryptorWithPrevious 用显式密钥构造加密器（供测试与轮换工具使用）。
func NewEncryptorWithPrevious(current string, previous ...string) (Encryptor, error) {
	ring, err := NewKeyRing(current, previous...)
	if err != nil {
		return nil, err
	}
	return &aesEncryptor{ring: ring}, nil
}

// Encrypt 加密字符串（密文带 key-id 前缀）
func (e *aesEncryptor) Encrypt(plaintext string) (string, error) {
	return e.ring.Encrypt(plaintext)
}

// Decrypt 解密字符串（兼容无 key-id 的历史密文）
func (e *aesEncryptor) Decrypt(ciphertext string) (string, error) {
	return e.ring.Decrypt(ciphertext)
}

// CurrentKeyID 返回当前密钥标识，供运维确认轮换是否生效。
func (e *aesEncryptor) CurrentKeyID() string { return e.ring.CurrentKeyID() }

// HashPassword 使用 bcrypt 哈希密码
func HashPassword(password string) (string, error) {
	// 使用 cost 10（默认值）
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// VerifyPassword 验证密码
func VerifyPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// Service 加密服务
type Service struct {
	encryptor Encryptor
}

// NewService 创建加密服务实例
func NewService(key string) (*Service, error) {
	ring, err := NewKeyRing(key)
	if err != nil {
		return nil, err
	}

	return &Service{encryptor: &aesEncryptor{ring: ring}}, nil
}

// Encrypt 加密数据
func (s *Service) Encrypt(data []byte) (string, error) {
	return s.encryptor.Encrypt(string(data))
}

// Decrypt 解密数据
func (s *Service) Decrypt(ciphertext string) ([]byte, error) {
	plaintext, err := s.encryptor.Decrypt(ciphertext)
	if err != nil {
		return nil, err
	}
	return []byte(plaintext), nil
}
