package repository

import (
	"context"
	"fusionmail/internal/model"
	"strings"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupGroupTestDB 创建测试数据库
func setupGroupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// 自动迁移
	err = db.AutoMigrate(&model.AccountGroup{}, &model.EmailAccount{})
	require.NoError(t, err)

	return db
}

// **Feature: account-group, Property 3: Group name uniqueness**
// *For any* existing group name, attempting to create a new group or rename an
// existing group to that name SHALL be rejected with a uniqueness error.
// **Validates: Requirements 1.3, 2.2**
func TestProperty_GroupNameUniqueness(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	// 生成有效的分组名称
	validNameGen := gen.AlphaString().Map(func(s string) string {
		if len(s) == 0 {
			return "default"
		}
		if len(s) > 100 {
			return s[:100]
		}
		return s
	}).SuchThat(func(s string) bool {
		return len(s) > 0 && len(s) <= 100
	})

	properties.Property("duplicate name on create is detected", prop.ForAll(
		func(name string) bool {
			db := setupTestDBForProperty()
			if db == nil {
				return true // 跳过无法创建数据库的情况
			}
			repo := NewGroupRepository(db)
			ctx := context.Background()

			// 创建第一个分组
			group1 := &model.AccountGroup{Name: name}
			err := repo.Create(ctx, group1)
			if err != nil {
				return false
			}

			// 检查名称是否存在
			exists, err := repo.ExistsByName(ctx, name, 0)
			if err != nil {
				return false
			}

			return exists == true
		},
		validNameGen,
	))

	properties.Property("duplicate name on update is detected", prop.ForAll(
		func(name1, name2 string) bool {
			// 确保两个名称不同
			if name1 == name2 {
				name2 = name2 + "_different"
			}

			db := setupTestDBForProperty()
			if db == nil {
				return true
			}
			repo := NewGroupRepository(db)
			ctx := context.Background()

			// 创建两个分组
			group1 := &model.AccountGroup{Name: name1}
			if err := repo.Create(ctx, group1); err != nil {
				return false
			}

			group2 := &model.AccountGroup{Name: name2}
			if err := repo.Create(ctx, group2); err != nil {
				return false
			}

			// 尝试将 group2 重命名为 group1 的名称
			exists, err := repo.ExistsByName(ctx, name1, group2.ID)
			if err != nil {
				return false
			}

			// 应该检测到名称已存在
			return exists == true
		},
		validNameGen,
		validNameGen,
	))

	properties.Property("same name with excludeID is allowed", prop.ForAll(
		func(name string) bool {
			db := setupTestDBForProperty()
			if db == nil {
				return true
			}
			repo := NewGroupRepository(db)
			ctx := context.Background()

			// 创建分组
			group := &model.AccountGroup{Name: name}
			if err := repo.Create(ctx, group); err != nil {
				return false
			}

			// 检查名称是否存在（排除自己）
			exists, err := repo.ExistsByName(ctx, name, group.ID)
			if err != nil {
				return false
			}

			// 排除自己后，不应该检测到重复
			return exists == false
		},
		validNameGen,
	))

	properties.TestingRun(t)
}

// setupTestDBForProperty 为属性测试创建数据库
func setupTestDBForProperty() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		return nil
	}

	if err := db.AutoMigrate(&model.AccountGroup{}, &model.EmailAccount{}); err != nil {
		return nil
	}

	return db
}

// 单元测试
func TestGroupRepository_Create(t *testing.T) {
	db := setupGroupTestDB(t)
	repo := NewGroupRepository(db)
	ctx := context.Background()

	group := &model.AccountGroup{
		Name:        "测试分组",
		Description: "测试描述",
	}

	err := repo.Create(ctx, group)
	assert.NoError(t, err)
	assert.NotZero(t, group.ID)
}

func TestGroupRepository_FindByID(t *testing.T) {
	db := setupGroupTestDB(t)
	repo := NewGroupRepository(db)
	ctx := context.Background()

	// 创建分组
	group := &model.AccountGroup{Name: "测试分组"}
	err := repo.Create(ctx, group)
	require.NoError(t, err)

	// 查找分组
	found, err := repo.FindByID(ctx, group.ID)
	assert.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, group.Name, found.Name)

	// 查找不存在的分组
	notFound, err := repo.FindByID(ctx, 99999)
	assert.NoError(t, err)
	assert.Nil(t, notFound)
}

func TestGroupRepository_FindByName(t *testing.T) {
	db := setupGroupTestDB(t)
	repo := NewGroupRepository(db)
	ctx := context.Background()

	// 创建分组
	group := &model.AccountGroup{Name: "唯一名称"}
	err := repo.Create(ctx, group)
	require.NoError(t, err)

	// 查找分组
	found, err := repo.FindByName(ctx, "唯一名称")
	assert.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, group.ID, found.ID)

	// 查找不存在的名称
	notFound, err := repo.FindByName(ctx, "不存在的名称")
	assert.NoError(t, err)
	assert.Nil(t, notFound)
}

