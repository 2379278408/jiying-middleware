package core

import (
	"errors"
	"strings"
	"testing"
)

func TestSafeExecuteRecoversPanic(t *testing.T) {
	_, err := safeExecute(func() (string, error) { panic("boom") })
	if err == nil || !strings.Contains(err.Error(), "功能执行异常") {
		t.Fatalf("panic 应转换为错误, got %v", err)
	}
}

func TestSafeExecutePassesThrough(t *testing.T) {
	got, err := safeExecute(func() (string, error) { return "ok", nil })
	if err != nil || got != "ok" {
		t.Fatalf("正常路径应透传, got %q err=%v", got, err)
	}

	sentinel := errors.New("exec failed")
	if _, err := safeExecute(func() (string, error) { return "", sentinel }); !errors.Is(err, sentinel) {
		t.Fatalf("业务错误应透传, got %v", err)
	}
}
