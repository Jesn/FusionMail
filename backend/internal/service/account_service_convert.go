package service

import (
	"time"

	"fusionmail/internal/dto/response"
	"fusionmail/internal/model"

	"gorm.io/gorm"
)

// toAccountResponse 将 model.EmailAccount 转换为 AccountResponse DTO
func toAccountResponse(a *model.EmailAccount) *response.AccountResponse {
	if a == nil {
		return nil
	}
	return &response.AccountResponse{
		ID:                 a.ID,
		UID:                a.UID,
		Email:              a.Email,
		ProviderID:         a.ProviderID,
		ProviderRef:        toProviderRef(a.ProviderRef),
		AdapterID:          a.AdapterID,
		AdapterRef:         toAdapterRef(a.AdapterRef),
		Provider:           a.GetProviderName(),
		Protocol:           a.GetProtocol(),
		AuthType:           a.GetAuthType(),
		SyncMode:           a.GetSyncMode(),
		SMTPEnabled:        a.SMTPEnabled,
		ProxyEnabled:       a.ProxyEnabled,
		ProxyType:          a.ProxyType,
		ProxyHost:          a.ProxyHost,
		ProxyPort:          a.ProxyPort,
		ProxyUsername:      a.ProxyUsername,
		Status:             a.Status,
		AutoDisabledAt:     a.AutoDisabledAt,
		DisableReason:      a.DisableReason,
		SyncEnabled:        a.SyncEnabled,
		SyncInterval:       a.SyncInterval,
		LastSyncAt:         a.LastSyncAt,
		LastSyncStatus:     a.LastSyncStatus,
		LastSyncError:      a.LastSyncError,
		FirstSyncDays:      a.FirstSyncDays,
		BatchSize:          a.BatchSize,
		MaxEmailsPerSync:   a.MaxEmailsPerSync,
		ServerDeletePolicy: a.ServerDeletePolicy,
		GroupID:            a.GroupID,
		ParentAccountUID:   a.ParentAccountUID,
		TotalEmails:        a.TotalEmails,
		UnreadCount:        a.UnreadCount,
		CreatedAt:          a.CreatedAt,
		UpdatedAt:          a.UpdatedAt,
		DeletedAt:          deletedAtPtr(a.DeletedAt),
	}
}

// deletedAtPtr 将 gorm.DeletedAt 转为 *time.Time，供前端识别回收站账号
func deletedAtPtr(d gorm.DeletedAt) *time.Time {
	if !d.Valid {
		return nil
	}
	t := d.Time
	return &t
}

func toProviderRef(p *model.Provider) *response.Provider {
	if p == nil {
		return nil
	}
	return &response.Provider{
		ID:          p.ID,
		Name:        p.Name,
		DisplayName: p.DisplayName,
	}
}

func toAdapterRef(a *model.Adapter) *response.Adapter {
	if a == nil {
		return nil
	}
	return &response.Adapter{
		ID:   a.ID,
		Name: a.Name,
	}
}

func toAccountResponseList(accounts []*model.EmailAccount) []*response.AccountResponse {
	result := make([]*response.AccountResponse, len(accounts))
	for i, a := range accounts {
		result[i] = toAccountResponse(a)
	}
	return result
}
