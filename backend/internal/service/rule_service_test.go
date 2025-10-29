package service

import (
	"testing"

	"fusionmail/internal/model"
)

// TestMatchCondition 测试条件匹配
func TestMatchCondition(t *testing.T) {
	s := &ruleService{}

	tests := []struct {
		name      string
		condition *model.RuleCondition
		email     *model.Email
		want      bool
	}{
		{
			name: "contains - 匹配",
			condition: &model.RuleCondition{
				Field:    "subject",
				Operator: "contains",
				Value:    "测试",
			},
			email: &model.Email{
				Subject: "这是一封测试邮件",
			},
			want: true,
		},
		{
			name: "contains - 不匹配",
			condition: &model.RuleCondition{
				Field:    "subject",
				Operator: "contains",
				Value:    "重要",
			},
			email: &model.Email{
				Subject: "这是一封测试邮件",
			},
			want: false,
		},
		{
			name: "equals - 匹配",
			condition: &model.RuleCondition{
				Field:    "from",
				Operator: "equals",
				Value:    "test@example.com",
			},
			email: &model.Email{
				FromAddress: "test@example.com",
			},
			want: true,
		},
		{
			name: "starts_with - 匹配",
			condition: &model.RuleCondition{
				Field:    "subject",
				Operator: "starts_with",
				Value:    "【通知】",
			},
			email: &model.Email{
				Subject: "【通知】系统维护",
			},
			want: true,
		},
		{
			name: "has_attachment - 有附件",
			condition: &model.RuleCondition{
				Field:    "has_attachment",
				Operator: "equals",
				Value:    "true",
			},
			email: &model.Email{
				HasAttachment: true,
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := s.matchCondition(tt.condition, tt.email)
			if err != nil {
				t.Errorf("matchCondition() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("matchCondition() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestMatchRule 测试规则匹配
func TestMatchRule(t *testing.T) {
	s := &ruleService{}

	tests := []struct {
		name  string
		rule  *model.EmailRule
		email *model.Email
		want  bool
	}{
		{
			name: "all 模式 - 所有条件匹配",
			rule: &model.EmailRule{
				MatchMode: "all",
				Conditions: []model.RuleCondition{
					{
						Field:    "from",
						Operator: "contains",
						Value:    "example.com",
					},
					{
						Field:    "subject",
						Operator: "contains",
						Value:    "测试",
					},
				},
			},
			email: &model.Email{
				FromAddress: "test@example.com",
				Subject:     "这是测试邮件",
			},
			want: true,
		},
		{
			name: "all 模式 - 部分条件不匹配",
			rule: &model.EmailRule{
				MatchMode: "all",
				Conditions: []model.RuleCondition{
					{
						Field:    "from",
						Operator: "contains",
						Value:    "example.com",
					},
					{
						Field:    "subject",
						Operator: "contains",
						Value:    "重要",
					},
				},
			},
			email: &model.Email{
				FromAddress: "test@example.com",
				Subject:     "这是测试邮件",
			},
			want: false,
		},
		{
			name: "any 模式 - 任意条件匹配",
			rule: &model.EmailRule{
				MatchMode: "any",
				Conditions: []model.RuleCondition{
					{
						Field:    "from",
						Operator: "contains",
						Value:    "other.com",
					},
					{
						Field:    "subject",
						Operator: "contains",
						Value:    "测试",
					},
				},
			},
			email: &model.Email{
				FromAddress: "test@example.com",
				Subject:     "这是测试邮件",
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := s.matchRule(tt.rule, tt.email)
			if err != nil {
				t.Errorf("matchRule() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("matchRule() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestValidateRule 测试规则验证
func TestValidateRule(t *testing.T) {
	s := &ruleService{}

	tests := []struct {
		name    string
		rule    *model.EmailRule
		wantErr bool
	}{
		{
			name: "有效规则",
			rule: &model.EmailRule{
				Name:       "测试规则",
				AccountUID: "test-account",
				Conditions: []model.RuleCondition{
					{
						Field:    "subject",
						Operator: "contains",
						Value:    "测试",
					},
				},
				Actions: []model.RuleAction{
					{
						Type: "mark_read",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "缺少名称",
			rule: &model.EmailRule{
				AccountUID: "test-account",
				Conditions: []model.RuleCondition{
					{
						Field:    "subject",
						Operator: "contains",
						Value:    "测试",
					},
				},
				Actions: []model.RuleAction{
					{
						Type: "mark_read",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "缺少条件",
			rule: &model.EmailRule{
				Name:       "测试规则",
				AccountUID: "test-account",
				Conditions: []model.RuleCondition{},
				Actions: []model.RuleAction{
					{
						Type: "mark_read",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "无效的操作符",
			rule: &model.EmailRule{
				Name:       "测试规则",
				AccountUID: "test-account",
				Conditions: []model.RuleCondition{
					{
						Field:    "subject",
						Operator: "invalid_operator",
						Value:    "测试",
					},
				},
				Actions: []model.RuleAction{
					{
						Type: "mark_read",
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := s.validateRule(tt.rule)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateRule() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestExecuteAction 测试动作执行
func TestExecuteAction(t *testing.T) {
	// 这个测试需要 mock repository，暂时跳过
	t.Skip("需要 mock repository 才能测试")
}
