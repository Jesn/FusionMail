package webhook

import (
	"crypto/rand"
	"encoding/hex"
	"net/mail"
	"regexp"
	"strings"
	"time"
)

// ParseEmailAddress 解析邮件地址字符串
// 支持以下格式：
//   - "Name <email@example.com>"
//   - "<email@example.com>"
//   - "email@example.com"
//
// 返回解析后的名称和地址
func ParseEmailAddress(raw string) (name, address string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", ""
	}

	// 尝试使用标准库解析
	addr, err := mail.ParseAddress(raw)
	if err == nil {
		return addr.Name, addr.Address
	}

	// 标准库解析失败，尝试手动解析
	// 格式: "Name <email>" 或 "<email>"
	if idx := strings.LastIndex(raw, "<"); idx != -1 {
		if endIdx := strings.LastIndex(raw, ">"); endIdx > idx {
			address = strings.TrimSpace(raw[idx+1 : endIdx])
			name = strings.TrimSpace(raw[:idx])
			// 移除名称两端的引号
			name = strings.Trim(name, "\"'")
			return name, address
		}
	}

	// 没有尖括号，假设整个字符串就是邮件地址
	// 验证是否像邮件地址
	if strings.Contains(raw, "@") {
		return "", raw
	}

	// 无法解析
	return "", raw
}

// ParseEmailAddressList 解析邮件地址列表
// 支持逗号或分号分隔的多个地址
func ParseEmailAddressList(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}

	// 尝试使用标准库解析
	addrs, err := mail.ParseAddressList(raw)
	if err == nil {
		result := make([]string, 0, len(addrs))
		for _, addr := range addrs {
			result = append(result, addr.Address)
		}
		return result
	}

	// 标准库解析失败，手动分割
	// 支持逗号和分号分隔
	separators := regexp.MustCompile(`[,;]`)
	parts := separators.Split(raw, -1)

	result := make([]string, 0, len(parts))
	for _, part := range parts {
		_, addr := ParseEmailAddress(part)
		if addr != "" {
			result = append(result, addr)
		}
	}

	return result
}

// ExtractDomain 从邮件地址中提取域名
// 例如: "user@example.com" -> "example.com"
func ExtractDomain(email string) string {
	email = strings.TrimSpace(email)
	if email == "" {
		return ""
	}

	// 如果是完整格式，先提取地址部分
	_, addr := ParseEmailAddress(email)
	if addr != "" {
		email = addr
	}

	// 提取 @ 后面的部分
	if idx := strings.LastIndex(email, "@"); idx != -1 {
		return strings.ToLower(email[idx+1:])
	}

	return ""
}

// GenerateWebhookSecret 生成随机的 Webhook Secret
// 返回 32 字节的十六进制字符串（64 个字符）
func GenerateWebhookSecret() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		// 如果随机数生成失败，使用时间戳作为备选
		// 这种情况极少发生
		return hex.EncodeToString([]byte(time.Now().String()))[:64]
	}
	return hex.EncodeToString(bytes)
}

// GenerateWebhookSecretShort 生成较短的 Webhook Secret
// 返回 16 字节的十六进制字符串（32 个字符）
func GenerateWebhookSecretShort() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return hex.EncodeToString([]byte(time.Now().String()))[:32]
	}
	return hex.EncodeToString(bytes)
}

// NormalizeEmailAddress 标准化邮件地址
// 转换为小写并去除空格
func NormalizeEmailAddress(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// IsValidEmailAddress 验证邮件地址格式
// 使用简单的正则表达式验证
func IsValidEmailAddress(email string) bool {
	email = strings.TrimSpace(email)
	if email == "" {
		return false
	}

	// 简单的邮件地址正则
	// 不追求完美匹配 RFC 5322，只做基本验证
	pattern := `^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`
	matched, _ := regexp.MatchString(pattern, email)
	return matched
}

// TruncateString 截断字符串到指定长度
// 如果字符串超过最大长度，会在末尾添加 "..."
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// SanitizeSubject 清理邮件主题
// 移除换行符和多余空格
func SanitizeSubject(subject string) string {
	// 替换换行符为空格
	subject = strings.ReplaceAll(subject, "\r\n", " ")
	subject = strings.ReplaceAll(subject, "\n", " ")
	subject = strings.ReplaceAll(subject, "\r", " ")

	// 压缩多个空格为一个
	space := regexp.MustCompile(`\s+`)
	subject = space.ReplaceAllString(subject, " ")

	return strings.TrimSpace(subject)
}

// ExtractSnippet 从邮件正文提取摘要
// 优先使用纯文本，如果没有则从 HTML 提取
func ExtractSnippet(textBody, htmlBody string, maxLen int) string {
	var content string

	if textBody != "" {
		content = textBody
	} else if htmlBody != "" {
		// 简单移除 HTML 标签
		content = StripHTMLTags(htmlBody)
	}

	if content == "" {
		return ""
	}

	// 清理空白字符
	content = SanitizeSubject(content)

	return TruncateString(content, maxLen)
}

// StripHTMLTags 移除 HTML 标签
// 简单实现，不处理复杂的 HTML 实体
func StripHTMLTags(html string) string {
	// 移除 script 和 style 标签及其内容
	scriptPattern := regexp.MustCompile(`(?i)<script[^>]*>[\s\S]*?</script>`)
	html = scriptPattern.ReplaceAllString(html, "")

	stylePattern := regexp.MustCompile(`(?i)<style[^>]*>[\s\S]*?</style>`)
	html = stylePattern.ReplaceAllString(html, "")

	// 移除所有 HTML 标签
	tagPattern := regexp.MustCompile(`<[^>]*>`)
	html = tagPattern.ReplaceAllString(html, " ")

	// 解码常见的 HTML 实体
	html = strings.ReplaceAll(html, "&nbsp;", " ")
	html = strings.ReplaceAll(html, "&amp;", "&")
	html = strings.ReplaceAll(html, "&lt;", "<")
	html = strings.ReplaceAll(html, "&gt;", ">")
	html = strings.ReplaceAll(html, "&quot;", "\"")
	html = strings.ReplaceAll(html, "&#39;", "'")

	return html
}

// ParseRFC822Date 解析 RFC822 格式的日期
// 支持多种常见的日期格式
func ParseRFC822Date(dateStr string) *time.Time {
	dateStr = strings.TrimSpace(dateStr)
	if dateStr == "" {
		return nil
	}

	// 常见的日期格式
	formats := []string{
		time.RFC1123Z,
		time.RFC1123,
		time.RFC822Z,
		time.RFC822,
		time.RFC3339,
		"Mon, 2 Jan 2006 15:04:05 -0700",
		"Mon, 2 Jan 2006 15:04:05 MST",
		"2 Jan 2006 15:04:05 -0700",
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02 15:04:05",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return &t
		}
	}

	return nil
}

// BuildWebhookURL 构建 Webhook URL
// baseURL: FusionMail 的基础 URL，如 "https://mail.example.com"
// providerType: 服务商类型，如 "cloudflare_temp_email"
func BuildWebhookURL(baseURL, providerType string) string {
	baseURL = strings.TrimSuffix(baseURL, "/")
	return baseURL + "/api/v1/webhook/" + providerType
}

// MaskSecret 遮蔽 Secret 用于日志输出
// 只显示前 4 位和后 4 位
func MaskSecret(secret string) string {
	if len(secret) <= 8 {
		return "****"
	}
	return secret[:4] + "****" + secret[len(secret)-4:]
}
