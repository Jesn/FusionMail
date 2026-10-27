package adapter

import (
	"context"
	"fmt"
	"time"

	"github.com/emersion/go-imap/v2"
)

// FetchEmails 拉取邮件列表
func (a *IMAPAdapter) FetchEmails(ctx context.Context, since time.Time, limit int) ([]*Email, error) {
	fmt.Printf("[IMAP] FetchEmails called, since=%v, limit=%d\n", since, limit)

	if a.client == nil {
		fmt.Printf("[IMAP] Error: client is nil\n")
		return nil, fmt.Errorf("not connected")
	}

	// 选择 INBOX
	fmt.Printf("[IMAP] Selecting INBOX...\n")
	mailbox, err := a.client.Select("INBOX", nil).Wait()
	if err != nil {
		fmt.Printf("[IMAP] Error selecting INBOX: %v\n", err)
		return nil, fmt.Errorf("failed to select INBOX: %w", err)
	}

	fmt.Printf("[IMAP] Mailbox INBOX: %d messages, %d recent\n",
		mailbox.NumMessages, mailbox.NumRecent)

	if mailbox.NumMessages == 0 {
		fmt.Printf("[IMAP] No messages in INBOX\n")
		return []*Email{}, nil
	}

	// 使用搜索来根据时间过滤邮件
	var seqSet imap.SeqSet

	if !since.IsZero() {
		// 暂时跳过时间过滤，直接获取所有邮件
		fmt.Printf("[IMAP] Time filtering requested but not implemented, fetching all emails\n")
		seqSet.AddRange(1, mailbox.NumMessages)
	} else {
		// 没有时间限制，获取所有邮件或最新的 limit 封
		start := uint32(1)
		end := mailbox.NumMessages

		if limit > 0 && int(mailbox.NumMessages) > limit {
			// 只获取最新的 limit 封邮件
			start = mailbox.NumMessages - uint32(limit) + 1
			fmt.Printf("[IMAP] Limiting to last %d emails (from %d to %d)\n", limit, start, end)
		} else {
			fmt.Printf("[IMAP] Fetching all %d emails\n", mailbox.NumMessages)
		}

		seqSet.AddRange(start, end)
	}

	fmt.Printf("[IMAP] Created SeqSet\n")

	// 获取邮件信息
	fetchOptions := &imap.FetchOptions{
		Envelope:     true,
		BodySection:  []*imap.FetchItemBodySection{{}},
		UID:          true,
		InternalDate: true,
		RFC822Size:   true,
	}

	emails := make([]*Email, 0)
	fmt.Printf("[IMAP] Starting to fetch messages...\n")
	fetchCmd := a.client.Fetch(seqSet, fetchOptions)

	for {
		msg := fetchCmd.Next()
		if msg == nil {
			break
		}

		// 使用 Collect() 获取完整的消息数据
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
	}

	if err := fetchCmd.Close(); err != nil {
		return nil, fmt.Errorf("failed to fetch emails: %w", err)
	}

	fmt.Printf("[IMAP] Successfully fetched %d emails\n", len(emails))
	return emails, nil
}

// FetchEmailDetail 获取邮件详情
func (a *IMAPAdapter) FetchEmailDetail(ctx context.Context, providerID string) (*Email, error) {
	if a.client == nil {
		return nil, fmt.Errorf("not connected")
	}

	// providerID 是 UID
	uid, err := parseUID(providerID)
	if err != nil {
		return nil, fmt.Errorf("invalid provider ID: %w", err)
	}

	// 选择 INBOX
	_, err = a.client.Select("INBOX", nil).Wait()
	if err != nil {
		return nil, fmt.Errorf("failed to select INBOX: %w", err)
	}

	// 获取邮件详情
	seqSet := imap.UIDSetNum(uid)
	fetchOptions := &imap.FetchOptions{
		Envelope:     true,
		BodySection:  []*imap.FetchItemBodySection{{}},
		UID:          true,
		InternalDate: true,
		RFC822Size:   true,
	}

	fetchCmd := a.client.Fetch(seqSet, fetchOptions)
	msg := fetchCmd.Next()
	if msg == nil {
		return nil, fmt.Errorf("email not found")
	}

	buf, err := msg.Collect()
	if err != nil {
		return nil, fmt.Errorf("failed to collect message: %w", err)
	}

	email, err := a.parseMessageBuffer(buf)
	if err != nil {
		return nil, err
	}

	if err := fetchCmd.Close(); err != nil {
		return nil, fmt.Errorf("failed to fetch email: %w", err)
	}

	return email, nil
}

// ============================================================================
// BatchFetcher 接口实现 - 支持分批拉取邮件
// Requirements: 3.1, 4.1, 4.2, 4.3
// ============================================================================

