package email

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"
)

// =============================================================================
// 属性测试：邮箱地址格式验证（Property 2）
// **Feature: email-sending, Property 2: 邮箱地址格式验证**
// **Validates: Requirements 1.3**
// =============================================================================

// TestProperty2_EmailAddressValidation 邮箱地址格式验证属性测试
// 对于任意邮箱地址，验证函数应正确识别有效和无效格式
func TestProperty2_EmailAddressValidation(t *testing.T) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	t.Run("属性2.1: 有效邮箱地址验证", func(t *testing.T) {
		// 生成随机有效邮箱地址并验证
		for i := 0; i < 100; i++ {
			email := generateValidEmail(rng)
			result := ValidateEmail(email)

			if !result.Valid {
				t.Errorf("有效邮箱被判定为无效: %q, 错误: %s", email, result.Error)
			}
		}
	})

	t.Run("属性2.2: 无效邮箱地址验证", func(t *testing.T) {
		invalidEmails := []string{
			"",                      // 空字符串
			"   ",                   // 只有空格
			"invalid",               // 没有 @
			"@example.com",          // 没有本地部分
			"user@",                 // 没有域名
			"user@@example.com",     // 双 @
			"user@.com",             // 域名以点开头
			"user@example.",         // 域名以点结尾
			"user@exam ple.com",     // 域名包含空格
			"user name@example.com", // 本地部分包含空格（无引号）
		}

		for _, email := range invalidEmails {
			result := ValidateEmail(email)
			if result.Valid {
				t.Errorf("无效邮箱被判定为有效: %q", email)
			}
		}
	})

	t.Run("属性2.3: 邮箱地址规范化", func(t *testing.T) {
		// 验证邮箱地址被正确规范化（转小写）
		testCases := []struct {
			input    string
			expected string
		}{
			{"User@Example.COM", "user@example.com"},
			{"USER@EXAMPLE.COM", "user@example.com"},
			{"Test.User@Domain.Org", "test.user@domain.org"},
		}

		for _, tc := range testCases {
			result := ValidateEmail(tc.input)
			if !result.Valid {
				t.Errorf("有效邮箱验证失败: %q", tc.input)
				continue
			}
			if result.Address != tc.expected {
				t.Errorf("邮箱规范化失败: 输入=%q, 期望=%q, 实际=%q",
					tc.input, tc.expected, result.Address)
			}
		}
	})

	t.Run("属性2.4: 带名称的邮箱地址解析", func(t *testing.T) {
		testCases := []struct {
			input    string
			expected string
		}{
			{"John Doe <john@example.com>", "john@example.com"},
			{"<user@example.com>", "user@example.com"},
			{"\"John Doe\" <john@example.com>", "john@example.com"},
		}

		for _, tc := range testCases {
			result := ValidateEmail(tc.input)
			if !result.Valid {
				t.Errorf("带名称邮箱验证失败: %q, 错误: %s", tc.input, result.Error)
				continue
			}
			if result.Address != tc.expected {
				t.Errorf("邮箱地址提取失败: 输入=%q, 期望=%q, 实际=%q",
					tc.input, tc.expected, result.Address)
			}
		}
	})

	t.Run("属性2.5: 邮箱长度限制", func(t *testing.T) {
		// RFC 5321 规定邮箱地址最大 254 字符
		// 生成超长邮箱地址（确保超过 254 字符）
		longLocal := strings.Repeat("a", 250)
		longEmail := longLocal + "@example.com" // 总长度 262 字符

		result := ValidateEmail(longEmail)
		if result.Valid {
			t.Errorf("超长邮箱应被判定为无效: len=%d", len(longEmail))
		}
	})

	t.Run("属性2.6: 批量验证一致性", func(t *testing.T) {
		// 批量验证结果应与单个验证一致
		emails := make([]string, 20)
		for i := 0; i < 10; i++ {
			emails[i] = generateValidEmail(rng)
		}
		for i := 10; i < 20; i++ {
			emails[i] = generateInvalidEmail(rng)
		}

		results, allValid := ValidateEmails(emails)

		// 验证结果数量
		if len(results) != len(emails) {
			t.Errorf("批量验证结果数量不匹配: 期望=%d, 实际=%d", len(emails), len(results))
		}

		// 验证每个结果与单独验证一致
		for i, email := range emails {
			singleResult := ValidateEmail(email)
			if results[i].Valid != singleResult.Valid {
				t.Errorf("批量验证与单独验证结果不一致: email=%q", email)
			}
		}

		// 验证 allValid 标志
		expectedAllValid := true
		for _, r := range results {
			if !r.Valid {
				expectedAllValid = false
				break
			}
		}
		if allValid != expectedAllValid {
			t.Errorf("allValid 标志不正确: 期望=%v, 实际=%v", expectedAllValid, allValid)
		}
	})

	t.Run("属性2.7: IsValidEmail 与 ValidateEmail 一致性", func(t *testing.T) {
		for i := 0; i < 50; i++ {
			var email string
			if i%2 == 0 {
				email = generateValidEmail(rng)
			} else {
				email = generateInvalidEmail(rng)
			}

			isValid := IsValidEmail(email)
			result := ValidateEmail(email)

			if isValid != result.Valid {
				t.Errorf("IsValidEmail 与 ValidateEmail 结果不一致: email=%q", email)
			}
		}
	})

	t.Run("属性2.8: ExtractEmailAddress 提取正确性", func(t *testing.T) {
		testCases := []struct {
			input    string
			expected string
		}{
			{"user@example.com", "user@example.com"},
			{"John <john@example.com>", "john@example.com"},
			{"<test@test.org>", "test@test.org"},
			{"", ""},
		}

		for _, tc := range testCases {
			result := ExtractEmailAddress(tc.input)
			if result != tc.expected {
				t.Errorf("ExtractEmailAddress 失败: 输入=%q, 期望=%q, 实际=%q",
					tc.input, tc.expected, result)
			}
		}
	})
}

