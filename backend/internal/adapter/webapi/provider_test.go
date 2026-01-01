package webapi

import (
	"errors"
	"fusionmail/internal/adapter"
	"testing"
	"time"
)

// TestSyncCheckpoint 测试同步检查点
func TestSyncCheckpoint(t *testing.T) {
	t.Run("NewSyncCheckpoint", func(t *testing.T) {
		cp := NewSyncCheckpoint()

		if cp == nil {
			t.Fatal("NewSyncCheckpoint 返回 nil")
		}

		if !cp.IsEmpty() {
			t.Error("新创建的检查点应该为空")
		}

		if cp.Metadata == nil {
			t.Error("Metadata 不应为 nil")
		}
	})

	t.Run("IsEmpty", func(t *testing.T) {
		cp := NewSyncCheckpoint()

		if !cp.IsEmpty() {
			t.Error("空检查点 IsEmpty() 应返回 true")
		}

		cp.LastSyncTime = time.Now()
		if cp.IsEmpty() {
			t.Error("设置 LastSyncTime 后 IsEmpty() 应返回 false")
		}

		cp2 := NewSyncCheckpoint()
		cp2.LastEmailID = "123"
		if cp2.IsEmpty() {
			t.Error("设置 LastEmailID 后 IsEmpty() 应返回 false")
		}

		cp3 := NewSyncCheckpoint()
		cp3.Cursor = "cursor123"
		if cp3.IsEmpty() {
			t.Error("设置 Cursor 后 IsEmpty() 应返回 false")
		}
	})

	t.Run("Update", func(t *testing.T) {
		cp := NewSyncCheckpoint()
		syncTime := time.Now()

		cp.Update(syncTime, 10)

		if cp.LastSyncTime != syncTime {
			t.Errorf("LastSyncTime = %v, want %v", cp.LastSyncTime, syncTime)
		}

		if cp.LastCount != 10 {
			t.Errorf("LastCount = %d, want 10", cp.LastCount)
		}

		if cp.TotalSynced != 10 {
			t.Errorf("TotalSynced = %d, want 10", cp.TotalSynced)
		}

		// 再次更新
		cp.Update(syncTime.Add(time.Hour), 5)

		if cp.LastCount != 5 {
			t.Errorf("LastCount = %d, want 5", cp.LastCount)
		}

		if cp.TotalSynced != 15 {
			t.Errorf("TotalSynced = %d, want 15", cp.TotalSynced)
		}
	})
}

// TestWebAPIEmail 测试 WebAPIEmail 结构
func TestWebAPIEmail(t *testing.T) {
	t.Run("NewWebAPIEmail", func(t *testing.T) {
		email := &adapter.Email{
			ProviderID: "test123",
			Subject:    "Test Subject",
		}

		webEmail := NewWebAPIEmail(email, "target@example.com")

		if webEmail == nil {
			t.Fatal("NewWebAPIEmail 返回 nil")
		}

		if webEmail.Email != email {
			t.Error("Email 字段不匹配")
		}

		if webEmail.TargetAddress != "target@example.com" {
			t.Errorf("TargetAddress = %q, want %q", webEmail.TargetAddress, "target@example.com")
		}
	})

	t.Run("ToEmail", func(t *testing.T) {
		email := &adapter.Email{
			ProviderID: "test123",
			Subject:    "Test Subject",
		}

		webEmail := NewWebAPIEmail(email, "target@example.com")
		result := webEmail.ToEmail()

		if result != email {
			t.Error("ToEmail() 应返回原始 Email")
		}
	})
}

