package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"fusionmail/internal/adapter"
	"fusionmail/internal/model"
)

// seedOriginalEmail 把一封入站邮件写进真实 PostgreSQL（回复/转发要先能查到它）。
func seedOriginalEmail(t *testing.T, env *sendTestEnv, mutate func(*model.Email)) *model.Email {
	t.Helper()

	original := &model.Email{
		ProviderID:  "orig-1",
		AccountUID:  testSendAccount().UID,
		MessageID:   "orig@example.com",
		Subject:     "原主题",
		FromAddress: "alice@example.com",
		FromName:    "Alice",
		ToAddress:   "me@example.com",
		ToAddresses: `["me@example.com"]`,
		TextBody:    "原文正文",
		SentAt:      time.Date(2026, 10, 26, 10, 0, 0, 0, time.UTC),
		ReceivedAt:  time.Date(2026, 10, 26, 10, 0, 5, 0, time.UTC),
	}
	if mutate != nil {
		mutate(original)
	}
	if err := env.db.Create(original).Error; err != nil {
		t.Fatalf("写入原邮件失败: %v", err)
	}
	return original
}

// TestReplyBuildsHeadersAndPersists 覆盖回复主路径：
// 主题加 Re: 前缀、带上 In-Reply-To/References、落库记录关联原邮件。
func TestReplyBuildsHeadersAndPersists(t *testing.T) {
	account := testSendAccount()
	env := newSendTestEnv(t, account)
	original := seedOriginalEmail(t, env, func(e *model.Email) {
		e.References = "<root@example.com>"
	})
	env.sender.result = &adapter.SendResult{
		Success:    true,
		MessageID:  "<reply@example.com>",
		SenderType: adapter.SenderTypeSMTP,
	}

	resp, err := env.svc.Reply(context.Background(), original.ID, account.UID, &ReplyRequest{
		TextBody: "收到，谢谢",
	})
	if err != nil {
		t.Fatalf("回复失败: %v", err)
	}
	if !resp.Success {
		t.Fatalf("resp.Success = false，Error=%q", resp.Error)
	}

	if len(env.sender.sent) != 1 {
		t.Fatalf("发送器被调用 %d 次，期望 1 次", len(env.sender.sent))
	}
	out := env.sender.sent[0]
	if out.Subject != "Re: 原主题" {
		t.Errorf("Subject = %q，期望 %q", out.Subject, "Re: 原主题")
	}
	if out.InReplyTo != "orig@example.com" {
		t.Errorf("InReplyTo = %q，期望原邮件 Message-ID", out.InReplyTo)
	}
	if out.References != "<root@example.com> <orig@example.com>" {
		t.Errorf("References = %q，期望原引用链 + 原 Message-ID", out.References)
	}
	if len(out.To) != 1 || out.To[0] != "alice@example.com" {
		t.Errorf("To = %v，期望回复给原发件人", out.To)
	}
	if !strings.Contains(out.TextBody, "收到，谢谢") || !strings.Contains(out.TextBody, "-------- 原始邮件 --------") {
		t.Errorf("回复正文未包含新内容或原文引用: %q", out.TextBody)
	}

	rows := loadSentEmails(t, env.db)
	if len(rows) != 1 {
		t.Fatalf("sent_emails 行数 = %d，期望 1", len(rows))
	}
	if rows[0].ReplyToEmailID == nil || *rows[0].ReplyToEmailID != original.ID {
		t.Errorf("ReplyToEmailID = %v，期望 %d", rows[0].ReplyToEmailID, original.ID)
	}
	if rows[0].InReplyTo != "orig@example.com" {
		t.Errorf("落库 InReplyTo = %q", rows[0].InReplyTo)
	}
	if rows[0].Status != model.SentEmailStatusSent {
		t.Errorf("Status = %q，期望 sent", rows[0].Status)
	}
}

// TestReplyOnEmailWithoutMessageID 覆盖边界：原邮件没有 Message-ID 时
// 不得伪造 In-Reply-To/References（伪造会让邮件客户端串错会话）。
func TestReplyOnEmailWithoutMessageID(t *testing.T) {
	account := testSendAccount()
	env := newSendTestEnv(t, account)
	original := seedOriginalEmail(t, env, func(e *model.Email) {
		e.MessageID = ""
		e.References = ""
	})
	env.sender.result = &adapter.SendResult{Success: true, MessageID: "<r@example.com>"}

	if _, err := env.svc.Reply(context.Background(), original.ID, account.UID, &ReplyRequest{TextBody: "好的"}); err != nil {
		t.Fatalf("回复失败: %v", err)
	}

	out := env.sender.sent[0]
	if out.InReplyTo != "" {
		t.Errorf("InReplyTo = %q，期望空", out.InReplyTo)
	}
	if out.References != "" {
		t.Errorf("References = %q，期望空", out.References)
	}
}

// TestReplyExistingRePrefix 覆盖边界：原主题已带 Re: 时不得叠加成 Re: Re:。
func TestReplyExistingRePrefix(t *testing.T) {
	account := testSendAccount()
	env := newSendTestEnv(t, account)
	original := seedOriginalEmail(t, env, func(e *model.Email) {
		e.Subject = "Re: 已经回复过的主题"
	})
	env.sender.result = &adapter.SendResult{Success: true, MessageID: "<r@example.com>"}

	if _, err := env.svc.Reply(context.Background(), original.ID, account.UID, &ReplyRequest{TextBody: "再回一次"}); err != nil {
		t.Fatalf("回复失败: %v", err)
	}
	if got := env.sender.sent[0].Subject; got != "Re: 已经回复过的主题" {
		t.Errorf("Subject = %q，期望不叠加 Re: 前缀", got)
	}
}

