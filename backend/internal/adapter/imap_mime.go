package adapter

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/quotedprintable"
	"strings"

	"golang.org/x/net/html/charset"
)

// unfoldHeaders 展开多行邮件头
func unfoldHeaders(headers string) string {
	// RFC 2822: 以空白（空格或制表符）开头的行是前一行的延续
	lines := strings.Split(headers, "\n")
	var result []string
	for i, line := range lines {
		line = strings.TrimRight(line, "\r")
		if i > 0 && len(line) > 0 && (line[0] == ' ' || line[0] == '\t') {
			// 这是延续行，追加到前一行
			if len(result) > 0 {
				result[len(result)-1] += " " + strings.TrimSpace(line)
			}
		} else {
			result = append(result, line)
		}
	}
	return strings.Join(result, "\n")
}

// decodeBase64Content 解码 Base64 内容
func decodeBase64Content(s string) ([]byte, error) {
	// 移除空白字符
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "\n", "")
	s = strings.ReplaceAll(s, " ", "")

	// 标准 Base64 解码
	return base64.StdEncoding.DecodeString(s)
}

// decodeQuotedPrintableContent 解码 Quoted-Printable 内容
func decodeQuotedPrintableContent(s string) (string, error) {
	reader := quotedprintable.NewReader(strings.NewReader(s))
	decoded, err := io.ReadAll(reader)
	if err != nil {
		return s, err
	}
	return string(decoded), nil
}

// decodeMIMEHeader 解码 MIME 编码的邮件头（如 =?gbk?b?...?= 格式）
// 支持 GBK、GB2312、GB18030、Big5 等中文编码
func decodeMIMEHeader(header string) string {
	if header == "" {
		return ""
	}

	// 检查是否包含 MIME 编码
	if !strings.Contains(header, "=?") {
		return header
	}

	// 创建支持中文编码的 MIME 解码器
	decoder := &mime.WordDecoder{
		CharsetReader: func(charsetName string, input io.Reader) (io.Reader, error) {
			name := strings.ToLower(strings.TrimSpace(charsetName))
			// 常见中文编码别名归一化
			if name == "gb2312" || name == "gbk" {
				name = "gb18030"
			}
			return charset.NewReaderLabel(name, input)
		},
	}

	decoded, err := decoder.DecodeHeader(header)
	if err != nil {
		// 解码失败，返回原始内容
		fmt.Printf("[IMAP] Failed to decode MIME header '%s': %v\n", header[:min(50, len(header))], err)
		return header
	}

	return decoded
}

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// decodeToUTF8 将按声明的 charset 的字节流转换为 UTF-8 字符串
func decodeToUTF8(b []byte, charsetName string) (string, error) {
	if len(b) == 0 {
		return "", nil
	}
	name := strings.ToLower(strings.TrimSpace(charsetName))
	if name == "" || name == "utf-8" || name == "utf8" || name == "us-ascii" {
		return string(b), nil
	}
	// 常见别名归一化
	if name == "gb2312" || name == "gbk" {
		name = "gb18030"
	}
	r, err := charset.NewReaderLabel(name, bytes.NewReader(b))
	if err != nil {
		// 回退：直接返回原文（避免阻塞）
		return string(b), err
	}
	out, err := io.ReadAll(r)
	if err != nil {
		return string(b), err
	}
	return string(out), nil
}
