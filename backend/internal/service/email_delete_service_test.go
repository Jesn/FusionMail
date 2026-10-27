package service

import (
	"context"
	"testing"

	"fusionmail/internal/model"
	"fusionmail/internal/repository"
)

// countingDeleteRepo 记录仓库调用次数，验证批量永久删除/清空回收站
// 的查询次数与 N 无关（N+1 回归测试）。
type countingDeleteRepo struct {
	repository.EmailRepository // 未实现的方法 panic，见 panic 兜底

	findByIDsCalls        int
	permDeleteByIDsCalls  int
	permDeleteAllCalls    int
	permDeleteSingleCalls int // 旧实现的逐封删除调用，新实现应为 0
}

func (r *countingDeleteRepo) FindByIDs(_ context.Context, ids []int64) ([]*model.Email, error) {
	r.findByIDsCalls++
	out := make([]*model.Email, 0, len(ids))
	for _, id := range ids {
		// 一半在回收站，一半不在 —— 验证服务层过滤
		out = append(out, &model.Email{ID: id, IsDeleted: id%2 == 0})
	}
	return out, nil
}

func (r *countingDeleteRepo) PermanentDeleteByIDs(_ context.Context, ids []int64) (int64, error) {
	r.permDeleteByIDsCalls++
	return int64(len(ids)), nil
}

func (r *countingDeleteRepo) PermanentDeleteAllTrash(context.Context) (int64, error) {
	r.permDeleteAllCalls++
	return 7, nil
}

func (r *countingDeleteRepo) PermanentDelete(context.Context, int64) error {
	r.permDeleteSingleCalls++
	return nil
}

// TestBatchPermanentDeleteEmailsConstantQueries 验证 1000 个 ID 的批量永久删除
// 仓库调用次数为常数 2（FindByIDs + PermanentDeleteByIDs），且过滤掉非回收站邮件。
func TestBatchPermanentDeleteEmailsConstantQueries(t *testing.T) {
	repo := &countingDeleteRepo{}
	svc := NewEmailDeleteService(repo, nil, nil, nil)

	ids := make([]int64, 1000)
	for i := range ids {
		ids[i] = int64(i + 1)
	}

	n, err := svc.BatchPermanentDeleteEmails(context.Background(), ids)
	if err != nil {
		t.Fatalf("批量永久删除失败: %v", err)
	}
	// 1000 个 ID 中偶数 ID 在回收站（id%2==0）→ 500
	if n != 500 {
		t.Fatalf("应删 500 封，got %d", n)
	}
	if repo.findByIDsCalls != 1 || repo.permDeleteByIDsCalls != 1 {
		t.Fatalf("仓库调用应恒为 2 次（1 FindByIDs + 1 批量删），got find=%d delete=%d",
			repo.findByIDsCalls, repo.permDeleteByIDsCalls)
	}
	if repo.permDeleteSingleCalls != 0 {
		t.Fatalf("不应再逐封删除（旧 N+1 路径），got %d 次", repo.permDeleteSingleCalls)
	}
}

// TestEmptyTrashSingleQuery 验证清空回收站只调 1 次仓库方法（原实现分页 + 逐封删）。
func TestEmptyTrashSingleQuery(t *testing.T) {
	repo := &countingDeleteRepo{}
	svc := NewEmailDeleteService(repo, nil, nil, nil)

	n, err := svc.EmptyTrash(context.Background())
	if err != nil {
		t.Fatalf("清空回收站失败: %v", err)
	}
	if n != 7 {
		t.Fatalf("应返回仓库删除行数 7，got %d", n)
	}
	if repo.permDeleteAllCalls != 1 || repo.permDeleteSingleCalls != 0 || repo.findByIDsCalls != 0 {
		t.Fatalf("应只调 1 次 PermanentDeleteAllTrash，got all=%d single=%d find=%d",
			repo.permDeleteAllCalls, repo.permDeleteSingleCalls, repo.findByIDsCalls)
	}
}
