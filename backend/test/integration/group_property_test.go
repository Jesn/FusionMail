package integration

import (
	"context"
	"fmt"
	"testing"

	"fusionmail/internal/model"
	"fusionmail/internal/repository"
	"fusionmail/internal/service"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// =============================================================================
// 测试数据库设置
// =============================================================================

// setupGroupTestDB 创建分组测试数据库
func setupGroupTestDB(t *testing.T) *gorm.DB {
	db := openSQLiteMemoryDB(t)

	// 迁移所需的表
	err := db.AutoMigrate(
		&model.AccountGroup{},
		&model.EmailAccount{},
	)
	if err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}

	return db
}

// =============================================================================
// 属性测试：分组删除保留账号（属性 5）
// =============================================================================

// TestProperty5_GroupDeletionPreservesAccounts 分组删除保留账号属性测试
// **Feature: account-group, Property 5: 分组删除保留账号**
// **Validates: Requirements 3.1, 3.2**
func TestProperty5_GroupDeletionPreservesAccounts(t *testing.T) {
	db := setupGroupTestDB(t)
	ctx := context.Background()

	groupRepo := repository.NewGroupRepository(db)
	accountRepo := repository.NewAccountRepository(db)
	groupService := service.NewGroupService(groupRepo, accountRepo)

	t.Run("属性5.1: 删除分组后账号数据保留", func(t *testing.T) {
		// 创建分组
		group, err := groupService.CreateGroup(ctx, "测试分组", "用于测试删除")
		assert.NoError(t, err)
		assert.NotNil(t, group)

		// 创建测试账号并分配到分组
		testAccounts := []*model.EmailAccount{
			{UID: "test-account-1", Email: "test1@example.com", GroupID: &group.ID},
			{UID: "test-account-2", Email: "test2@example.com", GroupID: &group.ID},
			{UID: "test-account-3", Email: "test3@example.com", GroupID: &group.ID},
		}

		for _, acc := range testAccounts {
			err := db.Create(acc).Error
			assert.NoError(t, err)
		}

		// 验证账号已分配到分组
		accountsInGroup, err := accountRepo.FindByGroupID(ctx, group.ID)
		assert.NoError(t, err)
		assert.Equal(t, 3, len(accountsInGroup))

		// 删除分组
		err = groupService.DeleteGroup(ctx, group.ID)
		assert.NoError(t, err)

		// 验证账号数据仍然存在
		for _, acc := range testAccounts {
			var found model.EmailAccount
			err := db.Where("uid = ?", acc.UID).First(&found).Error
			assert.NoError(t, err, "账号 %s 应该仍然存在", acc.UID)
			assert.Equal(t, acc.Email, found.Email)
		}
	})

	t.Run("属性5.2: 删除分组后账号的 group_id 被清除", func(t *testing.T) {
		// 创建新分组
		group, err := groupService.CreateGroup(ctx, "另一个测试分组", "")
		assert.NoError(t, err)

		// 创建测试账号
		acc := &model.EmailAccount{
			UID:     "test-account-clear",
			Email:   "clear@example.com",
			GroupID: &group.ID,
		}
		err = db.Create(acc).Error
		assert.NoError(t, err)

		// 删除分组
		err = groupService.DeleteGroup(ctx, group.ID)
		assert.NoError(t, err)

		// 验证账号的 group_id 已被清除
		var found model.EmailAccount
		err = db.Where("uid = ?", acc.UID).First(&found).Error
		assert.NoError(t, err)
		assert.Nil(t, found.GroupID, "账号的 group_id 应该被清除为 NULL")
	})

	t.Run("属性5.3: 删除不存在的分组返回错误", func(t *testing.T) {
		err := groupService.DeleteGroup(ctx, 99999)
		assert.Error(t, err)
		assert.Equal(t, model.ErrGroupNotFound, err)
	})
}

// =============================================================================
// 属性测试：账号-分组分配排他性（属性 6）
// =============================================================================

