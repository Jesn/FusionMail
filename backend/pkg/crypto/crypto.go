package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"os"

	"golang.org/x/crypto/bcrypt"
)

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
		keyStr = "fusionmail-default-key-32-bytes"
	}

	// 确保密钥长度为 32 字节（AES-256）
	key := []byte(keyStr)
	if len(key) < 32 {
		// 填充到 32 字节
		padded := make([]byte, 32)
		copy(padded, key)
		key = padded
	} else if len(key) > 32 {
		// 截断到 32 字节
		key = key[:32]
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
