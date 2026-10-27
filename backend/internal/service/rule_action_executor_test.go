package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"fusionmail/internal/model"
)

// newTestRuleService 构造一个只带 mock 仓库与真实 http.Client 的 ruleService。
func newTestRuleService(repo *mockEmailRepo) *ruleService {
	return &ruleService{
		emailRepo:  repo,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

// TestParseEmailLabels 覆盖 Labels 列的解析边界。
func TestParseEmailLabels(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want []string
	}{
		{name: "空字符串", raw: "", want: nil},
		{name: "仅空白", raw: "   ", want: nil},
		{name: "合法 JSON 数组", raw: `["工作","重要"]`, want: []string{"工作", "重要"}},
		{name: "空数组", raw: `[]`, want: []string{}},
		{name: "非法 JSON 视为无标签", raw: `{not json`, want: nil},
		{name: "逗号拼接的脏数据视为无标签", raw: `工作,重要`, want: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseEmailLabels(tt.raw)
			if len(got) != len(tt.want) {
				t.Fatalf("parseEmailLabels(%q) = %v, want %v", tt.raw, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("parseEmailLabels(%q)[%d] = %q, want %q", tt.raw, i, got[i], tt.want[i])
				}
			}
		})
	}
}

// TestAddLabel 验证 add_label 动作真正写库（此前是静默 no-op）。
func TestAddLabel(t *testing.T) {
	t.Run("追加到已有标签", func(t *testing.T) {
		repo := newMockEmailRepo()
		svc := newTestRuleService(repo)
		email := &model.Email{ID: 1, Labels: `["工作"]`}

		if err := svc.addLabel(context.Background(), email, "重要"); err != nil {
			t.Fatalf("addLabel() error = %v", err)
		}

		got := parseEmailLabels(email.Labels)
		if len(got) != 2 || got[0] != "工作" || got[1] != "重要" {
			t.Errorf("Labels = %v, want [工作 重要]", got)
		}
	})

	t.Run("从空标签开始", func(t *testing.T) {
		repo := newMockEmailRepo()
		svc := newTestRuleService(repo)
		email := &model.Email{ID: 1, Labels: ""}

		if err := svc.addLabel(context.Background(), email, "新标签"); err != nil {
			t.Fatalf("addLabel() error = %v", err)
		}
		if got := parseEmailLabels(email.Labels); len(got) != 1 || got[0] != "新标签" {
			t.Errorf("Labels = %v, want [新标签]", got)
		}
	})

	t.Run("幂等：重复添加不产生重复项", func(t *testing.T) {
		repo := newMockEmailRepo()
		svc := newTestRuleService(repo)
		email := &model.Email{ID: 1, Labels: `["工作"]`}

		if err := svc.addLabel(context.Background(), email, "工作"); err != nil {
			t.Fatalf("addLabel() error = %v", err)
		}
		if got := parseEmailLabels(email.Labels); len(got) != 1 {
			t.Errorf("Labels = %v, want 单元素 [工作]", got)
		}
	})

	t.Run("空标签报错", func(t *testing.T) {
		repo := newMockEmailRepo()
		svc := newTestRuleService(repo)
		if err := svc.addLabel(context.Background(), &model.Email{ID: 1}, "  "); err == nil {
			t.Error("addLabel() 空标签应当报错，实际返回 nil")
		}
	})
}

// TestRemoveLabel 验证 remove_label 动作真正写库。
func TestRemoveLabel(t *testing.T) {
	t.Run("移除存在的标签", func(t *testing.T) {
		repo := newMockEmailRepo()
		svc := newTestRuleService(repo)
		email := &model.Email{ID: 1, Labels: `["工作","重要","待办"]`}

		if err := svc.removeLabel(context.Background(), email, "重要"); err != nil {
			t.Fatalf("removeLabel() error = %v", err)
		}

		got := parseEmailLabels(email.Labels)
		if len(got) != 2 || got[0] != "工作" || got[1] != "待办" {
			t.Errorf("Labels = %v, want [工作 待办]", got)
		}
	})

	t.Run("移除最后一个标签后编码为空串", func(t *testing.T) {
		repo := newMockEmailRepo()
		svc := newTestRuleService(repo)
		email := &model.Email{ID: 1, Labels: `["唯一"]`}

		if err := svc.removeLabel(context.Background(), email, "唯一"); err != nil {
			t.Fatalf("removeLabel() error = %v", err)
		}
		if email.Labels != "" {
			t.Errorf("Labels = %q, want 空串", email.Labels)
		}
	})

	t.Run("幂等：移除不存在的标签不报错", func(t *testing.T) {
		repo := newMockEmailRepo()
		svc := newTestRuleService(repo)
		email := &model.Email{ID: 1, Labels: `["工作"]`}

		if err := svc.removeLabel(context.Background(), email, "不存在"); err != nil {
			t.Fatalf("removeLabel() error = %v", err)
		}
		if got := parseEmailLabels(email.Labels); len(got) != 1 || got[0] != "工作" {
			t.Errorf("Labels = %v, want 保持不变 [工作]", got)
		}
	})
}