// TestReplyMissingOriginalEmail 覆盖错误路径：原邮件不存在时不发信、不落库。
func TestReplyMissingOriginalEmail(t *testing.T) {
	env := newSendTestEnv(t, testSendAccount())

	resp, err := env.svc.Reply(context.Background(), 999999, testSendAccount().UID, &ReplyRequest{TextBody: "x"})
	if err == nil {
		t.Fatal("原邮件不存在时应返回 error")
	}
	if resp != nil {
		t.Errorf("原邮件不存在时 resp 应为 nil，实际 %+v", resp)
	}
	if len(env.sender.sent) != 0 {
		t.Error("原邮件不存在却调用了发送器")
	}
	if rows := loadSentEmails(t, env.db); len(rows) != 0 {
		t.Errorf("原邮件不存在却落库 %d 条", len(rows))
	}
}

// TestReplyAllRecipientsExcludeSelf 覆盖 ReplyAll 的收件人组装：
// 原发件人进 To，原收件人/抄送进 Cc，自己必须被剔除。
func TestReplyAllRecipientsExcludeSelf(t *testing.T) {
	account := testSendAccount()
	account.Email = "me@example.com"
	env := newSendTestEnv(t, account)
	original := seedOriginalEmail(t, env, func(e *model.Email) {
		e.FromAddress = "alice@example.com"
		e.ToAddresses = `["me@example.com","bob@example.com"]`
		e.CcAddresses = `["carol@example.com","me@example.com"]`
	})
	env.sender.result = &adapter.SendResult{Success: true, MessageID: "<ra@example.com>"}

	if _, err := env.svc.ReplyAll(context.Background(), original.ID, account.UID, &ReplyRequest{TextBody: "all"}); err != nil {
		t.Fatalf("全部回复失败: %v", err)
	}

	out := env.sender.sent[0]
	if len(out.To) != 1 || out.To[0] != "alice@example.com" {
		t.Errorf("To = %v，期望 [alice@example.com]", out.To)
	}
	if len(out.Cc) != 2 || out.Cc[0] != "bob@example.com" || out.Cc[1] != "carol@example.com" {
		t.Errorf("Cc = %v，期望 [bob@example.com carol@example.com]", out.Cc)
	}
	for _, addr := range append(append([]string{}, out.To...), out.Cc...) {
		if addr == "me@example.com" {
			t.Error("全部回复把自己放进了收件人")
		}
	}

	rows := loadSentEmails(t, env.db)
	if len(rows) != 1 || rows[0].ReplyToEmailID == nil || *rows[0].ReplyToEmailID != original.ID {
		t.Fatalf("全部回复未正确落库: %+v", rows)
	}
}

// TestForwardBuildsSubjectAndBody 覆盖转发：Fwd: 前缀、正文引用原邮件、
// 使用请求里的收件人，且不添加回复头。
func TestForwardBuildsSubjectAndBody(t *testing.T) {
	account := testSendAccount()
	env := newSendTestEnv(t, account)
	original := seedOriginalEmail(t, env, nil)
	env.sender.result = &adapter.SendResult{Success: true, MessageID: "<fwd@example.com>"}

	resp, err := env.svc.Forward(context.Background(), original.ID, account.UID, &ForwardRequest{
		To:       []string{"dave@example.com"},
		Cc:       []string{"erin@example.com"},
		TextBody: "请参考",
	})
	if err != nil {
		t.Fatalf("转发失败: %v", err)
	}
	if !resp.Success {
		t.Fatalf("resp.Success = false，Error=%q", resp.Error)
	}

	out := env.sender.sent[0]
	if out.Subject != "Fwd: 原主题" {
		t.Errorf("Subject = %q，期望 Fwd: 原主题", out.Subject)
	}
	if len(out.To) != 1 || out.To[0] != "dave@example.com" {
		t.Errorf("To = %v，期望请求里的收件人", out.To)
	}
	if !strings.Contains(out.TextBody, "请参考") || !strings.Contains(out.TextBody, "-------- 转发邮件 --------") {
		t.Errorf("转发正文未包含附加内容或原文: %q", out.TextBody)
	}
	if out.InReplyTo != "" || out.References != "" {
		t.Errorf("转发不应带回复头: InReplyTo=%q References=%q", out.InReplyTo, out.References)
	}

	rows := loadSentEmails(t, env.db)
	if len(rows) != 1 {
		t.Fatalf("sent_emails 行数 = %d，期望 1", len(rows))
	}
	if rows[0].ForwardFromID == nil || *rows[0].ForwardFromID != original.ID {
		t.Errorf("ForwardFromID = %v，期望 %d", rows[0].ForwardFromID, original.ID)
	}
	if rows[0].ReplyToEmailID != nil {
		t.Errorf("转发的 ReplyToEmailID 应为 nil，实际 %v", *rows[0].ReplyToEmailID)
	}
}

// TestForwardMissingOriginalEmail 覆盖错误路径：原邮件不存在。
func TestForwardMissingOriginalEmail(t *testing.T) {
	env := newSendTestEnv(t, testSendAccount())

	resp, err := env.svc.Forward(context.Background(), 999999, testSendAccount().UID, &ForwardRequest{
		To: []string{"dave@example.com"},
	})
	if err == nil {
		t.Fatal("原邮件不存在时应返回 error")
	}
	if resp != nil {
		t.Errorf("原邮件不存在时 resp 应为 nil，实际 %+v", resp)
	}
	if len(env.sender.sent) != 0 {
		t.Error("原邮件不存在却调用了发送器")
	}
}
