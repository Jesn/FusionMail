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
