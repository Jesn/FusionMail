package service

import (
	"context"
	"testing"
	"time"

	"fusionmail/config"
	"fusionmail/internal/dto/response"
	"fusionmail/internal/model"
	"fusionmail/internal/repository"
	"fusionmail/pkg/crypto"
	"fusionmail/pkg/logger"
)

// mockAccountRepo 轻量 mock
type mockAccountRepo struct {
	accounts map[string]*model.EmailAccount
	nextID   int64
}

func newMockAccountRepo() *mockAccountRepo {
	return &mockAccountRepo{accounts: make(map[string]*model.EmailAccount), nextID: 1}
}

func (r *mockAccountRepo) Create(ctx context.Context, account *model.EmailAccount) error {
	account.ID = r.nextID
	r.nextID++
	r.accounts[account.UID] = account
	return nil
}
func (r *mockAccountRepo) FindByID(context.Context, int64) (*model.EmailAccount, error) {
	return nil, nil
}
func (r *mockAccountRepo) FindByUID(ctx context.Context, uid string) (*model.EmailAccount, error) {
	a, ok := r.accounts[uid]
	if !ok {
		return nil, nil
	}
	return a, nil
}
func (r *mockAccountRepo) FindByEmail(context.Context, string) (*model.EmailAccount, error) {
	return nil, nil
}
func (r *mockAccountRepo) Update(ctx context.Context, account *model.EmailAccount) error {
	r.accounts[account.UID] = account
	return nil
}
func (r *mockAccountRepo) Delete(context.Context, int64) error { return nil }
func (r *mockAccountRepo) List(context.Context, int, int) ([]*model.EmailAccount, int64, error) {
	return nil, 0, nil
}
func (r *mockAccountRepo) ListWithFilter(context.Context, *repository.AccountListFilter) ([]*model.EmailAccount, int64, error) {
	return nil, 0, nil
}
func (r *mockAccountRepo) ListSyncEnabled(context.Context) ([]*model.EmailAccount, error) {
	return nil, nil
}
func (r *mockAccountRepo) HealWebhookChildPollingFlags(context.Context) (int64, error) {
	return 0, nil
}
func (r *mockAccountRepo) MarkRemoteMailboxDeleted(context.Context, string) error { return nil }
func (r *mockAccountRepo) ReactivateFromRemoteOrphan(context.Context, string) error {
	return nil
}
func (r *mockAccountRepo) UpdateSyncStatus(context.Context, string, string, string) error {
	return nil
}
func (r *mockAccountRepo) IncrementEmailCount(context.Context, string, int) error { return nil }
func (r *mockAccountRepo) UpdateUnreadCount(context.Context, string, int) error   { return nil }
func (r *mockAccountRepo) FindAll(context.Context) ([]*model.EmailAccount, error) { return nil, nil }
func (r *mockAccountRepo) Count(context.Context) (int64, error)                   { return 0, nil }
func (r *mockAccountRepo) CountActive(context.Context) (int64, error)             { return 0, nil }
func (r *mockAccountRepo) IncrementConsecutiveFailures(context.Context, string) (int, error) {
	return 0, nil
}
func (r *mockAccountRepo) ResetConsecutiveFailures(context.Context, string) error { return nil }
func (r *mockAccountRepo) AutoDisableAccount(context.Context, string, string) error {
	return nil
}
func (r *mockAccountRepo) AutoSoftDeleteAccount(context.Context, string, string) error {
	return nil
}
func (r *mockAccountRepo) UpdateSyncProgress(context.Context, string, string, string) error {
	return nil
}
func (r *mockAccountRepo) UpdateUIDSyncState(context.Context, string, int64, int64) error {
	return nil
}
func (r *mockAccountRepo) FindAllWithDeleted(context.Context) ([]*model.EmailAccount, error) {
	return nil, nil
}
func (r *mockAccountRepo) FindDeleted(context.Context) ([]*model.EmailAccount, error) {
	return nil, nil
}
func (r *mockAccountRepo) FindDeletedByEmail(context.Context, string) ([]*model.EmailAccount, error) {
	return nil, nil
}
func (r *mockAccountRepo) FindDeletedBefore(context.Context, time.Time) ([]*model.EmailAccount, error) {
	return nil, nil
}
func (r *mockAccountRepo) FindByUIDIncludingDeleted(context.Context, string) (*model.EmailAccount, error) {
	return nil, nil
}
func (r *mockAccountRepo) Restore(context.Context, string) error     { return nil }
func (r *mockAccountRepo) ForceDelete(context.Context, string) error { return nil }
func (r *mockAccountRepo) FindByGroupID(context.Context, int64) ([]*model.EmailAccount, error) {
	return nil, nil
}
func (r *mockAccountRepo) FindAllByGroupID(context.Context, int64) ([]*model.EmailAccount, error) {
	return nil, nil
}
func (r *mockAccountRepo) FindUngrouped(context.Context) ([]*model.EmailAccount, error) {
	return nil, nil
}
func (r *mockAccountRepo) UpdateGroupID(context.Context, string, *int64) error { return nil }
func (r *mockAccountRepo) BatchUpdateGroupID(context.Context, []string, *int64) error {
	return nil
}
func (r *mockAccountRepo) FindByUIDWithRelations(context.Context, string) (*model.EmailAccount, error) {
	return nil, nil
}
func (r *mockAccountRepo) FindByIDWithRelations(context.Context, int64) (*model.EmailAccount, error) {
	return nil, nil
}
func (r *mockAccountRepo) ListWithRelations(context.Context, int, int) ([]*model.EmailAccount, int64, error) {
	return nil, 0, nil
}
func (r *mockAccountRepo) ListSyncEnabledWithRelations(context.Context) ([]*model.EmailAccount, error) {
	return nil, nil
}
func (r *mockAccountRepo) FindByProviderID(context.Context, int64) ([]*model.EmailAccount, error) {
	return nil, nil
}
func (r *mockAccountRepo) FindByAdapterID(context.Context, int64) ([]*model.EmailAccount, error) {
	return nil, nil
}
func (r *mockAccountRepo) FindByProviderIDs(context.Context, []int64, int, int) ([]*model.EmailAccount, int64, error) {
	return nil, 0, nil
}
func (r *mockAccountRepo) FindByParentAccountUID(context.Context, string) ([]*model.EmailAccount, error) {
	return nil, nil
}
func (r *mockAccountRepo) FindChildrenByParent(context.Context, *repository.ChildAccountListFilter) ([]*model.EmailAccount, int64, error) {
	return nil, 0, nil
}
func (r *mockAccountRepo) FindByDomain(context.Context, string) ([]*model.EmailAccount, error) {
	return nil, nil
}

