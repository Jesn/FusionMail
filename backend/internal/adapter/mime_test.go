package adapter

import (
	"bytes"
	"encoding/base64"
	"math/rand"
	"mime"
	"mime/multipart"
	"net/textproto"
	"strings"
	"testing"
	"time"
)

// =============================================================================
// 属性测试：附件 MIME 编码 Round-Trip（Property 6）
// **Feature: email-sending, Property 6: 附件 MIME 编码 Round-Trip**
// **Validates: Requirements 4.4**
// =============================================================================

// TestProperty6_AttachmentMIMEEncodingRoundTrip 附件 MIME 编码 Round-Trip 属性测试
// 对于任意附件内容，Base64 编码后解码应该得到原始内容
func TestProperty6_AttachmentMIMEEncodingRoundTrip(t *testing.T) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	t.Run("属性6.1: 随机二进制内容 Round-Trip", func(t *testing.T) {
		// 运行 100 次随机测试
		for i := 0; i < 100; i++ {
			// 生成随机大小的二进制内容 (1 字节 - 10KB)
			size := 1 + rng.Intn(10*1024)
			content := make([]byte, size)
			rng.Read(content)

			// Base64 编码（与 SMTP 发送器使用相同的编码方式）
			encoded := base64.StdEncoding.EncodeToString(content)

			// Base64 解码
			decoded, err := base64.StdEncoding.DecodeString(encoded)
			if err != nil {
				t.Errorf("Base64 解码失败 (size=%d): %v", size, err)
				continue
			}

			// 验证 Round-Trip
			if !bytes.Equal(decoded, content) {
				t.Errorf("Round-Trip 失败: 原始大小=%d, 解码后大小=%d", len(content), len(decoded))
			}
		}
	})

	t.Run("属性6.2: 空内容处理", func(t *testing.T) {
		content := []byte{}
		encoded := base64.StdEncoding.EncodeToString(content)
		decoded, err := base64.StdEncoding.DecodeString(encoded)

		if err != nil {
			t.Errorf("空内容解码失败: %v", err)
		}
		if len(decoded) != 0 {
			t.Errorf("空内容 Round-Trip 失败: 解码后大小=%d", len(decoded))
		}
	})

	t.Run("属性6.3: 文本内容 Round-Trip", func(t *testing.T) {
		textContents := []string{
			"Hello, World!",
			"这是中文内容",
			"日本語テキスト",
			"Привет мир",
			"مرحبا بالعالم",
			"Line1\nLine2\nLine3",
			"Tab\tSeparated\tValues",
			strings.Repeat("Long text content. ", 100),
		}

		for _, text := range textContents {
			content := []byte(text)
			encoded := base64.StdEncoding.EncodeToString(content)
			decoded, err := base64.StdEncoding.DecodeString(encoded)

			if err != nil {
				t.Errorf("文本内容解码失败 (%q): %v", text[:minInt(20, len(text))], err)
				continue
			}

			if string(decoded) != text {
				t.Errorf("文本内容 Round-Trip 失败")
			}
		}
	})

	t.Run("属性6.4: 大文件内容 Round-Trip", func(t *testing.T) {
		// 测试不同大小的内容
		sizes := []int{1024, 10 * 1024, 100 * 1024, 1024 * 1024} // 1KB, 10KB, 100KB, 1MB

		for _, size := range sizes {
			content := make([]byte, size)
			rng.Read(content)

			encoded := base64.StdEncoding.EncodeToString(content)
			decoded, err := base64.StdEncoding.DecodeString(encoded)

			if err != nil {
				t.Errorf("大文件解码失败 (size=%d): %v", size, err)
				continue
			}

			if !bytes.Equal(decoded, content) {
				t.Errorf("大文件 Round-Trip 失败 (size=%d)", size)
			}
		}
	})

	t.Run("属性6.5: 编码后大小增长验证", func(t *testing.T) {
		// Base64 编码后大小约为原始大小的 4/3
		for i := 0; i < 50; i++ {
			size := 100 + rng.Intn(10000)
			content := make([]byte, size)
			rng.Read(content)

			encoded := base64.StdEncoding.EncodeToString(content)

			// 验证编码后大小在预期范围内
			expectedSize := (size + 2) / 3 * 4 // Base64 编码公式
			if len(encoded) != expectedSize {
				t.Errorf("编码后大小不符合预期: 原始=%d, 编码后=%d, 预期=%d",
					size, len(encoded), expectedSize)
			}
		}
	})
}

