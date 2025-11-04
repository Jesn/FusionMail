package adapter

import (
	"fmt"
	"testing"
	"time"
)

// TestAdapterManager_GetAdapter 测试获取适配器
func TestAdapterManager_GetAdapter(t *testing.T) {
	manager := NewAdapterManager(nil)

	config := &Config{
		Email:    "test@outlook.com",
		Provider: "outlook",
		AuthType: "quick",
		Credentials: &Credentials{
			ClientID:     "test_client_id",
			RefreshToken: "test_refresh_token",
		},
		Timeout: 30 * time.Second,
	}

	// 第一次获取适配器
	adapter1, err := manager.GetAdapter("account1", config, AdapterTypeQuick)
	if err != nil {
		t.Fatalf("Failed to get adapter: %v", err)
	}

	if adapter1 == nil {
		t.Fatal("Expected adapter but got nil")
	}

	// 第二次获取相同适配器（应该使用缓存）
	adapter2, err := manager.GetAdapter("account1", config, AdapterTypeQuick)
	if err != nil {
		t.Fatalf("Failed to get cached adapter: %v", err)
	}

	if adapter1 != adapter2 {
		t.Error("Expected same adapter instance from cache")
	}

	// 验证适配器类型
	if adapter1.GetProtocol() != "graph_quick" {
		t.Errorf("Expected protocol 'graph_quick', got %s", adapter1.GetProtocol())
	}
}

// TestAdapterManager_SwitchAdapter 测试切换适配器
func TestAdapterManager_SwitchAdapter(t *testing.T) {
	manager := NewAdapterManager(nil)

	quickConfig := &Config{
		Email:    "test@outlook.com",
		Provider: "outlook",
		AuthType: "quick",
		Credentials: &Credentials{
			ClientID:     "test_client_id",
			RefreshToken: "test_refresh_token",
		},
		Timeout: 30 * time.Second,
	}

	standardConfig := &Config{
		Email:    "test@outlook.com",
		Provider: "outlook",
		AuthType: "standard",
		Credentials: &Credentials{
			AccessToken:  "test_access_token",
			ClientID:     "test_client_id",
			ClientSecret: "test_client_secret",
			RefreshToken: "test_refresh_token",
		},
		Timeout: 30 * time.Second,
	}

	// 创建短效适配器
	quickAdapter, err := manager.GetAdapter("account1", quickConfig, AdapterTypeQuick)
	if err != nil {
		t.Fatalf("Failed to get quick adapter: %v", err)
	}

	if quickAdapter.GetProtocol() != "graph_quick" {
		t.Errorf("Expected protocol 'graph_quick', got %s", quickAdapter.GetProtocol())
	}

	// 切换到标准适配器
	standardAdapter, err := manager.SwitchAdapter("account1", standardConfig, AdapterTypeStandard)
	if err != nil {
		t.Fatalf("Failed to switch to standard adapter: %v", err)
	}

	if standardAdapter.GetProtocol() != "graph" {
		t.Errorf("Expected protocol 'graph', got %s", standardAdapter.GetProtocol())
	}

	// 验证适配器已切换
	if quickAdapter == standardAdapter {
		t.Error("Expected different adapter instances after switch")
	}
}

// TestAdapterManager_AutoSelection 测试自动适配器选择
func TestAdapterManager_AutoSelection(t *testing.T) {
	manager := NewAdapterManager(nil)

	tests := []struct {
		name             string
		config           *Config
		expectedType     string
		expectedProtocol string
	}{
		{
			name: "自动选择短效适配器",
			config: &Config{
				Email:    "test@outlook.com",
				Provider: "outlook",
				Credentials: &Credentials{
					ClientID:     "test_client_id",
					RefreshToken: "test_refresh_token",
					// 没有 ClientSecret
				},
				Timeout: 30 * time.Second,
			},
			expectedType:     "*adapter.GraphQuickAdapter",
			expectedProtocol: "graph_quick",
		},
		{
			name: "自动选择标准适配器",
			config: &Config{
				Email:    "test@outlook.com",
				Provider: "outlook",
				Credentials: &Credentials{
					AccessToken: "test_access_token",
					ClientID:    "test_client_id",
				},
				Timeout: 30 * time.Second,
			},
			expectedType:     "*adapter.GraphAdapter",
			expectedProtocol: "graph",
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			accountID := fmt.Sprintf("account%d", i+1)

			adapter, err := manager.GetAdapter(accountID, tt.config, AdapterTypeAuto)
			if err != nil {
				t.Fatalf("Failed to get auto adapter: %v", err)
			}

			actualType := fmt.Sprintf("%T", adapter)
			if actualType != tt.expectedType {
				t.Errorf("Expected type %s, got %s", tt.expectedType, actualType)
			}

			if adapter.GetProtocol() != tt.expectedProtocol {
				t.Errorf("Expected protocol %s, got %s", tt.expectedProtocol, adapter.GetProtocol())
			}
		})
	}
}

