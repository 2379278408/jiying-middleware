package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExecutionLedgerRecordsFiltersAndReloads(t *testing.T) {
	oldHome := DataHome
	DataHome = t.TempDir()
	defer func() { DataHome = oldHome }()
	executionLedger.Lock()
	oldItems := executionLedger.items
	executionLedger.items = nil
	executionLedger.Unlock()
	defer func() {
		executionLedger.Lock()
		executionLedger.items = oldItems
		executionLedger.Unlock()
	}()

	id := beginExecution(executionKindCron, "重置额度", "status", "admin")
	finishExecution(id, executionSuccess, "ok", "")
	id2 := beginExecution(executionKindManual, "手动", "help", "admin")
	finishExecution(id2, executionFailed, "", "boom")

	items := listExecutions(ExecutionFilter{Limit: 10})
	if len(items) != 2 || items[0].ID != id2 || items[0].Status != executionFailed {
		t.Fatalf("执行记录顺序或状态错误: %+v", items)
	}
	if got := listExecutions(ExecutionFilter{Kind: executionKindCron, Limit: 10}); len(got) != 1 || got[0].Name != "重置额度" {
		t.Fatalf("按类型筛选失败: %+v", got)
	}
	if got := listExecutions(ExecutionFilter{Name: "手动", Limit: 10}); len(got) != 1 {
		t.Fatalf("按名称筛选失败: %+v", got)
	}

	persistExecutions()
	executionLedger.Lock()
	executionLedger.items = nil
	executionLedger.Unlock()
	loadExecutions()
	items = listExecutions(ExecutionFilter{Limit: 10})
	if len(items) != 2 || items[0].DurationMS < 0 {
		t.Fatalf("落盘恢复失败: %+v", items)
	}
	info, err := os.Stat(filepath.Join(DataHome, "executions.json"))
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("执行记录权限错误: %v %v", info, err)
	}
}
