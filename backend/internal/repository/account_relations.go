package repository

import (
	"context"
	"errors"
	"fusionmail/internal/model"
	"strings"

	"gorm.io/gorm"
)

// FindByUIDWithRelations 根据 UID 查找账户并预加载 Provider 和 Adapter 关联
func (r *accountRepository) FindByUIDWithRelations(ctx context.Context, uid string) (*model.EmailAccount, error) {
	var account model.EmailAccount
	err := r.db.WithContext(ctx).
		Preload("ProviderRef").
		Preload("AdapterRef").
		Where("uid = ?", uid).
		First(&account).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &account, nil
}

// FindByIDWithRelations 根据 ID 查找账户并预加载 Provider 和 Adapter 关联
func (r *accountRepository) FindByIDWithRelations(ctx context.Context, id int64) (*model.EmailAccount, error) {
	var account model.EmailAccount
	err := r.db.WithContext(ctx).
		Preload("ProviderRef").
		Preload("AdapterRef").
		First(&account, id).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &account, nil
}

// ListWithRelations 获取账户列表并预加载 Provider 和 Adapter 关联
// 注意：不包括子账户（parent_account_uid 不为空的账户）
func (r *accountRepository) ListWithRelations(ctx context.Context, offset, limit int) ([]*model.EmailAccount, int64, error) {
	var accounts []*model.EmailAccount
	var total int64

	// 获取总数（不包括软删除的，不包括子账户）
	if err := r.db.WithContext(ctx).Model(&model.EmailAccount{}).
		Where("parent_account_uid IS NULL").
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取列表并预加载关联（不包括子账户）
	err := r.db.WithContext(ctx).
		Preload("ProviderRef").
		Preload("AdapterRef").
		Where("parent_account_uid IS NULL").
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&accounts).Error

	return accounts, total, err
}

// FindByProviderID 根据 Provider ID 查找账户列表
func (r *accountRepository) FindByProviderID(ctx context.Context, providerID int64) ([]*model.EmailAccount, error) {
	var accounts []*model.EmailAccount
	err := r.db.WithContext(ctx).
		Where("provider_id = ?", providerID).
		Order("created_at DESC").
		Find(&accounts).Error
	return accounts, err
}

// FindByAdapterID 根据 Adapter ID 查找账户列表
func (r *accountRepository) FindByAdapterID(ctx context.Context, adapterID int64) ([]*model.EmailAccount, error) {
	var accounts []*model.EmailAccount
	err := r.db.WithContext(ctx).
		Where("adapter_id = ?", adapterID).
		Order("created_at DESC").
		Find(&accounts).Error
	return accounts, err
}

// FindByProviderIDs 根据多个 Provider ID 查找账户列表（分页）
func (r *accountRepository) FindByProviderIDs(ctx context.Context, providerIDs []int64, page, pageSize int) ([]*model.EmailAccount, int64, error) {
	if len(providerIDs) == 0 {
		return []*model.EmailAccount{}, 0, nil
	}

	var accounts []*model.EmailAccount
	var total int64

	// 获取总数
	if err := r.db.WithContext(ctx).Model(&model.EmailAccount{}).
		Where("provider_id IN ?", providerIDs).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}

	err := r.db.WithContext(ctx).
		Preload("ProviderRef").
		Preload("AdapterRef").
		Where("provider_id IN ?", providerIDs).
		Offset(offset).
		Limit(pageSize).
		Order("created_at DESC").
		Find(&accounts).Error

	return accounts, total, err
}

// FindByParentAccountUID 根据父账户 UID 查找子邮箱账户列表
func (r *accountRepository) FindByParentAccountUID(ctx context.Context, parentUID string) ([]*model.EmailAccount, error) {
	var accounts []*model.EmailAccount
	err := r.db.WithContext(ctx).
		Where("parent_account_uid = ?", parentUID).
		Order("created_at DESC").
		Find(&accounts).Error
	return accounts, err
}

// ChildAccountListFilter 子邮箱分页查询条件
type ChildAccountListFilter struct {
	ParentUID string // 父账户 UID（必填）
	Include   string // active | orphaned | all
	Email     string // 邮箱模糊搜索（可选）
	Page      int    // 页码，从 1 起
	PageSize  int    // 每页数量
}

// FindChildrenByParent 分页查询父账户下的子邮箱
func (r *accountRepository) FindChildrenByParent(ctx context.Context, filter *ChildAccountListFilter) ([]*model.EmailAccount, int64, error) {
	if filter == nil || filter.ParentUID == "" {
		return []*model.EmailAccount{}, 0, nil
	}

	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	query := r.db.WithContext(ctx).Model(&model.EmailAccount{}).
		Where("parent_account_uid = ?", filter.ParentUID)

	switch strings.ToLower(strings.TrimSpace(filter.Include)) {
	case "orphaned":
		query = query.Where("status = ? AND disable_reason = ?",
			model.AccountStatusDisabled, model.DisableReasonRemoteMailboxDeleted)
	case "all":
		// 不过滤状态
	default: // active
		query = query.Where("status = ?", model.AccountStatusActive)
	}

	if kw := strings.TrimSpace(filter.Email); kw != "" {
		// 转义 ILIKE 通配符，避免用户输入 %/_ 扩大匹配面
		escaped := strings.ReplaceAll(kw, `\`, `\\`)
		escaped = strings.ReplaceAll(escaped, `%`, `\%`)
		escaped = strings.ReplaceAll(escaped, `_`, `\_`)
		query = query.Where("email ILIKE ? ESCAPE '\\'", "%"+escaped+"%")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var accounts []*model.EmailAccount
	offset := (page - 1) * pageSize
	err := query.Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&accounts).Error
	if err != nil {
		return nil, 0, err
	}
	return accounts, total, nil
}

// FindByDomain 按域名查找账户（用于 Webhook Admin 模式）
// 查找 EncryptedCredentials 中包含指定域名的账户
// 主要用于 Webhook 接收时，根据收件人域名匹配 Admin 模式的账户
func (r *accountRepository) FindByDomain(ctx context.Context, domain string) ([]*model.EmailAccount, error) {
	var accounts []*model.EmailAccount

	// 使用 PostgreSQL 的 JSON 操作符查询
	// 查找 domains 字段包含指定域名的账户
	// 注意：domains 字段格式为 "example.com, test.org"（逗号分隔）
	err := r.db.WithContext(ctx).
		Where("encrypted_credentials LIKE ?", "%"+domain+"%").
		Where("deleted_at IS NULL").
		Find(&accounts).Error

	if err != nil {
		return nil, err
	}

	return accounts, nil
}
