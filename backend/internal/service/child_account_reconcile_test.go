package service

import (
	"testing"

	"fusionmail/internal/model"
)

func TestNormalizeChildInclude(t *testing.T) {
	if got := normalizeChildInclude(""); got != ChildIncludeActive {
		t.Fatalf("empty -> active, got %s", got)
	}
	if got := normalizeChildInclude("ALL"); got != ChildIncludeAll {
		t.Fatalf("ALL -> all, got %s", got)
	}
	if got := normalizeChildInclude("orphaned"); got != ChildIncludeOrphaned {
		t.Fatalf("orphaned, got %s", got)
	}
}

func TestChildAccountListQueryDefaults(t *testing.T) {
	// 仅验证常量与过滤匹配仍可用（分页默认值在 service 内处理）
	if ChildIncludeActive != "active" {
		t.Fatal("unexpected include constant")
	}
}

func TestMatchChildInclude(t *testing.T) {
	active := &model.EmailAccount{Status: model.AccountStatusActive}
	orphan := &model.EmailAccount{
		Status:        model.AccountStatusDisabled,
		DisableReason: model.DisableReasonRemoteMailboxDeleted,
	}
	otherDisabled := &model.EmailAccount{
		Status:        model.AccountStatusDisabled,
		DisableReason: "auto_disabled_auth_failure",
	}

	if !matchChildInclude(active, ChildIncludeActive) {
		t.Fatal("active should match active filter")
	}
	if matchChildInclude(orphan, ChildIncludeActive) {
		t.Fatal("orphan should not match active filter")
	}
	if !matchChildInclude(orphan, ChildIncludeOrphaned) {
		t.Fatal("orphan should match orphaned filter")
	}
	if matchChildInclude(otherDisabled, ChildIncludeOrphaned) {
		t.Fatal("other disabled should not match orphaned")
	}
	if !matchChildInclude(otherDisabled, ChildIncludeAll) {
		t.Fatal("all should include other disabled")
	}
}

func TestRemoteEmailPresent(t *testing.T) {
	set := buildRemoteEmailMatchSet([]*SubAccountInfo{
		{Email: "a@example.com"},
		{Email: "*@wildcard.com"},
	})
	if !remoteEmailPresent("a@example.com", set) {
		t.Fatal("exact match failed")
	}
	if !remoteEmailPresent("x@wildcard.com", set) {
		t.Fatal("domain wildcard match failed")
	}
	if remoteEmailPresent("gone@other.com", set) {
		t.Fatal("should not match missing email")
	}
}
