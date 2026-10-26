package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
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

// aesEncryptor AES 加密器
type aesEncryptor struct {
	key []byte
}

// NewEncryptor 创建加密器实例
func NewEncryptor() (Encryptor, error) {
	// 从环境变量获取加密密钥
	keyStr := os.Getenv("ENCRYPTION_KEY")
	if keyStr == "" {
		// 开发环境使用默认密钥
		keyStr = DefaultEncryptionKey
	}

	key, err := deriveKey(keyStr)
	if err != nil {
		return nil, err
	}

	return &aesEncryptor{key: key}, nil
}

// Encrypt 加密字符串
func (e *aesEncryptor) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	// 创建 AES cipher
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", err
	}

	// 创建 GCM
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// 生成随机 nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	// 加密
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	// 返回 base64 编码的结果
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt 解密字符串
func (e *aesEncryptor) Decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	// base64 解码
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	// 创建 AES cipher
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", err
	}

	// 创建 GCM
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// 检查数据长度
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	// 分离 nonce 和 ciphertext
	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]

	// 解密
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

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
	keyBytes, err := deriveKey(key)
	if err != nil {
		return nil, err
	}

	encryptor := &aesEncryptor{key: keyBytes}
	return &Service{encryptor: encryptor}, nil
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
