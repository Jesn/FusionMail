package spam

import (
	"context"
	"testing"

	"fusionmail/internal/model"
)

type fakeSpamRuleRepository struct {
	rules []*model.SpamRule
}

func (f *fakeSpamRuleRepository) Create(ctx context.Context, rule *model.SpamRule) error {
	f.rules = append(f.rules, rule)
	return nil
}

func (f *fakeSpamRuleRepository) FindByID(ctx context.Context, id int64) (*model.SpamRule, error) {
	for _, rule := range f.rules {
		if rule.ID == id {
			return rule, nil
		}
	}
	return nil, nil
}

func (f *fakeSpamRuleRepository) Update(ctx context.Context, rule *model.SpamRule) error {
	return nil
}

func (f *fakeSpamRuleRepository) Delete(ctx context.Context, id int64) error {
	return nil
}

func (f *fakeSpamRuleRepository) FindAll(ctx context.Context) ([]*model.SpamRule, error) {
	return f.rules, nil
}

func (f *fakeSpamRuleRepository) FindEnabled(ctx context.Context) ([]*model.SpamRule, error) {
	enabled := make([]*model.SpamRule, 0, len(f.rules))
	for _, rule := range f.rules {
		if rule.Enabled {
			enabled = append(enabled, rule)
		}
	}
	return enabled, nil
}

func (f *fakeSpamRuleRepository) FindByCategory(ctx context.Context, category string) ([]*model.SpamRule, error) {
	rules := make([]*model.SpamRule, 0, len(f.rules))
	for _, rule := range f.rules {
		if rule.Category == category {
			rules = append(rules, rule)
		}
	}
	return rules, nil
}

func (f *fakeSpamRuleRepository) ToggleEnabled(ctx context.Context, id int64) error {
	return nil
}

func (f *fakeSpamRuleRepository) IncrementHitCount(ctx context.Context, id int64) error {
	return nil
}

func (f *fakeSpamRuleRepository) List(ctx context.Context, offset, limit int) ([]*model.SpamRule, int64, error) {
	return f.rules, int64(len(f.rules)), nil
}

func (f *fakeSpamRuleRepository) ListByCategory(ctx context.Context, category string, offset, limit int) ([]*model.SpamRule, int64, error) {
	rules, err := f.FindByCategory(ctx, category)
	return rules, int64(len(rules)), err
}

func (f *fakeSpamRuleRepository) CountBuiltin(ctx context.Context) (int64, error) {
	return 0, nil
}

func (f *fakeSpamRuleRepository) CountCustom(ctx context.Context) (int64, error) {
	return int64(len(f.rules)), nil
}

func TestRuleEngineCheck_AllowsNilSURBLChecker(t *testing.T) {
	repo := &fakeSpamRuleRepository{
		rules: []*model.SpamRule{
			{
				ID:       1,
				Name:     "链接数量过多",
				Category: "url",
				Score:    20,
				Enabled:  true,
			},
		},
	}
	engine := NewRuleEngine(repo, nil, nil)

	result, err := engine.Check(context.Background(), &model.Email{
		Subject:  "链接检测",
		TextBody: "https://a.example/a https://b.example/b https://c.example/c https://d.example/d https://e.example/e https://f.example/f",
	})
	if err != nil {
		t.Fatalf("Check returned error: %v", err)
	}
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if result.SURBLResult != nil {
		t.Fatal("Expected nil SURBL result when checker is not configured")
	}
	if result.Score != 20 {
		t.Fatalf("Expected URL rule score 20, got %d", result.Score)
	}
	if len(result.HitRules) != 1 {
		t.Fatalf("Expected 1 hit rule, got %d", len(result.HitRules))
	}
}

// ============ matchContent：Pattern 驱动 + legacy 回退 ============

func TestMatchContent_PatternDriven(t *testing.T) {
	engine := &RuleEngine{}

	tests := []struct {
		name    string
		pattern string
		email   *model.Email
		want    bool
	}{
		{
			name:    "uppercase 超阈值命中（规则名无关）",
			pattern: "uppercase:0.3",
			email:   &model.Email{TextBody: "AAAAAAAAAA aaaaaaaaaa"}, // 50% 大写
			want:    true,
		},
		{
			name:    "uppercase 低于阈值不命中",
			pattern: "uppercase:0.9",
			email:   &model.Email{TextBody: "AAAAAAAAAA aaaaaaaaaa"},
			want:    false,
		},
		{
			name:    "special 超阈值命中",
			pattern: "special:0.1",
			email:   &model.Email{TextBody: "!!!???### abc"},
			want:    true,
		},
		{
			name:    "html_tags 超阈值命中",
			pattern: "html_tags:5",
			email:   &model.Email{TextBody: "x", HTMLBody: "<div><p><b><i><u><span></span></u></i></b></p></div>"},
			want:    true,
		},
		{
			name:    "裸浮点 Pattern（历史内置规则形状）走 legacy 名字路由，名字无关时不命中",
			pattern: "0.3",
			email:   &model.Email{TextBody: "AAAAAAAAAA aaaaaaaaaa"},
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := &model.SpamRule{Name: "完全无关的规则名", Category: "content", Pattern: tt.pattern}
			got, _ := engine.matchContent(rule, tt.email)
			if got != tt.want {
				t.Fatalf("matchContent(pattern=%s) = %v, want %v", tt.pattern, got, tt.want)
			}
		})
	}
}

