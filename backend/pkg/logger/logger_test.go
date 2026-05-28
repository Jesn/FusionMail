package logger

import (
	"errors"
	"strings"
	"testing"
)

func TestFormatMessageSupportsStructuredFields(t *testing.T) {
	log := NewWithModule("Test")
	args := []interface{}{
		"provider", "google",
		"code_length", 12,
		"error", errors.New("boom"),
	}
	message := log.formatMessage(LevelInfo, "OAuth2 callback received",
		args...,
	)

	if strings.Contains(message, "%!(EXTRA") {
		t.Fatalf("结构化日志不应按 printf 参数渲染: %s", message)
	}
	for _, want := range []string{
		"OAuth2 callback received",
		"provider=google",
		"code_length=12",
		"error=boom",
	} {
		if !strings.Contains(message, want) {
			t.Fatalf("日志缺少 %q: %s", want, message)
		}
	}
}

func TestFormatMessageKeepsPrintfStyle(t *testing.T) {
	log := NewWithModule("Test")
	message := log.formatMessage(LevelWarn, "连接失败: count=%d, err=%v", 3, errors.New("boom"))

	if strings.Contains(message, "%!(EXTRA") {
		t.Fatalf("printf 风格日志不应产生额外参数: %s", message)
	}
	for _, want := range []string{
		"连接失败: count=3, err=boom",
		"[WARN][Test]",
	} {
		if !strings.Contains(message, want) {
			t.Fatalf("日志缺少 %q: %s", want, message)
		}
	}
}
