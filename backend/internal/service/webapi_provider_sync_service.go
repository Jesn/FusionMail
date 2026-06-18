package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"fusionmail/internal/adapter/webapi"
	"fusionmail/internal/model"
)

// TriggerSync 手动触发同步
func (s *WebAPIProviderService) TriggerSync(ctx context.Context, uid string) error {
	// 1. 获取账户
	account, err := s.GetByUID(ctx, uid)
	if err != nil {
		return err
	}

	// 2. 解密认证数据
	authDataBytes, err := s.cryptoSvc.Decrypt(account.EncryptedCredentials)
	if err != nil {
		return fmt.Errorf("解密认证数据失败: %w", err)
	}

	// 3. 获取服务类型
	serviceType, err := s.getServiceTypeFromAccount(ctx, account)
	if err != nil {
		return err
	}

	// 4. 创建适配器
	adapter, err := s.factory.CreateAdapter(serviceType, string(authDataBytes))
	if err != nil {
		return fmt.Errorf("创建适配器失败: %w", err)
	}

	// 5. 为 Cloudflare Temp Email 适配器设置 token 更新回调
	s.setupTokenUpdateCallback(adapter, account.UID, serviceType, string(authDataBytes))

	// 6. 异步执行同步
	go func() {
		syncCtx := context.Background()
		_, syncErr := s.syncService.SyncProvider(syncCtx, adapter, account.UID)
		if syncErr != nil {
			s.log.Error("同步失败: uid=%s, error=%v", uid, syncErr)
		}
	}()

	s.log.Info("触发同步: uid=%s", uid)
	return nil
}

// setupTokenUpdateCallback 为适配器设置 token 更新回调
// 当 Cloudflare Temp Email 的 user_token 被刷新时，自动保存新 token
func (s *WebAPIProviderService) setupTokenUpdateCallback(adapter webapi.WebAPIProvider, accountUID, serviceType, authDataJSON string) {
	// 只有 Cloudflare Temp Email 需要设置回调
	if serviceType != model.WebAPIServiceTypeCloudflareTempEmail {
		return
	}

	// 类型断言获取 Cloudflare 适配器
	cfAdapter, ok := adapter.(interface {
		SetTokenUpdateCallback(func(string) error)
	})
	if !ok {
		return
	}

	// 设置回调函数
	cfAdapter.SetTokenUpdateCallback(func(newUserToken string) error {
		return s.updateUserToken(context.Background(), accountUID, authDataJSON, newUserToken)
	})
}

// updateUserToken 更新账户的 user_token
func (s *WebAPIProviderService) updateUserToken(ctx context.Context, accountUID, oldAuthDataJSON, newUserToken string) error {
	// 1. 获取账户
	account, err := s.accountRepo.FindByUID(ctx, accountUID)
	if err != nil {
		return fmt.Errorf("账户未找到: %w", err)
	}

	// 2. 解析原有配置
	var config model.CloudflareTempEmailAuthData
	if err := json.Unmarshal([]byte(oldAuthDataJSON), &config); err != nil {
		return fmt.Errorf("解析配置失败: %w", err)
	}

	// 3. 更新 user_token
	config.UserToken = newUserToken

	// 4. 序列化新配置
	newAuthDataJSON, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	// 5. 加密并保存
	encryptedAuthData, err := s.cryptoSvc.Encrypt(newAuthDataJSON)
	if err != nil {
		return fmt.Errorf("加密认证数据失败: %w", err)
	}

	account.EncryptedCredentials = encryptedAuthData
	account.UpdatedAt = time.Now()

	if err := s.accountRepo.Update(ctx, account); err != nil {
		return fmt.Errorf("更新账户失败: %w", err)
	}

	s.log.Info("user_token 已更新: uid=%s", accountUID)
	return nil
}

// SyncStatus 同步状态
type SyncStatus struct {
	Status       string     `json:"status"`                   // 同步状态
	LastSyncAt   *time.Time `json:"last_sync_at,omitempty"`   // 上次同步时间
	LastSyncedID string     `json:"last_synced_id,omitempty"` // 上次同步的 ID
	EmailCount   int64      `json:"email_count"`              // 邮件数量
	ErrorMessage string     `json:"error_message,omitempty"`  // 错误信息
}

// GetSyncStatus 获取同步状态
func (s *WebAPIProviderService) GetSyncStatus(ctx context.Context, uid string) (*SyncStatus, error) {
	account, err := s.GetByUID(ctx, uid)
	if err != nil {
		return nil, err
	}

	// 获取邮件数量
	emailCount, err := s.emailRepo.CountByAccount(ctx, uid)
	if err != nil {
		emailCount = 0
	}

	return &SyncStatus{
		Status:       account.LastSyncStatus,
		LastSyncAt:   account.LastSyncAt,
		EmailCount:   emailCount,
		ErrorMessage: account.LastSyncError,
	}, nil
}
