package email

import (
	"net/mail"
	"regexp"
	"strings"
)

// RFC 5322 兼容的邮箱地址正则表达式（简化版本）
// 完整的 RFC 5322 正则非常复杂，这里使用实用的简化版本
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9.!#$%&'*+/=?^_` + "`" + `{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$`)

// ValidationResult 验证结果
type ValidationResult struct {
	Valid   bool   // 是否有效
	Address string // 规范化后的地址
	Error   string // 错误信息（如果无效）
}

// ValidateEmail 验证单个邮箱地址格式
// 使用 RFC 5322 兼容的验证规则
// Requirements: 1.2, 1.3
func ValidateEmail(email string) ValidationResult {
	// 去除首尾空白
	email = strings.TrimSpace(email)

	// 检查是否为空
	if email == "" {
		return ValidationResult{
			Valid: false,
			Error: "邮箱地址不能为空",
		}
	}

	// 检查长度限制（RFC 5321 规定最大 254 字符）
	if len(email) > 254 {
		return ValidationResult{
			Valid: false,
			Error: "邮箱地址长度超过限制（最大 254 字符）",
		}
	}

	// 使用 Go 标准库的 mail.ParseAddress 进行解析
	// 这可以处理 "Name <email@example.com>" 格式
	addr, err := mail.ParseAddress(email)
	if err != nil {
		// 如果标准库解析失败，尝试使用正则表达式
		if !emailRegex.MatchString(email) {
			return ValidationResult{
				Valid: false,
				Error: "邮箱地址格式不正确",
			}
		}
		// 正则匹配成功，返回原始地址
		return ValidationResult{
			Valid:   true,
			Address: strings.ToLower(email),
		}
	}

	// 标准库解析成功，返回规范化的地址
	return ValidationResult{
		Valid:   true,
		Address: strings.ToLower(addr.Address),
	}
}

// ValidateEmails 批量验证邮箱地址
// 返回所有验证结果和是否全部有效
// Requirements: 1.2, 1.3
func ValidateEmails(emails []string) ([]ValidationResult, bool) {
	results := make([]ValidationResult, len(emails))
	allValid := true

	for i, email := range emails {
		results[i] = ValidateEmail(email)
		if !results[i].Valid {
			allValid = false
		}
	}

	return results, allValid
}

// IsValidEmail 快速检查邮箱地址是否有效
// 返回 true 表示有效，false 表示无效
func IsValidEmail(email string) bool {
	return ValidateEmail(email).Valid
}

// NormalizeEmail 规范化邮箱地址
// 去除空白、转换为小写
func NormalizeEmail(email string) string {
	result := ValidateEmail(email)
	if result.Valid {
		return result.Address
	}
	return strings.ToLower(strings.TrimSpace(email))
}

// ExtractEmailAddress 从可能包含名称的地址中提取纯邮箱地址
// 例如: "John Doe <john@example.com>" -> "john@example.com"
func ExtractEmailAddress(input string) string {
	input = strings.TrimSpace(input)
	if input == "" {
		return ""
	}

	// 尝试使用标准库解析
	addr, err := mail.ParseAddress(input)
	if err != nil {
		// 解析失败，返回原始输入（去除空白并转小写）
		return strings.ToLower(input)
	}

	return strings.ToLower(addr.Address)
}

// FormatEmailAddress 格式化邮箱地址（带名称）
// 例如: ("John Doe", "john@example.com") -> "John Doe <john@example.com>"
func FormatEmailAddress(name, email string) string {
	email = strings.TrimSpace(email)
	name = strings.TrimSpace(name)

	if name == "" {
		return email
	}

	addr := mail.Address{
		Name:    name,
		Address: email,
	}
	return addr.String()
}