// TestAdapterManager_GetAdapterInfo 测试获取适配器信息
func TestAdapterManager_GetAdapterInfo(t *testing.T) {
	manager := NewAdapterManager(nil)

	config := &Config{
		Email:    "test@outlook.com",
		Provider: "outlook",
		AuthType: "quick",
		Credentials: &Credentials{
			ClientID:     "test_client_id",
			RefreshToken: "test_refresh_token",
		},
		Timeout: 30 * time.Second,
	}

	// 创建适配器
	_, err := manager.GetAdapter("account1", config, AdapterTypeQuick)
	if err != nil {
		t.Fatalf("Failed to create adapter: %v", err)
	}

	// 获取适配器信息
	info, err := manager.GetAdapterInfo("account1")
	if err != nil {
		t.Fatalf("Failed to get adapter info: %v", err)
	}

	if info.AccountID != "account1" {
		t.Errorf("Expected account ID 'account1', got %s", info.AccountID)
	}

	if info.Email != "test@outlook.com" {
		t.Errorf("Expected email 'test@outlook.com', got %s", info.Email)
	}

	if info.Provider != "outlook" {
		t.Errorf("Expected provider 'outlook', got %s", info.Provider)
	}

	if info.Protocol != "graph_quick" {
		t.Errorf("Expected protocol 'graph_quick', got %s", info.Protocol)
	}
}

// TestAdapterManager_ListAdapters 测试列出适配器
func TestAdapterManager_ListAdapters(t *testing.T) {
	manager := NewAdapterManager(nil)

	configs := []*Config{
		{
			Email:    "test1@outlook.com",
			Provider: "outlook",
			AuthType: "quick",
			Credentials: &Credentials{
				ClientID:     "client1",
				RefreshToken: "token1",
			},
			Timeout: 30 * time.Second,
		},
		{
			Email:    "test2@outlook.com",
			Provider: "outlook",
			AuthType: "standard",
			Credentials: &Credentials{
				AccessToken: "access_token2",
				ClientID:    "client2",
			},
			Timeout: 30 * time.Second,
		},
	}

	// 创建多个适配器
	for i, config := range configs {
		accountID := fmt.Sprintf("account%d", i+1)
		adapterType := AdapterTypeQuick
		if i == 1 {
			adapterType = AdapterTypeStandard
		}

		_, err := manager.GetAdapter(accountID, config, adapterType)
		if err != nil {
			t.Fatalf("Failed to create adapter %d: %v", i+1, err)
		}
	}

	// 列出所有适配器
	infos, err := manager.ListAdapters()
	if err != nil {
		t.Fatalf("Failed to list adapters: %v", err)
	}

	if len(infos) != 2 {
		t.Errorf("Expected 2 adapters, got %d", len(infos))
	}

	// 验证适配器信息
	accountIDs := make(map[string]bool)
	for _, info := range infos {
		accountIDs[info.AccountID] = true
	}

	if !accountIDs["account1"] || !accountIDs["account2"] {
		t.Error("Expected both account1 and account2 in adapter list")
	}
}

// TestAdapterManager_RemoveAdapter 测试移除适配器
func TestAdapterManager_RemoveAdapter(t *testing.T) {
	manager := NewAdapterManager(nil)

	config := &Config{
		Email:    "test@outlook.com",
		Provider: "outlook",
		AuthType: "quick",
		Credentials: &Credentials{
			ClientID:     "test_client_id",
			RefreshToken: "test_refresh_token",
		},
		Timeout: 30 * time.Second,
	}

	// 创建适配器
	_, err := manager.GetAdapter("account1", config, AdapterTypeQuick)
	if err != nil {
		t.Fatalf("Failed to create adapter: %v", err)
	}

	// 验证适配器存在
	_, err = manager.GetAdapterInfo("account1")
	if err != nil {
		t.Fatalf("Adapter should exist: %v", err)
	}

	// 移除适配器
	err = manager.RemoveAdapter("account1")
	if err != nil {
		t.Fatalf("Failed to remove adapter: %v", err)
	}

	// 验证适配器已移除
	_, err = manager.GetAdapterInfo("account1")
	if err == nil {
		t.Error("Expected error when getting removed adapter")
	}
}

