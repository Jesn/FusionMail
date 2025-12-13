package email

import (
	"strings"
)

// ParseRecipients 解析收件人字符串
// 支持逗号和分号分隔，去除空白字符和重复地址
// Requirements: 6.2
func ParseRecipients(input string) []string {
	if strings.TrimSpace(input) == "" {
		return []string{}
	}

	// 统一分隔符：将分号替换为逗号
	input = strings.ReplaceAll(input, ";", ",")

	// 按逗号分割
	parts := strings.Split(input, ",")

	// 使用 map 去重
	seen := make(map[string]bool)
	result := make([]string, 0, len(parts))

	for _, part := range parts {
		// 去除空白
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// 提取纯邮箱地址（处理 "Name <email>" 格式）
		email := ExtractEmailAddress(part)
		if email == "" {
			continue
		}

		// 规范化为小写
		email = strings.ToLower(email)

		// 去重
		if !seen[email] {
			seen[email] = true
			result = append(result, email)
		}
	}

	return result
}

// ParseAndValidateRecipients 解析并验证收件人字符串
// 返回有效的收件人列表和无效的地址列表
// Requirements: 6.2, 1.2, 1.3
func ParseAndValidateRecipients(input string) (valid []string, invalid []string) {
	recipients := ParseRecipients(input)

	valid = make([]string, 0, len(recipients))
	invalid = make([]string, 0)

	for _, email := range recipients {
		if IsValidEmail(email) {
			valid = append(valid, email)
		} else {
			invalid = append(invalid, email)
		}
	}

	return valid, invalid
}

// JoinRecipients 将收件人列表合并为字符串
// 使用逗号分隔
func JoinRecipients(recipients []string) string {
	return strings.Join(recipients, ", ")
}

// SplitRecipientsByType 根据类型分割收件人
// 用于处理 To、Cc、Bcc 字段
type RecipientsByType struct {
	To  []string
	Cc  []string
	Bcc []string
}

// ParseAllRecipients 解析所有类型的收件人
// 返回分类后的收件人列表
func ParseAllRecipients(to, cc, bcc string) RecipientsByType {
	return RecipientsByType{
		To:  ParseRecipients(to),
		Cc:  ParseRecipients(cc),
		Bcc: ParseRecipients(bcc),
	}
}

// GetAllRecipients 获取所有收件人（去重）
func (r RecipientsByType) GetAllRecipients() []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(r.To)+len(r.Cc)+len(r.Bcc))

	for _, email := range r.To {
		if !seen[email] {
			seen[email] = true
			result = append(result, email)
		}
	}

	for _, email := range r.Cc {
		if !seen[email] {
			seen[email] = true
			result = append(result, email)
		}
	}

	for _, email := range r.Bcc {
		if !seen[email] {
			seen[email] = true
			result = append(result, email)
		}
	}

	return result
}

// TotalCount 获取收件人总数（不去重）
func (r RecipientsByType) TotalCount() int {
	return len(r.To) + len(r.Cc) + len(r.Bcc)
}

// IsEmpty 检查是否没有任何收件人
func (r RecipientsByType) IsEmpty() bool {
	return len(r.To) == 0 && len(r.Cc) == 0 && len(r.Bcc) == 0
}

// HasTo 检查是否有主要收件人
func (r RecipientsByType) HasTo() bool {
	return len(r.To) > 0
}
