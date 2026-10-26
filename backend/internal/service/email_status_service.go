package service

import (
	"context"
	"fmt"

	"fusionmail/internal/dto"
	"fusionmail/internal/repository"
	"fusionmail/pkg/logger"
)

// EmailStatusService 负责邮件的本地状态变更（已读/星标/归档）。
//
// 为什么从 emailService 里拆出来：见 EmailQueryService 的说明。
// 本类型只依赖 emailRepo 与 accountRepo，不涉及 adapter 与凭证解析。
type EmailStatusService struct {
	emailRepo   repository.EmailRepository
	accountRepo repository.AccountRepository
	logger      *logger.Logger
}

// NewEmailStatusService 创建邮件状态服务。
func NewEmailStatusService(
	emailRepo repository.EmailRepository,
	accountRepo repository.AccountRepository,
) *EmailStatusService {
	return &EmailStatusService{
		emailRepo:   emailRepo,
		accountRepo: accountRepo,
		logger:      logger.NewWithModule("EmailStatus"),
	}
}

// 编译期断言：确保实现 EmailStatusService 接口。
var _ EmailStatusServiceInterface = (*EmailStatusService)(nil)

func (s *EmailStatusService) MarkAsRead(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return dto.NewAPIErrorWithMessage(dto.ErrInvalidRequest, "邮件 ID 列表不能为空")
	}
	if err := s.emailRepo.MarkAsRead(ctx, ids); err != nil {
		s.logger.Error("标记已读失败: count=%d, error=%v", len(ids), err)
		return fmt.Errorf("database error: %w", err)
	}
	return nil
}

// MarkAsUnread 标记邮件为未读
func (s *EmailStatusService) MarkAsUnread(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return dto.NewAPIErrorWithMessage(dto.ErrInvalidRequest, "邮件 ID 列表不能为空")
	}
	if err := s.emailRepo.MarkAsUnread(ctx, ids); err != nil {
		s.logger.Error("标记未读失败: count=%d, error=%v", len(ids), err)
		return fmt.Errorf("database error: %w", err)
	}
	return nil
}

// MarkAllAsRead 批量标记所有未读邮件为已读
func (s *EmailStatusService) MarkAllAsRead(ctx context.Context, accountUID *string) (int64, error) {
	// 如果指定了账号，验证账号是否存在
	if accountUID != nil && *accountUID != "" {
		account, err := s.accountRepo.FindByUID(ctx, *accountUID)
		if err != nil {
			s.logger.Error("查询账户失败: uid=%s, error=%v", *accountUID, err)
			return 0, fmt.Errorf("database error: %w", err)
		}
		if account == nil {
			return 0, dto.NewAPIError(dto.ErrAccountNotFound)
		}
	}

	// 批量更新
	count, err := s.emailRepo.MarkAllAsRead(ctx, accountUID)
	if err != nil {
		s.logger.Error("批量标记已读失败: error=%v", err)
		return 0, fmt.Errorf("database error: %w", err)
	}

	return count, nil
}

// ToggleStar 切换星标状态
func (s *EmailStatusService) ToggleStar(ctx context.Context, id int64) error {
	email, err := s.emailRepo.FindByID(ctx, id)
	if err != nil {
		s.logger.Error("查询邮件失败: id=%d, error=%v", id, err)
		return fmt.Errorf("database error: %w", err)
	}
	if email == nil {
		return dto.NewAPIError(dto.ErrEmailNotFound)
	}

	// 切换星标状态
	newStarred := !email.IsStarred
	if err := s.emailRepo.UpdateLocalStatus(ctx, id, nil, &newStarred, nil, nil); err != nil {
		s.logger.Error("切换星标失败: id=%d, error=%v", id, err)
		return fmt.Errorf("database error: %w", err)
	}
	return nil
}

// ArchiveEmail 归档邮件
func (s *EmailStatusService) ArchiveEmail(ctx context.Context, id int64) error {
	// 验证邮件是否存在
	email, err := s.emailRepo.FindByID(ctx, id)
	if err != nil {
		s.logger.Error("查询邮件失败: id=%d, error=%v", id, err)
		return fmt.Errorf("database error: %w", err)
	}
	if email == nil {
		return dto.NewAPIError(dto.ErrEmailNotFound)
	}

	archived := true
	if err := s.emailRepo.UpdateLocalStatus(ctx, id, nil, nil, &archived, nil); err != nil {
		s.logger.Error("归档邮件失败: id=%d, error=%v", id, err)
		return fmt.Errorf("database error: %w", err)
	}
	return nil
}

// GetUnreadCount 获取未读邮件数
