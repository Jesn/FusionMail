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
)

// TOTPService TOTP 双因素认证服务
type TOTPService struct {
	issuer string
	digits int
	period int
}

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

// GenerateBackupCodes 生成恢复码
func (s *TOTPService) GenerateBackupCodes(count int) ([]string, error) {
	codes := make([]string, count)
	for i := 0; i < count; i++ {
		code := make([]byte, 4)
		if _, err := rand.Read(code); err != nil {
			return nil, fmt.Errorf("生成恢复码失败: %w", err)
		}
		// 生成 8 位数字恢复码
		codes[i] = fmt.Sprintf("%08d", binary.BigEndian.Uint32(code)%100000000)
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

// ValidateBackupCode 验证恢复码
func (s *TOTPService) ValidateBackupCode(backupCodes []string, code string) (bool, []string) {
	for i, bc := range backupCodes {
		if bc == code {
			// 移除已使用的恢复码
			remaining := append(backupCodes[:i], backupCodes[i+1:]...)
			return true, remaining
		}
	}
	return false, backupCodes
}