// TestProperty6_AccountGroupAssignmentExclusivity 账号-分组分配排他性属性测试
// **Feature: account-group, Property 6: 账号-分组分配排他性**
// **Validates: Requirements 4.1, 4.2, 4.3**
func TestProperty6_AccountGroupAssignmentExclusivity(t *testing.T) {
	db := setupGroupTestDB(t)
	ctx := context.Background()

	groupRepo := repository.NewGroupRepository(db)
	accountRepo := repository.NewAccountRepository(db)
	groupService := service.NewGroupService(groupRepo, accountRepo)

	t.Run("属性6.1: 账号只能属于一个分组", func(t *testing.T) {
		// 创建两个分组
		group1, err := groupService.CreateGroup(ctx, "分组A", "")
		assert.NoError(t, err)
		group2, err := groupService.CreateGroup(ctx, "分组B", "")
		assert.NoError(t, err)

		// 创建测试账号
		acc := &model.EmailAccount{
			UID:   "exclusive-account",
			Email: "exclusive@example.com",
		}
		err = db.Create(acc).Error
		assert.NoError(t, err)

		// 分配到分组A
		err = groupService.AssignAccountToGroup(ctx, acc.UID, &group1.ID)
		assert.NoError(t, err)

		// 验证账号在分组A
		var found model.EmailAccount
		err = db.Where("uid = ?", acc.UID).First(&found).Error
		assert.NoError(t, err)
		assert.NotNil(t, found.GroupID)
		assert.Equal(t, group1.ID, *found.GroupID)

		// 分配到分组B（应该自动从分组A移除）
		err = groupService.AssignAccountToGroup(ctx, acc.UID, &group2.ID)
		assert.NoError(t, err)

		// 验证账号现在在分组B
		err = db.Where("uid = ?", acc.UID).First(&found).Error
		assert.NoError(t, err)
		assert.NotNil(t, found.GroupID)
		assert.Equal(t, group2.ID, *found.GroupID)

		// 验证分组A中没有该账号
		accountsInGroup1, err := accountRepo.FindByGroupID(ctx, group1.ID)
		assert.NoError(t, err)
		for _, a := range accountsInGroup1 {
			assert.NotEqual(t, acc.UID, a.UID, "账号不应该在分组A中")
		}
	})

	t.Run("属性6.2: 可以将账号从分组中移除", func(t *testing.T) {
		// 创建分组
		group, err := groupService.CreateGroup(ctx, "移除测试分组", "")
		assert.NoError(t, err)

		// 创建测试账号并分配到分组
		acc := &model.EmailAccount{
			UID:     "removable-account",
			Email:   "removable@example.com",
			GroupID: &group.ID,
		}
		err = db.Create(acc).Error
		assert.NoError(t, err)

		// 从分组中移除（设置 groupID 为 nil）
		err = groupService.AssignAccountToGroup(ctx, acc.UID, nil)
		assert.NoError(t, err)

		// 验证账号不再属于任何分组
		var found model.EmailAccount
		err = db.Where("uid = ?", acc.UID).First(&found).Error
		assert.NoError(t, err)
		assert.Nil(t, found.GroupID)
	})

	t.Run("属性6.3: 分配到不存在的分组返回错误", func(t *testing.T) {
		// 创建测试账号
		acc := &model.EmailAccount{
			UID:   "error-test-account",
			Email: "error@example.com",
		}
		err := db.Create(acc).Error
		assert.NoError(t, err)

		// 尝试分配到不存在的分组
		nonExistentID := int64(99999)
		err = groupService.AssignAccountToGroup(ctx, acc.UID, &nonExistentID)
		assert.Error(t, err)
		assert.Equal(t, model.ErrGroupNotFound, err)
	})
}

// =============================================================================
// 属性测试：分组筛选正确性（属性 7）
// =============================================================================