// TestTriggerWebhook 验证 trigger_webhook 动作真正发出请求（此前是静默 no-op）。
func TestTriggerWebhook(t *testing.T) {
	t.Run("成功 POST 并携带邮件摘要", func(t *testing.T) {
		var (
			gotBody   []byte
			gotMethod string
			gotCT     string
			serverHit bool
		)
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			serverHit = true
			gotMethod = r.Method
			gotCT = r.Header.Get("Content-Type")
			gotBody, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()

		repo := newMockEmailRepo()
		svc := newTestRuleService(repo)
		email := &model.Email{
			ID:          42,
			AccountUID:  "acc-1",
			MessageID:   "<msg@example.com>",
			Subject:     "测试主题",
			FromAddress: "a@example.com",
			Labels:      `["工作"]`,
		}

		if err := svc.triggerWebhook(context.Background(), email, srv.URL); err != nil {
			t.Fatalf("triggerWebhook() error = %v", err)
		}
		if !serverHit {
			t.Fatal("webhook 服务端未被调用")
		}
		if gotMethod != http.MethodPost {
			t.Errorf("method = %s, want POST", gotMethod)
		}
		if gotCT != "application/json" {
			t.Errorf("Content-Type = %s, want application/json", gotCT)
		}

		var payload map[string]any
		if err := json.Unmarshal(gotBody, &payload); err != nil {
			t.Fatalf("payload 不是合法 JSON: %v (body=%s)", err, gotBody)
		}
		if payload["event"] != "rule.matched" {
			t.Errorf("event = %v, want rule.matched", payload["event"])
		}
		if payload["subject"] != "测试主题" {
			t.Errorf("subject = %v, want 测试主题", payload["subject"])
		}
		if payload["email_id"] != float64(42) {
			t.Errorf("email_id = %v, want 42", payload["email_id"])
		}
	})

	t.Run("非 2xx 返回错误", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer srv.Close()

		svc := newTestRuleService(newMockEmailRepo())
		err := svc.triggerWebhook(context.Background(), &model.Email{ID: 1}, srv.URL)
		if err == nil {
			t.Fatal("非 2xx 应当返回错误，实际 nil")
		}
		if !strings.Contains(err.Error(), "500") {
			t.Errorf("错误信息应包含状态码 500，实际: %v", err)
		}
	})

	t.Run("拒绝非 http/https 协议", func(t *testing.T) {
		svc := newTestRuleService(newMockEmailRepo())
		for _, u := range []string{"file:///etc/passwd", "gopher://x", "ftp://x/y"} {
			if err := svc.triggerWebhook(context.Background(), &model.Email{ID: 1}, u); err == nil {
				t.Errorf("协议 %s 应当被拒绝，实际通过", u)
			}
		}
	})

	t.Run("空 URL 与缺失 host 报错", func(t *testing.T) {
		svc := newTestRuleService(newMockEmailRepo())
		for _, u := range []string{"", "   ", "http://"} {
			if err := svc.triggerWebhook(context.Background(), &model.Email{ID: 1}, u); err == nil {
				t.Errorf("URL %q 应当报错，实际通过", u)
			}
		}
	})
}

// TestValidateActionAcceptsTriggerWebhook 锁定前后端动作名契约。
//
// 前端 RuleForm.tsx 与 types/index.ts 提交的是 trigger_webhook，
// 而历史后端只认 webhook —— 用户在前端创建「触发 Webhook」规则必然失败。
func TestValidateActionAcceptsTriggerWebhook(t *testing.T) {
	svc := &ruleService{}

	for _, typ := range []string{"webhook", "trigger_webhook"} {
		t.Run(typ, func(t *testing.T) {
			action := &model.RuleAction{Type: typ, Value: "https://example.com/hook"}
			if err := svc.validateAction(action); err != nil {
				t.Errorf("validateAction(%s) 应当通过，实际: %v", typ, err)
			}
		})
	}

	t.Run("trigger_webhook 缺少 value 时报错", func(t *testing.T) {
		action := &model.RuleAction{Type: "trigger_webhook", Value: ""}
		if err := svc.validateAction(action); err == nil {
			t.Error("trigger_webhook 缺少 value 应当报错，实际通过")
		}
	})

	t.Run("未知动作仍然被拒绝", func(t *testing.T) {
		action := &model.RuleAction{Type: "not_a_real_action", Value: "x"}
		if err := svc.validateAction(action); err == nil {
			t.Error("未知动作应当被拒绝，实际通过")
		}
	})
}

// TestExecuteActionDispatchesLabelAndWebhook 验证 executeAction 的分派不再落到 no-op。
func TestExecuteActionDispatchesLabelAndWebhook(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	repo := newMockEmailRepo()
	svc := newTestRuleService(repo)
	email := &model.Email{ID: 1, Labels: `["工作"]`}

	if err := svc.executeAction(context.Background(), &model.RuleAction{Type: "add_label", Value: "重要"}, email); err != nil {
		t.Fatalf("executeAction(add_label) error = %v", err)
	}
	if got := parseEmailLabels(email.Labels); len(got) != 2 {
		t.Errorf("add_label 未生效，Labels = %v", got)
	}

	if err := svc.executeAction(context.Background(), &model.RuleAction{Type: "trigger_webhook", Value: srv.URL}, email); err != nil {
		t.Fatalf("executeAction(trigger_webhook) error = %v", err)
	}

	if err := svc.executeAction(context.Background(), &model.RuleAction{Type: "remove_label", Value: "工作"}, email); err != nil {
		t.Fatalf("executeAction(remove_label) error = %v", err)
	}
	if got := parseEmailLabels(email.Labels); len(got) != 1 || got[0] != "重要" {
		t.Errorf("remove_label 未生效，Labels = %v", got)
	}
}