func TestMatchContent_LegacyNameFallback_PreservesBuiltinBehavior(t *testing.T) {
	engine := &RuleEngine{}
	// 内置规则形状：名字含"大写字母"，Pattern 是裸阈值
	rule := &model.SpamRule{Name: "大写字母比例过高", Category: "content", Pattern: "0.3"}
	email := &model.Email{TextBody: "AAAAAAAAAA aaaaaaaaaa"} // 50% 大写

	got, reason := engine.matchContent(rule, email)
	if !got {
		t.Fatalf("legacy 内置规则应保持原行为：命中")
	}
	if reason == "" {
		t.Fatalf("命中原因不应为空")
	}
}

// ============ matchURL：Pattern 驱动 ============

func TestMatchURL_PatternDriven(t *testing.T) {
	engine := &RuleEngine{}

	tests := []struct {
		name    string
		pattern string
		email   *model.Email
		want    bool
	}{
		{
			name:    "整数 Pattern = 链接数阈值（内置 链接数量过多 形状，Pattern=5 终于生效）",
			pattern: "5",
			email:   &model.Email{TextBody: "a http://x.com b http://y.com c http://z.com d http://a.com e http://b.com f http://c.com"},
			want:    true,
		},
		{
			name:    "链接数等于阈值不命中",
			pattern: "5",
			email:   &model.Email{TextBody: "a http://x.com b http://y.com c http://z.com d http://a.com e http://b.com"},
			want:    false,
		},
		{
			name:    "域名 Pattern 匹配 URL（内置 短链接检测 形状）",
			pattern: "bit.ly",
			email:   &model.Email{TextBody: "点此 https://bit.ly/xyz 查看"},
			want:    true,
		},
		{
			name:    "域名出现在纯文本但不在 URL 中不命中（不再误伤）",
			pattern: "bit.ly",
			email:   &model.Email{TextBody: "我们讨论 bit.ly 这个域名，但邮件里没有链接"},
			want:    false,
		},
		{
			name:    "空 Pattern 不命中",
			pattern: "",
			email:   &model.Email{TextBody: "http://x.com"},
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := &model.SpamRule{Name: "完全无关的规则名", Category: "url", Pattern: tt.pattern}
			got, _ := engine.matchURL(rule, tt.email)
			if got != tt.want {
				t.Fatalf("matchURL(pattern=%s) = %v, want %v", tt.pattern, got, tt.want)
			}
		})
	}
}

// ============ matchAttachment：Pattern 驱动 ============

func TestMatchAttachment_PatternDriven(t *testing.T) {
	engine := &RuleEngine{}

	newEmailWith := func(att *model.EmailAttachment) *model.Email {
		return &model.Email{HasAttachment: true, Attachments: []model.EmailAttachment{*att}}
	}

	tests := []struct {
		name    string
		pattern string
		email   *model.Email
		want    bool
	}{
		{
			name:    "扩展名匹配（内置 可执行附件检测 形状，终于生效）",
			pattern: ".exe",
			email:   newEmailWith(&model.EmailAttachment{Filename: "invoice.exe", ContentType: "application/x-msdownload"}),
			want:    true,
		},
		{
			name:    "扩展名大小写不敏感",
			pattern: ".exe",
			email:   newEmailWith(&model.EmailAttachment{Filename: "PHOTO.EXE"}),
			want:    true,
		},
		{
			name:    "MIME 前缀匹配",
			pattern: "image/",
			email:   newEmailWith(&model.EmailAttachment{Filename: "pic.png", ContentType: "image/png"}),
			want:    true,
		},
		{
			name:    "无附件不命中",
			pattern: ".exe",
			email:   &model.Email{HasAttachment: false},
			want:    false,
		},
		{
			name:    "附件存在但扩展名不符不命中",
			pattern: ".exe",
			email:   newEmailWith(&model.EmailAttachment{Filename: "report.pdf", ContentType: "application/pdf"}),
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := &model.SpamRule{Name: "完全无关的规则名", Category: "attachment", Pattern: tt.pattern}
			got, _ := engine.matchAttachment(rule, tt.email)
			if got != tt.want {
				t.Fatalf("matchAttachment(pattern=%s) = %v, want %v", tt.pattern, got, tt.want)
			}
		})
	}
}