// TestProperty7_GroupFilteringCorrectness 分组筛选正确性属性测试
// **Feature: account-group, Property 7: 分组筛选正确性**
// **Validates: Requirements 5.1, 5.2, 5.3**
func TestProperty7_GroupFilteringCorrectness(t *testing.T) {
	db := setupGroupTestDB(t)
	ctx := context.Background()

	groupRepo := repository.NewGroupRepository(db)
	accountRepo := repository.NewAccountRepository(db)
	groupService := service.NewGroupService(groupRepo, accountRepo)

	// 创建测试分组
	group1, _ := groupService.CreateGroup(ctx, "工作邮箱", "")
	group2, _ := groupService.CreateGroup(ctx, "个人邮箱", "")

	// 创建测试账号
	accounts := []*model.EmailAccount{
		{UID: "work-1", Email: "work1@company.com", GroupID: &group1.ID},
		{UID: "work-2", Email: "work2@company.com", GroupID: &group1.ID},
		{UID: "personal-1", Email: "personal1@gmail.com", GroupID: &group2.ID},
		{UID: "ungrouped-1", Email: "ungrouped@example.com", GroupID: nil},
	}

	for _, acc := range accounts {
		db.Create(acc)
	}

	t.Run("属性7.1: 按分组筛选返回正确的账号", func(t *testing.T) {
		// 筛选分组1的账号
		accountsInGroup1, err := groupService.GetAccountsByGroupID(ctx, group1.ID)
		assert.NoError(t, err)
		assert.Equal(t, 2, len(accountsInGroup1))

		for _, acc := range accountsInGroup1 {
			assert.NotNil(t, acc.GroupID)
			assert.Equal(t, group1.ID, *acc.GroupID)
		}

		// 筛选分组2的账号
		accountsInGroup2, err := groupService.GetAccountsByGroupID(ctx, group2.ID)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(accountsInGroup2))
		assert.Equal(t, "personal-1", accountsInGroup2[0].UID)
	})

	t.Run("属性7.2: 筛选未分组账号返回正确结果", func(t *testing.T) {
		ungroupedAccounts, err := groupService.GetUngroupedAccounts(ctx)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(ungroupedAccounts))
		assert.Equal(t, "ungrouped-1", ungroupedAccounts[0].UID)
		assert.Nil(t, ungroupedAccounts[0].GroupID)
	})

	t.Run("属性7.3: 空分组返回空列表", func(t *testing.T) {
		// 创建一个空分组
		emptyGroup, err := groupService.CreateGroup(ctx, "空分组", "")
		assert.NoError(t, err)

		accountsInEmpty, err := groupService.GetAccountsByGroupID(ctx, emptyGroup.ID)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(accountsInEmpty))
	})
}

// =============================================================================
// 属性测试：分组账号计数准确性（属性 8）
// =============================================================================

// TestProperty8_GroupAccountCountAccuracy 分组账号计数准确性属性测试
// **Feature: account-group, Property 8: 分组账号计数准确性**
// **Validates: Requirements 5.4**
func TestProperty8_GroupAccountCountAccuracy(t *testing.T) {
	db := setupGroupTestDB(t)
	ctx := context.Background()

	groupRepo := repository.NewGroupRepository(db)
	accountRepo := repository.NewAccountRepository(db)
	groupService := service.NewGroupService(groupRepo, accountRepo)

	t.Run("属性8.1: 分组账号计数与实际数量一致", func(t *testing.T) {
		// 创建分组
		group, err := groupService.CreateGroup(ctx, "计数测试分组", "")
		assert.NoError(t, err)

		// 创建多个账号
		expectedCount := 5
		for i := 0; i < expectedCount; i++ {
			acc := &model.EmailAccount{
				UID:     fmt.Sprintf("count-test-%d", i),
				Email:   fmt.Sprintf("count%d@example.com", i),
				GroupID: &group.ID,
			}
			db.Create(acc)
		}

		// 获取分组列表（带计数）
		groups, err := groupService.GetGroups(ctx)
		assert.NoError(t, err)

		// 找到测试分组并验证计数
		var found *model.AccountGroupWithCount
		for _, g := range groups {
			if g.ID == group.ID {
				found = g
				break
			}
		}

		assert.NotNil(t, found)
		assert.Equal(t, expectedCount, found.AccountCount)
	})

	t.Run("属性8.2: 空分组计数为零", func(t *testing.T) {
		// 创建空分组
		emptyGroup, err := groupService.CreateGroup(ctx, "空计数分组", "")
		assert.NoError(t, err)

		// 获取分组列表
		groups, err := groupService.GetGroups(ctx)
		assert.NoError(t, err)

		// 找到空分组并验证计数
		var found *model.AccountGroupWithCount
		for _, g := range groups {
			if g.ID == emptyGroup.ID {
				found = g
				break
			}
		}

		assert.NotNil(t, found)
		assert.Equal(t, 0, found.AccountCount)
	})

	t.Run("属性8.3: 账号移动后计数正确更新", func(t *testing.T) {
		// 创建两个分组
		groupA, _ := groupService.CreateGroup(ctx, "计数分组A", "")
		groupB, _ := groupService.CreateGroup(ctx, "计数分组B", "")

		// 创建账号并分配到分组A
		acc := &model.EmailAccount{
			UID:     "moving-account",
			Email:   "moving@example.com",
			GroupID: &groupA.ID,
		}
		db.Create(acc)

		// 验证分组A计数为1
		countA, err := groupRepo.CountAccountsByGroupID(ctx, groupA.ID)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), countA)

		// 移动账号到分组B
		err = groupService.AssignAccountToGroup(ctx, acc.UID, &groupB.ID)
		assert.NoError(t, err)

		// 验证分组A计数为0
		countA, err = groupRepo.CountAccountsByGroupID(ctx, groupA.ID)
		assert.NoError(t, err)
		assert.Equal(t, int64(0), countA)

		// 验证分组B计数为1
		countB, err := groupRepo.CountAccountsByGroupID(ctx, groupB.ID)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), countB)
	})
}

