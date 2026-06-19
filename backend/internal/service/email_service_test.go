package service

import (
	"context"
	"testing"
	"time"

	"fusionmail/internal/dto/response"
	"fusionmail/internal/model"
	"fusionmail/internal/repository"
	"fusionmail/pkg/logger"
)

// mockEmailRepo 轻量 mock，仅实现测试需要的方法
type mockEmailRepo struct {
	emails     map[int64]*model.Email
	nextID     int64
	listResult []*model.Email
	listTotal  int64
}

func newMockEmailRepo() *mockEmailRepo {
	return &mockEmailRepo{emails: make(map[int64]*model.Email), nextID: 1}
}

func (r *mockEmailRepo) Create(ctx context.Context, email *model.Email) error {
	email.ID = r.nextID
	r.nextID++
	r.emails[email.ID] = email
	return nil
}
func (r *mockEmailRepo) CreateBatch(ctx context.Context, emails []*model.Email) error {
	for _, e := range emails {
		if err := r.Create(ctx, e); err != nil {
			return err
		}
	}
	return nil
}
func (r *mockEmailRepo) FindByID(ctx context.Context, id int64) (*model.Email, error) {
	e, ok := r.emails[id]
	if !ok {
		return nil, nil
	}
	return e, nil
}
func (r *mockEmailRepo) FindByIDs(context.Context, []int64) ([]*model.Email, error) {
	return nil, nil
}
func (r *mockEmailRepo) FindByProviderID(context.Context, string, string) (*model.Email, error) {
	return nil, nil
}
func (r *mockEmailRepo) FindByDedupeKey(context.Context, string, string) (*model.Email, error) {
	return nil, nil
}
func (r *mockEmailRepo) Update(context.Context, *model.Email) error { return nil }
func (r *mockEmailRepo) UpdateLocalStatus(context.Context, int64, *bool, *bool, *bool, *bool) error {
	return nil
}
func (r *mockEmailRepo) BatchUpdateLocalDeleted(context.Context, []int64, bool) (int64, error) {
	return 0, nil
}
func (r *mockEmailRepo) Delete(context.Context, int64) error              { return nil }
func (r *mockEmailRepo) DeleteByAccountUID(context.Context, string) error { return nil }
func (r *mockEmailRepo) SoftDeleteByAccountUID(context.Context, string) error {
	return nil
}
func (r *mockEmailRepo) RestoreByAccountUID(context.Context, string) error { return nil }
func (r *mockEmailRepo) List(context.Context, *repository.EmailFilter, int, int) ([]*model.Email, int64, error) {
	return r.listResult, r.listTotal, nil
}
func (r *mockEmailRepo) Search(context.Context, string, string, int, int) ([]*model.Email, int64, error) {
	return nil, 0, nil
}
func (r *mockEmailRepo) CountUnread(context.Context, string) (int64, error)    { return 0, nil }
func (r *mockEmailRepo) MarkAsRead(context.Context, []int64) error             { return nil }
func (r *mockEmailRepo) MarkAsUnread(context.Context, []int64) error           { return nil }
func (r *mockEmailRepo) MarkAllAsRead(context.Context, *string) (int64, error) { return 0, nil }
func (r *mockEmailRepo) Count(context.Context, *repository.EmailFilter) (int64, error) {
	return 0, nil
}
func (r *mockEmailRepo) CountByDateRange(context.Context, time.Time, time.Time) (int64, error) {
	return 0, nil
}
func (r *mockEmailRepo) CountByAccount(context.Context, string) (int64, error) { return 0, nil }
func (r *mockEmailRepo) GetGlobalStats(context.Context) (*repository.GlobalEmailStats, error) {
	return nil, nil
}
func (r *mockEmailRepo) GetAccountStats(context.Context, string) (*repository.AccountEmailStats, error) {
	return nil, nil
}

func TestEmailQueryParams_ToFilter(t *testing.T) {
	isRead := true
	isSpam := false
	groupID := int64(0)

	params := &EmailQueryParams{
		AccountUID:  "test-uid",
		GroupID:     &groupID,
		IsRead:      &isRead,
		IsSpam:      &isSpam,
		FromAddress: "sender@test.com",
		Subject:     "test subject",
		StartDate:   "2024-01-01",
		EndDate:     "2024-12-31",
		SearchQuery: "keyword",
	}

	filter := params.toFilter()
	if filter.AccountUID != "test-uid" {
		t.Errorf("AccountUID = %s, want test-uid", filter.AccountUID)
	}
	if filter.GroupID == nil || *filter.GroupID != 0 {
		t.Errorf("GroupID = %v, want 0", filter.GroupID)
	}
	if filter.IsRead == nil || *filter.IsRead != true {
		t.Errorf("IsRead = %v, want true", filter.IsRead)
	}
	if filter.IsSpam == nil || *filter.IsSpam != false {
		t.Errorf("IsSpam = %v, want false", filter.IsSpam)
	}
	if filter.FromAddress != "sender@test.com" {
		t.Errorf("FromAddress = %s, want sender@test.com", filter.FromAddress)
	}
	if filter.SearchQuery != "keyword" {
		t.Errorf("SearchQuery = %s, want keyword", filter.SearchQuery)
	}
}

