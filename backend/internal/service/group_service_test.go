package service

import (
	"context"
	"testing"

	"fusionmail/internal/model"
)

// countingGroupRepo 记录调用次数的 GroupRepository mock
// 用于验证 GetGroups 使用批量聚合方法，不再循环内逐分组查询（N+1 回归测试）
type countingGroupRepo struct {
	groups []*model.AccountGroup

	accountCounts map[int64]int64
	unreadCounts  map[int64]int64

	// 调用次数记录
	findAllCalls                     int
	countAccountsByGroupIDCalls      int
	countUnreadEmailsByGroupIDCalls  int
	countAccountsByGroupIDsCalls     int
	countUnreadEmailsByGroupIDsCalls int

	// 记录最近一次批量入参，验证一次调用携带全部分组 ID
	lastBatchAccountIDs []int64
	lastBatchUnreadIDs  []int64

	// 置为 true 时批量方法返回错误（验证降级行为）
	failBatch bool
}

func (r *countingGroupRepo) Create(ctx context.Context, group *model.AccountGroup) error {
	return nil
}
func (r *countingGroupRepo) Update(ctx context.Context, group *model.AccountGroup) error {
	return nil
}
func (r *countingGroupRepo) Delete(ctx context.Context, id int64) error { return nil }
func (r *countingGroupRepo) FindByID(ctx context.Context, id int64) (*model.AccountGroup, error) {
	return nil, nil
}
func (r *countingGroupRepo) FindAll(ctx context.Context) ([]*model.AccountGroup, error) {
	r.findAllCalls++
	return r.groups, nil
}
func (r *countingGroupRepo) FindByName(ctx context.Context, name string) (*model.AccountGroup, error) {
	return nil, nil
}
func (r *countingGroupRepo) ExistsByName(ctx context.Context, name string, excludeID int64) (bool, error) {
	return false, nil
}
func (r *countingGroupRepo) CountAccountsByGroupID(ctx context.Context, groupID int64) (int64, error) {
	r.countAccountsByGroupIDCalls++
	return 0, nil
}
func (r *countingGroupRepo) CountAccountsByGroupIDs(ctx context.Context, groupIDs []int64) (map[int64]int64, error) {
	r.countAccountsByGroupIDsCalls++
	r.lastBatchAccountIDs = groupIDs
	if r.failBatch {
		return nil, context.DeadlineExceeded
	}
	return r.accountCounts, nil
}
func (r *countingGroupRepo) CountUnreadEmailsByGroupID(ctx context.Context, groupID int64) (int64, error) {
	r.countUnreadEmailsByGroupIDCalls++
	return 0, nil
}
func (r *countingGroupRepo) CountUnreadEmailsByGroupIDs(ctx context.Context, groupIDs []int64) (map[int64]int64, error) {
	r.countUnreadEmailsByGroupIDsCalls++
	r.lastBatchUnreadIDs = groupIDs
	if r.failBatch {
		return nil, context.DeadlineExceeded
	}
	return r.unreadCounts, nil
}
func (r *countingGroupRepo) CountUnreadEmailsByUngrouped(ctx context.Context) (int64, error) {
	return 0, nil
}
func (r *countingGroupRepo) CountUnreadEmailsTotal(ctx context.Context) (int64, error) {
	return 0, nil
}
func (r *countingGroupRepo) UpdateDisplayOrders(ctx context.Context, groupIDs []int64) error {
	return nil
}
func (r *countingGroupRepo) GetMaxDisplayOrder(ctx context.Context) (int, error) { return 0, nil }
func (r *countingGroupRepo) ClearGroupIDForAccounts(ctx context.Context, groupID int64) error {
	return nil
}

