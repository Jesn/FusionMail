package service

import (
	"context"
	"fusionmail/internal/dto"
	"fusionmail/internal/model"
	"fusionmail/internal/repository"
	"fusionmail/pkg/logger"
)

// 模块日志记录器
var groupServiceLog = logger.NewWithModule("GroupService")

// GroupService 分组服务接口
type GroupService interface {
	// 分组 CRUD
	CreateGroup(ctx context.Context, name, description string) (*model.AccountGroup, error)
	UpdateGroup(ctx context.Context, id int64, name, description string) (*model.AccountGroup, error)
	DeleteGroup(ctx context.Context, id int64) error
	GetGroups(ctx context.Context) ([]*model.AccountGroupWithCount, error)
	GetGroupsWithStats(ctx context.Context) (*model.GroupListResponse, error) // 新增：带统计信息的分组列表
	GetGroupByID(ctx context.Context, id int64) (*model.AccountGroupWithAccounts, error)

	// 账号分配
	AssignAccountToGroup(ctx context.Context, accountUID string, groupID *int64) error
	BatchAssignAccounts(ctx context.Context, accountUIDs []string, groupID *int64) error
	GetAccountsByGroupID(ctx context.Context, groupID int64) ([]*model.EmailAccount, error)
	GetUngroupedAccounts(ctx context.Context) ([]*model.EmailAccount, error)

	// 排序
	ReorderGroups(ctx context.Context, groupIDs []int64) error
}

// groupService 分组服务实现
type groupService struct {
	groupRepo   repository.GroupRepository
	accountRepo repository.AccountRepository
}

// NewGroupService 创建分组服务实例
func NewGroupService(groupRepo repository.GroupRepository, accountRepo repository.AccountRepository) GroupService {
	return &groupService{
		groupRepo:   groupRepo,
		accountRepo: accountRepo,
	}
}

// CreateGroup 创建分组
func (s *groupService) CreateGroup(ctx context.Context, name, description string) (*model.AccountGroup, error) {
	// 检查名称唯一性
	exists, err := s.groupRepo.ExistsByName(ctx, name, 0)
	if err != nil {
		groupServiceLog.Error("检查分组名称失败: %v", err)
		return nil, err
	}
	if exists {
		return nil, model.ErrGroupNameExists
	}

	// 获取最大显示顺序
	maxOrder, err := s.groupRepo.GetMaxDisplayOrder(ctx)
	if err != nil {
		groupServiceLog.Error("获取最大显示顺序失败: %v", err)
		return nil, err
	}

	group := &model.AccountGroup{
		Name:         name,
		Description:  description,
		DisplayOrder: maxOrder + 1,
	}

	if err := s.groupRepo.Create(ctx, group); err != nil {
		groupServiceLog.Error("创建分组失败: %v", err)
		return nil, err
	}

	groupServiceLog.Info("创建分组成功: id=%d, name=%s", group.ID, group.Name)
	return group, nil
}

// UpdateGroup 更新分组
func (s *groupService) UpdateGroup(ctx context.Context, id int64, name, description string) (*model.AccountGroup, error) {
	// 查找分组
	group, err := s.groupRepo.FindByID(ctx, id)
	if err != nil {
		groupServiceLog.Error("查找分组失败: %v", err)
		return nil, err
	}
	if group == nil {
		return nil, model.ErrGroupNotFound
	}

	// 检查名称唯一性（排除自己）
	exists, err := s.groupRepo.ExistsByName(ctx, name, id)
	if err != nil {
		groupServiceLog.Error("检查分组名称失败: %v", err)
		return nil, err
	}
	if exists {
		return nil, model.ErrGroupNameExists
	}

	// 更新字段
	group.Name = name
	group.Description = description

	if err := s.groupRepo.Update(ctx, group); err != nil {
		groupServiceLog.Error("更新分组失败: %v", err)
		return nil, err
	}

	groupServiceLog.Info("更新分组成功: id=%d, name=%s", group.ID, group.Name)
	return group, nil
}

// DeleteGroup 删除分组
func (s *groupService) DeleteGroup(ctx context.Context, id int64) error {
	// 查找分组
	group, err := s.groupRepo.FindByID(ctx, id)
	if err != nil {
		groupServiceLog.Error("查找分组失败: %v", err)
		return err
	}
	if group == nil {
		return model.ErrGroupNotFound
	}

	// 清除关联账号的 group_id（设为 NULL）
	if err := s.groupRepo.ClearGroupIDForAccounts(ctx, id); err != nil {
		groupServiceLog.Error("清除账号分组关联失败: %v", err)
		return err
	}

	// 删除分组
	if err := s.groupRepo.Delete(ctx, id); err != nil {
		groupServiceLog.Error("删除分组失败: %v", err)
		return err
	}

	groupServiceLog.Info("删除分组成功: id=%d, name=%s", id, group.Name)
	return nil
}

// GetGroups 获取所有分组（带账号数量）
func (s *groupService) GetGroups(ctx context.Context) ([]*model.AccountGroupWithCount, error) {
	groups, err := s.groupRepo.FindAll(ctx)
	if err != nil {
		groupServiceLog.Error("获取分组列表失败: %v", err)
		return nil, err
	}

	result := make([]*model.AccountGroupWithCount, len(groups))
	for i, group := range groups {
		count, err := s.groupRepo.CountAccountsByGroupID(ctx, group.ID)
		if err != nil {
			groupServiceLog.Error("统计分组账号数量失败: groupID=%d, err=%v", group.ID, err)
			count = 0
		}
		result[i] = &model.AccountGroupWithCount{
			AccountGroup: *group,
			AccountCount: int(count),
		}
	}

	return result, nil
}