// generateValidEmail 生成随机有效邮箱地址
func generateValidEmail(rng *rand.Rand) string {
	// 本地部分字符集
	localChars := "abcdefghijklmnopqrstuvwxyz0123456789._-"
	// 域名字符集
	domainChars := "abcdefghijklmnopqrstuvwxyz0123456789"
	// 顶级域名
	tlds := []string{"com", "org", "net", "io", "co", "cn", "edu"}

	// 生成本地部分 (1-20 字符)
	localLen := 1 + rng.Intn(20)
	local := make([]byte, localLen)
	// 确保第一个字符是字母
	local[0] = "abcdefghijklmnopqrstuvwxyz"[rng.Intn(26)]
	for i := 1; i < localLen; i++ {
		local[i] = localChars[rng.Intn(len(localChars))]
	}
	// 确保最后一个字符不是点
	if local[localLen-1] == '.' {
		local[localLen-1] = 'x'
	}

	// 生成域名 (3-15 字符)
	domainLen := 3 + rng.Intn(13)
	domain := make([]byte, domainLen)
	for i := 0; i < domainLen; i++ {
		domain[i] = domainChars[rng.Intn(len(domainChars))]
	}

	// 选择顶级域名
	tld := tlds[rng.Intn(len(tlds))]

	return fmt.Sprintf("%s@%s.%s", string(local), string(domain), tld)
}

// generateInvalidEmail 生成随机无效邮箱地址
func generateInvalidEmail(rng *rand.Rand) string {
	invalidPatterns := []func(*rand.Rand) string{
		// 没有 @
		func(r *rand.Rand) string {
			return fmt.Sprintf("user%d.example.com", r.Intn(1000))
		},
		// 没有域名
		func(r *rand.Rand) string {
			return fmt.Sprintf("user%d@", r.Intn(1000))
		},
		// 没有本地部分
		func(r *rand.Rand) string {
			return "@example.com"
		},
		// 双 @
		func(r *rand.Rand) string {
			return fmt.Sprintf("user%d@@example.com", r.Intn(1000))
		},
		// 域名以点开头
		func(r *rand.Rand) string {
			return fmt.Sprintf("user%d@.example.com", r.Intn(1000))
		},
		// 空字符串
		func(r *rand.Rand) string {
			return ""
		},
		// 只有空格
		func(r *rand.Rand) string {
			return "   "
		},
	}

	pattern := invalidPatterns[rng.Intn(len(invalidPatterns))]
	return pattern(rng)
}
