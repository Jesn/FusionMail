package service

import (
	"context"
	"time"

	"fusionmail/internal/model"
	"fusionmail/internal/repository"
)

// SentEmailService 已发送邮件服务
// Requirements: 7.1, 7.4
type SentEmailService struct {
	sentEmailRepo repository.SentEmailRepository
}

// NewSentEmailService 创建已发送邮件服务实例
func NewSentEmailService(sentEmailRepo repository.SentEmailRepository) *SentEmailService {
	return &SentEmailService{
		sentEmailRepo: sentEmailRepo,
	}
}

// ListSentEmailsRequest 列出已发送邮件请求
type ListSentEmailsRequest struct {
	AccountUID  string     `form:"account_uid"`
	Status      string     `form:"status"`
	StartDate   *time.Time `form:"start_date"`
	EndDate     *time.Time `form:"end_date"`
	SearchQuery string     `form:"search"`
	Page        int        `form:"page"`
	PageSize    int        `form:"page_size"`
}

// ListSentEmailsResponse 列出已发送邮件响应
type ListSentEmailsResponse struct {
	Emails     []*model.SentEmail `json:"emails"`
	Total      int64              `json:"total"`
	Page       int                `json:"page"`
	PageSize   int                `json:"page_size"`
	TotalPages int                `json:"total_pages"`
}

// ListSentEmails 列出已发送邮件
// Requirements: 7.1
func (s *SentEmailService) ListSentEmails(ctx context.Context, req *ListSentEmailsRequest) (*ListSentEmailsResponse, error) {
	// 设置默认分页参数
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}
	if req.PageSize > 100 {
		req.PageSize = 100
	}

	// 构建过滤条件
	filter := &repository.SentEmailFilter{
		AccountUID:  req.AccountUID,
		Status:      req.Status,
		StartDate:   req.StartDate,
		EndDate:     req.EndDate,
		SearchQuery: req.SearchQuery,
	}

	// 计算偏移量
	offset := (req.Page - 1) * req.PageSize

	// 查询数据
	emails, total, err := s.sentEmailRepo.List(ctx, filter, offset, req.PageSize)
	if err != nil {
		return nil, err
	}

	// 计算总页数
	totalPages := int(total) / req.PageSize
	if int(total)%req.PageSize > 0 {
		totalPages++
	}

	return &ListSentEmailsResponse{
		Emails:     emails,
		Total:      total,
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: totalPages,
	}, nil
}

// GetSentEmail 获取已发送邮件详情
// Requirements: 7.2
func (s *SentEmailService) GetSentEmail(ctx context.Context, id int64) (*model.SentEmail, error) {
	return s.sentEmailRepo.FindByID(ctx, id)
}

// DeleteSentEmail 删除已发送邮件
func (s *SentEmailService) DeleteSentEmail(ctx context.Context, id int64) error {
	return s.sentEmailRepo.Delete(ctx, id)
}

// CountSentEmails 统计已发送邮件数量
func (s *SentEmailService) CountSentEmails(ctx context.Context, accountUID string) (int64, error) {
	return s.sentEmailRepo.CountByAccount(ctx, accountUID)
}

// GetSentEmailStats 获取已发送邮件统计
type SentEmailStats struct {
	Total   int64 `json:"total"`
	Success int64 `json:"success"`
	Failed  int64 `json:"failed"`
}

// GetStats 获取已发送邮件统计
func (s *SentEmailService) GetStats(ctx context.Context, accountUID string) (*SentEmailStats, error) {
	// 获取总数
	total, err := s.sentEmailRepo.Count(ctx, &repository.SentEmailFilter{
		AccountUID: accountUID,
	})
	if err != nil {
		return nil, err
	}

	// 获取成功数
	success, err := s.sentEmailRepo.Count(ctx, &repository.SentEmailFilter{
		AccountUID: accountUID,
		Status:     model.SentEmailStatusSent,
	})
	if err != nil {
		return nil, err
	}

	// 获取失败数
	failed, err := s.sentEmailRepo.Count(ctx, &repository.SentEmailFilter{
		AccountUID: accountUID,
		Status:     model.SentEmailStatusFailed,
	})
	if err != nil {
		return nil, err
	}

	return &SentEmailStats{
		Total:   total,
		Success: success,
		Failed:  failed,
	}, nil
}
