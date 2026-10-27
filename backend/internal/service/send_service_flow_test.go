package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"fusionmail/internal/adapter"
	"fusionmail/internal/model"
)

// TestSendEmailPersistsSentRecord 覆盖发信主路径：
// 收件人/抄送/密送/附件齐全 → 发送成功 → 落库一条 sent 记录，
// 且密送只进 bcc 列（不能混进 to_addresses，否则会泄露给收件人）。
func TestSendEmailPersistsSentRecord(t *testing.T) {
	account := testSendAccount()
	env := newSendTestEnv(t, account)
	env.sender.result = &adapter.SendResult{
		Success:       true,
		MessageID:     "<msg-1@example.com>",
		ProviderMsgID: "provider-1",
		SenderType:    adapter.SenderTypeSMTP,
	}

	resp, err := env.svc.SendEmail(context.Background(), &SendEmailRequest{
		AccountUID: account.UID,
		To:         []string{"to@example.com"},
		Cc:         []string{"cc@example.com"},
		Bcc:        []string{"bcc@example.com"},
		Subject:    "季度报告",
		TextBody:   "正文",
		Attachments: []adapter.OutgoingAttachment{
			{Filename: "report.pdf", ContentType: "application/pdf", Content: []byte("pdf-bytes")},
		},
	})
	if err != nil {
		t.Fatalf("发送失败: %v", err)
	}
	if !resp.Success {
		t.Fatalf("resp.Success = false，Error=%q", resp.Error)
	}
	if resp.MessageID != "<msg-1@example.com>" {
		t.Errorf("resp.MessageID = %q，期望 %q", resp.MessageID, "<msg-1@example.com>")
	}
	if resp.SentEmailID == 0 {
		t.Fatal("resp.SentEmailID = 0，成功发送必须留下审计记录")
	}

	// 交给发送器的邮件必须带上密送（SMTP 靠信封投递，不能丢）
	if len(env.sender.sent) != 1 {
		t.Fatalf("发送器被调用 %d 次，期望 1 次", len(env.sender.sent))
	}
	sent := env.sender.sent[0]
	if len(sent.Bcc) != 1 || sent.Bcc[0] != "bcc@example.com" {
		t.Errorf("发送器收到的 Bcc = %v，期望 [bcc@example.com]", sent.Bcc)
	}
	if sent.From != account.Email {
		t.Errorf("发送器收到的 From = %q，期望 %q", sent.From, account.Email)
	}

	rows := loadSentEmails(t, env.db)
	if len(rows) != 1 {
		t.Fatalf("sent_emails 行数 = %d，期望 1", len(rows))
	}
	row := rows[0]
	if row.ID != resp.SentEmailID {
		t.Errorf("落库 ID = %d，resp.SentEmailID = %d", row.ID, resp.SentEmailID)
	}
	if row.Status != model.SentEmailStatusSent {
		t.Errorf("Status = %q，期望 %q", row.Status, model.SentEmailStatusSent)
	}
	if row.Subject != "季度报告" || row.TextBody != "正文" {
		t.Errorf("落库主题/正文 = %q/%q", row.Subject, row.TextBody)
	}
	if row.FromAddress != account.Email || row.AccountUID != account.UID {
		t.Errorf("落库 From/AccountUID = %q/%q", row.FromAddress, row.AccountUID)
	}
	if row.MessageID != "<msg-1@example.com>" || row.ProviderMsgID != "provider-1" {
		t.Errorf("落库 MessageID/ProviderMsgID = %q/%q", row.MessageID, row.ProviderMsgID)
	}
	if row.SenderType != adapter.SenderTypeSMTP {
		t.Errorf("落库 SenderType = %q，期望 %q", row.SenderType, adapter.SenderTypeSMTP)
	}
	if !row.HasAttachments || row.AttachmentCount != 1 {
		t.Errorf("附件标记 = %v/%d，期望 true/1", row.HasAttachments, row.AttachmentCount)
	}
	if !strings.Contains(row.AttachmentInfo, "report.pdf") {
		t.Errorf("AttachmentInfo = %q，未包含附件名", row.AttachmentInfo)
	}
	if row.ErrorMessage != "" {
		t.Errorf("���功发送的 ErrorMessage = %q，期望空", row.ErrorMessage)
	}

	if got := decodeAddresses(t, row.ToAddresses); len(got) != 1 || got[0] != "to@example.com" {
		t.Errorf("to_addresses = %v，期望 [to@example.com]", got)
	}
	if got := decodeAddresses(t, row.CcAddresses); len(got) != 1 || got[0] != "cc@example.com" {
		t.Errorf("cc_addresses = %v，期望 [cc@example.com]", got)
	}
	if got := decodeAddresses(t, row.BccAddresses); len(got) != 1 || got[0] != "bcc@example.com" {
		t.Errorf("bcc_addresses = %v，期望 [bcc@example.com]", got)
	}
	if strings.Contains(row.ToAddresses, "bcc@example.com") {
		t.Error("密送地址出现在 to_addresses 里，会泄露给收件人")
	}

	// SMTP 账户无加密凭证 → 解析结果应为 nil，发送器不该拿到伪造凭证
	if env.resolver.creds != nil {
		t.Errorf("无凭证账户解析出 %+v，期望 nil", env.resolver.creds)
	}
}

