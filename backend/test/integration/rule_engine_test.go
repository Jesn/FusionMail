package integration

import (
	"context"
	"testing"
	"time"

	"fusionmail/internal/model"
	"fusionmail/internal/repository"
	"fusionmail/internal/service"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDB 创建测试数据库
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// 自动迁移
	err = db.AutoMigrate(
		&model.EmailRule{},
		&model.Email{},
		&model.EmailAttachment{},
	)
	if err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	return db
}

// TestRuleEngineIntegration 规则引擎集成测试
func TestRuleEngineIntegration(t *testing.T) {
	// 设置测试数据库
	db := setupTestDB(t)

	// 创建仓库
	ruleRepo := repository.NewRuleRepository(db)
	emailRepo := repository.NewEmailRepository(db)

	// 创建服务
	ruleService := service.NewRuleService(ruleRepo, emailRepo)

	ctx := context.Background()

	t.Run("创建和应用规则", func(t *testing.T) {
		// 1. 创建测试账户
		accountUID := "test-account-001"

		// 2. 创建规则：自动归档通知邮件
		rule := &model.EmailRule{
			Name:        "自动归档通知邮件",
			AccountUID:  accountUID,
			Description: "将所有通知邮件自动归档",
			Enabled:     true,
			Priority:    10,
			MatchMode:   "all",
			Conditions: []model.RuleCondition{
				{
					Field:    "subject",
					Operator: "starts_with",
					Value:    "【通知】",
				},
			},
			Actions: []model.RuleAction{
				{
					Type: "mark_read",
				},
				{
					Type: "archive",
				},
			},
		}

		err := ruleService.Create(ctx, rule)
		if err != nil {
			t.Fatalf("Failed to create rule: %v", err)
		}

		if rule.ID == 0 {
			t.Fatal("Rule ID should not be 0 after creation")
		}

		// 3. 创建测试邮件
		email := &model.Email{
			ProviderID:  "test-email-001",
			AccountUID:  accountUID,
			Subject:     "【通知】系统维护通知",
			FromAddress: "system@example.com",
			ToAddress:   "user@example.com",
			TextBody:    "系统将于今晚进行维护",
			IsRead:      false,
			IsArchived:  false,
			SentAt:      time.Now(),
			ReceivedAt:  time.Now(),
		}

		err = emailRepo.Create(ctx, email)
		if err != nil {
			t.Fatalf("Failed to create email: %v", err)
		}

		// 4. 应用规则
		err = ruleService.ApplyRules(ctx, email)
		if err != nil {
			t.Fatalf("Failed to apply rules: %v", err)
		}

		// 5. 验证邮件状态
		updatedEmail, err := emailRepo.FindByID(ctx, email.ID)
		if err != nil {
			t.Fatalf("Failed to get updated email: %v", err)
		}

		if !updatedEmail.IsRead {
			t.Error("Email should be marked as read")
		}

		if !updatedEmail.IsArchived {
			t.Error("Email should be archived")
		}

		// 6. 验证规则统计
		updatedRule, err := ruleService.GetByID(ctx, rule.ID)
		if err != nil {
			t.Fatalf("Failed to get updated rule: %v", err)
		}

		if updatedRule.MatchedCount != 1 {
			t.Errorf("Rule matched_count should be 1, got %d", updatedRule.MatchedCount)
		}
	})

	t.Run("规则优先级和停止处理", func(t *testing.T) {
		accountUID := "test-account-002"

		// 创建高优先级规则（删除垃圾邮件）
		rule1 := &model.EmailRule{
			Name:           "删除垃圾邮件",
			AccountUID:     accountUID,
			Enabled:        true,
			Priority:       1,
			MatchMode:      "any",
			StopProcessing: true,
			Conditions: []model.RuleCondition{
				{
					Field:    "subject",
					Operator: "contains",
					Value:    "中奖",
				},
			},
			Actions: []model.RuleAction{
				{
					Type: "delete",
				},
			},
		}

		err := ruleService.Create(ctx, rule1)
		if err != nil {
			t.Fatalf("Failed to create rule1: %v", err)
		}

		// 创建低优先级规则（标记已读）
		rule2 := &model.EmailRule{
			Name:       "标记所有邮件已读",
			AccountUID: accountUID,
			Enabled:    true,
			Priority:   100,
			MatchMode:  "all",
			Conditions: []model.RuleCondition{
				{
					Field:    "subject",
					Operator: "contains",
					Value:    "",
				},
			},
			Actions: []model.RuleAction{
				{
					Type: "mark_read",
				},
			},
		}

		err = ruleService.Create(ctx, rule2)
		if err != nil {
			t.Fatalf("Failed to create rule2: %v", err)
		}

		// 创建垃圾邮件
		email := &model.Email{
			ProviderID:  "test-email-002",
			AccountUID:  accountUID,
			Subject:     "恭喜您中奖了！",
			FromAddress: "spam@example.com",
			ToAddress:   "user@example.com",
			IsRead:      false,
			IsDeleted:   false,
			SentAt:      time.Now(),
			ReceivedAt:  time.Now(),
		}

		err = emailRepo.Create(ctx, email)
		if err != nil {
			t.Fatalf("Failed to create email: %v", err)
		}

		// 应用规则
		err = ruleService.ApplyRules(ctx, email)
		if err != nil {
			t.Fatalf("Failed to apply rules: %v", err)
		}

		// 验证：应该被删除，但不应该被标记已读（因为 StopProcessing）
		updatedEmail, err := emailRepo.FindByID(ctx, email.ID)
		if err != nil {
			t.Fatalf("Failed to get updated email: %v", err)
		}

		if !updatedEmail.IsDeleted {
			t.Error("Email should be deleted")
		}

		if updatedEmail.IsRead {
			t.Error("Email should not be marked as read (rule2 should not execute)")
		}
	})

	t.Run("匹配模式测试", func(t *testing.T) {
		accountUID := "test-account-003"

		// 测试 all 模式
		t.Run("all 模式", func(t *testing.T) {
			rule := &model.EmailRule{
				Name:       "重要工作邮件",
				AccountUID: accountUID,
				Enabled:    true,
				Priority:   10,
				MatchMode:  "all",
				Conditions: []model.RuleCondition{
					{
						Field:    "from",
						Operator: "contains",
						Value:    "@company.com",
					},
					{
						Field:    "subject",
						Operator: "contains",
						Value:    "重要",
					},
				},
				Actions: []model.RuleAction{
					{
						Type: "star",
					},
				},
			}

			err := ruleService.Create(ctx, rule)
			if err != nil {
				t.Fatalf("Failed to create rule: %v", err)
			}

			// 创建匹配的邮件
			email1 := &model.Email{
				ProviderID:  "test-email-003-1",
				AccountUID:  accountUID,
				Subject:     "重要：项目进度",
				FromAddress: "boss@company.com",
				ToAddress:   "user@example.com",
				IsStarred:   false,
				SentAt:      time.Now(),
				ReceivedAt:  time.Now(),
			}

			err = emailRepo.Create(ctx, email1)
			if err != nil {
				t.Fatalf("Failed to create email1: %v", err)
			}

			err = ruleService.ApplyRules(ctx, email1)
			if err != nil {
				t.Fatalf("Failed to apply rules: %v", err)
			}

			updatedEmail1, _ := emailRepo.FindByID(ctx, email1.ID)
			if !updatedEmail1.IsStarred {
				t.Error("Email1 should be starred (all conditions matched)")
			}

			// 创建不完全匹配的邮件
			email2 := &model.Email{
				ProviderID:  "test-email-003-2",
				AccountUID:  accountUID,
				Subject:     "普通邮件",
				FromAddress: "boss@company.com",
				ToAddress:   "user@example.com",
				IsStarred:   false,
				SentAt:      time.Now(),
				ReceivedAt:  time.Now(),
			}

			err = emailRepo.Create(ctx, email2)
			if err != nil {
				t.Fatalf("Failed to create email2: %v", err)
			}

			err = ruleService.ApplyRules(ctx, email2)
			if err != nil {
				t.Fatalf("Failed to apply rules: %v", err)
			}

			updatedEmail2, _ := emailRepo.FindByID(ctx, email2.ID)
			if updatedEmail2.IsStarred {
				t.Error("Email2 should not be starred (not all conditions matched)")
			}
		})

		// 测试 any 模式
		t.Run("any 模式", func(t *testing.T) {
			rule := &model.EmailRule{
				Name:       "紧急或重要邮件",
				AccountUID: accountUID,
				Enabled:    true,
				Priority:   20,
				MatchMode:  "any",
				Conditions: []model.RuleCondition{
					{
						Field:    "subject",
						Operator: "contains",
						Value:    "紧急",
					},
					{
						Field:    "subject",
						Operator: "contains",
						Value:    "urgent",
					},
				},
				Actions: []model.RuleAction{
					{
						Type: "star",
					},
				},
			}

			err := ruleService.Create(ctx, rule)
			if err != nil {
				t.Fatalf("Failed to create rule: %v", err)
			}

			// 创建匹配第一个条件的邮件
			email := &model.Email{
				ProviderID:  "test-email-003-3",
				AccountUID:  accountUID,
				Subject:     "紧急：服务器故障",
				FromAddress: "ops@example.com",
				ToAddress:   "user@example.com",
				IsStarred:   false,
				SentAt:      time.Now(),
				ReceivedAt:  time.Now(),
			}

			err = emailRepo.Create(ctx, email)
			if err != nil {
				t.Fatalf("Failed to create email: %v", err)
			}

			err = ruleService.ApplyRules(ctx, email)
			if err != nil {
				t.Fatalf("Failed to apply rules: %v", err)
			}

			updatedEmail, _ := emailRepo.FindByID(ctx, email.ID)
			if !updatedEmail.IsStarred {
				t.Error("Email should be starred (any condition matched)")
			}
		})
	})

	t.Run("正则表达式匹配", func(t *testing.T) {
		accountUID := "test-account-004"

		rule := &model.EmailRule{
			Name:       "匹配订单号邮件",
			AccountUID: accountUID,
			Enabled:    true,
			Priority:   10,
			MatchMode:  "all",
			Conditions: []model.RuleCondition{
				{
					Field:    "subject",
					Operator: "regex",
					Value:    "订单号[：:][0-9]{8,}",
				},
			},
			Actions: []model.RuleAction{
				{
					Type:  "move_folder",
					Value: "订单",
				},
			},
		}

		err := ruleService.Create(ctx, rule)
		if err != nil {
			t.Fatalf("Failed to create rule: %v", err)
		}

		// 创建匹配的邮件
		email := &model.Email{
			ProviderID:  "test-email-004",
			AccountUID:  accountUID,
			Subject:     "您的订单号：12345678 已发货",
			FromAddress: "shop@example.com",
			ToAddress:   "user@example.com",
			Folder:      "",
			SentAt:      time.Now(),
			ReceivedAt:  time.Now(),
		}

		err = emailRepo.Create(ctx, email)
		if err != nil {
			t.Fatalf("Failed to create email: %v", err)
		}

		err = ruleService.ApplyRules(ctx, email)
		if err != nil {
			t.Fatalf("Failed to apply rules: %v", err)
		}

		updatedEmail, _ := emailRepo.FindByID(ctx, email.ID)
		if updatedEmail.Folder != "订单" {
			t.Errorf("Email should be moved to '订单' folder, got '%s'", updatedEmail.Folder)
		}
	})

	t.Run("规则测试功能", func(t *testing.T) {
		accountUID := "test-account-005"

		rule := &model.EmailRule{
			Name:       "测试规则",
			AccountUID: accountUID,
			Enabled:    true,
			Priority:   10,
			MatchMode:  "all",
			Conditions: []model.RuleCondition{
				{
					Field:    "from",
					Operator: "contains",
					Value:    "test",
				},
			},
			Actions: []model.RuleAction{
				{
					Type: "mark_read",
				},
			},
		}

		err := ruleService.Create(ctx, rule)
		if err != nil {
			t.Fatalf("Failed to create rule: %v", err)
		}

		// 测试匹配的邮件
		testEmail1 := &model.Email{
			FromAddress: "test@example.com",
			Subject:     "Test email",
		}

		matched, err := ruleService.TestRule(ctx, rule, testEmail1)
		if err != nil {
			t.Fatalf("Failed to test rule: %v", err)
		}

		if !matched {
			t.Error("Rule should match test email 1")
		}

		// 测试不匹配的邮件
		testEmail2 := &model.Email{
			FromAddress: "user@example.com",
			Subject:     "Normal email",
		}

		matched, err = ruleService.TestRule(ctx, rule, testEmail2)
		if err != nil {
			t.Fatalf("Failed to test rule: %v", err)
		}

		if matched {
			t.Error("Rule should not match test email 2")
		}
	})
}

