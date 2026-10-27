package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"fusionmail/internal/model"
	"fusionmail/internal/repository"
)

// 子邮箱列表 include 过滤
const (
	ChildIncludeActive   = "active"   // 仅有效（默认）
	ChildIncludeOrphaned = "orphaned" // 仅远端已删除标记
	ChildIncludeAll      = "all"      // 全部
)

// ChildAccountInfo 子邮箱账户信息
type ChildAccountInfo struct {
	UID           string     `json:"uid"`            // 账户 UID
	Email         string     `json:"email"`          // 邮箱地址
	Status        string     `json:"status"`         // 状态
	DisableReason string     `json:"disable_reason"` // 禁用原因（如 remote_mailbox_deleted）
	TotalEmails   int64      `json:"total_emails"`   // 邮件总数
	UnreadCount   int        `json:"unread_count"`   // 未读数量
	LastSyncAt    *time.Time `json:"last_sync_at"`   // 上次同步时间
	CreatedAt     time.Time  `json:"created_at"`     // 创建时间
	Orphaned      bool       `json:"orphaned"`       // 是否为远端已删除的本地归档
}

// ChildReconcileResult 子邮箱与远端对账结果
type ChildReconcileResult struct {
	RemoteCount       int      `json:"remote_count"`    // 远端有效地址数
	LocalCount        int      `json:"local_count"`     // 本地子账户总数
	MarkedOrphaned    int      `json:"marked_orphaned"` // 本次标记为孤儿的数量
	Reactivated       int      `json:"reactivated"`     // 本次恢复为 active 的数量
	Unchanged         int      `json:"unchanged"`       // 无变化数量
	OrphanedEmails    []string `json:"orphaned_emails"` // 本次标记的邮箱（最多返回前 50）
	ReactivatedEmails []string `json:"reactivated_emails"`
	SkippedRemote     bool     `json:"skipped_remote"` // 是否因无法获取远端列表而跳过
	Message           string   `json:"message,omitempty"`
}

// ChildAccountListQuery 子邮箱分页查询参数
type ChildAccountListQuery struct {
	Include  string // active | orphaned | all
	Email    string // 邮箱关键词
	Page     int
	PageSize int
}

// ChildAccountListResult 子邮箱分页结果
type ChildAccountListResult struct {
	Items    []*ChildAccountInfo `json:"items"`
	Total    int64               `json:"total"`
	Page     int                 `json:"page"`
	PageSize int                 `json:"page_size"`
}

// GetChildAccounts 获取 WebAPI 账户关联的子邮箱列表（分页 + 邮箱搜索）
// include: active（默认）| orphaned | all
func (s *WebAPIProviderChildAccountService) GetChildAccounts(ctx context.Context, parentUID string, q ChildAccountListQuery) (*ChildAccountListResult, error) {
	q.Include = normalizeChildInclude(q.Include)
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 {
		q.PageSize = 20
	}
	if q.PageSize > 100 {
		q.PageSize = 100
	}

	parentAccount, err := s.accountRepo.FindByUID(ctx, parentUID)
	if err != nil {
		return nil, fmt.Errorf("父账户未找到: %w", err)
	}
	if parentAccount == nil {
		return nil, errors.New("父账户不存在")
	}
	if !isWebAPIAccount(ctx, s.providerRepo, parentAccount) {
		return nil, errors.New("该账户不是 WebAPI 类型")
	}

	childAccountList, total, err := s.accountRepo.FindChildrenByParent(ctx, &repository.ChildAccountListFilter{
		ParentUID: parentUID,
		Include:   q.Include,
		Email:     q.Email,
		Page:      q.Page,
		PageSize:  q.PageSize,
	})
	if err != nil {
		return nil, fmt.Errorf("查询子账户失败: %w", err)
	}

	// 一次聚合查询统计各子邮箱的邮件总数，避免循环内逐账户查询（N+1）
	childUIDs := make([]string, 0, len(childAccountList))
	for _, acc := range childAccountList {
		childUIDs = append(childUIDs, acc.UID)
	}
	emailCounts, err := s.emailRepo.CountByAccounts(ctx, childUIDs)
	if err != nil {
		// 与原逐账户统计失败时的行为一致：计数按 0 处理，不阻断列表返回
		s.log.Warn("批量统计子邮箱邮件总数失败: parentUID=%s, err=%v", parentUID, err)
		emailCounts = map[string]int64{}
	}

	items := make([]*ChildAccountInfo, 0, len(childAccountList))
	for _, acc := range childAccountList {
		items = append(items, &ChildAccountInfo{
			UID:           acc.UID,
			Email:         acc.Email,
			Status:        acc.Status,
			DisableReason: acc.DisableReason,
			TotalEmails:   emailCounts[acc.UID],
			UnreadCount:   acc.UnreadCount,
			LastSyncAt:    acc.LastSyncAt,
			CreatedAt:     acc.CreatedAt,
			Orphaned:      acc.IsOrphanRemoteDeleted(),
		})
	}

	s.log.Info("获取子邮箱列表: parentUID=%s, include=%s, email=%q, page=%d/%d, total=%d",
		parentUID, q.Include, q.Email, q.Page, q.PageSize, total)
	return &ChildAccountListResult{
		Items:    items,
		Total:    total,
		Page:     q.Page,
		PageSize: q.PageSize,
	}, nil
}

func normalizeChildInclude(include string) string {
	switch strings.ToLower(strings.TrimSpace(include)) {
	case ChildIncludeOrphaned:
		return ChildIncludeOrphaned
	case ChildIncludeAll:
		return ChildIncludeAll
	default:
		return ChildIncludeActive
	}
}

