package spam

import (
	"context"
	"testing"
	"time"
)

func TestSURBLCheckerCheck_AllowsNilRedisClient(t *testing.T) {
	checker := NewSURBLChecker(nil)
	checker.timeout = time.Nanosecond

	result, err := checker.Check(context.Background(), "请访问 https://example.com", "")
	if err != nil {
		t.Fatalf("Check returned error: %v", err)
	}
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if result.CheckedURLs != 1 {
		t.Fatalf("Expected 1 checked URL, got %d", result.CheckedURLs)
	}
}