// TestGetGroups_BatchAggregation 验证 GetGroups 仅用两次批量聚合查询完成统计，
// 仓库调用次数与分组数量无关（N+1 回归测试）
func TestGetGroups_BatchAggregation(t *testing.T) {
	const groupCount = 30
	groups := make([]*model.AccountGroup, groupCount)
	for i := range groups {
		groups[i] = &model.AccountGroup{ID: int64(i + 1), Name: "分组"}
	}

	repo := &countingGroupRepo{
		groups:        groups,
		accountCounts: map[int64]int64{1: 5, 2: 3},
		unreadCounts:  map[int64]int64{1: 7},
	}
	svc := NewGroupService(repo, nil)

	result, err := svc.GetGroups(context.Background())
	if err != nil {
		t.Fatalf("GetGroups() error = %v", err)
	}

	// 仓库调用次数应为常数（1 次 FindAll + 各 1 次批量聚合），与分组数量无关
	if repo.findAllCalls != 1 {
		t.Errorf("FindAll 调用次数 = %d, want 1", repo.findAllCalls)
	}
	if repo.countAccountsByGroupIDsCalls != 1 {
		t.Errorf("CountAccountsByGroupIDs 调用次数 = %d, want 1", repo.countAccountsByGroupIDsCalls)
	}
	if repo.countUnreadEmailsByGroupIDsCalls != 1 {
		t.Errorf("CountUnreadEmailsByGroupIDs 调用次数 = %d, want 1", repo.countUnreadEmailsByGroupIDsCalls)
	}
	if repo.countAccountsByGroupIDCalls != 0 || repo.countUnreadEmailsByGroupIDCalls != 0 {
		t.Errorf("不应再逐分组调用单组统计: accounts=%d, unread=%d",
			repo.countAccountsByGroupIDCalls, repo.countUnreadEmailsByGroupIDCalls)
	}

	// 批量调用应一次性携带全部分组 ID
	if len(repo.lastBatchAccountIDs) != groupCount || len(repo.lastBatchUnreadIDs) != groupCount {
		t.Errorf("批量入参应携带全部 %d 个分组 ID, got accounts=%d, unread=%d",
			groupCount, len(repo.lastBatchAccountIDs), len(repo.lastBatchUnreadIDs))
	}

	// 结果统计值应来自批量聚合的 map
	if len(result) != groupCount {
		t.Fatalf("结果分组数 = %d, want %d", len(result), groupCount)
	}
	if result[0].AccountCount != 5 || result[0].UnreadCount != 7 {
		t.Errorf("分组1 统计 = (accounts=%d, unread=%d), want (5, 7)",
			result[0].AccountCount, result[0].UnreadCount)
	}
	if result[1].AccountCount != 3 || result[1].UnreadCount != 0 {
		t.Errorf("分组2 统计 = (accounts=%d, unread=%d), want (3, 0)",
			result[1].AccountCount, result[1].UnreadCount)
	}
	if result[2].AccountCount != 0 || result[2].UnreadCount != 0 {
		t.Errorf("无统计数据的分组应取默认值 0, got (accounts=%d, unread=%d)",
			result[2].AccountCount, result[2].UnreadCount)
	}
}

// TestGetGroups_EmptyGroups 验证空分组列表时不触发批量统计查询
func TestGetGroups_EmptyGroups(t *testing.T) {
	repo := &countingGroupRepo{groups: []*model.AccountGroup{}}
	svc := NewGroupService(repo, nil)

	result, err := svc.GetGroups(context.Background())
	if err != nil {
		t.Fatalf("GetGroups() error = %v", err)
	}
	if len(result) != 0 {
		t.Errorf("空分组应返回空切片, got %d", len(result))
	}
	if repo.countAccountsByGroupIDsCalls != 0 || repo.countUnreadEmailsByGroupIDsCalls != 0 {
		t.Errorf("空分组列表不应触发批量统计查询: accounts=%d, unread=%d",
			repo.countAccountsByGroupIDsCalls, repo.countUnreadEmailsByGroupIDsCalls)
	}
}

// TestGetGroups_BatchErrorDegradesToZero 验证批量统计失败时保持原有降级行为：
// 计数按 0 处理，列表正常返回，不向上传播错误
func TestGetGroups_BatchErrorDegradesToZero(t *testing.T) {
	groups := []*model.AccountGroup{{ID: 1, Name: "分组1"}}
	repo := &countingGroupRepo{
		groups:    groups,
		failBatch: true,
	}
	svc := NewGroupService(repo, nil)

	result, err := svc.GetGroups(context.Background())
	if err != nil {
		t.Fatalf("批量统计失败时列表仍应返回, got error = %v", err)
	}
	if len(result) != 1 || result[0].AccountCount != 0 || result[0].UnreadCount != 0 {
		t.Errorf("统计失败时应降级为 0, got (accounts=%d, unread=%d)",
			result[0].AccountCount, result[0].UnreadCount)
	}
}
