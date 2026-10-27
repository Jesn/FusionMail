package service

import (
	"fmt"
	"strings"

	"fusionmail/internal/model"
)

// buildReplySubject 构建回复主题
func (s *SendService) buildReplySubject(originalSubject string) string {
	if strings.HasPrefix(strings.ToLower(originalSubject), "re:") {
		return originalSubject
	}
	return "Re: " + originalSubject
}

// buildForwardSubject 构建转发主题
// Requirements: 5.5
func (s *SendService) buildForwardSubject(originalSubject string) string {
	if strings.HasPrefix(strings.ToLower(originalSubject), "fwd:") {
		return originalSubject
	}
	return "Fwd: " + originalSubject
}

// buildReferences 构建 References 头
func (s *SendService) buildReferences(originalEmail *model.Email) string {
	refs := originalEmail.References
	if refs != "" {
		refs += " "
	}
	if originalEmail.MessageID != "" {
		refs += "<" + originalEmail.MessageID + ">"
	}
	return refs
}

// buildReplyBody 构建回复正文
func (s *SendService) buildReplyBody(newBody string, originalEmail *model.Email) string {
	return fmt.Sprintf("%s\n\n-------- 原始邮件 --------\n发件人: %s\n日期: %s\n主题: %s\n\n%s",
		newBody,
		originalEmail.FromAddress,
		originalEmail.SentAt.Format("2006-01-02 15:04:05"),
		originalEmail.Subject,
		originalEmail.TextBody,
	)
}

// buildReplyHTMLBody 构建回复 HTML 正文
func (s *SendService) buildReplyHTMLBody(newBody string, originalEmail *model.Email) string {
	if newBody == "" && originalEmail.HTMLBody == "" {
		return ""
	}

	originalContent := originalEmail.HTMLBody
	if originalContent == "" {
		originalContent = "<pre>" + originalEmail.TextBody + "</pre>"
	}

	return fmt.Sprintf(`%s
<br><br>
<div style="border-left: 2px solid #ccc; padding-left: 10px; margin-left: 10px;">
<p><strong>发件人:</strong> %s<br>
<strong>日期:</strong> %s<br>
<strong>主题:</strong> %s</p>
%s
</div>`,
		newBody,
		originalEmail.FromAddress,
		originalEmail.SentAt.Format("2006-01-02 15:04:05"),
		originalEmail.Subject,
		originalContent,
	)
}

// buildForwardBody 构建转发正文
func (s *SendService) buildForwardBody(newBody string, originalEmail *model.Email) string {
	return fmt.Sprintf("%s\n\n-------- 转发邮件 --------\n发件人: %s\n日期: %s\n收件人: %s\n主题: %s\n\n%s",
		newBody,
		originalEmail.FromAddress,
		originalEmail.SentAt.Format("2006-01-02 15:04:05"),
		originalEmail.ToAddress,
		originalEmail.Subject,
		originalEmail.TextBody,
	)
}

// buildForwardHTMLBody 构建转发 HTML 正文
func (s *SendService) buildForwardHTMLBody(newBody string, originalEmail *model.Email) string {
	if newBody == "" && originalEmail.HTMLBody == "" {
		return ""
	}

	originalContent := originalEmail.HTMLBody
	if originalContent == "" {
		originalContent = "<pre>" + originalEmail.TextBody + "</pre>"
	}

	return fmt.Sprintf(`%s
<br><br>
<div style="border: 1px solid #ccc; padding: 10px; margin: 10px 0;">
<p><strong>---------- 转发邮件 ----------</strong><br>
<strong>发件人:</strong> %s<br>
<strong>日期:</strong> %s<br>
<strong>收件人:</strong> %s<br>
<strong>主题:</strong> %s</p>
<hr>
%s
</div>`,
		newBody,
		originalEmail.FromAddress,
		originalEmail.SentAt.Format("2006-01-02 15:04:05"),
		originalEmail.ToAddress,
		originalEmail.Subject,
		originalContent,
	)
}
