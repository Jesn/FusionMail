package handler

import (
	"context"
	"log"
	"strings"
	"time"

	"fusionmail/internal/repository"
	"fusionmail/internal/service"
	pkgredis "fusionmail/pkg/redis"
	"fusionmail/pkg/synclock"
)

// PublicHandler 公共接口处理器
type PublicHandler struct {
	emailService     service.EmailService
	accountService   service.AccountService
	syncService      service.SyncService
	sendService      *service.SendService
	sentEmailService *service.SentEmailService
	emailRepo        repository.EmailRepository
	oauth2Service    *service.OAuth2Service
	groupService     service.GroupService
}

// NewPublicHandler 创建公共接口处理器实例
func NewPublicHandler(
	emailService service.EmailService,
	accountService service.AccountService,
	syncService service.SyncService,
	sendService *service.SendService,
	sentEmailService *service.SentEmailService,
	emailRepo repository.EmailRepository,
	oauth2Service *service.OAuth2Service,
	groupService service.GroupService,
) *PublicHandler {
	return &PublicHandler{
		emailService:     emailService,
		accountService:   accountService,
		syncService:      syncService,
		sendService:      sendService,
		sentEmailService: sentEmailService,
		emailRepo:        emailRepo,
		oauth2Service:    oauth2Service,
		groupService:     groupService,
	}
}

func isSyncInProgressError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "sync already in progress")
}

func isWebhookModeError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "webhook mode")
}

func isSyncDisabledError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "sync is disabled for account")
}

// waitForSyncComplete 等待指定账号的同步完成（用于 sync already in progress 的场景）
func (h *PublicHandler) waitForSyncComplete(ctx context.Context, accountUID string) error {
	redisClient := pkgredis.GetClient()

	// 优先使用 Redis 锁判断（支持多实例部署）
	if redisClient != nil {
		lock := synclock.NewSyncLock(redisClient)
		ticker := time.NewTicker(300 * time.Millisecond)
		defer ticker.Stop()

		for {
			locked, err := lock.IsLocked(ctx, accountUID)
			if err != nil {
				return err
			}
			if !locked {
				return nil
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-ticker.C:
			}
		}
	}

	// 无 Redis 时降级：使用本地进度追踪器判断（仅适用于单实例）
	if h.syncService == nil {
		return nil
	}

	ticker := time.NewTicker(300 * time.Millisecond)
	defer ticker.Stop()
	for {
		if h.syncService.GetSyncProgress(accountUID) == nil {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

// logAPIKeyUsage 记录 API Key 使用情况（占位符）
func (h *PublicHandler) logAPIKeyUsage(ctx interface{}, apiKeyID int64, email string, emailCount int) error {
	// TODO: 实现 API Key 使用日志记录
	// 可以在这里记录 API Key 的使用情况，用于审计和分析
	log.Printf("[INFO] API Key %d used to fetch %d emails from %s", apiKeyID, emailCount, email)
	return nil
}
