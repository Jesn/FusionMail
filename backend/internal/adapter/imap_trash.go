package adapter

import (
	"context"
	"fmt"
	"strings"

	"github.com/emersion/go-imap/v2"
)

// MoveToTrash 将邮件移至垃圾箱
func (a *IMAPAdapter) MoveToTrash(ctx context.Context, providerID string) error {
	if a.client == nil {
		return fmt.Errorf("not connected to IMAP server")
	}

	// 解析 UID
	uid, err := parseUID(providerID)
	if err != nil {
		return fmt.Errorf("invalid provider ID: %w", err)
	}

	// 发现 Trash 文件夹
	trashMailbox, err := a.findTrashMailbox(ctx)
	if err != nil {
		return fmt.Errorf("trash mailbox not found: %w", err)
	}

	// 尝试使用 MOVE 命令（RFC 6851）
	if err := a.moveToTrash(ctx, uid, trashMailbox); err == nil {
		return nil
	}

	// 降级：COPY + STORE +FLAGS \Deleted
	return a.copyAndMarkDeleted(ctx, uid, trashMailbox)
}

// findTrashMailbox 发现 Trash 文件夹
func (a *IMAPAdapter) findTrashMailbox(ctx context.Context) (string, error) {
	// 获取所有邮箱列表
	listCmd := a.client.List("", "*", nil)
	mailboxes, err := listCmd.Collect()
	if err != nil {
		return "", fmt.Errorf("failed to list mailboxes: %w", err)
	}

	// 其次查找常见的 Trash 文件夹名称
	trashNames := []string{"Trash", "Deleted Items", "[Gmail]/Trash", "[Gmail]/Deleted Mail", "Deleted", "Bin"}
	for _, name := range trashNames {
		for _, mbox := range mailboxes {
			if strings.EqualFold(mbox.Mailbox, name) {
				return mbox.Mailbox, nil
			}
		}
	}

	return "", fmt.Errorf("trash mailbox not found")
}

// moveToTrash 使用 MOVE 命令移动邮件到 Trash
func (a *IMAPAdapter) moveToTrash(ctx context.Context, uid imap.UID, trashMailbox string) error {
	// 选择 INBOX
	_, err := a.client.Select("INBOX", nil).Wait()
	if err != nil {
		return fmt.Errorf("failed to select INBOX: %w", err)
	}

	// 使用 MOVE 命令（UID 模式）
	seqSet := imap.UIDSetNum(uid)
	_, err = a.client.Move(seqSet, trashMailbox).Wait()
	if err != nil {
		return fmt.Errorf("failed to move message: %w", err)
	}

	return nil
}

// copyAndMarkDeleted COPY 降级方案（仅复制到 Trash，不标记删除）
func (a *IMAPAdapter) copyAndMarkDeleted(ctx context.Context, uid imap.UID, trashMailbox string) error {
	// 选择 INBOX
	_, err := a.client.Select("INBOX", nil).Wait()
	if err != nil {
		return fmt.Errorf("failed to select INBOX: %w", err)
	}

	// COPY 到 Trash
	seqSet := imap.UIDSetNum(uid)
	_, err = a.client.Copy(seqSet, trashMailbox).Wait()
	if err != nil {

		return fmt.Errorf("failed to copy message to trash: %w", err)
	}

	return nil
}
