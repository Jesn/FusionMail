package response

import (
	"fusionmail/internal/model"
	"time"
)

// EmailListEntry 是白名单/黑名单条目的响应 DTO。
//
// 与 model.EmailList 逐字段 JSON 等价（model 未声明 json tag，
// 序列化键为 Go 字段名，这里用显式 tag 保持一致，含 CreatedAt/UpdatedAt）。
// 仅用于 Swagger 契约声明，避免带 gorm tag 的持久化模型进入 API 文档。
type EmailListEntry struct {
	ID         int64     `json:"ID"`
	UserUID    string    `json:"UserUID"`
	Type       string    `json:"Type"`
	Target     string    `json:"Target"`
	TargetType string    `json:"TargetType"`
	Reason     string    `json:"Reason"`
	CreatedAt  time.Time `json:"CreatedAt"`
	UpdatedAt  time.Time `json:"UpdatedAt"`
}

// NewEmailListEntry 把存储模型映射为响应 DTO。
func NewEmailListEntry(e *model.EmailList) *EmailListEntry {
	if e == nil {
		return nil
	}
	return &EmailListEntry{
		ID:         e.ID,
		UserUID:    e.UserUID,
		Type:       e.Type,
		Target:     e.Target,
		TargetType: e.TargetType,
		Reason:     e.Reason,
		CreatedAt:  e.CreatedAt,
		UpdatedAt:  e.UpdatedAt,
	}
}

// NewEmailListEntries 批量映射（保持 nil/空切片语义不变）。
func NewEmailListEntries(lists []*model.EmailList) []*EmailListEntry {
	if lists == nil {
		return nil
	}
	out := make([]*EmailListEntry, 0, len(lists))
	for _, e := range lists {
		out = append(out, NewEmailListEntry(e))
	}
	return out
}
