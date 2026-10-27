package response

import (
	"time"

	"fusionmail/internal/model"
)

// WebAPI Provider 模块响应 DTO（P16 swagger 注解补全）。
//
// model.EmailAccount 无 json tag，序列化键为 Go 字段名 —— AccountRecord
// 逐字段镜像该序列化形状（含嵌套 Provider/Adapter 的 Go 字段名键，
// 分别镜像为 ProviderRecord / AdapterRecord）。仅用于 Swagger 契约声明。

// AccountRecord model.EmailAccount 序列化镜像（Go 字段名键）。
// ProviderRef/AdapterRef 在查询时 Preload，运行时可能为对象或 null。
type AccountRecord struct {
	ID    int64  `json:"ID"`
	UID   string `json:"UID"`
	Email string `json:"Email"`

	ProviderID  int64           `json:"ProviderID"`
	ProviderRef *ProviderRecord `json:"ProviderRef"`
	AdapterID   int64           `json:"AdapterID"`
	AdapterRef  *AdapterRecord  `json:"AdapterRef"`

	EncryptedCredentials string `json:"EncryptedCredentials"`
	SMTPEnabled          bool   `json:"SMTPEnabled"`

	ProxyEnabled           bool   `json:"ProxyEnabled"`
	ProxyType              string `json:"ProxyType"`
	ProxyHost              string `json:"ProxyHost"`
	ProxyPort              int    `json:"ProxyPort"`
	ProxyUsername          string `json:"ProxyUsername"`
	EncryptedProxyPassword string `json:"EncryptedProxyPassword"`

	Status                  string     `json:"Status"`
	ConsecutiveAuthFailures int        `json:"ConsecutiveAuthFailures"`
	AutoDisabledAt          *time.Time `json:"AutoDisabledAt"`
	DisableReason           string     `json:"DisableReason"`

	SyncEnabled    bool       `json:"SyncEnabled"`
	SyncInterval   int        `json:"SyncInterval"`
	SyncModeField  string     `json:"SyncModeField"`
	LastSyncAt     *time.Time `json:"LastSyncAt"`
	LastSyncStatus string     `json:"LastSyncStatus"`
	LastSyncError  string     `json:"LastSyncError"`

	UIDValidity int64 `json:"UIDValidity"`
	LastUID     int64 `json:"LastUID"`

	FirstSyncDays    int    `json:"FirstSyncDays"`
	BatchSize        int    `json:"BatchSize"`
	MaxEmailsPerSync int    `json:"MaxEmailsPerSync"`
	SyncCursor       string `json:"SyncCursor"`
	SyncProgressJSON string `json:"SyncProgressJSON"`

	ServerDeletePolicy string  `json:"ServerDeletePolicy"`
	GroupID            *int64  `json:"GroupID"`
	ParentAccountUID   *string `json:"ParentAccountUID"`

	TotalEmails int `json:"TotalEmails"`
	UnreadCount int `json:"UnreadCount"`

	CreatedAt time.Time `json:"CreatedAt"`
	UpdatedAt time.Time `json:"UpdatedAt"`
}

// TestConnectionResponse 连接测试结果（镜像 service.TestConnectionResult）。
type TestConnectionResponse struct {
	Success     bool   `json:"success"`
	Message     string `json:"message"`
	ServiceName string `json:"service_name"`
	EmailCount  int    `json:"email_count,omitempty"`
	Error       string `json:"error,omitempty"`
}

// SyncStatusResponse 同步状态（镜像 service.SyncStatus）。
type SyncStatusResponse struct {
	Status       string     `json:"status"`
	LastSyncAt   *time.Time `json:"last_sync_at,omitempty"`
	LastSyncedID string     `json:"last_synced_id,omitempty"`
	EmailCount   int64      `json:"email_count"`
	ErrorMessage string     `json:"error_message,omitempty"`
}

// AccountConfigResponse 账户 WebAPI 配置（镜像 service.AccountConfigResponse）。
type AccountConfigResponse struct {
	ServiceType string         `json:"service_type"`
	AuthData    map[string]any `json:"auth_data"`
}

// CloudflareSettingsResponse Cloudflare Temp Email 设置（镜像 service.CloudflareTempEmailSettings）。
type CloudflareSettingsResponse struct {
	Email   string   `json:"email"`
	Domains []string `json:"domains,omitempty"`
}

// ChildReconcileResultResponse 子账户对账结果（镜像 service.ChildReconcileResult）。
type ChildReconcileResultResponse struct {
	RemoteCount       int      `json:"remote_count"`
	LocalCount        int      `json:"local_count"`
	MarkedOrphaned    int      `json:"marked_orphaned"`
	Reactivated       int      `json:"reactivated"`
	Unchanged         int      `json:"unchanged"`
	OrphanedEmails    []string `json:"orphaned_emails"`
	ReactivatedEmails []string `json:"reactivated_emails"`
	SkippedRemote     bool     `json:"skipped_remote"`
	Message           string   `json:"message,omitempty"`
}

