package core

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
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

func TestWaitForTaskDoneReturnsTerminalStatus(t *testing.T) {
	resetTaskQueue()
	defer resetTaskQueue()
	tasks.mu.Lock()
	tasks.items["timeout-task"] = &Task{ID: "timeout-task", Status: taskTimeout, Result: "执行超时"}
	tasks.mu.Unlock()
	got := WaitForTaskDone("timeout-task", 10*time.Millisecond)
	if got == nil || got.Status != taskTimeout {
		t.Fatalf("terminal timeout should return immediately, got %+v", got)
	}
}

func TestWaitForTaskDoneContextKeepsRunningSnapshot(t *testing.T) {
	resetTaskQueue()
	defer resetTaskQueue()
	tasks.mu.Lock()
	tasks.items["running-task"] = &Task{ID: "running-task", Status: taskRunning}
	tasks.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	got := WaitForTaskDoneContext(ctx, "running-task", time.Second)
	if got == nil || got.Status != taskRunning {
		t.Fatalf("context cancel should return running snapshot, got %+v", got)
	}
}