// TestSendEmailRecordsFailedAttempt 覆盖发送失败路径：
// 失败也必须落库（status=failed + 错误原因），否则失败发送在审计里凭空消失。
func TestSendEmailRecordsFailedAttempt(t *testing.T) {
	account := testSendAccount()
	env := newSendTestEnv(t, account)
	env.sender.result = &adapter.SendResult{
		Success:    false,
		Error:      "SMTP 535 认证失败",
		SenderType: adapter.SenderTypeSMTP,
	}
	env.sender.err = errors.New("smtp: 535 authentication failed")

	resp, err := env.svc.SendEmail(context.Background(), &SendEmailRequest{
		AccountUID: account.UID,
		To:         []string{"to@example.com"},
		Subject:    "会失败的邮件",
		TextBody:   "正文",
	})
	if err == nil {
		t.Fatal("发送器报错时 SendEmail 应返回 error")
	}
	if resp == nil || resp.Success {
		t.Fatalf("resp = %+v，期望 Success=false", resp)
	}
	if resp.Error != "SMTP 535 认证失败" {
		t.Errorf("resp.Error = %q，期望发送器给出的原因", resp.Error)
	}

	rows := loadSentEmails(t, env.db)
	if len(rows) != 1 {
		t.Fatalf("失败发送后 sent_emails 行数 = %d，期望 1", len(rows))
	}
	if rows[0].Status != model.SentEmailStatusFailed {
		t.Errorf("Status = %q，期望 %q", rows[0].Status, model.SentEmailStatusFailed)
	}
	if rows[0].ErrorMessage != "SMTP 535 认证失败" {
		t.Errorf("ErrorMessage = %q，期望 %q", rows[0].ErrorMessage, "SMTP 535 认证失败")
	}
}

// TestSendEmailNilResultDoesNotPanic 覆盖适配器违约：
// MailSender.Send 的契约是始终返回非 nil 结果，但四条发送分支都会解引用它。
// 违约时必须是「一条 failed 记录 + 错误响应」，而不是 panic。
func TestSendEmailNilResultDoesNotPanic(t *testing.T) {
	account := testSendAccount()
	env := newSendTestEnv(t, account)
	env.sender.result = nil
	env.sender.err = errors.New("smtp 连接被拒绝")

	resp, err := env.svc.SendEmail(context.Background(), &SendEmailRequest{
		AccountUID: account.UID,
		To:         []string{"to@example.com"},
		Subject:    "违约发送器",
	})
	if err == nil {
		t.Fatal("发送器返回 nil 结果 + error 时 SendEmail 应返回 error")
	}
	if resp == nil || resp.Success {
		t.Fatalf("resp = %+v，期望 Success=false", resp)
	}
	if resp.Error != "smtp 连接被拒绝" {
		t.Errorf("resp.Error = %q，期望回落到 err 文本", resp.Error)
	}

	rows := loadSentEmails(t, env.db)
	if len(rows) != 1 || rows[0].Status != model.SentEmailStatusFailed {
		t.Fatalf("违约发送器未留下 failed 记录: %+v", rows)
	}
}

