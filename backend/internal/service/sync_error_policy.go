package service

import (
	"context"
	"strings"

	"fusionmail/internal/model"
	"fusionmail/internal/repository"
	"fusionmail/pkg/logger"
)

// SyncErrorPolicy 负责同步失败的分类与落库（认证失效 vs 其他错误）。
//
// 为什么从 syncService 里拆出来：原类型跨 6 个文件、1782 行、30 个方法、15 个字段。
// 本类型只依赖 accountRepo / syncLogRepo / logger，是依赖图上的叶子。
//
// 注意：CleanupStaleSyncLogs（现属 SyncScheduler）刻意**没有**拆进来 —— 它需要
// 读写活跃同步的共享可变状态（syncMu / activeSyncs / activeTrackers / syncLock），
// 这组状态已收敛到 SyncProgressRegistry，由调度器经注册表清理，与本类型无关。
type SyncErrorPolicy struct {
	accountRepo repository.AccountRepository
	syncLogRepo repository.SyncLogRepository
	logger      *logger.Logger
}

// NewSyncErrorPolicy 创建同步错误策略服务。
func NewSyncErrorPolicy(
	accountRepo repository.AccountRepository,
	syncLogRepo repository.SyncLogRepository,
) *SyncErrorPolicy {
	return &SyncErrorPolicy{
		accountRepo: accountRepo,
		syncLogRepo: syncLogRepo,
		logger:      logger.NewWithModule("SyncErrorPolicy"),
	}
}

// isAuthError 判断错误是否为认证错误
// 认证错误包括：HTTP 401、token 过期、invalid_grant、WebAPI 连接失败等
func (s *SyncErrorPolicy) isAuthError(err error) bool {
	if err == nil {
		return false
	}

	errMsg := strings.ToLower(err.Error())

	// 检查 HTTP 401 状态码
	if strings.Contains(errMsg, "401") || strings.Contains(errMsg, "unauthorized") {
		return true
	}

	// 检查 WebAPI 连接失败（包括解析错误、网络错误等）
	if strings.Contains(errMsg, "webapi 连接失败") || strings.Contains(errMsg, "webapi connection failed") {
		return true
	}

	// 检查 token 过期相关错误
	authErrorPatterns := []string{
		"token expired",
		"token has been expired",
		"invalid_grant",
		"authentication failed",
		"authenticate failed",
		"invalid credentials",
		"access denied",
		"auth",
	}

	for _, pattern := range authErrorPatterns {
		if strings.Contains(errMsg, pattern) {
			return true
		}
	}

	return false
}

// handleSyncError 处理同步错误
// handleSyncError 处理同步错误
// 对于认证错误，进行失败计数和自动处理：
// - quick 类型账号：连续 3 次失败后自动禁用
// - oauth2 类型的 Outlook 账号：连续 10 次失败后自动软删除（放入回收站）
// - 其他类型账号（SMTP、WebAPI、IMAP、POP3 等）：连续 10 次失败后自动禁用
func (s *SyncErrorPolicy) handleSyncError(ctx context.Context, account *model.EmailAccount, err error) error {
	// 判断是否为认证错误
	if !s.isAuthError(err) {
		return err
	}

	// 判断账号类型，决定处理策略
	authType := account.GetAuthType()
	providerName := account.GetProviderName()
	protocol := account.GetProtocol()

	isQuickAccount := authType == "quick"
	isOAuth2Outlook := authType == "oauth2" && providerName == "outlook"
	isOtherAccount := !isQuickAccount && !isOAuth2Outlook // SMTP、WebAPI、IMAP、POP3 等其他类型

	// 增加失败计数（所有账号类型都记录）
	failureCount, incErr := s.accountRepo.IncrementConsecutiveFailures(ctx, account.UID)
	if incErr != nil {
		s.logger.Error("增加失败计数失败: account=%s, err=%v", account.UID, incErr)
		return err
	}

	// 根据账号类型设置阈值和处理方式
	if isQuickAccount {
		// Quick 账号：3 次失败后自动禁用
		threshold := 3
		s.logger.Warn("Quick账号认证失败: account=%s, count=%d/%d", account.UID, failureCount, threshold)

		if failureCount >= threshold {
			disableErr := s.accountRepo.AutoDisableAccount(ctx, account.UID, "auto_disabled_auth_failure")
			if disableErr != nil {
				s.logger.Error("自动禁用账号失败: account=%s, err=%v", account.UID, disableErr)
			} else {
				s.logger.Info("已自动禁用Quick账号: account=%s, failures=%d", account.UID, failureCount)
			}
		}
	} else if isOAuth2Outlook {
		// OAuth2 Outlook 账号：10 次失败后自动软删除
		threshold := 10
		s.logger.Warn("OAuth2 Outlook账号认证失败: account=%s, count=%d/%d", account.UID, failureCount, threshold)

		if failureCount >= threshold {
			softDeleteErr := s.accountRepo.AutoSoftDeleteAccount(ctx, account.UID, "auto_recycled_token_invalid")
			if softDeleteErr != nil {
				s.logger.Error("自动回收账号失败: account=%s, err=%v", account.UID, softDeleteErr)
			} else {
				s.logger.Info("已自动回收OAuth2 Outlook账号: account=%s, failures=%d", account.UID, failureCount)
			}
		}
	} else if isOtherAccount {
		// 其他类型账号（SMTP、WebAPI、IMAP、POP3 等）：10 次失败后自动禁用
		threshold := 10
		s.logger.Warn("账号认证失败: account=%s, type=%s, protocol=%s, count=%d/%d",
			account.UID, authType, protocol, failureCount, threshold)

		if failureCount >= threshold {
			disableErr := s.accountRepo.AutoDisableAccount(ctx, account.UID, "auto_disabled_auth_failure")
			if disableErr != nil {
				s.logger.Error("自动禁用账号失败: account=%s, err=%v", account.UID, disableErr)
			} else {
				s.logger.Info("已自动禁用账号: account=%s, type=%s, protocol=%s, failures=%d",
					account.UID, authType, protocol, failureCount)
			}
		}
	}

	return err
}