// GetGroupsWithStats 获取分组列表（带统计信息：总数、未分组数）
func (s *groupService) GetGroupsWithStats(ctx context.Context) (*model.GroupListResponse, error) {
	// 获取分组列表（带各分组账号数）
	groups, err := s.GetGroups(ctx)
	if err != nil {
		return nil, err
	}

	// 统计未分组账号数
	ungroupedAccounts, err := s.accountRepo.FindUngrouped(ctx)
	if err != nil {
		groupServiceLog.Error("获取未分组账号失败: %v", err)
		return nil, err
	}
	ungroupedCount := len(ungroupedAccounts)

	// 计算总数 = 各分组账号数之和 + 未分组数
	totalCount := ungroupedCount
	for _, g := range groups {
		totalCount += g.AccountCount
	}

	return &model.GroupListResponse{
		Groups:         groups,
		TotalCount:     totalCount,
		UngroupedCount: ungroupedCount,
	}, nil
}

// GetGroupByID 根据 ID 获取分组详情（带账号列表）
func (s *groupService) GetGroupByID(ctx context.Context, id int64) (*model.AccountGroupWithAccounts, error) {
	group, err := s.groupRepo.FindByID(ctx, id)
	if err != nil {
		groupServiceLog.Error("查找分组失败: %v", err)
		return nil, err
	}
	if group == nil {
		return nil, model.ErrGroupNotFound
	}

	accounts, err := s.accountRepo.FindByGroupID(ctx, id)
	if err != nil {
		groupServiceLog.Error("获取分组账号列表失败: %v", err)
		return nil, err
	}

	return &model.AccountGroupWithAccounts{
		AccountGroup: *group,
		Accounts:     accounts,
	}, nil
}

// AssignAccountToGroup 将账号分配到分组
func (s *groupService) AssignAccountToGroup(ctx context.Context, accountUID string, groupID *int64) error {
	// 验证账号存在
	account, err := s.accountRepo.FindByUID(ctx, accountUID)
	if err != nil {
		groupServiceLog.Error("查找账号失败: %v", err)
		return err
	}
	if account == nil {
		return dto.NewAPIError(dto.ErrAccountNotFound)
	}

	// 如果指定了分组，验证分组存在
	if groupID != nil {
		group, err := s.groupRepo.FindByID(ctx, *groupID)
		if err != nil {
			groupServiceLog.Error("查找分组失败: %v", err)
			return err
		}
		if group == nil {
			return model.ErrGroupNotFound
		}
	}

	// 更新账号的分组
	if err := s.accountRepo.UpdateGroupID(ctx, accountUID, groupID); err != nil {
		groupServiceLog.Error("更新账号分组失败: %v", err)
		return err
	}

	if groupID != nil {
		groupServiceLog.Info("账号分配到分组: accountUID=%s, groupID=%d", accountUID, *groupID)
	} else {
		groupServiceLog.Info("账号移出分组: accountUID=%s", accountUID)
	}

	return nil
}

// BatchAssignAccounts 批量将账号分配到分组
func (s *groupService) BatchAssignAccounts(ctx context.Context, accountUIDs []string, groupID *int64) error {
	if len(accountUIDs) == 0 {
		return nil
	}

	// 如果指定了分组，验证分组存在
	if groupID != nil {
		group, err := s.groupRepo.FindByID(ctx, *groupID)
		if err != nil {
			groupServiceLog.Error("查找分组失败: %v", err)
			return err
		}
		if group == nil {
			return model.ErrGroupNotFound
		}
	}

	// 批量更新
	if err := s.accountRepo.BatchUpdateGroupID(ctx, accountUIDs, groupID); err != nil {
		groupServiceLog.Error("批量更新账号分组失败: %v", err)
		return err
	}

	if groupID != nil {
		groupServiceLog.Info("批量分配账号到分组: count=%d, groupID=%d", len(accountUIDs), *groupID)
	} else {
		groupServiceLog.Info("批量移出账号分组: count=%d", len(accountUIDs))
	}

	return nil
}

// GetAccountsByGroupID 获取分组中的账号列表
func (s *groupService) GetAccountsByGroupID(ctx context.Context, groupID int64) ([]*model.EmailAccount, error) {
	return s.accountRepo.FindByGroupID(ctx, groupID)
}

// GetUngroupedAccounts 获取未分组的账号列表
func (s *groupService) GetUngroupedAccounts(ctx context.Context) ([]*model.EmailAccount, error) {
	return s.accountRepo.FindUngrouped(ctx)
}

// ReorderGroups 重新排序分组
func (s *groupService) ReorderGroups(ctx context.Context, groupIDs []int64) error {
	// 验证所有分组都存在
	for _, id := range groupIDs {
		group, err := s.groupRepo.FindByID(ctx, id)
		if err != nil {
			groupServiceLog.Error("查找分组失败: %v", err)
			return err
		}
		if group == nil {
			return model.ErrGroupNotFound
		}
	}

	if err := s.groupRepo.UpdateDisplayOrders(ctx, groupIDs); err != nil {
		groupServiceLog.Error("更新分组顺序失败: %v", err)
		return err
	}

	groupServiceLog.Info("重新排序分组成功: count=%d", len(groupIDs))
	return nil
}