// TestSendEmailAccountNotFound 覆盖账户缺失：不得调用发送器，也不得落库。
func TestSendEmailAccountNotFound(t *testing.T) {
	env := newSendTestEnv(t, nil) // 账户查询返回 nil

	resp, err := env.svc.SendEmail(context.Background(), &SendEmailRequest{
		AccountUID: "missing-account",
		To:         []string{"to@example.com"},
		Subject:    "无账户",
	})
	if err == nil {
		t.Fatal("账户不存在时应返回 error")
	}
	if resp == nil || resp.Success {
		t.Fatalf("resp = %+v，期望 Success=false", resp)
	}
	if resp.Error != "账户不存在" {
		t.Errorf("resp.Error = %q，期望「账户不存在」", resp.Error)
	}
	if len(env.sender.sent) != 0 {
		t.Errorf("账户不存在却调用了发送器 %d 次", len(env.sender.sent))
	}
	if rows := loadSentEmails(t, env.db); len(rows) != 0 {
		t.Errorf("账户不存在却落库了 %d 条记录", len(rows))
	}
}

// TestSendEmailValidatesRequest 覆盖入参校验边界：非法请求在触达发送器与
// 数据库之前就被拒绝。
func TestSendEmailValidatesRequest(t *testing.T) {
	cases := []struct {
		name    string
		req     *SendEmailRequest
		wantErr string
	}{
		{
			name:    "空账户",
			req:     &SendEmailRequest{To: []string{"to@example.com"}},
			wantErr: "发送账户不能为空",
		},
		{
			name:    "空收件人",
			req:     &SendEmailRequest{AccountUID: "acct-send-1"},
			wantErr: "收件人不能为空",
		},
		{
			name:    "收件人格式非法",
			req:     &SendEmailRequest{AccountUID: "acct-send-1", To: []string{"not-an-email"}},
			wantErr: "收件人邮箱格式不正确",
		},
		{
			name: "抄送格式非法",
			req: &SendEmailRequest{
				AccountUID: "acct-send-1",
				To:         []string{"to@example.com"},
				Cc:         []string{"bad@@example.com"},
			},
			wantErr: "抄送邮箱格式不正确",
		},
		{
			name: "密送格式非法",
			req: &SendEmailRequest{
				AccountUID: "acct-send-1",
				To:         []string{"to@example.com"},
				Bcc:        []string{"nope"},
			},
			wantErr: "密送邮箱格式不正确",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			env := newSendTestEnv(t, testSendAccount())

			resp, err := env.svc.SendEmail(context.Background(), c.req)
			if err == nil {
				t.Fatalf("非法请求应返回 error")
			}
			if !strings.Contains(err.Error(), c.wantErr) {
				t.Errorf("error = %q，期望包含 %q", err.Error(), c.wantErr)
			}
			if resp == nil || resp.Success {
				t.Errorf("resp = %+v，期望 Success=false", resp)
			}
			if len(env.sender.sent) != 0 {
				t.Errorf("非法请求却调用了发送器")
			}
			if rows := loadSentEmails(t, env.db); len(rows) != 0 {
				t.Errorf("非法请求却落库了 %d 条记录", len(rows))
			}
		})
	}
}

// TestSendEmailRejectsOversizedAttachment 覆盖附件大小上限（边界）。
func TestSendEmailRejectsOversizedAttachment(t *testing.T) {
	env := newSendTestEnv(t, testSendAccount())

	oversized := make([]byte, adapter.MaxSingleAttachmentSize+1)
	resp, err := env.svc.SendEmail(context.Background(), &SendEmailRequest{
		AccountUID: "acct-send-1",
		To:         []string{"to@example.com"},
		Subject:    "超大附件",
		Attachments: []adapter.OutgoingAttachment{
			{Filename: "huge.bin", ContentType: "application/octet-stream", Content: oversized},
		},
	})
	if err == nil {
		t.Fatal("超过单附件上限时应返回 error")
	}
	if !strings.Contains(err.Error(), "大小超过限制") {
		t.Errorf("error = %q，期望提示附件超限", err.Error())
	}
	if resp == nil || resp.Success {
		t.Errorf("resp = %+v，期望 Success=false", resp)
	}
	if len(env.sender.sent) != 0 {
		t.Error("附件超限却调用了发送器")
	}
}

func decodeAddresses(t *testing.T, raw string) []string {
	t.Helper()

	if raw == "" {
		return nil
	}
	var out []string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		t.Fatalf("解析地址 JSON %q 失败: %v", raw, err)
	}
	return out
}
