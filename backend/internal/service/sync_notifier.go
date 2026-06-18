package service

import (
	"encoding/json"

	"fusionmail/internal/sse"
)

const SSEEventEmailCountsMaybeChanged = "email_counts_maybe_changed"

// SyncNotifier 收敛 service 层对前端实时事件的依赖。
type SyncNotifier interface {
	Notify(eventType string, data any)
}

type noopSyncNotifier struct{}

type sseSyncNotifier struct{}

func NewNoopSyncNotifier() SyncNotifier {
	return noopSyncNotifier{}
}

func NewSSESyncNotifier() SyncNotifier {
	return sseSyncNotifier{}
}

func resolveSyncNotifier(notifier SyncNotifier) SyncNotifier {
	if notifier == nil {
		return noopSyncNotifier{}
	}
	return notifier
}

func NotifyEmailCountsMaybeChanged(notifier SyncNotifier, data any) {
	if data == nil {
		data = map[string]any{}
	}
	resolveSyncNotifier(notifier).Notify(SSEEventEmailCountsMaybeChanged, data)
}

func (noopSyncNotifier) Notify(string, any) {}

func (sseSyncNotifier) Notify(eventType string, data any) {
	switch v := data.(type) {
	case string:
		sse.Broadcast(eventType, v)
	case []byte:
		sse.Broadcast(eventType, string(v))
	default:
		jsonData, err := json.Marshal(data)
		if err != nil {
			return
		}
		sse.Broadcast(eventType, string(jsonData))
	}
}