func TestToAccountResponse_ExcludesInternalFields(t *testing.T) {
	account := &model.EmailAccount{
		ID:                      1,
		UID:                     "test-uid",
		Email:                   "test@test.com",
		Status:                  "active",
		EncryptedCredentials:    "secret-creds",
		SyncProgressJSON:        `{"progress":50}`,
		SyncCursor:              "cursor-data",
		ConsecutiveAuthFailures: 3,
		UIDValidity:             12345,
		LastUID:                 67890,
	}

	result := toAccountResponse(account)
	if result.ID != 1 {
		t.Errorf("ID = %d, want 1", result.ID)
	}
	if result.UID != "test-uid" {
		t.Errorf("UID = %s, want test-uid", result.UID)
	}
	if result.Status != "active" {
		t.Errorf("Status = %s, want active", result.Status)
	}
	// Verify DTO type doesn't have internal fields — enforced at compile time
	var _ *response.AccountResponse = result
}

func TestToAccountResponse_IncludesCompatibilityFields(t *testing.T) {
	account := &model.EmailAccount{
		ID:            1,
		UID:           "test-uid",
		Email:         "test@gmail.com",
		ProviderID:    10,
		ProviderRef:   &model.Provider{ID: 10, Name: "gmail", DisplayName: "Gmail"},
		AdapterID:     20,
		AdapterRef:    &model.Adapter{ID: 20, Name: model.AdapterNameGmail, AuthType: model.AdapterAuthTypeOAuth2},
		SyncModeField: model.SyncModeWebhook,
	}

	result := toAccountResponse(account)
	if result.Provider != "gmail" {
		t.Errorf("Provider = %s, want gmail", result.Provider)
	}
	if result.Protocol != "oauth2" {
		t.Errorf("Protocol = %s, want oauth2", result.Protocol)
	}
	if result.AuthType != "oauth2" {
		t.Errorf("AuthType = %s, want oauth2", result.AuthType)
	}
	if result.SyncMode != "webhook" {
		t.Errorf("SyncMode = %s, want webhook", result.SyncMode)
	}
	if result.ProviderRef == nil || result.ProviderRef.DisplayName != "Gmail" {
		t.Fatalf("ProviderRef = %#v, want display name Gmail", result.ProviderRef)
	}
}