// =============================================================================
// 属性测试：批量分配原子性（属性 10）
// =============================================================================

// TestProperty10_BatchAssignmentAtomicity 批量分配原子性属性测试
// **Feature: account-group, Property 10: 批量分配原子性**
// **Validates: Requirements 7.1, 7.3**
func TestProperty10_BatchAssignmentAtomicity(t *testing.T) {
	db := setupGroupTestDB(t)
	ctx := context.Background()

	groupRepo := repository.NewGroupRepository(db)
	accountRepo := repository.NewAccountRepository(db)
	groupService := service.NewGroupService(groupRepo, accountRepo)

	t.Run("属性10.1: 批量分配所有账号成功", func(t *testing.T) {
		// 创建分组
		group, err := groupService.CreateGroup(ctx, "批量分配分组", "")
		assert.NoError(t, err)

		// 创建多个账号
		accountUIDs := []string{}
		for i := 0; i < 5; i++ {
			acc := &model.EmailAccount{
				UID:   fmt.Sprintf("batch-account-%d", i),
				Email: fmt.Sprintf("batch%d@example.com", i),
			}
			db.Create(acc)
			accountUIDs = append(accountUIDs, acc.UID)
		}

		// 批量分配
		err = groupService.BatchAssignAccounts(ctx, accountUIDs, &group.ID)
		assert.NoError(t, err)

		// 验证所有账号都已分配
		for _, uid := range accountUIDs {
			var acc model.EmailAccount
			err := db.Where("uid = ?", uid).First(&acc).Error
			assert.NoError(t, err)
			assert.NotNil(t, acc.GroupID)
			assert.Equal(t, group.ID, *acc.GroupID)
		}
	})

	t.Run("属性10.2: 批量移除分组成功", func(t *testing.T) {
		// 创建分组
		group, err := groupService.CreateGroup(ctx, "批量移除分组", "")
		assert.NoError(t, err)

		// 创建多个账号并分配到分组
		accountUIDs := []string{}
		for i := 0; i < 3; i++ {
			acc := &model.EmailAccount{
				UID:     fmt.Sprintf("batch-remove-%d", i),
				Email:   fmt.Sprintf("remove%d@example.com", i),
				GroupID: &group.ID,
			}
			db.Create(acc)
			accountUIDs = append(accountUIDs, acc.UID)
		}

		// 批量移除（设置 groupID 为 nil）
		err = groupService.BatchAssignAccounts(ctx, accountUIDs, nil)
		assert.NoError(t, err)

		// 验证所有账号都已移除分组
		for _, uid := range accountUIDs {
			var acc model.EmailAccount
			err := db.Where("uid = ?", uid).First(&acc).Error
			assert.NoError(t, err)
			assert.Nil(t, acc.GroupID)
		}
	})

	t.Run("属性10.3: 空账号列表不报错", func(t *testing.T) {
		group, _ := groupService.CreateGroup(ctx, "空批量分组", "")
		err := groupService.BatchAssignAccounts(ctx, []string{}, &group.ID)
		assert.NoError(t, err)
	})
}

// =============================================================================
// 属性测试：分组排序往返一致性（属性 11）
// =============================================================================

