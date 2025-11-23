package model

import (
	"testing"
)

func TestProvider_GetSupportedProtocols(t *testing.T) {
	tests := []struct {
		name     string
		provider *Provider
		want     []string
		wantErr  bool
	}{
		{
			name:     "有效的 JSON 数组",
			provider: &Provider{SupportedProtocols: `["oauth2","imap","pop3"]`},
			want:     []string{"oauth2", "imap", "pop3"},
			wantErr:  false,
		},
		{
			name:     "空数组",
			provider: &Provider{SupportedProtocols: `[]`},
			want:     []string{},
			wantErr:  false,
		},
		{
			name:     "空字符串",
			provider: &Provider{SupportedProtocols: ""},
			want:     []string{},
			wantErr:  false,
		},
		{
			name:     "无效的 JSON",
			provider: &Provider{SupportedProtocols: `["oauth2","imap"`},
			want:     nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.provider.GetSupportedProtocols()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetSupportedProtocols() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if len(got) != len(tt.want) {
					t.Errorf("GetSupportedProtocols() got = %v, want %v", got, tt.want)
					return
				}
				for i, v := range got {
					if v != tt.want[i] {
						t.Errorf("GetSupportedProtocols() got = %v, want %v", got, tt.want)
						break
					}
				}
			}
		})
	}
}

func TestProvider_SetSupportedProtocols(t *testing.T) {
	tests := []struct {
		name      string
		provider  *Provider
		protocols []string
		want      string
		wantErr   bool
	}{
		{
			name:      "设置有效的协议列表",
			provider:  &Provider{},
			protocols: []string{"oauth2", "imap"},
			want:      `["oauth2","imap"]`,
			wantErr:   false,
		},
		{
			name:      "设置空协议列表",
			provider:  &Provider{},
			protocols: []string{},
			want:      `[]`,
			wantErr:   false,
		},
		{
			name:      "设置 nil",
			provider:  &Provider{},
			protocols: nil,
			want:      `[]`,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.provider.SetSupportedProtocols(tt.protocols)
			if (err != nil) != tt.wantErr {
				t.Errorf("SetSupportedProtocols() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.provider.SupportedProtocols != tt.want {
				t.Errorf("SetSupportedProtocols() got = %v, want %v", tt.provider.SupportedProtocols, tt.want)
			}
		})
	}
}

func TestProvider_GetMetadata(t *testing.T) {
	tests := []struct {
		name     string
		provider *Provider
		want     map[string]interface{}
		wantErr  bool
	}{
		{
			name:     "有效的 JSON 对象",
			provider: &Provider{Metadata: `{"key1":"value1","key2":2}`},
			want:     map[string]interface{}{"key1": "value1", "key2": float64(2)},
			wantErr:  false,
		},
		{
			name:     "空对象",
			provider: &Provider{Metadata: `{}`},
			want:     map[string]interface{}{},
			wantErr:  false,
		},
		{
			name:     "空字符串",
			provider: &Provider{Metadata: ""},
			want:     map[string]interface{}{},
			wantErr:  false,
		},
		{
			name:     "无效的 JSON",
			provider: &Provider{Metadata: `{"key1":"value1"`},
			want:     nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.provider.GetMetadata()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetMetadata() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if len(got) != len(tt.want) {
					t.Errorf("GetMetadata() got = %v, want %v", got, tt.want)
					return
				}
			}
		})
	}
}

func TestProvider_SetMetadata(t *testing.T) {
	tests := []struct {
		name     string
		provider *Provider
		metadata map[string]interface{}
		want     string
		wantErr  bool
	}{
		{
			name:     "设置有效的元数据",
			provider: &Provider{},
			metadata: map[string]interface{}{"key1": "value1", "key2": 2},
			want:     `{"key1":"value1","key2":2}`,
			wantErr:  false,
		},
		{
			name:     "设置空元数据",
			provider: &Provider{},
			metadata: map[string]interface{}{},
			want:     `{}`,
			wantErr:  false,
		},
		{
			name:     "设置 nil",
			provider: &Provider{},
			metadata: nil,
			want:     `{}`,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.provider.SetMetadata(tt.metadata)
			if (err != nil) != tt.wantErr {
				t.Errorf("SetMetadata() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.provider.Metadata != tt.want {
				t.Errorf("SetMetadata() got = %v, want %v", tt.provider.Metadata, tt.want)
			}
		})
	}
}