func TestEmailQueryParams_ToFilter_Nil(t *testing.T) {
	var params *EmailQueryParams
	filter := params.toFilter()
	if filter != nil {
		t.Errorf("expected nil filter for nil params, got %v", filter)
	}
}

func TestGetEmailByID_ReturnsDTO(t *testing.T) {
	repo := newMockEmailRepo()
	email := &model.Email{
		ID:          1,
		ProviderID:  "provider-1",
		AccountUID:  "account-1",
		Subject:     "Test Subject",
		FromAddress: "sender@test.com",
		HTMLBody:    "<p>Test body</p>",
		IsRead:      false,
		IsDeleted:   false,
		IsSpam:      false,
		DedupeKey:   "secret-dedupe-key",
		SyncedAt:    time.Now(),
	}
	repo.emails[1] = email

	svc := &emailService{
		emailRepo: repo,
		logger:    logger.NewWithModule("Test"),
	}

	result, err := svc.GetEmailByID(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetEmailByID failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.ID != 1 {
		t.Errorf("ID = %d, want 1", result.ID)
	}
	if result.Subject != "Test Subject" {
		t.Errorf("Subject = %s, want 'Test Subject'", result.Subject)
	}
	if result.FromAddress != "sender@test.com" {
		t.Errorf("FromAddress = %s, want 'sender@test.com'", result.FromAddress)
	}
}

func TestGetEmailByID_NotFound(t *testing.T) {
	repo := newMockEmailRepo()
	svc := &emailService{
		emailRepo: repo,
		logger:    logger.NewWithModule("Test"),
	}

	_, err := svc.GetEmailByID(context.Background(), 999)
	if err == nil {
		t.Fatal("expected error for non-existent email, got nil")
	}
}

func TestGetEmailList_Pagination(t *testing.T) {
	repo := newMockEmailRepo()
	// 模拟 25 封邮件
	repo.listResult = make([]*model.Email, 20)
	repo.listTotal = 25

	svc := &emailService{
		emailRepo: repo,
		logger:    logger.NewWithModule("Test"),
	}

	params := &EmailQueryParams{AccountUID: "test-uid"}
	result, err := svc.GetEmailList(context.Background(), params, 1, 20)
	if err != nil {
		t.Fatalf("GetEmailList failed: %v", err)
	}
	if result.Total != 25 {
		t.Errorf("Total = %d, want 25", result.Total)
	}
	if result.Page != 1 {
		t.Errorf("Page = %d, want 1", result.Page)
	}
	if result.PageSize != 20 {
		t.Errorf("PageSize = %d, want 20", result.PageSize)
	}
	expectedPages := 2 // 25 / 20 = 1.25 → 2 pages
	if result.TotalPages != expectedPages {
		t.Errorf("TotalPages = %d, want %d", result.TotalPages, expectedPages)
	}
}

func TestGetEmailList_PageClamping(t *testing.T) {
	repo := newMockEmailRepo()
	svc := &emailService{
		emailRepo: repo,
		logger:    logger.NewWithModule("Test"),
	}

	params := &EmailQueryParams{}
	// page < 1 should be clamped to 1
	result, err := svc.GetEmailList(context.Background(), params, 0, 20)
	if err != nil {
		t.Fatalf("GetEmailList failed: %v", err)
	}
	if result.Page != 1 {
		t.Errorf("Page = %d, want 1 (clamped)", result.Page)
	}
}

func TestToEmailDetailResponse_ExcludesInternalFields(t *testing.T) {
	email := &model.Email{
		ID:        1,
		DedupeKey: "secret-key",
		SyncedAt:  time.Now(),
		Subject:   "Test",
	}

	result := toEmailDetailResponse(email)
	if result.Subject != "Test" {
		t.Errorf("Subject = %s, want 'Test'", result.Subject)
	}
	// Verify DTO type — it should not have DedupeKey/SyncedAt/DeletedAt fields
	// This is enforced at compile time by the struct definition
	var _ *response.EmailDetailResponse = result
}
