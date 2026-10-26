package service

import (
	"context"
	"fmt"

	"fusionmail/internal/adapter"
	"fusionmail/internal/dto"
	"fusionmail/internal/dto/response"
	"fusionmail/internal/model"
	"fusionmail/internal/repository"
	"fusionmail/pkg/logger"
)

// EmailQueryService 负责邮件的读取与统计。
//
// 为什么从 emailService 里拆出来：原 emailService 跨 3 个文件、985 行、22 个方法，
// 同时承担查询、状态变更、删除、修复四类职责。Go 没有 partial class，
// 跨文件拆分只带来"文件级"错觉——依赖关系、字段可见性、测试边界��是一个大类型。
//
// 本类型只依赖读取路径需要的东西，因此可以独立构造与测试。
type EmailQueryService struct {
	emailRepo   repository.EmailRepository
	accountRepo repository.AccountRepository
	// adapterFactory 与 credentialResolver 供正文修复使用：
	// GetEmailByID 在发现正文疑似原始 MIME 时会即时从远端重新拉取解析。
	adapterFactory     *adapter.Factory
	credentialResolver *CredentialResolver
	logger             *logger.Logger
}

// NewEmailQueryService 创建邮件查询服务。
func NewEmailQueryService(
	emailRepo repository.EmailRepository,
	accountRepo repository.AccountRepository,
	adapterFactory *adapter.Factory,
	credentialResolver *CredentialResolver,
) *EmailQueryService {
	return &EmailQueryService{
		emailRepo:          emailRepo,
		accountRepo:        accountRepo,
		adapterFactory:     adapterFactory,
		credentialResolver: credentialResolver,
		logger:             logger.NewWithModule("EmailQuery"),
	}
}

// 编译期断言：确保实现 EmailQueryService 接口。
var _ EmailQueryServiceInterface = (*EmailQueryService)(nil)

func (s *EmailQueryService) GetEmailByID(ctx context.Context, id int64) (*response.EmailDetailResponse, error) {
	email, err := s.emailRepo.FindByID(ctx, id)
	if err != nil {
		s.logger.Error("查询邮件失败: id=%d, error=%v", id, err)
		return nil, fmt.Errorf("database error: %w", err)
	}
	if email == nil {
		return nil, dto.NewAPIError(dto.ErrEmailNotFound)
	}

	// 已删除邮件也允许查看详情（便于在详情中执行恢复操作）

	// 若检测到正文未解析或疑似原始 MIME 文本，尝试即时修复
	if email.HTMLBody == "" || looksLikeRawMIME(email.TextBody) {
		if updated, rerr := s.tryRepairEmailBody(ctx, email); rerr != nil {
			s.logger.Debug("修复邮件内容失败: id=%d, err=%v", id, rerr)
		} else if updated {
			s.logger.Debug("邮件内容已修复: id=%d", id)
		}
	}

	return toEmailDetailResponse(email), nil
}