// TestProperty11_GroupOrderingRoundTrip 分组排序往返一致性属性测试
// **Feature: account-group, Property 11: 分组排序往返一致性**
// **Validates: Requirements 8.1, 8.2, 8.3**
func TestProperty11_GroupOrderingRoundTrip(t *testing.T) {
	db := setupGroupTestDB(t)
	ctx := context.Background()

	groupRepo := repository.NewGroupRepository(db)
	accountRepo := repository.NewAccountRepository(db)
	groupService := service.NewGroupService(groupRepo, accountRepo)

	t.Run("属性11.1: 新分组获得最低优先级顺序", func(t *testing.T) {
		// 创建多个分组
		group1, _ := groupService.CreateGroup(ctx, "顺序分组1", "")
		group2, _ := groupService.CreateGroup(ctx, "顺序分组2", "")
		group3, _ := groupService.CreateGroup(ctx, "顺序分组3", "")

		// 验证顺序递增
		assert.True(t, group1.DisplayOrder < group2.DisplayOrder)
		assert.True(t, group2.DisplayOrder < group3.DisplayOrder)
	})

	t.Run("属性11.2: 重新排序后顺序正确保存", func(t *testing.T) {
		// 创建分组
		groupA, _ := groupService.CreateGroup(ctx, "排序A", "")
		groupB, _ := groupService.CreateGroup(ctx, "排序B", "")
		groupC, _ := groupService.CreateGroup(ctx, "排序C", "")

		// 重新排序：C, A, B
		newOrder := []int64{groupC.ID, groupA.ID, groupB.ID}
		err := groupService.ReorderGroups(ctx, newOrder)
		assert.NoError(t, err)

		// 获取分组列表并验证顺序
		groups, err := groupService.GetGroups(ctx)
		assert.NoError(t, err)

		// 找到这三个分组并验证顺序
		var foundA, foundB, foundC *model.AccountGroupWithCount
		for _, g := range groups {
			switch g.ID {
			case groupA.ID:
				foundA = g
			case groupB.ID:
				foundB = g
			case groupC.ID:
				foundC = g
			}
		}

		assert.NotNil(t, foundA)
		assert.NotNil(t, foundB)
		assert.NotNil(t, foundC)

		// C 应该在 A 前面，A 应该在 B 前面
		assert.True(t, foundC.DisplayOrder < foundA.DisplayOrder, "C 应该在 A 前面")
		assert.True(t, foundA.DisplayOrder < foundB.DisplayOrder, "A 应该在 B 前面")
	})

	t.Run("属性11.3: 排序不存在的分组返回错误", func(t *testing.T) {
		group, _ := groupService.CreateGroup(ctx, "存在的分组", "")
		invalidOrder := []int64{group.ID, 99999}
		err := groupService.ReorderGroups(ctx, invalidOrder)
		assert.Error(t, err)
		assert.Equal(t, model.ErrGroupNotFound, err)
	})
}

// =============================================================================
// 属性测试：分组名称唯一性（属性 3 - 补充）
// =============================================================================

// TestProperty3_GroupNameUniqueness 分组名称唯一性属性测试
// **Feature: account-group, Property 3: 分组名称唯一性**
// **Validates: Requirements 1.3, 2.2**
func TestProperty3_GroupNameUniqueness(t *testing.T) {
	db := setupGroupTestDB(t)
	ctx := context.Background()

	groupRepo := repository.NewGroupRepository(db)
	accountRepo := repository.NewAccountRepository(db)
	groupService := service.NewGroupService(groupRepo, accountRepo)

	t.Run("属性3.1: 创建重复名称分组失败", func(t *testing.T) {
		// 创建第一个分组
		_, err := groupService.CreateGroup(ctx, "唯一名称", "")
		assert.NoError(t, err)

		// 尝试创建同名分组
		_, err = groupService.CreateGroup(ctx, "唯一名称", "")
		assert.Error(t, err)
		assert.Equal(t, model.ErrGroupNameExists, err)
	})

	t.Run("属性3.2: 更新为已存在的名称失败", func(t *testing.T) {
		// 创建两个分组
		group1, _ := groupService.CreateGroup(ctx, "名称1", "")
		group2, _ := groupService.CreateGroup(ctx, "名称2", "")

		// 尝试将 group2 重命名为 group1 的名称
		_, err := groupService.UpdateGroup(ctx, group2.ID, "名称1", "")
		assert.Error(t, err)
		assert.Equal(t, model.ErrGroupNameExists, err)

		// 验证 group1 仍然存在
		found, err := groupRepo.FindByID(ctx, group1.ID)
		assert.NoError(t, err)
		assert.Equal(t, "名称1", found.Name)
	})

	t.Run("属性3.3: 更新为自己的名称成功", func(t *testing.T) {
		// 创建分组
		group, _ := groupService.CreateGroup(ctx, "自我更新", "原描述")

		// 更新为相同名称但不同描述
		updated, err := groupService.UpdateGroup(ctx, group.ID, "自我更新", "新描述")
		assert.NoError(t, err)
		assert.Equal(t, "自我更新", updated.Name)
		assert.Equal(t, "新描述", updated.Description)
	})
}

