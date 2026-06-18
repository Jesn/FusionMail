package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"fusionmail/internal/model"
)

// isAuthError 判断错误是否为认证错误
// 认证错误包括：HTTP 401、token 过期、invalid_grant、WebAPI 连接失败等
func (s *syncService) isAuthError(err error) bool {
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
func (s *syncService) handleSyncError(ctx context.Context, account *model.EmailAccount, err error) error {
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

// CleanupStaleSyncLogs 清理卡住的同步日志
// 将超过指定时间仍处于 running 状态的同步日志标记为失败
// 同时更新对应账户的同步状态
func (s *syncService) CleanupStaleSyncLogs(ctx context.Context, maxAge time.Duration) (int, error) {
	// 查找所有卡住的同步日志
	staleLogs, err := s.syncLogRepo.FindStaleRunning(ctx, maxAge)
	if err != nil {
		return 0, fmt.Errorf("failed to find stale sync logs: %w", err)
	}

	if len(staleLogs) == 0 {
		return 0, nil
	}

	cleanedCount := 0
	errorMsg := fmt.Sprintf("同步超时 - 任务运行超过 %v 未完成，已自动清理", maxAge)

	for _, syncLog := range staleLogs {
		// 更新同步日志状态
		syncLog.Status = "failed"
		syncLog.ErrorMessage = errorMsg
		now := time.Now()
		syncLog.CompletedAt = &now
		syncLog.DurationMs = time.Since(syncLog.StartedAt).Milliseconds()

		if updateErr := s.syncLogRepo.Update(ctx, syncLog); updateErr != nil {
			s.logger.Error("更新过期同步日志失败: logId=%d, err=%v", syncLog.ID, updateErr)
			continue
		}

		// 更新账户的同步状态（保存错误信息）
		if updateErr := s.accountRepo.UpdateSyncStatus(ctx, syncLog.AccountUID, "failed", errorMsg); updateErr != nil {
			s.logger.Error("更新账户同步状态失败: account=%s, err=%v", syncLog.AccountUID, updateErr)
		}

		// 清理内存中的锁（如果存在）
		s.syncMu.Lock()
		if lockInfo, exists := s.activeSyncs[syncLog.AccountUID]; exists {
			if lockInfo.CancelFunc != nil {
				lockInfo.CancelFunc()
			}
			delete(s.activeSyncs, syncLog.AccountUID)
			delete(s.activeTrackers, syncLog.AccountUID)
		}
		s.syncMu.Unlock()

		// 清理 Redis 锁（如果存在）
		if s.syncLock != nil {
			_ = s.syncLock.ForceReleaseLock(ctx, syncLog.AccountUID)
		}

		s.logger.Info("已清理过期同步日志: logId=%d, account=%s", syncLog.ID, syncLog.AccountUID)
		cleanedCount++
	}

	return cleanedCount, nil
}
