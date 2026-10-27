package adapter

import (
	"context"
	"fmt"

	"github.com/emersion/go-imap/v2"
)

// parseUID 解析 UID
func parseUID(providerID string) (imap.UID, error) {
	var uid uint32
	_, err := fmt.Sscanf(providerID, "%d", &uid)
	if err != nil {
		return 0, err
	}
	return imap.UID(uid), nil
}

// ============================================================================
// UID 增量同步接口实现
// Requirements: 1.1, 1.2, 1.3, 6.1
// ============================================================================

// GetUIDValidity 获取当前邮箱的 UIDVALIDITY
// Requirements: 1.3, 6.1 - 获取 UIDVALIDITY 用于检测邮箱重建
func (a *IMAPAdapter) GetUIDValidity(ctx context.Context) (uint32, error) {
	if a.client == nil {
		return 0, fmt.Errorf("not connected")
	}

	// 选择 INBOX
	mailbox, err := a.client.Select("INBOX", nil).Wait()
	if err != nil {
		return 0, fmt.Errorf("failed to select INBOX: %w", err)
	}

	return mailbox.UIDValidity, nil
}

// GetUIDSyncState 获取完整的 UID 同步状态
// 包括 UIDVALIDITY 和当前最大 UID
func (a *IMAPAdapter) GetUIDSyncState(ctx context.Context) (*UIDSyncState, error) {
	if a.client == nil {
		return nil, fmt.Errorf("not connected")
	}

	// 选择 INBOX
	mailbox, err := a.client.Select("INBOX", nil).Wait()
	if err != nil {
		return nil, fmt.Errorf("failed to select INBOX: %w", err)
	}

	state := &UIDSyncState{
		UIDValidity: mailbox.UIDValidity,
	}

	// 获取最大 UID
	if mailbox.NumMessages > 0 {
		maxUID, err := a.getMaxUID(ctx, mailbox.NumMessages)
		if err != nil {
			fmt.Printf("[IMAP] Warning: failed to get max UID: %v\n", err)
		} else {
			state.MaxUID = maxUID
		}
	}

	return state, nil
}

// getMaxUID 获取邮箱中的最大 UID
func (a *IMAPAdapter) getMaxUID(ctx context.Context, numMessages uint32) (uint32, error) {
	// 获取最后一封邮件的 UID
	seqSet := imap.SeqSetNum(numMessages)
	fetchOptions := &imap.FetchOptions{
		UID: true,
	}

	fetchCmd := a.client.Fetch(seqSet, fetchOptions)
	msg := fetchCmd.Next()
	if msg == nil {
		return 0, fmt.Errorf("no message found")
	}

	buf, err := msg.Collect()
	if err != nil {
		return 0, fmt.Errorf("failed to collect message: %w", err)
	}

	if err := fetchCmd.Close(); err != nil {
		return 0, fmt.Errorf("failed to close fetch: %w", err)
	}

	return uint32(buf.UID), nil
}

// FetchEmailsSinceUID 获取指定 UID 之后的邮件（增量同步）
// Requirements: 1.1 - 只拉取新邮件
// 参数：
//   - sinceUID: 上次同步的最大 UID，只拉取 UID > sinceUID 的邮件
//   - limit: 最大拉取数量，0 表示不限制
//
// 返回：
//   - emails: 邮件列表
//   - maxUID: 本次拉取的最大 UID（用于更新 LastUID）
//   - error: 错误信息
func (a *IMAPAdapter) FetchEmailsSinceUID(ctx context.Context, sinceUID uint32, limit int) ([]*Email, uint32, error) {
	if a.client == nil {
		return nil, 0, fmt.Errorf("not connected")
	}

	// 选择 INBOX
	mailbox, err := a.client.Select("INBOX", nil).Wait()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to select INBOX: %w", err)
	}

	fmt.Printf("[IMAP] FetchEmailsSinceUID: sinceUID=%d, limit=%d, mailbox.NumMessages=%d\n",
		sinceUID, limit, mailbox.NumMessages)

	if mailbox.NumMessages == 0 {
		return []*Email{}, 0, nil
	}

	// 使用 UID SEARCH 查找 UID > sinceUID 的邮件
	// IMAP 命令: UID SEARCH UID sinceUID+1:*
	searchCriteria := &imap.SearchCriteria{
		UID: []imap.UIDSet{
			{imap.UIDRange{Start: imap.UID(sinceUID + 1), Stop: 0}}, // 0 表示 *（最大值）
		},
	}

	searchCmd := a.client.UIDSearch(searchCriteria, nil)
	searchData, err := searchCmd.Wait()
	if err != nil {
		return nil, 0, fmt.Errorf("UID SEARCH failed: %w", err)
	}

	uids := searchData.AllUIDs()
	fmt.Printf("[IMAP] UID SEARCH found %d new emails\n", len(uids))

	if len(uids) == 0 {
		return []*Email{}, sinceUID, nil
	}

	// 应用 limit
	if limit > 0 && len(uids) > limit {
		uids = uids[:limit]
		fmt.Printf("[IMAP] Limited to %d emails\n", limit)
	}

	// 构建 UID 集合
	var uidSet imap.UIDSet
	for _, uid := range uids {
		uidSet = append(uidSet, imap.UIDRange{Start: uid, Stop: uid})
	}

	// 获取邮件详情
	fetchOptions := &imap.FetchOptions{
		Envelope:     true,
		BodySection:  []*imap.FetchItemBodySection{{}},
		UID:          true,
		InternalDate: true,
		RFC822Size:   true,
	}

	emails := make([]*Email, 0, len(uids))
	var maxUID uint32 = sinceUID

	fetchCmd := a.client.Fetch(uidSet, fetchOptions)
	for {
		msg := fetchCmd.Next()
		if msg == nil {
			break
		}

		buf, err := msg.Collect()
		if err != nil {
			fmt.Printf("[IMAP] Failed to collect message: %v\n", err)
			continue
		}

		email, err := a.parseMessageBuffer(buf)
		if err != nil {
			fmt.Printf("[IMAP] Failed to parse message: %v\n", err)
			continue
		}

		emails = append(emails, email)

		// 更新最大 UID
		if uint32(buf.UID) > maxUID {
			maxUID = uint32(buf.UID)
		}
	}

	if err := fetchCmd.Close(); err != nil {
		return nil, 0, fmt.Errorf("failed to fetch emails: %w", err)
	}

	fmt.Printf("[IMAP] Successfully fetched %d emails, maxUID=%d\n", len(emails), maxUID)
	return emails, maxUID, nil
}

// FetchEmailsSinceUIDBatch 分批获取指定 UID 之后的邮件
// 用于大量新邮件的场景，避免一次性拉取过多
func (a *IMAPAdapter) FetchEmailsSinceUIDBatch(ctx context.Context, sinceUID uint32, batchSize int) ([]*Email, uint32, bool, error) {
	emails, maxUID, err := a.FetchEmailsSinceUID(ctx, sinceUID, batchSize)
	if err != nil {
		return nil, 0, false, err
	}

	// 检查是否还有更多邮件
	hasMore := len(emails) >= batchSize

	return emails, maxUID, hasMore, nil
}

// ShouldFullSync 判断是否需要全量同步
// Requirements: 1.3 - UIDVALIDITY 变化时需要全量同步
func (a *IMAPAdapter) ShouldFullSync(storedValidity, currentValidity uint32) bool {
	// 如果存储的 UIDVALIDITY 为 0，说明是首次同步
	if storedValidity == 0 {
		return true
	}
	// 如果 UIDVALIDITY 变化，需要全量同步
	return storedValidity != currentValidity
}