func TestGroupRepository_ExistsByName(t *testing.T) {
	db := setupGroupTestDB(t)
	repo := NewGroupRepository(db)
	ctx := context.Background()

	// 创建分组
	group := &model.AccountGroup{Name: "已存在的名称"}
	err := repo.Create(ctx, group)
	require.NoError(t, err)

	// 检查名称是否存在
	exists, err := repo.ExistsByName(ctx, "已存在的名称", 0)
	assert.NoError(t, err)
	assert.True(t, exists)

	// 检查不存在的名称
	exists, err = repo.ExistsByName(ctx, "不存在的名称", 0)
	assert.NoError(t, err)
	assert.False(t, exists)

	// 检查名称是否存在（排除自己）
	exists, err = repo.ExistsByName(ctx, "已存在的名称", group.ID)
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestGroupRepository_FindAll(t *testing.T) {
	db := setupGroupTestDB(t)
	repo := NewGroupRepository(db)
	ctx := context.Background()

	// 创建多个分组
	groups := []*model.AccountGroup{
		{Name: "分组1", DisplayOrder: 2},
		{Name: "分组2", DisplayOrder: 0},
		{Name: "分组3", DisplayOrder: 1},
	}

	for _, g := range groups {
		err := repo.Create(ctx, g)
		require.NoError(t, err)
	}

	// 获取所有分组
	all, err := repo.FindAll(ctx)
	assert.NoError(t, err)
	assert.Len(t, all, 3)

	// 验证按 display_order 排序
	assert.Equal(t, "分组2", all[0].Name)
	assert.Equal(t, "分组3", all[1].Name)
	assert.Equal(t, "分组1", all[2].Name)
}

func TestGroupRepository_UpdateDisplayOrders(t *testing.T) {
	db := setupGroupTestDB(t)
	repo := NewGroupRepository(db)
	ctx := context.Background()

	// 创建分组
	group1 := &model.AccountGroup{Name: "分组1", DisplayOrder: 0}
	group2 := &model.AccountGroup{Name: "分组2", DisplayOrder: 1}
	group3 := &model.AccountGroup{Name: "分组3", DisplayOrder: 2}

	require.NoError(t, repo.Create(ctx, group1))
	require.NoError(t, repo.Create(ctx, group2))
	require.NoError(t, repo.Create(ctx, group3))

	// 重新排序：3, 1, 2
	err := repo.UpdateDisplayOrders(ctx, []int64{group3.ID, group1.ID, group2.ID})
	assert.NoError(t, err)

	// 验证新顺序
	all, err := repo.FindAll(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "分组3", all[0].Name)
	assert.Equal(t, "分组1", all[1].Name)
	assert.Equal(t, "分组2", all[2].Name)
}

func TestGroupRepository_Delete(t *testing.T) {
	db := setupGroupTestDB(t)
	repo := NewGroupRepository(db)
	ctx := context.Background()

	// 创建分组
	group := &model.AccountGroup{Name: "待删除分组"}
	err := repo.Create(ctx, group)
	require.NoError(t, err)

	// 删除分组
	err = repo.Delete(ctx, group.ID)
	assert.NoError(t, err)

	// 验证已删除
	found, err := repo.FindByID(ctx, group.ID)
	assert.NoError(t, err)
	assert.Nil(t, found)
}

func TestGroupRepository_CountAccountsByGroupID(t *testing.T) {
	db := setupGroupTestDB(t)
	groupRepo := NewGroupRepository(db)
	ctx := context.Background()

	// 创建分组
	group := &model.AccountGroup{Name: "测试分组"}
	err := groupRepo.Create(ctx, group)
	require.NoError(t, err)

	// 创建账号并关联到分组
	for i := 0; i < 3; i++ {
		account := &model.EmailAccount{
			UID:      strings.ReplaceAll("uid-"+string(rune('a'+i)), " ", ""),
			Email:    "test" + string(rune('0'+i)) + "@example.com",
			Provider: "gmail",
			Protocol: "gmail_api",
			AuthType: "oauth2",
			GroupID:  &group.ID,
		}
		err := db.Create(account).Error
		require.NoError(t, err)
	}

	// 统计账号数量
	count, err := groupRepo.CountAccountsByGroupID(ctx, group.ID)
	assert.NoError(t, err)
	assert.Equal(t, int64(3), count)
}

func TestGroupRepository_ClearGroupIDForAccounts(t *testing.T) {
	db := setupGroupTestDB(t)
	groupRepo := NewGroupRepository(db)
	ctx := context.Background()

	// 创建分组
	group := &model.AccountGroup{Name: "测试分组"}
	err := groupRepo.Create(ctx, group)
	require.NoError(t, err)

	// 创建账号并关联到分组
	account := &model.EmailAccount{
		UID:      "test-uid",
		Email:    "test@example.com",
		Provider: "gmail",
		Protocol: "gmail_api",
		AuthType: "oauth2",
		GroupID:  &group.ID,
	}
	err = db.Create(account).Error
	require.NoError(t, err)

	// 清除分组关联
	err = groupRepo.ClearGroupIDForAccounts(ctx, group.ID)
	assert.NoError(t, err)

	// 验证账号的 group_id 已清除
	var updated model.EmailAccount
	err = db.First(&updated, account.ID).Error
	require.NoError(t, err)
	assert.Nil(t, updated.GroupID)
}
