package crypto

import (
	"math/rand"
	"testing"
	"time"
	"unicode"
)

// =============================================================================
// 属性测试：SMTP 密码加密 Round-Trip（Property 5）
// **Feature: email-sending, Property 5: SMTP 密码加密 Round-Trip**
// **Validates: Requirements 3.5**
// =============================================================================

// TestProperty5_SMTPPasswordEncryptionRoundTrip SMTP 密码加密 Round-Trip 属性测试
// 对于任意有效密码，加密后解密应该得到原始密码
func TestProperty5_SMTPPasswordEncryptionRoundTrip(t *testing.T) {
	// 使用固定密钥创建加密器，确保测试可重复
	encryptor, err := NewSMTPPasswordEncryptorWithKey("test-key-for-property-testing-32")
	if err != nil {
		t.Fatalf("创建加密器失败: %v", err)
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	t.Run("属性5.1: 随机密码 Round-Trip", func(t *testing.T) {
		// 运行 100 次随机测试
		for i := 0; i < 100; i++ {
			password := generateRandomPassword(rng, 1, 128)

			// 加密
			encrypted, err := encryptor.Encrypt(password)
			if err != nil {
				t.Errorf("加密失败 (password len=%d): %v", len(password), err)
				continue
			}

			// 解密
			decrypted, err := encryptor.Decrypt(encrypted)
			if err != nil {
				t.Errorf("解密失败 (password len=%d): %v", len(password), err)
				continue
			}

			// 验证 Round-Trip
			if decrypted != password {
				t.Errorf("Round-Trip 失败: 原始=%q, 解密后=%q", password, decrypted)
			}
		}
	})

	t.Run("属性5.2: 空密码处理", func(t *testing.T) {
		// 空密码应该返回空字符串
		encrypted, err := encryptor.Encrypt("")
		if err != nil {
			t.Errorf("加密空密码失败: %v", err)
		}
		if encrypted != "" {
			t.Errorf("空密码加密后应为空，实际=%q", encrypted)
		}

		decrypted, err := encryptor.Decrypt("")
		if err != nil {
			t.Errorf("解密空密码失败: %v", err)
		}
		if decrypted != "" {
			t.Errorf("空密码解密后应为空，实际=%q", decrypted)
		}
	})

	t.Run("属性5.3: 特殊字符密码 Round-Trip", func(t *testing.T) {
		specialPasswords := []string{
			"!@#$%^&*()_+-=[]{}|;':\",./<>?",
			"密码123",
			"パスワード",
			"пароль",
			"🔐🔑🔒",
			"a\nb\tc\rd",
			"   spaces   ",
			"null\x00byte",
		}

		for _, password := range specialPasswords {
			encrypted, err := encryptor.Encrypt(password)
			if err != nil {
				t.Errorf("加密特殊密码失败 (%q): %v", password, err)
				continue
			}

			decrypted, err := encryptor.Decrypt(encrypted)
			if err != nil {
				t.Errorf("解密特殊密码失败 (%q): %v", password, err)
				continue
			}

			if decrypted != password {
				t.Errorf("特殊密码 Round-Trip 失败: 原始=%q, 解密后=%q", password, decrypted)
			}
		}
	})

	t.Run("属性5.4: 长密码 Round-Trip", func(t *testing.T) {
		// 测试不同长度的密码
		lengths := []int{1, 10, 50, 100, 256, 512, 1024}

		for _, length := range lengths {
			password := generateRandomPassword(rng, length, length)

			encrypted, err := encryptor.Encrypt(password)
			if err != nil {
				t.Errorf("加密长密码失败 (len=%d): %v", length, err)
				continue
			}

			decrypted, err := encryptor.Decrypt(encrypted)
			if err != nil {
				t.Errorf("解密长密码失败 (len=%d): %v", length, err)
				continue
			}

			if decrypted != password {
				t.Errorf("长密码 Round-Trip 失败 (len=%d)", length)
			}
		}
	})

	t.Run("属性5.5: 加密结果不等于原文", func(t *testing.T) {
		// 对于非空密码，加密结果不应等于原文
		for i := 0; i < 50; i++ {
			password := generateRandomPassword(rng, 8, 32)

			encrypted, err := encryptor.Encrypt(password)
			if err != nil {
				t.Errorf("加密失败: %v", err)
				continue
			}

			if encrypted == password {
				t.Errorf("加密结果不应等于原文: %q", password)
			}
		}
	})

	t.Run("属性5.6: 相同密码多次加密结果不同", func(t *testing.T) {
		// 由于使用随机 nonce，相同密码多次加密应产生不同结果
		password := "test-password-123"
		results := make(map[string]bool)

		for i := 0; i < 10; i++ {
			encrypted, err := encryptor.Encrypt(password)
			if err != nil {
				t.Errorf("加密失败: %v", err)
				continue
			}

			if results[encrypted] {
				t.Errorf("相同密码多次加密产生了相同结果")
			}
			results[encrypted] = true
		}
	})
}

// generateRandomPassword 生成随机密码
func generateRandomPassword(rng *rand.Rand, minLen, maxLen int) string {
	length := minLen
	if maxLen > minLen {
		length = minLen + rng.Intn(maxLen-minLen+1)
	}

	// 字符集：ASCII 可打印字符 + 一些 Unicode 字符
	chars := []rune{}
	for r := rune(32); r < 127; r++ {
		if unicode.IsPrint(r) {
			chars = append(chars, r)
		}
	}
	// 添加一些中文字符
	for r := rune(0x4E00); r < rune(0x4E20); r++ {
		chars = append(chars, r)
	}

	result := make([]rune, length)
	for i := range result {
		result[i] = chars[rng.Intn(len(chars))]
	}

	return string(result)
}
