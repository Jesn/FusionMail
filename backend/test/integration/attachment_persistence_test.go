package integration

import (
	"context"
	"testing"
	"time"

	"fusionmail/internal/model"
	"fusionmail/internal/repository"
)

// TestAttachmentMetadataPersistence 附件元数据落库（P18）：
// 验证 GORM 级联（Create 携带关联自动写 email_id）与 ReplaceAttachments 整体替换。
// 存储语义：StorageType=external、StoragePath 空 —— 内容在服务商侧，本地只存元数据。
func TestAttachmentMetadataPersistence(t *testing.T) {
	db := setupTestDB(t)
	emailRepo := repository.NewEmailRepository(db)
	ctx := context.Background()

	email := &model.Email{
		ProviderID: "att-persist-001", AccountUID: "att-persist-account",
		Subject: "带附件", FromAddress: "s@example.com", ToAddress: "u@example.com",
		SentAt: time.Now(), ReceivedAt: time.Now(),
		HasAttachments:   true,
		AttachmentsCount: 2,
		Attachments: []model.EmailAttachment{
			{Filename: "invoice.pdf", ContentType: "application/pdf", SizeBytes: 1024, StorageType: "external"},
			{Filename: "photo.png", ContentType: "image/png", SizeBytes: 2048, StorageType: "external", IsInline: true, ContentID: "cid1"},
		},
	}
	if err := emailRepo.Create(ctx, email); err != nil {
		t.Fatalf("create with attachments: %v", err)
	}
	if email.ID == 0 {
		t.Fatal("email ID should be set")
	}

	// 级联落库 + email_id 自动填充
	rows, err := emailRepo.FindByID(ctx, email.ID)
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if rows == nil || len(rows.Attachments) != 2 {
		t.Fatalf("expected 2 persisted attachments, got %v", rows)
	}
	for _, a := range rows.Attachments {
		if a.EmailID != email.ID {
			t.Fatalf("cascade email_id: want %d got %d", email.ID, a.EmailID)
		}
		if a.StorageType != "external" || a.StoragePath != "" {
			t.Fatalf("storage semantics: %+v", a)
		}
	}

	// 整体替换：2 行 → 1 行
	replacement := []model.EmailAttachment{
		{Filename: "only.zip", ContentType: "application/zip", SizeBytes: 4096, StorageType: "external"},
	}
	if err := emailRepo.ReplaceAttachments(ctx, email.ID, replacement); err != nil {
		t.Fatalf("replace: %v", err)
	}
	rows, _ = emailRepo.FindByID(ctx, email.ID)
	if len(rows.Attachments) != 1 || rows.Attachments[0].Filename != "only.zip" {
		t.Fatalf("after replace: %+v", rows.Attachments)
	}

	// 替换为空 = 清空
	if err := emailRepo.ReplaceAttachments(ctx, email.ID, nil); err != nil {
		t.Fatalf("replace empty: %v", err)
	}
	rows, _ = emailRepo.FindByID(ctx, email.ID)
	if len(rows.Attachments) != 0 {
		t.Fatalf("after clear: %+v", rows.Attachments)
	}
}
