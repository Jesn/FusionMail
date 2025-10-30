package service

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"fusionmail/internal/model"
	"fusionmail/internal/repository"
	"fusionmail/pkg/storage"
)

// AttachmentService 附件服务
type AttachmentService struct {
	attachmentRepo  *repository.AttachmentRepository
	storageProvider storage.Provider
}

// NewAttachmentService 创建附件服务
func NewAttachmentService(
	attachmentRepo *repository.AttachmentRepository,
	storageProvider storage.Provider,
) *AttachmentService {
	return &AttachmentService{
		attachmentRepo:  attachmentRepo,
		storageProvider: storageProvider,
	}
}

// SaveAttachment 保存附件
func (s *AttachmentService) SaveAttachment(
	ctx context.Context,
	emailID int64,
	accountUID string,
	filename string,
	contentType string,
	size int64,
	reader io.Reader,
) (*model.EmailAttachment, error) {
	// 生成存储路径：{account_uid}/{email_id}/{filename}
	storagePath := filepath.Join(accountUID, fmt.Sprintf("%d", emailID), sanitizeFilename(filename))

	// 上传文件
	url, err := s.storageProvider.Upload(ctx, storagePath, reader, contentType)
	if err != nil {
		return nil, fmt.Errorf("failed to upload attachment: %w", err)
	}

	// 创建附件记录
	attachment := &model.EmailAttachment{
		EmailID:     emailID,
		Filename:    filename,
		ContentType: contentType,
		SizeBytes:   size,
		StorageType: "local",
		StoragePath: storagePath,
		URL:         url,
	}

	if err := s.attachmentRepo.Create(ctx, attachment); err != nil {
		// 如果数据库保存失败，尝试删除已上传的文件
		_ = s.storageProvider.Delete(ctx, storagePath)
		return nil, fmt.Errorf("failed to save attachment record: %w", err)
	}

	return attachment, nil
}

// GetAttachment 获取附件信息
func (s *AttachmentService) GetAttachment(ctx context.Context, id int64) (*model.EmailAttachment, error) {
	return s.attachmentRepo.FindByID(ctx, id)
}

// GetAttachmentsByEmailID 获取邮件的所有附件
func (s *AttachmentService) GetAttachmentsByEmailID(ctx context.Context, emailID int64) ([]*model.EmailAttachment, error) {
	return s.attachmentRepo.FindByEmailID(ctx, emailID)
}

// DownloadAttachment 下载附件
func (s *AttachmentService) DownloadAttachment(ctx context.Context, id int64) (io.ReadCloser, *model.EmailAttachment, error) {
	// 获取附件信息
	attachment, err := s.attachmentRepo.FindByID(ctx, id)
	if err != nil {
		return nil, nil, fmt.Errorf("attachment not found: %w", err)
	}

	// 下载文件
	reader, err := s.storageProvider.Download(ctx, attachment.StoragePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to download attachment: %w", err)
	}

	return reader, attachment, nil
}

// DeleteAttachment 删除附件
func (s *AttachmentService) DeleteAttachment(ctx context.Context, id int64) error {
	// 获取附件信息
	attachment, err := s.attachmentRepo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("attachment not found: %w", err)
	}

	// 删除文件
	if err := s.storageProvider.Delete(ctx, attachment.StoragePath); err != nil {
		return fmt.Errorf("failed to delete attachment file: %w", err)
	}

	// 删除数据库记录
	if err := s.attachmentRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete attachment record: %w", err)
	}

	return nil
}

// DeleteAttachmentsByEmailID 删除邮件的所有附件
func (s *AttachmentService) DeleteAttachmentsByEmailID(ctx context.Context, emailID int64) error {
	// 获取所有附件
	attachments, err := s.attachmentRepo.FindByEmailID(ctx, emailID)
	if err != nil {
		return err
	}

	// 删除每个附件
	for _, attachment := range attachments {
		// 删除文件（忽略错误，继续删除其他文件）
		_ = s.storageProvider.Delete(ctx, attachment.StoragePath)
	}

	// 批量删除数据库记录
	return s.attachmentRepo.DeleteByEmailID(ctx, emailID)
}

// sanitizeFilename 清理文件名，移除不安全的字符
func sanitizeFilename(filename string) string {
	// 替换路径分隔符
	filename = strings.ReplaceAll(filename, "/", "_")
	filename = strings.ReplaceAll(filename, "\\", "_")

	// 替换其他不安全字符
	filename = strings.ReplaceAll(filename, "..", "_")

	return filename
}
