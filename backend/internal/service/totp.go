package service

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"strings"
	"time"

	cryptoutil "fusionmail/pkg/crypto"
)

// TOTPService TOTP 双因素认证服务
type TOTPService struct {
	issuer string
	digits int
	period int
}

const encryptedTOTPSecretPrefix = "enc:v1:"

// NewTOTPService 创建 TOTP 服务
func NewTOTPService(issuer string) *TOTPService {
	return &TOTPService{
		issuer: issuer,
		digits: 6,
		period: 30,
	}
}

// GenerateSecret 生成 TOTP 密钥
func (s *TOTPService) GenerateSecret() (string, error) {
	// 生成 20 字节的随机密钥
	secret := make([]byte, 20)
	if _, err := rand.Read(secret); err != nil {
		return "", fmt.Errorf("生成随机密钥失败: %w", err)
	}

	// Base32 编码（去除填充）
	encoded := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secret)
	return strings.ToUpper(encoded), nil
}

// EncryptSecret 加密 TOTP 密钥后再持久化
func (s *TOTPService) EncryptSecret(secret string) (string, error) {
	if secret == "" {
		return "", nil
	}

	encryptor, err := cryptoutil.NewEncryptor()
	if err != nil {
		return "", fmt.Errorf("创建 2FA 密钥加密器失败: %w", err)
	}

	encrypted, err := encryptor.Encrypt(secret)
	if err != nil {
		return "", fmt.Errorf("加密 2FA 密钥失败: %w", err)
	}

	return encryptedTOTPSecretPrefix + encrypted, nil
}

// DecryptSecret 解密持久化的 TOTP 密钥，兼容历史明文记录
func (s *TOTPService) DecryptSecret(storedSecret string) (string, error) {
	if storedSecret == "" {
		return "", nil
	}
	if !strings.HasPrefix(storedSecret, encryptedTOTPSecretPrefix) {
		return storedSecret, nil
	}

	encryptor, err := cryptoutil.NewEncryptor()
	if err != nil {
		return "", fmt.Errorf("创建 2FA 密钥加密器失败: %w", err)
	}

	secret, err := encryptor.Decrypt(strings.TrimPrefix(storedSecret, encryptedTOTPSecretPrefix))
	if err != nil {
		return "", fmt.Errorf("解密 2FA 密钥失败: %w", err)
	}

	return secret, nil
}

func (s *TOTPService) IsEncryptedSecret(storedSecret string) bool {
	return strings.HasPrefix(storedSecret, encryptedTOTPSecretPrefix)
}

// GenerateBackupCodes 生成恢复码
func (s *TOTPService) GenerateBackupCodes(count int) ([]string, error) {
	if count <= 0 {
		return []string{}, nil
	}

	codes := make([]string, 0, count)
	seen := make(map[string]struct{}, count)
	for len(codes) < count {
		code := make([]byte, 4)
		if _, err := rand.Read(code); err != nil {
			return nil, fmt.Errorf("生成恢复码失败: %w", err)
		}
		codeString := fmt.Sprintf("%08d", binary.BigEndian.Uint32(code)%100000000)
		if _, exists := seen[codeString]; exists {
			continue
		}
		seen[codeString] = struct{}{}
		codes = append(codes, codeString)
	}
	return codes, nil
}

// GenerateOTPAuthURL 生成 OTP Auth URL（用于二维码）
func (s *TOTPService) GenerateOTPAuthURL(secret, accountName string) string {
	return fmt.Sprintf(
		"otpauth://totp/%s:%s?secret=%s&issuer=%s&algorithm=SHA1&digits=%d&period=%d",
		s.issuer,
		accountName,
		secret,
		s.issuer,
		s.digits,
		s.period,
	)
}

// ValidateCode 验证 TOTP 码
func (s *TOTPService) ValidateCode(secret, code string) bool {
	// 允许前后一个时间窗口的偏差
	currentTime := time.Now().Unix()
	for _, offset := range []int64{-1, 0, 1} {
		timestamp := currentTime + offset*int64(s.period)
		expectedCode := s.generateCode(secret, timestamp)
		if expectedCode == code {
			return true
		}
	}
	return false
}

// generateCode 生成指定时间戳的 TOTP 码
func (s *TOTPService) generateCode(secret string, timestamp int64) string {
	// 解码 Base32 密钥
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(secret))
	if err != nil {
		return ""
	}

	// 计算时间计数器
	counter := uint64(timestamp / int64(s.period))

	// 将计数器转换为 8 字节大端序
	counterBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(counterBytes, counter)

	// HMAC-SHA1
	h := hmac.New(sha1.New, key)
	h.Write(counterBytes)
	hash := h.Sum(nil)

	// 动态截断
	offset := hash[len(hash)-1] & 0x0f
	truncated := binary.BigEndian.Uint32(hash[offset:offset+4]) & 0x7fffffff

	// 取模得到指定位数的码
	otp := truncated % uint32(pow10(s.digits))

	return fmt.Sprintf("%0*d", s.digits, otp)
}

// pow10 计算 10 的 n 次方
func pow10(n int) int {
	result := 1
	for i := 0; i < n; i++ {
		result *= 10
	}
	return result
}

// HashBackupCodes 对恢复码做单向哈希后再持久化
func (s *TOTPService) HashBackupCodes(backupCodes []string) ([]string, error) {
	hashedCodes := make([]string, len(backupCodes))
	for i, code := range backupCodes {
		if isHashedBackupCode(code) {
			hashedCodes[i] = code
			continue
		}

		hashedCode, err := cryptoutil.HashPassword(code)
		if err != nil {
			return nil, fmt.Errorf("哈希恢复码失败: %w", err)
		}
		hashedCodes[i] = hashedCode
	}
	return hashedCodes, nil
}

// ValidateBackupCode 验证恢复码并返回移除已使用码后的剩余哈希
func (s *TOTPService) ValidateBackupCode(backupCodes []string, code string) (bool, []string, error) {
	remaining := make([]string, 0, len(backupCodes))
	valid := false

	for _, backupCode := range backupCodes {
		if !valid && backupCodeMatches(backupCode, code) {
			valid = true
			continue
		}
		remaining = append(remaining, backupCode)
	}

	if !valid {
		return false, backupCodes, nil
	}

	hashedRemaining, err := s.HashBackupCodes(remaining)
	if err != nil {
		return false, nil, err
	}
	return true, hashedRemaining, nil
}

func backupCodeMatches(storedCode, code string) bool {
	if isHashedBackupCode(storedCode) {
		return cryptoutil.VerifyPassword(code, storedCode)
	}
	return storedCode == code
}

func isHashedBackupCode(storedCode string) bool {
	return strings.HasPrefix(storedCode, "$2a$") ||
		strings.HasPrefix(storedCode, "$2b$") ||
		strings.HasPrefix(storedCode, "$2y$")
}
