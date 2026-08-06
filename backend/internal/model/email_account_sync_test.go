package model

import "testing"

func TestShouldSkipPollingSync(t *testing.T) {
	parent := "parent-uid"

	tests := []struct {
		name    string
		account *EmailAccount
		want    bool
	}{
		{
			name:    "普通轮询账户",
			account: &EmailAccount{UID: "normal-1", SyncModeField: SyncModePolling},
			want:    false,
		},
		{
			name:    "webhook 模式主账户",
			account: &EmailAccount{UID: "master-1", SyncModeField: SyncModeWebhook},
			want:    true,
		},
		{
			name:    "webhook_ 子账户",
			account: &EmailAccount{UID: "webhook_123", SyncModeField: SyncModePolling},
			want:    true,
		},
		{
			name:    "有父账户的子邮箱",
			account: &EmailAccount{UID: "child-1", ParentAccountUID: &parent},
			want:    true,
		},
		{
			name:    "空账户",
			account: nil,
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.account.ShouldSkipPollingSync(); got != tt.want {
				t.Fatalf("ShouldSkipPollingSync() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsWebhookOnlyAuthDataHelpers(t *testing.T) {
	// 行为由 service 包 isWebhookOnlyAuthData 覆盖；此处仅保证子账户识别
	if !(&EmailAccount{UID: "webhook_1"}).IsWebhookChildAccount() {
		t.Fatal("expected webhook child")
	}
	if (&EmailAccount{UID: "abc"}).IsWebhookChildAccount() {
		t.Fatal("expected non-webhook")
	}
}