// toEmailDetailResponse 将 model.Email 转换为 EmailDetailResponse DTO
func (s *EmailQueryService) GetEmailList(ctx context.Context, params *EmailQueryParams, page, pageSize int) (*EmailListResponse, error) {
	filter := params.toFilter()
	// 参数验证
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// 处理分组筛选：将 GroupID 转换为 AccountUIDs
	if filter.GroupID != nil {
		groupID := *filter.GroupID
		if groupID == -1 {
			// -1: 所有账号，不需要额外筛选
			filter.GroupID = nil
		} else if groupID == 0 {
			// 0: 未分组账号
			accounts, err := s.accountRepo.FindUngrouped(ctx)
			if err != nil {
				s.logger.Error("获取未分组账号失败: %v", err)
				return nil, fmt.Errorf("failed to get ungrouped accounts: %w", err)
			}
			if len(accounts) == 0 {
				// 没有未分组账号，返回空列表
				return &EmailListResponse{
					Emails:     []EmailListItem{},
					Total:      0,
					Page:       page,
					PageSize:   pageSize,
					TotalPages: 0,
				}, nil
			}
			// 提取账号 UID 列表
			accountUIDs := make([]string, len(accounts))
			for i, acc := range accounts {
				accountUIDs[i] = acc.UID
			}
			filter.AccountUIDs = accountUIDs
			filter.GroupID = nil
		} else {
			// >0: 具体分组 ID
			// 使用 FindAllByGroupID 获取分组下的所有账号（包括子账户）
			accounts, err := s.accountRepo.FindAllByGroupID(ctx, groupID)
			if err != nil {
				s.logger.Error("获取分组账号失败: groupID=%d, error=%v", groupID, err)
				return nil, fmt.Errorf("failed to get group accounts: %w", err)
			}
			if len(accounts) == 0 {
				// 分组中没有账号，返回空列表
				return &EmailListResponse{
					Emails:     []EmailListItem{},
					Total:      0,
					Page:       page,
					PageSize:   pageSize,
					TotalPages: 0,
				}, nil
			}
			// 提取账号 UID 列表
			accountUIDs := make([]string, len(accounts))
			for i, acc := range accounts {
				accountUIDs[i] = acc.UID
			}
			filter.AccountUIDs = accountUIDs
			filter.GroupID = nil
		}
	}

	// 计算偏移量
	offset := (page - 1) * pageSize

	// 查询邮件列表
	emails, total, err := s.emailRepo.List(ctx, filter, offset, pageSize)
	if err != nil {
		return nil, fmt.Errorf("failed to get email list: %w", err)
	}

	// 计算总页数
	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	return &EmailListResponse{
		Emails:     toEmailListItems(emails),
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// SearchEmails 全文搜索邮件
func (s *EmailQueryService) SearchEmails(ctx context.Context, query string, accountUID string, page, pageSize int) (*EmailListResponse, error) {
	// 参数验证
	if query == "" {
		return nil, fmt.Errorf("search query is required")
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// 计算偏移量
	offset := (page - 1) * pageSize

	// 搜索邮件
	emails, total, err := s.emailRepo.Search(ctx, query, accountUID, offset, pageSize)
	if err != nil {
		return nil, fmt.Errorf("failed to search emails: %w", err)
	}

	// 计算总页数
	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	return &EmailListResponse{
		Emails:     toEmailListItems(emails),
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// GetEmailListWithFilter 获取邮件列表（使用 offset/limit，用于 API）
func (s *EmailQueryService) GetEmailListWithFilter(ctx context.Context, params *EmailQueryParams, offset, limit int) ([]*model.Email, int64, error) {
	filter := params.toFilter()
	// 参数验证
	if offset < 0 {
		offset = 0
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	// 查询邮件列表
	emails, total, err := s.emailRepo.List(ctx, filter, offset, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get email list: %w", err)
	}

	return emails, total, nil
}

// SearchEmailsWithFilter 搜索邮件（使用 offset/limit，用于 API）
func (s *EmailQueryService) SearchEmailsWithFilter(ctx context.Context, query string, accountUID string, offset, limit int) ([]*model.Email, int64, error) {
	// 参数验证
	if query == "" {
		return nil, 0, fmt.Errorf("search query is required")
	}
	if offset < 0 {
		offset = 0
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	// 搜索邮件
	emails, total, err := s.emailRepo.Search(ctx, query, accountUID, offset, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to search emails: %w", err)
	}

	return emails, total, nil
}

// MarkAsRead 标记邮件为已读
func (s *EmailQueryService) GetUnreadCount(ctx context.Context, accountUID string) (int64, error) {
	return s.emailRepo.CountUnread(ctx, accountUID)
}

// GetAccountStats 获取账户邮件统计信息
// 优化：使用单条 SQL 聚合查询，减少数据库往返次数
func (s *EmailQueryService) GetAccountStats(ctx context.Context, accountUID string) (*AccountEmailStats, error) {
	repoStats, err := s.emailRepo.GetAccountStats(ctx, accountUID)
	if err != nil {
		s.logger.Error("获取账户统计失败: uid=%s, error=%v", accountUID, err)
		return nil, fmt.Errorf("failed to get account stats: %w", err)
	}

	// 转换为 service 层的类型
	return &AccountEmailStats{
		TotalCount:    repoStats.TotalCount,
		UnreadCount:   repoStats.UnreadCount,
		StarredCount:  repoStats.StarredCount,
		ArchivedCount: repoStats.ArchivedCount,
	}, nil
}

// GetGlobalStats 获取全局邮件统计信息
// 优化：使用单条 SQL 聚合查询，减少数据库往返次数
func (s *EmailQueryService) GetGlobalStats(ctx context.Context) (*GlobalEmailStats, error) {
	repoStats, err := s.emailRepo.GetGlobalStats(ctx)
	if err != nil {
		s.logger.Error("获取全局统计失败: %v", err)
		return nil, fmt.Errorf("failed to get global stats: %w", err)
	}

	// 转换为 service 层的类型
	return &GlobalEmailStats{
		TotalCount:    repoStats.TotalCount,
		UnreadCount:   repoStats.UnreadCount,
		StarredCount:  repoStats.StarredCount,
		ArchivedCount: repoStats.ArchivedCount,
		DeletedCount:  repoStats.DeletedCount,
		SpamCount:     repoStats.SpamCount,
	}, nil
}
