package service

import (
	"context"

	"fusionmail/internal/dto/response"
	"fusionmail/internal/model"
)

// 以下为转发方法：compositeAccountService 只做委托，不含业务逻辑。
func (c *compositeAccountService) Create(ctx context.Context, req *CreateAccountRequest) (*response.AccountResponse, error) {
	return c.crud.Create(ctx, req)
}
func (c *compositeAccountService) GetByUID(ctx context.Context, uid string) (*response.AccountResponse, error) {
	return c.crud.GetByUID(ctx, uid)
}
func (c *compositeAccountService) GetByEmail(ctx context.Context, email string) (*response.AccountResponse, error) {
	return c.crud.GetByEmail(ctx, email)
}
func (c *compositeAccountService) List(ctx context.Context) ([]*response.AccountResponse, error) {
	return c.crud.List(ctx)
}
func (c *compositeAccountService) ListWithFilter(ctx context.Context, filter *AccountListFilter) (*AccountListResponse, error) {
	return c.crud.ListWithFilter(ctx, filter)
}
func (c *compositeAccountService) Update(ctx context.Context, uid string, req *UpdateAccountRequest) (*response.AccountResponse, error) {
	return c.crud.Update(ctx, uid, req)
}
func (c *compositeAccountService) Delete(ctx context.Context, uid string) error {
	return c.crud.Delete(ctx, uid)
}
func (c *compositeAccountService) GetByUIDWithRelations(ctx context.Context, uid string) (*model.EmailAccount, error) {
	return c.crud.GetByUIDWithRelations(ctx, uid)
}
func (c *compositeAccountService) ListWithRelations(ctx context.Context) ([]*model.EmailAccount, error) {
	return c.crud.ListWithRelations(ctx)
}
func (c *compositeAccountService) ListSyncEnabledWithRelations(ctx context.Context) ([]*model.EmailAccount, error) {
	return c.crud.ListSyncEnabledWithRelations(ctx)
}
func (c *compositeAccountService) MatchProviderByEmail(ctx context.Context, email string) (*model.Provider, error) {
	return c.crud.MatchProviderByEmail(ctx, email)
}
func (c *compositeAccountService) TestConnection(ctx context.Context, uid string) error {
	return c.sync.TestConnection(ctx, uid)
}
func (c *compositeAccountService) SetStatus(ctx context.Context, uid string, status string) error {
	return c.sync.SetStatus(ctx, uid, status)
}
func (c *compositeAccountService) DisableAccount(ctx context.Context, uid string) error {
	return c.sync.DisableAccount(ctx, uid)
}
func (c *compositeAccountService) EnableAccount(ctx context.Context, uid string) error {
	return c.sync.EnableAccount(ctx, uid)
}
func (c *compositeAccountService) BatchEnableAccounts(ctx context.Context, uids []string) (*BatchOperationResult, error) {
	return c.sync.BatchEnableAccounts(ctx, uids)
}
func (c *compositeAccountService) BatchDisableAccounts(ctx context.Context, uids []string) (*BatchOperationResult, error) {
	return c.sync.BatchDisableAccounts(ctx, uids)
}
func (c *compositeAccountService) ClearSyncError(ctx context.Context, uid string) error {
	return c.sync.ClearSyncError(ctx, uid)
}
func (c *compositeAccountService) ListDeleted(ctx context.Context) ([]*response.AccountResponse, error) {
	return c.trash.ListDeleted(ctx)
}
func (c *compositeAccountService) Restore(ctx context.Context, uid string) error {
	return c.trash.Restore(ctx, uid)
}
func (c *compositeAccountService) ForceDelete(ctx context.Context, uid string) error {
	return c.trash.ForceDelete(ctx, uid)
}
func (c *compositeAccountService) CleanupTrash(ctx context.Context, days int) (int, error) {
	return c.trash.CleanupTrash(ctx, days)
}