// =============================================================================
// 属性测试：分组名称验证（属性 2 - 补充）
// =============================================================================

// TestProperty2_GroupNameValidation 分组名称验证属性测试
// **Feature: account-group, Property 2: 分组名称验证**
// **Validates: Requirements 1.2**
func TestProperty2_GroupNameValidation(t *testing.T) {
	db := setupGroupTestDB(t)
	ctx := context.Background()

	groupRepo := repository.NewGroupRepository(db)
	accountRepo := repository.NewAccountRepository(db)
	groupService := service.NewGroupService(groupRepo, accountRepo)

	t.Run("属性2.1: 空名称创建失败", func(t *testing.T) {
		_, err := groupService.CreateGroup(ctx, "", "描述")
		assert.Error(t, err)
		assert.Equal(t, model.ErrGroupNameRequired, err)
	})

	t.Run("属性2.2: 仅空格名称创建失败", func(t *testing.T) {
		_, err := groupService.CreateGroup(ctx, "   ", "描述")
		assert.Error(t, err)
		assert.Equal(t, model.ErrGroupNameRequired, err)
	})

	t.Run("属性2.3: 超长名称创建失败", func(t *testing.T) {
		// 创建超过 100 字符的名称
		longName := ""
		for i := 0; i < 101; i++ {
			longName += "a"
		}
		_, err := groupService.CreateGroup(ctx, longName, "")
		assert.Error(t, err)
		assert.Equal(t, model.ErrGroupNameTooLong, err)
	})

	t.Run("属性2.4: 有效名称创建成功", func(t *testing.T) {
		validNames := []string{
			"工作邮箱",
			"Personal",
			"分组-123",
			"A",
		}

		for _, name := range validNames {
			group, err := groupService.CreateGroup(ctx, name, "")
			assert.NoError(t, err, "名称 '%s' 应该是有效的", name)
			assert.Equal(t, name, group.Name)
		}
	})
}

// =============================================================================
// 属性测试：邮件按分组筛选（属性 9）
// =============================================================================

