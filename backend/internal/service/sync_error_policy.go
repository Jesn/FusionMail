package service

import (
	"context"
	"errors"
	"strings"

	"fusionmail/internal/adapter"
	"fusionmail/internal/adapter/webapi"
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

// isAuthError 判断错误是否为认证错误（决定是否累计失败并禁用/软删账户）。
//
// 三层归因，从可靠到兜底：
//  1. 结构化：WebAPI 适配器的 401/403 返回 WebAPIError{Code: AUTH_FAILED}
//     （webapi/custom|cloudmail|cloudflare/adapter.go），errors.As 穿透 %w 包装链
//  2. 结构化：Graph/OAuth2 的 token 失败返回 TokenError（graph_quick.go:97），
//     invalid_grant 或 HTTP 401/403 视为认证失效
//  3. 兜底：其余来源（IMAP/SMTP 等）仍是字符串错误，只匹配完整短语
//
// 历史教训（AI 分析文档 §1.9）：旧实现裸匹配 "401"（误伤 "1401 bytes"、
// 日志行号）与 "auth"（误伤 "author"），错误消息里偶然出现这些字样就会
// 把临时故障计成认证失败，quick 账号 3 次即禁用 —— 已移除这两个裸模式。
func (s *SyncErrorPolicy) isAuthError(err error) bool {
	if err == nil {
		return false
	}

	// 第 1 层：WebAPI 结构化错误
	if webapi.IsAuthError(err) {
		return true
	}

	// 第 2 层：Graph/OAuth2 TokenError
	var tokenErr *adapter.TokenError
	if errors.As(err, &tokenErr) {
		if tokenErr.Code == "invalid_grant" {
			return true
		}
		if tokenErr.StatusCode == 401 || tokenErr.StatusCode == 403 {
			return true
		}
		return false
	}

	// 第 3 层：字符串兜底（仅完整短语，不用裸子串）
	errMsg := strings.ToLower(err.Error())
	authErrorPatterns := []string{
		"unauthorized",
		"token expired",
		"token has been expired",
		"invalid_grant",
		"authentication failed",
		"authenticate failed",
		"invalid credentials",
		"access denied",
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
