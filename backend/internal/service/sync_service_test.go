package service

import (
	"context"
	"testing"
	"time"

	"fusionmail/internal/adapter"
	"fusionmail/internal/model"
	"fusionmail/internal/repository"
	"fusionmail/pkg/logger"
)

func TestProcessBatchEmailsReturnsBatchDelta(t *testing.T) {
	emailRepo := newSyncCountEmailRepo()
	service := &syncService{
		emailRepo:    emailRepo,
		dedupeKeyGen: NewDedupeKeyGenerator(),
		logger:       logger.NewWithModule("SyncTest"),
	}
	syncLog := &model.SyncLog{}

	firstNew, firstUpdated, firstFailed := service.processBatchEmails(context.Background(), "account-1", []*adapter.Email{
		newSyncCountAdapterEmail("provider-1", "message-1", "first"),
	}, syncLog)
	if firstNew != 1 || firstUpdated != 0 || firstFailed != 0 {
		t.Fatalf("first batch = new:%d updated:%d failed:%d, want new:1 updated:0 failed:0", firstNew, firstUpdated, firstFailed)
	}
	if syncLog.EmailsNew != 1 {
		t.Fatalf("syncLog.EmailsNew after first batch = %d, want 1", syncLog.EmailsNew)
	}

	secondNew, secondUpdated, secondFailed := service.processBatchEmails(context.Background(), "account-1", []*adapter.Email{
		newSyncCountAdapterEmail("provider-2", "message-2", "second"),
	}, syncLog)
	if secondNew != 1 || secondUpdated != 0 || secondFailed != 0 {
		t.Fatalf("second batch = new:%d updated:%d failed:%d, want new:1 updated:0 failed:0", secondNew, secondUpdated, secondFailed)
	}
	if syncLog.EmailsNew != 2 {
		t.Fatalf("syncLog.EmailsNew after second batch = %d, want 2", syncLog.EmailsNew)
	}
}

func newSyncCountAdapterEmail(providerID, messageID, subject string) *adapter.Email {
	return &adapter.Email{
		ProviderID:  providerID,
		MessageID:   messageID,
		Subject:     subject,
		FromAddress: "sender@example.com",
		SentAt:      time.Unix(1, 0).UTC(),
		ReceivedAt:  time.Unix(1, 0).UTC(),
	}
}

type syncCountEmailRepo struct {
	byDedupe map[string]*model.Email
	nextID   int64
}

func newSyncCountEmailRepo() *syncCountEmailRepo {
	return &syncCountEmailRepo{
		byDedupe: make(map[string]*model.Email),
		nextID:   1,
	}
}

func (r *syncCountEmailRepo) Create(_ context.Context, email *model.Email) error {
	email.ID = r.nextID
	r.nextID++
	r.byDedupe[email.AccountUID+"\x00"+email.DedupeKey] = email
	return nil
}

func (r *syncCountEmailRepo) FindByDedupeKey(_ context.Context, accountUID, dedupeKey string) (*model.Email, error) {
	return r.byDedupe[accountUID+"\x00"+dedupeKey], nil
}

func (r *syncCountEmailRepo) FindByProviderID(_ context.Context, providerID, accountUID string) (*model.Email, error) {
	for _, email := range r.byDedupe {
		if email.ProviderID == providerID && email.AccountUID == accountUID {
			return email, nil
		}
	}
	return nil, nil
}

func (r *syncCountEmailRepo) Update(_ context.Context, email *model.Email) error {
	r.byDedupe[email.AccountUID+"\x00"+email.DedupeKey] = email
	return nil
}

func (r *syncCountEmailRepo) CreateBatch(ctx context.Context, emails []*model.Email) error {
	for _, email := range emails {
		if err := r.Create(ctx, email); err != nil {
			return err
		}
	}
	return nil
}

func (r *syncCountEmailRepo) FindByID(context.Context, int64) (*model.Email, error) { return nil, nil }
func (r *syncCountEmailRepo) FindByIDs(context.Context, []int64) ([]*model.Email, error) {
	return nil, nil
}
func (r *syncCountEmailRepo) UpdateLocalStatus(context.Context, int64, *bool, *bool, *bool, *bool) error {
	return nil
}
func (r *syncCountEmailRepo) BatchUpdateLocalDeleted(context.Context, []int64, bool) (int64, error) {
	return 0, nil
}
func (r *syncCountEmailRepo) Delete(context.Context, int64) error                  { return nil }
func (r *syncCountEmailRepo) DeleteByAccountUID(context.Context, string) error     { return nil }
func (r *syncCountEmailRepo) SoftDeleteByAccountUID(context.Context, string) error { return nil }
func (r *syncCountEmailRepo) RestoreByAccountUID(context.Context, string) error    { return nil }
func (r *syncCountEmailRepo) List(context.Context, *repository.EmailFilter, int, int) ([]*model.Email, int64, error) {
	return nil, 0, nil
}
func (r *syncCountEmailRepo) Search(context.Context, string, string, int, int) ([]*model.Email, int64, error) {
	return nil, 0, nil
}
func (r *syncCountEmailRepo) CountUnread(context.Context, string) (int64, error)    { return 0, nil }
func (r *syncCountEmailRepo) MarkAsRead(context.Context, []int64) error             { return nil }
func (r *syncCountEmailRepo) MarkAsUnread(context.Context, []int64) error           { return nil }
func (r *syncCountEmailRepo) MarkAllAsRead(context.Context, *string) (int64, error) { return 0, nil }
func (r *syncCountEmailRepo) Count(context.Context, *repository.EmailFilter) (int64, error) {
	return 0, nil
}
func (r *syncCountEmailRepo) CountByDateRange(context.Context, time.Time, time.Time) (int64, error) {
	return 0, nil
}
func (r *syncCountEmailRepo) CountByAccount(context.Context, string) (int64, error) { return 0, nil }
func (r *syncCountEmailRepo) GetGlobalStats(context.Context) (*repository.GlobalEmailStats, error) {
	return nil, nil
}
func (r *syncCountEmailRepo) GetAccountStats(context.Context, string) (*repository.AccountEmailStats, error) {
	return nil, nil
}