// TestAdapterManager_GetStats 测试获取统计信息
func TestAdapterManager_GetStats(t *testing.T) {
	manager := NewAdapterManager(nil)

	configs := []*Config{
		{
			Email:    "test1@outlook.com",
			Provider: "outlook",
			AuthType: "quick",
			Credentials: &Credentials{
				ClientID:     "client1",
				RefreshToken: "token1",
			},
			Timeout: 30 * time.Second,
		},
		{
			Email:    "test2@outlook.com",
			Provider: "outlook",
			AuthType: "standard",
			Credentials: &Credentials{
				AccessToken: "access_token2",
				ClientID:    "client2",
			},
			Timeout: 30 * time.Second,
		},
	}

	// 创建不同类型的适配器
	_, err := manager.GetAdapter("account1", configs[0], AdapterTypeQuick)
	if err != nil {
		t.Fatalf("Failed to create quick adapter: %v", err)
	}

	_, err = manager.GetAdapter("account2", configs[1], AdapterTypeStandard)
	if err != nil {
		t.Fatalf("Failed to create standard adapter: %v", err)
	}

	// 获取统计信息
	stats := manager.GetStats()

	if stats.TotalAdapters != 2 {
		t.Errorf("Expected 2 total adapters, got %d", stats.TotalAdapters)
	}

	if stats.AdapterTypes["*adapter.GraphQuickAdapter"] != 1 {
		t.Errorf("Expected 1 GraphQuickAdapter, got %d", stats.AdapterTypes["*adapter.GraphQuickAdapter"])
	}

	if stats.AdapterTypes["*adapter.GraphAdapter"] != 1 {
		t.Errorf("Expected 1 GraphAdapter, got %d", stats.AdapterTypes["*adapter.GraphAdapter"])
	}
}

// TestAdapterManager_ConfigChange 测试配置变更
func TestAdapterManager_ConfigChange(t *testing.T) {
	manager := NewAdapterManager(nil)

	originalConfig := &Config{
		Email:    "test@outlook.com",
		Provider: "outlook",
		AuthType: "quick",
		Credentials: &Credentials{
			ClientID:     "original_client_id",
			RefreshToken: "original_refresh_token",
		},
		Timeout: 30 * time.Second,
	}

	// 创建适配器
	adapter1, err := manager.GetAdapter("account1", originalConfig, AdapterTypeQuick)
	if err != nil {
		t.Fatalf("Failed to create adapter: %v", err)
	}

	// 修改配置
	modifiedConfig := &Config{
		Email:    "test@outlook.com",
		Provider: "outlook",
		AuthType: "quick",
		Credentials: &Credentials{
			ClientID:     "modified_client_id", // 修改了 ClientID
			RefreshToken: "original_refresh_token",
		},
		Timeout: 30 * time.Second,
	}

	// 使用修改后的配置获取适配器
	adapter2, err := manager.GetAdapter("account1", modifiedConfig, AdapterTypeQuick)
	if err != nil {
		t.Fatalf("Failed to get adapter with modified config: %v", err)
	}

	// 应该创建新的适配器实例
	if adapter1 == adapter2 {
		t.Error("Expected different adapter instances when config changes")
	}
}

// TestAdapterManager_Cleanup 测试清理
func TestAdapterManager_Cleanup(t *testing.T) {
	manager := NewAdapterManager(nil)

	config := &Config{
		Email:    "test@outlook.com",
		Provider: "outlook",
		AuthType: "quick",
		Credentials: &Credentials{
			ClientID:     "test_client_id",
			RefreshToken: "test_refresh_token",
		},
		Timeout: 30 * time.Second,
	}

	// 创建多个适配器
	for i := 1; i <= 3; i++ {
		accountID := fmt.Sprintf("account%d", i)
		_, err := manager.GetAdapter(accountID, config, AdapterTypeQuick)
		if err != nil {
			t.Fatalf("Failed to create adapter %d: %v", i, err)
		}
	}

	// 验证适配器存在
	stats := manager.GetStats()
	if stats.TotalAdapters != 3 {
		t.Errorf("Expected 3 adapters before cleanup, got %d", stats.TotalAdapters)
	}

	// 清理所有适配器
	err := manager.Cleanup()
	if err != nil {
		t.Fatalf("Failed to cleanup adapters: %v", err)
	}

	// 验证适配器已清理
	stats = manager.GetStats()
	if stats.TotalAdapters != 0 {
		t.Errorf("Expected 0 adapters after cleanup, got %d", stats.TotalAdapters)
	}
}

// TestAdapterManager_ErrorHandling 测试错误处理
func TestAdapterManager_ErrorHandling(t *testing.T) {
	manager := NewAdapterManager(nil)

	tests := []struct {
		name      string
		accountID string
		config    *Config
		expectErr bool
	}{
		{
			name:      "空账户ID",
			accountID: "",
			config:    &Config{},
			expectErr: true,
		},
		{
			name:      "空配置",
			accountID: "account1",
			config:    nil,
			expectErr: true,
		},
		{
			name:      "无效配置",
			accountID: "account1",
			config: &Config{
				Email:       "test@outlook.com",
				Provider:    "outlook",
				Credentials: nil, // 缺少凭据
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := manager.GetAdapter(tt.accountID, tt.config, AdapterTypeAuto)

			if tt.expectErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}