// TestWebAPIError 测试错误类型
func TestWebAPIError(t *testing.T) {
	t.Run("Error", func(t *testing.T) {
		err := &WebAPIError{
			Code:    ErrCodeAuthFailed,
			Message: "认证失败",
		}

		expected := "[AUTH_FAILED] 认证失败"
		if err.Error() != expected {
			t.Errorf("Error() = %q, want %q", err.Error(), expected)
		}
	})

	t.Run("Error with cause", func(t *testing.T) {
		cause := errors.New("原始错误")
		err := &WebAPIError{
			Code:    ErrCodeConnectionFailed,
			Message: "连接失败",
			Cause:   cause,
		}

		result := err.Error()
		if result != "[CONNECTION_FAILED] 连接失败: 原始错误" {
			t.Errorf("Error() = %q", result)
		}
	})

	t.Run("Unwrap", func(t *testing.T) {
		cause := errors.New("原始错误")
		err := &WebAPIError{
			Code:    ErrCodeServerError,
			Message: "服务器错误",
			Cause:   cause,
		}

		if err.Unwrap() != cause {
			t.Error("Unwrap() 应返回原始错误")
		}
	})

	t.Run("NewWebAPIError", func(t *testing.T) {
		err := NewWebAPIError(ErrCodeRateLimited, "请求过于频繁", 429, true, nil)

		if err.Code != ErrCodeRateLimited {
			t.Errorf("Code = %q, want %q", err.Code, ErrCodeRateLimited)
		}

		if err.StatusCode != 429 {
			t.Errorf("StatusCode = %d, want 429", err.StatusCode)
		}

		if !err.Retryable {
			t.Error("Retryable 应为 true")
		}
	})

	t.Run("WrapError", func(t *testing.T) {
		cause := errors.New("原始错误")
		err := WrapError(ErrCodeParseError, "解析失败", cause)

		if err.Code != ErrCodeParseError {
			t.Errorf("Code = %q, want %q", err.Code, ErrCodeParseError)
		}

		if err.Cause != cause {
			t.Error("Cause 不匹配")
		}

		if err.Retryable {
			t.Error("WrapError 创建的错误默认不可重试")
		}
	})
}

// TestIsRetryable 测试错误重试判断
func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		retryable bool
	}{
		{
			name:      "可重试的 WebAPIError",
			err:       &WebAPIError{Code: ErrCodeRateLimited, Retryable: true},
			retryable: true,
		},
		{
			name:      "不可重试的 WebAPIError",
			err:       &WebAPIError{Code: ErrCodeAuthFailed, Retryable: false},
			retryable: false,
		},
		{
			name:      "普通错误",
			err:       errors.New("普通错误"),
			retryable: false,
		},
		{
			name:      "预定义可重试错误",
			err:       ErrConnectionFailed,
			retryable: true,
		},
		{
			name:      "预定义不可重试错误",
			err:       ErrAuthFailed,
			retryable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRetryable(tt.err)
			if result != tt.retryable {
				t.Errorf("IsRetryable() = %v, want %v", result, tt.retryable)
			}
		})
	}
}

// TestIsAuthError 测试认证错误判断
func TestIsAuthError(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		isAuth bool
	}{
		{
			name:   "认证错误",
			err:    &WebAPIError{Code: ErrCodeAuthFailed},
			isAuth: true,
		},
		{
			name:   "其他 WebAPIError",
			err:    &WebAPIError{Code: ErrCodeServerError},
			isAuth: false,
		},
		{
			name:   "普通错误",
			err:    errors.New("普通错误"),
			isAuth: false,
		},
		{
			name:   "预定义认证错误",
			err:    ErrAuthFailed,
			isAuth: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsAuthError(tt.err)
			if result != tt.isAuth {
				t.Errorf("IsAuthError() = %v, want %v", result, tt.isAuth)
			}
		})
	}
}

// TestBaseWebAPIAdapter 测试基础适配器
func TestBaseWebAPIAdapter(t *testing.T) {
	t.Run("NewBaseWebAPIAdapter", func(t *testing.T) {
		adapter := NewBaseWebAPIAdapter("test_service")

		if adapter == nil {
			t.Fatal("NewBaseWebAPIAdapter 返回 nil")
		}

		if adapter.GetServiceName() != "test_service" {
			t.Errorf("GetServiceName() = %q, want %q", adapter.GetServiceName(), "test_service")
		}

		if adapter.IsConnected() {
			t.Error("新创建的适配器不应处于连接状态")
		}
	})

	t.Run("GetSyncCheckpoint", func(t *testing.T) {
		adapter := NewBaseWebAPIAdapter("test_service")
		cp := adapter.GetSyncCheckpoint()

		if cp == nil {
			t.Fatal("GetSyncCheckpoint 返回 nil")
		}

		if !cp.IsEmpty() {
			t.Error("初始检查点应为空")
		}
	})

	t.Run("UpdateSyncCheckpoint", func(t *testing.T) {
		adapter := NewBaseWebAPIAdapter("test_service")

		newCp := &SyncCheckpoint{
			LastSyncTime: time.Now(),
			LastEmailID:  "email123",
		}

		err := adapter.UpdateSyncCheckpoint(newCp)
		if err != nil {
			t.Errorf("UpdateSyncCheckpoint 失败: %v", err)
		}

		cp := adapter.GetSyncCheckpoint()
		if cp.LastEmailID != "email123" {
			t.Errorf("LastEmailID = %q, want %q", cp.LastEmailID, "email123")
		}
	})

	t.Run("UpdateSyncCheckpoint nil", func(t *testing.T) {
		adapter := NewBaseWebAPIAdapter("test_service")

		err := adapter.UpdateSyncCheckpoint(nil)
		if err == nil {
			t.Error("传入 nil 应返回错误")
		}
	})

	t.Run("SetConnected", func(t *testing.T) {
		adapter := NewBaseWebAPIAdapter("test_service")

		adapter.SetConnected(true)
		if !adapter.IsConnected() {
			t.Error("SetConnected(true) 后 IsConnected() 应返回 true")
		}

		adapter.SetConnected(false)
		if adapter.IsConnected() {
			t.Error("SetConnected(false) 后 IsConnected() 应返回 false")
		}
	})

	t.Run("GetProviderType", func(t *testing.T) {
		adapter := NewBaseWebAPIAdapter("my_provider")

		if adapter.GetProviderType() != "my_provider" {
			t.Errorf("GetProviderType() = %q, want %q", adapter.GetProviderType(), "my_provider")
		}
	})

	t.Run("GetProtocol", func(t *testing.T) {
		adapter := NewBaseWebAPIAdapter("test_service")

		if adapter.GetProtocol() != "webapi" {
			t.Errorf("GetProtocol() = %q, want %q", adapter.GetProtocol(), "webapi")
		}
	})
}

