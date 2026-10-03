package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dop251/goja"
)

func TestInstallScriptTrialFailureLeavesNoFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("JIYING_RES", dir)

	if err := InstallScript("bad.js", "throw new Error('boom')"); err == nil {
		t.Fatal("试跑失败应返回错误")
	}
	if _, err := os.Stat(filepath.Join(dir, "scripts", "bad.js")); err == nil {
		t.Fatal("试跑失败不应写入脚本")
	}
	assertNoTempScript(t, filepath.Join(dir, "scripts"))
}

func TestInstallScriptSuccessIsAtomic(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("JIYING_RES", dir)

	if err := InstallScript("ok.js", "result='yes'"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "scripts", "ok.js"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "result='yes'" {
		t.Fatalf("脚本内容不符: %q", string(data))
	}
	assertNoTempScript(t, filepath.Join(dir, "scripts"))
}

func TestInstallScriptRollbackRemovesNewScript(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("JIYING_RES", dir)

	prev := ScriptEngine
	ScriptEngine = goja.New()
	defer func() { ScriptEngine = prev }()
	// 预置中断，使热加载阶段的 RunString 立即失败，触发回滚分支。
	ScriptEngine.Interrupt("boom")

	if err := InstallScript("hot.js", "result='x'"); err == nil {
		t.Fatal("热加载失败应返回错误")
	}
	if _, err := os.Stat(filepath.Join(dir, "scripts", "hot.js")); err == nil {
		t.Fatal("回滚应删除新装脚本")
	}
	assertNoTempScript(t, filepath.Join(dir, "scripts"))
}

func TestRollbackScriptRestoresPreviousContent(t *testing.T) {
	dir := t.TempDir()
	full := filepath.Join(dir, "keep.js")
	if err := os.WriteFile(full, []byte("v1"), 0o644); err != nil {
		t.Fatal(err)
	}
	rollbackScript(full, []byte("v0"), true)
	data, err := os.ReadFile(full)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "v0" {
		t.Fatalf("回滚应恢复旧内容，实际 %q", string(data))
	}
}

func assertNoTempScript(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".install-") {
			t.Fatalf("残留临时文件: %s", e.Name())
		}
	}
}