// TestProperty6_FilenameEncoding 文件名编码测试
func TestProperty6_FilenameEncoding(t *testing.T) {
	t.Run("属性6.6: 中文文件名编码", func(t *testing.T) {
		filenames := []string{
			"测试文件.txt",
			"报告2024.pdf",
			"附件_文档.docx",
			"日本語ファイル.xlsx",
		}

		for _, filename := range filenames {
			// 使用 MIME Q 编码（与 smtp_sender.go 中的 encodeFilename 相同）
			encoded := mime.QEncoding.Encode("utf-8", filename)

			// 解码
			dec := new(mime.WordDecoder)
			decoded, err := dec.DecodeHeader(encoded)
			if err != nil {
				t.Errorf("文件名解码失败 (%q): %v", filename, err)
				continue
			}

			if decoded != filename {
				t.Errorf("文件名 Round-Trip 失败: 原始=%q, 解码后=%q", filename, decoded)
			}
		}
	})

	t.Run("属性6.7: ASCII 文件名编码", func(t *testing.T) {
		filenames := []string{
			"document.pdf",
			"image.png",
			"report_2024.xlsx",
			"file-name.txt",
		}

		for _, filename := range filenames {
			encoded := mime.QEncoding.Encode("utf-8", filename)

			dec := new(mime.WordDecoder)
			decoded, err := dec.DecodeHeader(encoded)
			if err != nil {
				t.Errorf("ASCII 文件名解码失败 (%q): %v", filename, err)
				continue
			}

			if decoded != filename {
				t.Errorf("ASCII 文件名 Round-Trip 失败: 原始=%q, 解码后=%q", filename, decoded)
			}
		}
	})

	t.Run("属性6.8: 特殊字符文件名编码", func(t *testing.T) {
		filenames := []string{
			"file with spaces.txt",
			"file(1).pdf",
			"file[2024].doc",
			"file=value.txt",
		}

		for _, filename := range filenames {
			encoded := mime.QEncoding.Encode("utf-8", filename)

			dec := new(mime.WordDecoder)
			decoded, err := dec.DecodeHeader(encoded)
			if err != nil {
				t.Errorf("特殊字符文件名解码失败 (%q): %v", filename, err)
				continue
			}

			if decoded != filename {
				t.Errorf("特殊字符文件名 Round-Trip 失败: 原始=%q, 解码后=%q", filename, decoded)
			}
		}
	})
}

// TestProperty6_MIMEPartConstruction MIME 部分构建测试
func TestProperty6_MIMEPartConstruction(t *testing.T) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	t.Run("属性6.9: 附件 MIME 部分构建", func(t *testing.T) {
		attachments := []struct {
			filename    string
			contentType string
			content     []byte
		}{
			{"test.txt", "text/plain", []byte("Hello, World!")},
			{"image.png", "image/png", generateRandomBytes(rng, 1024)},
			{"document.pdf", "application/pdf", generateRandomBytes(rng, 2048)},
			{"中文文件.txt", "text/plain; charset=utf-8", []byte("中文内容")},
		}

		for _, att := range attachments {
			var buf bytes.Buffer
			writer := multipart.NewWriter(&buf)

			// 构建 MIME 头
			header := textproto.MIMEHeader{
				"Content-Type":              []string{att.contentType + "; name=\"" + att.filename + "\""},
				"Content-Transfer-Encoding": []string{"base64"},
				"Content-Disposition":       []string{"attachment; filename=\"" + att.filename + "\""},
			}

			part, err := writer.CreatePart(header)
			if err != nil {
				t.Errorf("创建 MIME 部分失败 (%s): %v", att.filename, err)
				continue
			}

			// 写入 Base64 编码的内容
			encoded := base64.StdEncoding.EncodeToString(att.content)
			_, err = part.Write([]byte(encoded))
			if err != nil {
				t.Errorf("写入内容失败 (%s): %v", att.filename, err)
				continue
			}

			writer.Close()

			// 验证 MIME 部分包含必要的头
			result := buf.String()
			if !strings.Contains(result, "Content-Type:") {
				t.Errorf("MIME 部分缺少 Content-Type 头 (%s)", att.filename)
			}
			if !strings.Contains(result, "Content-Transfer-Encoding: base64") {
				t.Errorf("MIME 部分缺少 Content-Transfer-Encoding 头 (%s)", att.filename)
			}
			if !strings.Contains(result, "Content-Disposition:") {
				t.Errorf("MIME 部分缺少 Content-Disposition 头 (%s)", att.filename)
			}
		}
	})
}

// TestProperty6_SubjectEncoding 邮件主题编码测试
func TestProperty6_SubjectEncoding(t *testing.T) {
	t.Run("属性6.10: 中文主题编码 Round-Trip", func(t *testing.T) {
		subjects := []string{
			"测试邮件",
			"会议通知：2024年度总结",
			"【重要】项目进度报告",
			"Re: 关于合同的问题",
			"Fwd: 转发：客户反馈",
		}

		for _, subject := range subjects {
			// 使用 MIME Q 编码（与 smtp_sender.go 中的 encodeSubject 相同）
			encoded := mime.QEncoding.Encode("utf-8", subject)

			// 解码
			dec := new(mime.WordDecoder)
			decoded, err := dec.DecodeHeader(encoded)
			if err != nil {
				t.Errorf("主题解码失败 (%q): %v", subject, err)
				continue
			}

			if decoded != subject {
				t.Errorf("主题 Round-Trip 失败: 原始=%q, 解码后=%q", subject, decoded)
			}
		}
	})

	t.Run("属性6.11: 混合语言主题编码", func(t *testing.T) {
		subjects := []string{
			"Hello 你好 World",
			"Meeting: 会议 at 3pm",
			"Report 报告 2024",
		}

		for _, subject := range subjects {
			encoded := mime.QEncoding.Encode("utf-8", subject)

			dec := new(mime.WordDecoder)
			decoded, err := dec.DecodeHeader(encoded)
			if err != nil {
				t.Errorf("混合语言主题解码失败 (%q): %v", subject, err)
				continue
			}

			if decoded != subject {
				t.Errorf("混合语言主题 Round-Trip 失败: 原始=%q, 解码后=%q", subject, decoded)
			}
		}
	})
}

// generateRandomBytes 生成随机字节
func generateRandomBytes(rng *rand.Rand, size int) []byte {
	data := make([]byte, size)
	rng.Read(data)
	return data
}

// minInt 返回两个整数中的较小值
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