func TestToAccountResponse_Nil(t *testing.T) {
	result := toAccountResponse(nil)
	if result != nil {
		t.Errorf("expected nil for nil input, got %v", result)
	}
}

func TestToAccountResponseList(t *testing.T) {
	accounts := []*model.EmailAccount{
		{ID: 1, UID: "uid-1", Email: "a@test.com"},
		{ID: 2, UID: "uid-2", Email: "b@test.com"},
		{},
	}

	result := toAccountResponseList(accounts)
	if len(result) != 3 {
		t.Fatalf("len = %d, want 3", len(result))
	}
	if result[0].UID != "uid-1" {
		t.Errorf("result[0].UID = %s, want uid-1", result[0].UID)
	}
	if result[1].UID != "uid-2" {
		t.Errorf("result[1].UID = %s, want uid-2", result[1].UID)
	}
	if result[2].ID != 0 {
		t.Errorf("result[2].ID = %d, want 0 for zero-value account", result[2].ID)
	}
}

func TestAccountService_GetByUID_ReturnsDTO(t *testing.T) {
	repo := newMockAccountRepo()
	repo.accounts["test-uid"] = &model.EmailAccount{
		ID:     1,
		UID:    "test-uid",
		Email:  "test@test.com",
		Status: "active",
	}

	encryptor, _ := crypto.NewService(config.DefaultEncryptionKey)
	svc := &accountService{
		accountRepo:   repo,
		cryptoService: encryptor,
		logger:        logger.NewWithModule("Test"),
	}

	result, err := svc.GetByUID(context.Background(), "test-uid")
	if err != nil {
		t.Fatalf("GetByUID failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.UID != "test-uid" {
		t.Errorf("UID = %s, want test-uid", result.UID)
	}
	if result.Email != "test@test.com" {
		t.Errorf("Email = %s, want test@test.com", result.Email)
	}
}

func TestAccountService_GetByUID_NotFound(t *testing.T) {
	repo := newMockAccountRepo()
	encryptor, _ := crypto.NewService(config.DefaultEncryptionKey)
	svc := &accountService{
		accountRepo:   repo,
		cryptoService: encryptor,
		logger:        logger.NewWithModule("Test"),
	}

	_, err := svc.GetByUID(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for non-existent account, got nil")
	}
}

func TestAccountService_SetStatus(t *testing.T) {
	repo := newMockAccountRepo()
	repo.accounts["test-uid"] = &model.EmailAccount{
		ID:     1,
		UID:    "test-uid",
		Status: "active",
	}

	encryptor, _ := crypto.NewService(config.DefaultEncryptionKey)
	svc := &accountService{
		accountRepo:   repo,
		cryptoService: encryptor,
		logger:        logger.NewWithModule("Test"),
	}

	err := svc.SetStatus(context.Background(), "test-uid", "disabled")
	if err != nil {
		t.Fatalf("SetStatus failed: %v", err)
	}
	if repo.accounts["test-uid"].Status != "disabled" {
		t.Errorf("Status = %s, want disabled", repo.accounts["test-uid"].Status)
	}
}