func TestProvider_IsOAuth2Supported(t *testing.T) {
	tests := []struct {
		name     string
		provider *Provider
		want     bool
		wantErr  bool
	}{
		{
			name:     "支持 OAuth2",
			provider: &Provider{SupportedProtocols: `["oauth2","imap"]`},
			want:     true,
			wantErr:  false,
		},
		{
			name:     "不支持 OAuth2",
			provider: &Provider{SupportedProtocols: `["imap","pop3"]`},
			want:     false,
			wantErr:  false,
		},
		{
			name:     "不支持任何协议",
			provider: &Provider{SupportedProtocols: `[]`},
			want:     false,
			wantErr:  false,
		},
		{
			name:     "无效的 JSON",
			provider: &Provider{SupportedProtocols: `["oauth2","imap"`},
			want:     false,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.provider.IsOAuth2Supported()
			if (err != nil) != tt.wantErr {
				t.Errorf("IsOAuth2Supported() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("IsOAuth2Supported() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProvider_IsIMAPSupported(t *testing.T) {
	tests := []struct {
		name     string
		provider *Provider
		want     bool
		wantErr  bool
	}{
		{
			name:     "支持 IMAP",
			provider: &Provider{SupportedProtocols: `["oauth2","imap"]`},
			want:     true,
			wantErr:  false,
		},
		{
			name:     "不支持 IMAP",
			provider: &Provider{SupportedProtocols: `["oauth2","pop3"]`},
			want:     false,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.provider.IsIMAPSupported()
			if (err != nil) != tt.wantErr {
				t.Errorf("IsIMAPSupported() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("IsIMAPSupported() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProvider_IsPOP3Supported(t *testing.T) {
	tests := []struct {
		name     string
		provider *Provider
		want     bool
		wantErr  bool
	}{
		{
			name:     "支持 POP3",
			provider: &Provider{SupportedProtocols: `["oauth2","imap","pop3"]`},
			want:     true,
			wantErr:  false,
		},
		{
			name:     "不支持 POP3",
			provider: &Provider{SupportedProtocols: `["oauth2","imap"]`},
			want:     false,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.provider.IsPOP3Supported()
			if (err != nil) != tt.wantErr {
				t.Errorf("IsPOP3Supported() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("IsPOP3Supported() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProvider_Validate(t *testing.T) {
	tests := []struct {
		name     string
		provider *Provider
		wantErr  bool
	}{
		{
			name:     "有效的配置",
			provider: &Provider{Name: "gmail", DisplayName: "Gmail", SupportedProtocols: `["oauth2","imap"]`, RecommendedProtocol: "oauth2"},
			wantErr:  false,
		},
		{
			name:     "缺少名称",
			provider: &Provider{DisplayName: "Gmail", SupportedProtocols: `["oauth2","imap"]`, RecommendedProtocol: "oauth2"},
			wantErr:  true,
		},
		{
			name:     "缺少显示名称",
			provider: &Provider{Name: "gmail", SupportedProtocols: `["oauth2","imap"]`, RecommendedProtocol: "oauth2"},
			wantErr:  true,
		},
		{
			name:     "缺少推荐协议",
			provider: &Provider{Name: "gmail", DisplayName: "Gmail", SupportedProtocols: `["oauth2","imap"]`},
			wantErr:  true,
		},
		{
			name:     "支持的协议列表为空",
			provider: &Provider{Name: "gmail", DisplayName: "Gmail", SupportedProtocols: `[]`, RecommendedProtocol: "oauth2"},
			wantErr:  true,
		},
		{
			name:     "推荐协议不在支持的协议中",
			provider: &Provider{Name: "gmail", DisplayName: "Gmail", SupportedProtocols: `["imap","pop3"]`, RecommendedProtocol: "oauth2"},
			wantErr:  true,
		},
		{
			name:     "设置了 requires_oauth 但不支持 oauth2",
			provider: &Provider{Name: "qq", DisplayName: "QQ Mail", SupportedProtocols: `["imap","pop3"]`, RecommendedProtocol: "imap", RequiresOAuth: true},
			wantErr:  true,
		},
		{
			name:     "端口超出范围",
			provider: &Provider{Name: "gmail", DisplayName: "Gmail", SupportedProtocols: `["oauth2","imap"]`, RecommendedProtocol: "oauth2", IMAPPort: 99999},
			wantErr:  true,
		},
		{
			name:     "设置 requires_oauth=true 且支持 oauth2",
			provider: &Provider{Name: "gmail", DisplayName: "Gmail", SupportedProtocols: `["oauth2","imap"]`, RecommendedProtocol: "oauth2", RequiresOAuth: true},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.provider.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestProvider_TableName(t *testing.T) {
	provider := &Provider{}
	if provider.TableName() != "providers" {
		t.Errorf("TableName() = %v, want %v", provider.TableName(), "providers")
	}
}

func TestValidationError_Error(t *testing.T) {
	err := &ValidationError{
		Field:   "name",
		Message: "name cannot be empty",
	}
	expected := "validation error on field 'name': name cannot be empty"
	if err.Error() != expected {
		t.Errorf("Error() = %v, want %v", err.Error(), expected)
	}
}