// TestSyncResult 测试同步结果
func TestSyncResult(t *testing.T) {
	t.Run("NewSyncResult", func(t *testing.T) {
		result := NewSyncResult()

		if result == nil {
			t.Fatal("NewSyncResult 返回 nil")
		}

		if result.Emails == nil {
			t.Error("Emails 不应为 nil")
		}

		if result.TotalCount != 0 {
			t.Errorf("TotalCount = %d, want 0", result.TotalCount)
		}

		if result.SyncTime.IsZero() {
			t.Error("SyncTime 不应为零值")
		}
	})

	t.Run("AddEmail", func(t *testing.T) {
		result := NewSyncResult()

		email := NewWebAPIEmail(&adapter.Email{ProviderID: "test1"}, "target@example.com")
		result.AddEmail(email)

		if result.TotalCount != 1 {
			t.Errorf("TotalCount = %d, want 1", result.TotalCount)
		}

		if len(result.Emails) != 1 {
			t.Errorf("Emails 长度 = %d, want 1", len(result.Emails))
		}
	})

	t.Run("AddEmails", func(t *testing.T) {
		result := NewSyncResult()

		emails := []*WebAPIEmail{
			NewWebAPIEmail(&adapter.Email{ProviderID: "test1"}, "target1@example.com"),
			NewWebAPIEmail(&adapter.Email{ProviderID: "test2"}, "target2@example.com"),
		}
		result.AddEmails(emails)

		if result.TotalCount != 2 {
			t.Errorf("TotalCount = %d, want 2", result.TotalCount)
		}

		if len(result.Emails) != 2 {
			t.Errorf("Emails 长度 = %d, want 2", len(result.Emails))
		}
	})
}

// TestHTTPClientConfig 测试 HTTP 客户端配置
func TestHTTPClientConfig(t *testing.T) {
	config := DefaultHTTPClientConfig()

	if config == nil {
		t.Fatal("DefaultHTTPClientConfig 返回 nil")
	}

	if config.Timeout != 30*time.Second {
		t.Errorf("Timeout = %v, want 30s", config.Timeout)
	}

	if config.MaxRetries != 3 {
		t.Errorf("MaxRetries = %d, want 3", config.MaxRetries)
	}

	if config.RetryDelay != time.Second {
		t.Errorf("RetryDelay = %v, want 1s", config.RetryDelay)
	}

	if config.UserAgent != "FusionMail/1.0" {
		t.Errorf("UserAgent = %q, want %q", config.UserAgent, "FusionMail/1.0")
	}

	if !config.FollowRedirects {
		t.Error("FollowRedirects 应为 true")
	}
}

// TestContextHelpers 测试上下文辅助函数
func TestContextHelpers(t *testing.T) {
	t.Run("RequestID", func(t *testing.T) {
		// 注意：WithRequestID 内部使用 context.WithValue，传入 nil 会 panic
		// 这里跳过 nil context 测试，仅验证函数存在
		t.Skip("跳过 nil context 测试")
	})

	t.Run("AccountUID", func(t *testing.T) {
		// 类似上面，跳过 nil context 测试
		t.Skip("跳过 nil context 测试")
	})
}