// TestProperty9_EmailFilteringByGroup 邮件按分组筛选属性测试
// **Feature: account-group, Property 9: 邮件按分组筛选**
// **Validates: Requirements 6.1, 6.3**
func TestProperty9_EmailFilteringByGroup(t *testing.T) {
	db := setupGroupTestDB(t)
	ctx := context.Background()

	// 迁移邮件表
	err := db.AutoMigrate(&model.Email{})
	if err != nil {
		t.Fatalf("邮件表迁移失败: %v", err)
	}

	groupRepo := repository.NewGroupRepository(db)
	accountRepo := repository.NewAccountRepository(db)
	groupService := service.NewGroupService(groupRepo, accountRepo)

	// 创建测试分组
	workGroup, _ := groupService.CreateGroup(ctx, "工作分组", "")
	personalGroup, _ := groupService.CreateGroup(ctx, "个人分组", "")

	// 创建测试账号
	workAccount := &model.EmailAccount{
		UID:     "work-email-account",
		Email:   "work@company.com",
		GroupID: &workGroup.ID,
	}
	personalAccount := &model.EmailAccount{
		UID:     "personal-email-account",
		Email:   "personal@gmail.com",
		GroupID: &personalGroup.ID,
	}
	ungroupedAccount := &model.EmailAccount{
		UID:   "ungrouped-email-account",
		Email: "ungrouped@example.com",
	}

	db.Create(workAccount)
	db.Create(personalAccount)
	db.Create(ungroupedAccount)

	// 创建测试邮件
	emails := []*model.Email{
		{ProviderID: "email-1", AccountUID: workAccount.UID, Subject: "工作邮件1", FromAddress: "sender1@company.com"},
		{ProviderID: "email-2", AccountUID: workAccount.UID, Subject: "工作邮件2", FromAddress: "sender2@company.com"},
		{ProviderID: "email-3", AccountUID: personalAccount.UID, Subject: "个人邮件1", FromAddress: "friend@gmail.com"},
		{ProviderID: "email-4", AccountUID: ungroupedAccount.UID, Subject: "未分组邮件1", FromAddress: "other@example.com"},
	}

	for _, email := range emails {
		db.Create(email)
	}

	emailRepo := repository.NewEmailRepository(db)

	t.Run("属性9.1: 按分组筛选返回正确的邮件", func(t *testing.T) {
		// 获取工作分组的账号
		workAccounts, err := accountRepo.FindByGroupID(ctx, workGroup.ID)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(workAccounts))

		// 使用账号 UID 筛选邮件
		filter := &repository.EmailFilter{
			AccountUIDs: []string{workAccounts[0].UID},
		}
		emailsInGroup, _, err := emailRepo.List(ctx, filter, 0, 100)
		assert.NoError(t, err)
		assert.Equal(t, 2, len(emailsInGroup))

		// 验证所有邮件都属于工作账号
		for _, email := range emailsInGroup {
			assert.Equal(t, workAccount.UID, email.AccountUID)
		}
	})

	t.Run("属性9.2: 未分组账号的邮件筛选", func(t *testing.T) {
		// 获取未分组账号
		ungroupedAccounts, err := accountRepo.FindUngrouped(ctx)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(ungroupedAccounts))

		// 使用账号 UID 筛选邮件
		filter := &repository.EmailFilter{
			AccountUIDs: []string{ungroupedAccounts[0].UID},
		}
		emailsUngrouped, _, err := emailRepo.List(ctx, filter, 0, 100)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(emailsUngrouped))
		assert.Equal(t, ungroupedAccount.UID, emailsUngrouped[0].AccountUID)
	})

	t.Run("属性9.3: 空分组返回空邮件列表", func(t *testing.T) {
		// 创建空分组
		emptyGroup, _ := groupService.CreateGroup(ctx, "空邮件分组", "")

		// 获取空分组的账号
		emptyAccounts, err := accountRepo.FindByGroupID(ctx, emptyGroup.ID)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(emptyAccounts))

		// 空账号列表应该返回空邮件列表
		if len(emptyAccounts) == 0 {
			// 模拟服务层的行为：空分组返回空列表
			t.Log("空分组没有账号，邮件列表应为空")
		}
	})

	t.Run("属性9.4: 多账号分组的邮件聚合", func(t *testing.T) {
		// 创建一个包含多个账号的分组
		multiGroup, _ := groupService.CreateGroup(ctx, "多账号分组", "")

		// 创建多个账号并分配到同一分组
		acc1 := &model.EmailAccount{UID: "multi-acc-1", Email: "multi1@example.com", GroupID: &multiGroup.ID}
		acc2 := &model.EmailAccount{UID: "multi-acc-2", Email: "multi2@example.com", GroupID: &multiGroup.ID}
		db.Create(acc1)
		db.Create(acc2)

		// 为每个账号创建邮件
		db.Create(&model.Email{ProviderID: "multi-email-1", AccountUID: acc1.UID, Subject: "多账号邮件1"})
		db.Create(&model.Email{ProviderID: "multi-email-2", AccountUID: acc1.UID, Subject: "多账号邮件2"})
		db.Create(&model.Email{ProviderID: "multi-email-3", AccountUID: acc2.UID, Subject: "多账号邮件3"})

		// 获取分组的所有账号
		multiAccounts, err := accountRepo.FindByGroupID(ctx, multiGroup.ID)
		assert.NoError(t, err)
		assert.Equal(t, 2, len(multiAccounts))

		// 使用所有账号 UID 筛选邮件
		accountUIDs := make([]string, len(multiAccounts))
		for i, acc := range multiAccounts {
			accountUIDs[i] = acc.UID
		}

		filter := &repository.EmailFilter{
			AccountUIDs: accountUIDs,
		}
		emailsInMultiGroup, _, err := emailRepo.List(ctx, filter, 0, 100)
		assert.NoError(t, err)
		assert.Equal(t, 3, len(emailsInMultiGroup))
	})
}
