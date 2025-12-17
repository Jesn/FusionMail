package service

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"fusionmail/internal/model"
)

// DedupeKeyGenerator 去重标识生成器
// 用于生成稳定的邮件去重标识，解决以下问题：
// 1. IMAP UID 可能因 UIDVALIDITY 变化而改变
// 2. 部分系统邮件（如 139 邮箱的 10086 通知）没有 Message-ID
type DedupeKeyGenerator struct{}

// NewDedupeKeyGenerator 创建去重标识生成器
func NewDedupeKeyGenerator() *DedupeKeyGenerator {
	return &DedupeKeyGenerator{}
}

// 系统通知域名列表
// 这些域名的邮件通常没有标准的 Message-ID，需要强制使用 hash 方式生成 dedupe_key
var systemNotificationDomains = []string{
	"@139.com",
	"@10086.cn",
	"@10086.com",
	"@chinamobile.com",
}

// Generate 生成去重标识
// 策略：
// 1. 如果是系统通知域名，强制使用 hash 方式（即使有 Message-ID）
// 2. 如果有有效的 Message-ID，使用 "mid:" 前缀
// 3. 如果没有 Message-ID，使用 "hash:" 前缀 + SHA256(from|subject|sent_at)
func (g *DedupeKeyGenerator) Generate(email *model.Email) string {
	// 检查是否为系统通知域名
	if g.IsSystemNotificationDomain(email.FromAddress) {
		return g.generateHashKey(email)
	}

	// 有 Message-ID 时使用 mid: 前缀
	if email.MessageID != "" {
		return g.generateMessageIDKey(email.MessageID)
	}

	// 无 Message-ID 时使用 hash: 前缀
	return g.generateHashKey(email)
}

// GenerateFromRaw 从原始数据生成去重标识
// 用于在邮件对象创建之前生成 dedupe_key
func (g *DedupeKeyGenerator) GenerateFromRaw(messageID, fromAddress, subject string, sentAt time.Time) string {
	// 检查是否为系统通知域名
	if g.IsSystemNotificationDomain(fromAddress) {
		return g.generateHashKeyFromRaw(fromAddress, subject, sentAt)
	}

	// 有 Message-ID 时使用 mid: 前缀
	if messageID != "" {
		return g.generateMessageIDKey(messageID)
	}

	// 无 Message-ID 时使用 hash: 前缀
	return g.generateHashKeyFromRaw(fromAddress, subject, sentAt)
}

// IsSystemNotificationDomain 检查是否为系统通知域名
func (g *DedupeKeyGenerator) IsSystemNotificationDomain(fromAddress string) bool {
	fromLower := strings.ToLower(fromAddress)
	for _, domain := range systemNotificationDomains {
		if strings.HasSuffix(fromLower, strings.ToLower(domain)) {
			return true
		}
	}
	return false
}

// generateMessageIDKey 生成基于 Message-ID 的去重标识
func (g *DedupeKeyGenerator) generateMessageIDKey(messageID string) string {
	// 清理 Message-ID（去除尖括号和空格）
	cleanID := strings.TrimSpace(messageID)
	cleanID = strings.TrimPrefix(cleanID, "<")
	cleanID = strings.TrimSuffix(cleanID, ">")

	// 如果 Message-ID 过长，使用 hash
	if len(cleanID) > 60 {
		hash := sha256.Sum256([]byte(cleanID))
		return "mid:" + hex.EncodeToString(hash[:])[:32]
	}

	return "mid:" + cleanID
}

// generateHashKey 生成基于 hash 的去重标识
func (g *DedupeKeyGenerator) generateHashKey(email *model.Email) string {
	return g.generateHashKeyFromRaw(email.FromAddress, email.Subject, email.SentAt)
}

// generateHashKeyFromRaw 从原始数据生成基于 hash 的去重标识
// 使用 from_address|subject|sent_at（精确到分钟）生成 SHA256 hash
func (g *DedupeKeyGenerator) generateHashKeyFromRaw(fromAddress, subject string, sentAt time.Time) string {
	// 标准化输入
	from := strings.ToLower(strings.TrimSpace(fromAddress))
	subj := strings.TrimSpace(subject)
	// 精确到分钟，避免秒级差异导致的重复
	timeStr := sentAt.UTC().Format("2006-01-02T15:04")

	// 生成 hash
	data := fmt.Sprintf("%s|%s|%s", from, subj, timeStr)
	hash := sha256.Sum256([]byte(data))

	// 返回前 32 个字符的 hex 编码
	return "hash:" + hex.EncodeToString(hash[:])[:32]
}

// ValidateDedupeKey 验证去重标识格式
func (g *DedupeKeyGenerator) ValidateDedupeKey(key string) bool {
	if key == "" {
		return false
	}
	return strings.HasPrefix(key, "mid:") || strings.HasPrefix(key, "hash:")
}

// GetKeyType 获取去重标识类型
func (g *DedupeKeyGenerator) GetKeyType(key string) string {
	if strings.HasPrefix(key, "mid:") {
		return "message_id"
	}
	if strings.HasPrefix(key, "hash:") {
		return "hash"
	}
	return "unknown"
}