// FetchEmailsBatch 分批拉取邮件
// Requirements: 3.1, 4.3 - 分批处理邮件
func (a *IMAPAdapter) FetchEmailsBatch(ctx context.Context, since time.Time, batchSize int, cursor string) ([]*Email, string, bool, error) {
	if a.client == nil {
		return nil, "", false, fmt.Errorf("not connected")
	}

	// 选择 INBOX
	mailbox, err := a.client.Select("INBOX", nil).Wait()
	if err != nil {
		return nil, "", false, fmt.Errorf("failed to select INBOX: %w", err)
	}

	if mailbox.NumMessages == 0 {
		return []*Email{}, "", false, nil
	}

	// 解析游标（上次处理到的序号）
	var startSeq uint32 = 1
	if cursor != "" {
		var lastSeq uint32
		if _, err := fmt.Sscanf(cursor, "%d", &lastSeq); err == nil && lastSeq > 0 {
			startSeq = lastSeq + 1
		}
	}

	// 如果起始序号超过邮件总数，说明已经处理完毕
	if startSeq > mailbox.NumMessages {
		return []*Email{}, "", false, nil
	}

	// 尝试使用 SEARCH 命令按日期过滤
	var seqNums []uint32
	if !since.IsZero() {
		seqNums, err = a.searchEmailsSince(ctx, since)
		if err != nil {
			// SEARCH 失败，降级到序号范围
			fmt.Printf("[IMAP] SEARCH failed, falling back to sequence range: %v\n", err)
			seqNums = nil
		}
	}

	// 如果 SEARCH 成功，过滤出从 startSeq 开始的序号
	if len(seqNums) > 0 {
		var filteredSeqs []uint32
		for _, seq := range seqNums {
			if seq >= startSeq {
				filteredSeqs = append(filteredSeqs, seq)
			}
		}
		seqNums = filteredSeqs
	}

	// 计算本批次要拉取的范围
	var seqSet imap.SeqSet
	var endSeq uint32
	var hasMore bool

	if len(seqNums) > 0 {
		// 使用 SEARCH 结果
		if len(seqNums) > batchSize {
			seqNums = seqNums[:batchSize]
			hasMore = true
		}
		for _, seq := range seqNums {
			seqSet.AddNum(seq)
		}
		if len(seqNums) > 0 {
			endSeq = seqNums[len(seqNums)-1]
		}
	} else {
		// 降级到序号范围
		endSeq = startSeq + uint32(batchSize) - 1
		if endSeq > mailbox.NumMessages {
			endSeq = mailbox.NumMessages
		} else {
			hasMore = true
		}
		seqSet.AddRange(startSeq, endSeq)
	}

	// 获取邮件
	fetchOptions := &imap.FetchOptions{
		Envelope:     true,
		BodySection:  []*imap.FetchItemBodySection{{}},
		UID:          true,
		InternalDate: true,
		RFC822Size:   true,
	}

	emails := make([]*Email, 0)
	fetchCmd := a.client.Fetch(seqSet, fetchOptions)

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

		// 注意：不再在适配器层做时间过滤
		// 原因：
		// 1. IMAP SEARCH 的 SINCE 只精确到日期，不含时间
		// 2. 时区问题可能导致误判（服务器时区 vs 邮件服务器时区）
		// 3. 完全依赖数据库层的 ProviderID 去重更可靠
		//
		// 邮件去重在 sync_service.processEmail 中通过 FindByProviderID 实现
		// 即使拉取了已存在的邮件，也只会更新而不会重复创建

		emails = append(emails, email)
	}

	if err := fetchCmd.Close(); err != nil {
		return nil, "", false, fmt.Errorf("failed to fetch emails: %w", err)
	}

	// 生成下一个游标
	nextCursor := ""
	if hasMore {
		nextCursor = fmt.Sprintf("%d", endSeq)
	}

	return emails, nextCursor, hasMore, nil
}

// searchEmailsSince 使用 SEARCH 命令搜索指定日期之后的邮件
// Requirements: 4.1 - 使用 IMAP SEARCH 命令
func (a *IMAPAdapter) searchEmailsSince(ctx context.Context, since time.Time) ([]uint32, error) {
	// 构建 SEARCH 条件
	criteria := &imap.SearchCriteria{
		Since: since,
	}

	// 执行搜索
	searchCmd := a.client.Search(criteria, nil)
	searchData, err := searchCmd.Wait()
	if err != nil {
		return nil, fmt.Errorf("SEARCH command failed: %w", err)
	}

	// 转换为 uint32 切片
	seqNums := make([]uint32, 0, len(searchData.AllSeqNums()))
	for _, num := range searchData.AllSeqNums() {
		seqNums = append(seqNums, num)
	}

	return seqNums, nil
}

// GetEstimatedCount 获取预估邮件数量
// Requirements: 2.1 - 提供预估总数
func (a *IMAPAdapter) GetEstimatedCount(ctx context.Context, since time.Time) (int, error) {
	if a.client == nil {
		return 0, fmt.Errorf("not connected")
	}

	// 选择 INBOX
	mailbox, err := a.client.Select("INBOX", nil).Wait()
	if err != nil {
		return 0, fmt.Errorf("failed to select INBOX: %w", err)
	}

	// 如果没有时间限制，返回总邮件数
	if since.IsZero() {
		return int(mailbox.NumMessages), nil
	}

	// 尝试使用 SEARCH 获取精确数量
	seqNums, err := a.searchEmailsSince(ctx, since)
	if err != nil {
		// SEARCH 失败，返回总数作为估计值
		fmt.Printf("[IMAP] SEARCH failed for count estimation, using total: %v\n", err)
		return int(mailbox.NumMessages), nil
	}

	return len(seqNums), nil
}
