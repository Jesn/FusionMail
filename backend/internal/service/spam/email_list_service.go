package spam

import (
	"context"
	"fmt"
	"fusionmail/internal/dto"
	"fusionmail/internal/model"
	"fusionmail/internal/repository"
	"fusionmail/pkg/logger"
	"regexp"
	"strings"
	"time"
)

// 模块日志记录器
var emailListLog = logger.NewWithModule("EmailList")

// EmailListService 白名单/黑名单管理服务
type EmailListService struct {
	repo             repository.EmailListRepository
	whitelistChecker *WhitelistChecker
}

// NewEmailListService 创建白名单/黑名单管理服务实例
func NewEmailListService(repo repository.EmailListRepository, whitelistChecker *WhitelistChecker) *EmailListService {
	return &EmailListService{
		repo:             repo,
		whitelistChecker: whitelistChecker,
	}
}

// AddToWhitelist 添加到白名单
func (s *EmailListService) AddToWhitelist(ctx context.Context, userUID string, target string, reason string) (*model.EmailList, error) {
	return s.addToList(ctx, userUID, target, "whitelist", reason)
}

// AddToBlacklist 添加到黑名单
func (s *EmailListService) AddToBlacklist(ctx context.Context, userUID string, target string, reason string) (*model.EmailList, error) {
	return s.addToList(ctx, userUID, target, "blacklist", reason)
}

// addToList 添加到列表（通用方法）
func (s *EmailListService) addToList(ctx context.Context, userUID string, target string, listType string, reason string) (*model.EmailList, error) {
	// 1. 验证目标格式
	targetType, err := validateTarget(target)
	if err != nil {
		return nil, err
	}

	// 2. 检查是否已存在
	existing, err := s.repo.FindByTarget(ctx, userUID, target, listType)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing entry: %w", err)
	}
	if existing != nil {
		return existing, nil // 已存在，直接返回
	}

	// 3. 创建新条目
	list := &model.EmailList{
		UserUID:    userUID,
		Type:       listType,
		Target:     strings.ToLower(target), // 统一转为小写
		TargetType: targetType,
		Reason:     reason,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := s.repo.Create(ctx, list); err != nil {
		return nil, fmt.Errorf("failed to create list entry: %w", err)
	}

	// 4. 使缓存失效
	if s.whitelistChecker != nil {
		if err := s.whitelistChecker.InvalidateCache(ctx, userUID, target); err != nil {
			// 缓存失效失败不影响主流程，只记录日志
			emailListLog.Warn("缓存失效失败: user=%s, target=%s, err=%v", userUID, target, err)
		}
	}

	return list, nil
}

// RemoveFromWhitelist 从白名单中移除
func (s *EmailListService) RemoveFromWhitelist(ctx context.Context, id int64, userUID string) error {
	return s.removeFromList(ctx, id, userUID, "whitelist")
}

// RemoveFromBlacklist 从黑名单中移除
func (s *EmailListService) RemoveFromBlacklist(ctx context.Context, id int64, userUID string) error {
	return s.removeFromList(ctx, id, userUID, "blacklist")
}

// removeFromList 从列表中移除（通用方法）
func (s *EmailListService) removeFromList(ctx context.Context, id int64, userUID string, listType string) error {
	// 1. 查找条目
	list, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to find list entry: %w", err)
	}
	if list == nil {
		return dto.NewAPIErrorWithMessage(dto.ErrResourceNotFound, "条目不存在")
	}

	// 2. 验证权限（确保是用户自己的条目）
	if list.UserUID != userUID {
		return dto.NewAPIErrorWithMessage(dto.ErrForbidden, "无权限删除此条目")
	}

	// 3. 验证类型
	if list.Type != listType {
		return dto.NewAPIErrorWithMessage(dto.ErrInvalidRequest, "列表类型不匹配")
	}

	// 4. 删除条目
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete list entry: %w", err)
	}

	// 5. 使缓存失效
	if s.whitelistChecker != nil {
		if err := s.whitelistChecker.InvalidateCache(ctx, userUID, list.Target); err != nil {
			// 缓存失效失败不影响主流程，只记录日志
			emailListLog.Warn("缓存失效失败: user=%s, target=%s, err=%v", userUID, list.Target, err)
		}
	}

	return nil
}

// GetWhitelist 获取白名单
func (s *EmailListService) GetWhitelist(ctx context.Context, userUID string, offset, limit int) ([]*model.EmailList, int64, error) {
	return s.repo.List(ctx, userUID, "whitelist", offset, limit)
}

// GetBlacklist 获取黑名单
func (s *EmailListService) GetBlacklist(ctx context.Context, userUID string, offset, limit int) ([]*model.EmailList, int64, error) {
	return s.repo.List(ctx, userUID, "blacklist", offset, limit)
}

// validateTarget 验证目标格式（邮箱地址或域名）
func validateTarget(target string) (string, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return "", fmt.Errorf("target cannot be empty")
	}

	// 转为小写
	target = strings.ToLower(target)

	// 检查是否为邮箱地址
	emailRegex := regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}$`)
	if emailRegex.MatchString(target) {
		return "email", nil
	}

	// 检查是否为域名
	domainRegex := regexp.MustCompile(`^[a-z0-9.\-]+\.[a-z]{2,}$`)
	if domainRegex.MatchString(target) {
		return "domain", nil
	}

	return "", fmt.Errorf("invalid email address or domain format")
}