// SubAccountInfoResponse 服务端子邮箱账户（镜像 service.SubAccountInfo）。
type SubAccountInfoResponse struct {
	AccountID int    `json:"account_id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
}

// OAuth2TokenRefreshResponse 令牌刷新结果（镜像 service.OAuth2TokenRefreshResponse）。
type OAuth2TokenRefreshResponse struct {
	AccessToken string    `json:"access_token"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// DefaultSMTPConfigResponse 默认 SMTP 配置（镜像 service.DefaultSMTPConfig）。
type DefaultSMTPConfigResponse struct {
	Host       string `json:"host"`
	Port       int    `json:"port"`
	Encryption string `json:"encryption"`
}

// AttachmentUploadResponse 附件上传结果。
type AttachmentUploadResponse struct {
	Filename    string `json:"filename"`
	Size        int64  `json:"size"`
	ContentType string `json:"content_type"`
	TempPath    string `json:"temp_path"`
}

// ValidateConfigResponse 配置校验响应（非标准信封：success/valid/message 平铺）。
type ValidateConfigResponse struct {
	Success bool   `json:"success"`
	Valid   bool   `json:"valid"`
	Message string `json:"message,omitempty"`
}

// WebAPIProviderAccountPage WebAPI Provider 列表分页信封
// （data 内含 items/total/page/page_size，gin.H 平铺形状）。
type WebAPIProviderAccountPage struct {
	Items    []AccountRecord `json:"items"`
	Total    int64           `json:"total"`
	Page     int             `json:"page"`
	PageSize int             `json:"page_size"`
}

// NewAccountRecords 把 []*model.EmailAccount 逐字段映射为序列化等价的
// AccountRecord 切片。仅覆盖非关联字段；ProviderRef/AdapterRef 的序列化
// 形状依赖 Preload（本查询已 Preload，运行时非 nil），保持指针传递。
func NewAccountRecords(accounts []*model.EmailAccount) []AccountRecord {
	out := make([]AccountRecord, 0, len(accounts))
	for _, a := range accounts {
		if a == nil {
			continue
		}
		out = append(out, AccountRecord{
			ID:    a.ID,
			UID:   a.UID,
			Email: a.Email,

			ProviderID: a.ProviderID,
			AdapterID:  a.AdapterID,

			EncryptedCredentials: a.EncryptedCredentials,
			SMTPEnabled:          a.SMTPEnabled,

			ProxyEnabled:           a.ProxyEnabled,
			ProxyType:              a.ProxyType,
			ProxyHost:              a.ProxyHost,
			ProxyPort:              a.ProxyPort,
			ProxyUsername:          a.ProxyUsername,
			EncryptedProxyPassword: a.EncryptedProxyPassword,

			Status:                  a.Status,
			ConsecutiveAuthFailures: a.ConsecutiveAuthFailures,
			AutoDisabledAt:          a.AutoDisabledAt,
			DisableReason:           a.DisableReason,

			SyncEnabled:    a.SyncEnabled,
			SyncInterval:   a.SyncInterval,
			SyncModeField:  a.SyncModeField,
			LastSyncAt:     a.LastSyncAt,
			LastSyncStatus: a.LastSyncStatus,
			LastSyncError:  a.LastSyncError,

			UIDValidity: a.UIDValidity,
			LastUID:     a.LastUID,

			FirstSyncDays:    a.FirstSyncDays,
			BatchSize:        a.BatchSize,
			MaxEmailsPerSync: a.MaxEmailsPerSync,
			SyncCursor:       a.SyncCursor,
			SyncProgressJSON: a.SyncProgressJSON,

			ServerDeletePolicy: a.ServerDeletePolicy,
			GroupID:            a.GroupID,
			ParentAccountUID:   a.ParentAccountUID,

			TotalEmails: a.TotalEmails,
			UnreadCount: a.UnreadCount,

			CreatedAt: a.CreatedAt,
			UpdatedAt: a.UpdatedAt,
		})
	}
	return out
}

// DefaultSMTPConfigMap 键为服务商代号的默认 SMTP 配置表。
type DefaultSMTPConfigMap map[string]DefaultSMTPConfigResponse

// NewAccountRecord 单个账户的序列化等价映射（复用批量映射）。
func NewAccountRecord(a *model.EmailAccount) *AccountRecord {
	if a == nil {
		return nil
	}
	records := NewAccountRecords([]*model.EmailAccount{a})
	return &records[0]
}