// TestRuleValidation 规则验证测试
func TestRuleValidation(t *testing.T) {
	db := setupTestDB(t)
	ruleRepo := repository.NewRuleRepository(db)
	emailRepo := repository.NewEmailRepository(db)
	ruleService := service.NewRuleService(ruleRepo, emailRepo)

	ctx := context.Background()

	tests := []struct {
		name    string
		rule    *model.EmailRule
		wantErr bool
		errMsg  string
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
						Value:    "test",
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
			name: "缺少规则名称",
			rule: &model.EmailRule{
				AccountUID: "test-account",
				Conditions: []model.RuleCondition{
					{
						Field:    "subject",
						Operator: "contains",
						Value:    "test",
					},
				},
				Actions: []model.RuleAction{
					{
						Type: "mark_read",
					},
				},
			},
			wantErr: true,
			errMsg:  "rule name is required",
		},
		{
			name: "无效的字段",
			rule: &model.EmailRule{
				Name:       "测试规则",
				AccountUID: "test-account",
				Conditions: []model.RuleCondition{
					{
						Field:    "invalid_field",
						Operator: "contains",
						Value:    "test",
					},
				},
				Actions: []model.RuleAction{
					{
						Type: "mark_read",
					},
				},
			},
			wantErr: true,
			errMsg:  "invalid condition field",
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
						Value:    "test",
					},
				},
				Actions: []model.RuleAction{
					{
						Type: "mark_read",
					},
				},
			},
			wantErr: true,
			errMsg:  "invalid condition operator",
		},
		{
			name: "无效的正则表达式",
			rule: &model.EmailRule{
				Name:       "测试规则",
				AccountUID: "test-account",
				Conditions: []model.RuleCondition{
					{
						Field:    "subject",
						Operator: "regex",
						Value:    "[invalid(regex",
					},
				},
				Actions: []model.RuleAction{
					{
						Type: "mark_read",
					},
				},
			},
			wantErr: true,
			errMsg:  "invalid regex pattern",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ruleService.Create(ctx, tt.rule)
			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil {
				if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("Error message should contain '%s', got '%s'", tt.errMsg, err.Error())
				}
			}
		})
	}
}

// 辅助函数
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
