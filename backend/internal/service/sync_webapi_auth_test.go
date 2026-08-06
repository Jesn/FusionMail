package service

import "testing"

func TestIsWebhookOnlyAuthData(t *testing.T) {
	tests := []struct {
		name string
		json string
		want bool
	}{
		{
			name: "完整轮询配置",
			json: `{"base_url":"https://example.com","access_mode":"single","jwt_token":"x"}`,
			want: false,
		},
		{
			name: "明确 webhook 模式",
			json: `{"sync_mode":"webhook","webhook_secret":"sec"}`,
			want: true,
		},
		{
			name: "仅有 webhook_secret 无 base_url",
			json: `{"webhook_secret":"sec","email":"a@b.com"}`,
			want: true,
		},
		{
			name: "空 JSON",
			json: `{}`,
			want: false,
		},
		{
			name: "非法 JSON",
			json: `not-json`,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isWebhookOnlyAuthData(tt.json); got != tt.want {
				t.Fatalf("isWebhookOnlyAuthData() = %v, want %v", got, tt.want)
			}
		})
	}
}