func matchChildInclude(acc *model.EmailAccount, include string) bool {
	if acc == nil {
		return false
	}
	switch include {
	case ChildIncludeOrphaned:
		return acc.IsOrphanRemoteDeleted()
	case ChildIncludeAll:
		return true
	default: // active
		return acc.IsActiveAccount()
	}
}

// ReconcileChildAccounts 将本地子账户与远端有效邮箱列表对账
// 策略：本地有、远端无 → 标记 disabled + remote_mailbox_deleted（保留邮件）
//
//	本地孤儿、远端又有 → 恢复 active
func (s *WebAPIProviderChildAccountService) ReconcileChildAccounts(ctx context.Context, parentUID string) (*ChildReconcileResult, error) {
	parentAccount, err := s.accountRepo.FindByUID(ctx, parentUID)
	if err != nil {
		return nil, fmt.Errorf("父账户未找到: %w", err)
	}
	if parentAccount == nil {
		return nil, errors.New("父账户不存在")
	}
	if !isWebAPIAccount(ctx, s.providerRepo, parentAccount) {
		return nil, errors.New("该账户不是 WebAPI 类型")
	}

	result := &ChildReconcileResult{}

	// 1. 拉取远端有效地址
	remoteList, remoteErr := s.GetSubAccounts(ctx, parentUID)
	if remoteErr != nil {
		result.SkippedRemote = true
		result.Message = fmt.Sprintf("无法获取远端邮箱列表，跳过对账: %v", remoteErr)
		s.log.Warn("子邮箱对账跳过: parentUID=%s, err=%v", parentUID, remoteErr)
		return result, nil
	}

	remoteSet := buildRemoteEmailMatchSet(remoteList)
	result.RemoteCount = len(remoteList)

	// 2. 本地子账户
	localChildren, err := s.accountRepo.FindByParentAccountUID(ctx, parentUID)
	if err != nil {
		return nil, fmt.Errorf("查询本地子账户失败: %w", err)
	}
	result.LocalCount = len(localChildren)

	const maxEmailSamples = 50
	for _, child := range localChildren {
		emailKey := strings.ToLower(strings.TrimSpace(child.Email))
		present := remoteEmailPresent(emailKey, remoteSet)

		if present {
			// 远端仍存在：若此前被标为远端删除，则恢复
			if child.IsOrphanRemoteDeleted() {
				if err := s.accountRepo.ReactivateFromRemoteOrphan(ctx, child.UID); err != nil {
					s.log.Warn("恢复子账户失败: uid=%s, err=%v", child.UID, err)
					continue
				}
				result.Reactivated++
				if len(result.ReactivatedEmails) < maxEmailSamples {
					result.ReactivatedEmails = append(result.ReactivatedEmails, child.Email)
				}
			} else {
				result.Unchanged++
			}
			continue
		}

		// 远端不存在
		if child.IsOrphanRemoteDeleted() {
			result.Unchanged++
			continue
		}
		// 仅处理 active 的子账户；其它禁用原因不动
		if !child.IsActiveAccount() {
			result.Unchanged++
			continue
		}
		if err := s.accountRepo.MarkRemoteMailboxDeleted(ctx, child.UID); err != nil {
			s.log.Warn("标记孤儿子账户失败: uid=%s, err=%v", child.UID, err)
			continue
		}
		result.MarkedOrphaned++
		if len(result.OrphanedEmails) < maxEmailSamples {
			result.OrphanedEmails = append(result.OrphanedEmails, child.Email)
		}
	}

	result.Message = fmt.Sprintf(
		"对账完成：远端 %d，本地 %d，新标记孤儿 %d，恢复 %d，无变化 %d",
		result.RemoteCount, result.LocalCount, result.MarkedOrphaned, result.Reactivated, result.Unchanged,
	)
	s.log.Info("子邮箱对账完成: parentUID=%s, %s", parentUID, result.Message)
	return result, nil
}

// remoteEmailMatchSet 远端地址匹配集
type remoteEmailMatchSet struct {
	exact   map[string]struct{} // 完整邮箱
	domains map[string]struct{} // 通配域名 *@example.com
}

func buildRemoteEmailMatchSet(list []*SubAccountInfo) remoteEmailMatchSet {
	set := remoteEmailMatchSet{
		exact:   make(map[string]struct{}),
		domains: make(map[string]struct{}),
	}
	for _, item := range list {
		if item == nil {
			continue
		}
		email := strings.ToLower(strings.TrimSpace(item.Email))
		if email == "" {
			continue
		}
		// 通配域名：*@domain.com 或 域名: x
		if strings.HasPrefix(email, "*@") {
			set.domains[strings.TrimPrefix(email, "*@")] = struct{}{}
			continue
		}
		if strings.HasPrefix(email, "域名:") {
			// 忽略展示项
			continue
		}
		set.exact[email] = struct{}{}
		// 也记录域名便于扩展
		if at := strings.LastIndex(email, "@"); at > 0 && at < len(email)-1 {
			// exact only for full addresses
		}
	}
	return set
}

func remoteEmailPresent(localEmail string, set remoteEmailMatchSet) bool {
	if localEmail == "" {
		return false
	}
	if _, ok := set.exact[localEmail]; ok {
		return true
	}
	at := strings.LastIndex(localEmail, "@")
	if at > 0 && at < len(localEmail)-1 {
		domain := localEmail[at+1:]
		if _, ok := set.domains[domain]; ok {
			return true
		}
	}
	return false
}
