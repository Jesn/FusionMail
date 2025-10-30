package repository

import (
	"context"

	"fusionmail/internal/model"

	"gorm.io/gorm"
)

// AttachmentRepository 附件数据仓库
type AttachmentRepository struct {
	db *gorm.DB
}

// NewAttachmentRepository 创建附件数据仓库
func NewAttachmentRepository(db *gorm.DB) *AttachmentRepository {
	return &AttachmentRepository{db: db}
}

// Create 创建附件
func (r *AttachmentRepository) Create(ctx context.Context, attachment *model.EmailAttachment) error {
	return r.db.WithContext(ctx).Create(attachment).Error
}

// FindByID 根据 ID 查找附件
func (r *AttachmentRepository) FindByID(ctx context.Context, id int64) (*model.EmailAttachment, error) {
	var attachment model.EmailAttachment
	err := r.db.WithContext(ctx).First(&attachment, id).Error
	if err != nil {
		return nil, err
	}
	return &attachment, nil
}

// FindByEmailID 根据邮件 ID 查找所有附件
func (r *AttachmentRepository) FindByEmailID(ctx context.Context, emailID int64) ([]*model.EmailAttachment, error) {
	var attachments []*model.EmailAttachment
	err := r.db.WithContext(ctx).
		Where("email_id = ?", emailID).
		Order("created_at ASC").
		Find(&attachments).Error
	return attachments, err
}

// Update 更新附件
func (r *AttachmentRepository) Update(ctx context.Context, attachment *model.EmailAttachment) error {
	return r.db.WithContext(ctx).Save(attachment).Error
}

// Delete 删除附件
func (r *AttachmentRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&model.EmailAttachment{}, id).Error
}

// DeleteByEmailID 删除邮件的所有附件
func (r *AttachmentRepository) DeleteByEmailID(ctx context.Context, emailID int64) error {
	return r.db.WithContext(ctx).
		Where("email_id = ?", emailID).
		Delete(&model.EmailAttachment{}).Error
}

// CountByEmailID 统计邮件的附件数量
func (r *AttachmentRepository) CountByEmailID(ctx context.Context, emailID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.EmailAttachment{}).
		Where("email_id = ?", emailID).
		Count(&count).Error
	return count, err
}
