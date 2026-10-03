package core

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestScriptLangOf(t *testing.T) {
	cases := map[string]string{
		"a.js":  "js",
		"a.mjs": "js",
		"a.py":  "python",
		"a.PY":  "python",
		"a.ts":  "",
		"a.txt": "",
		"a":     "",
	}
	for name, want := range cases {
		if got := scriptLangOf(name); got != want {
			t.Fatalf("scriptLangOf(%q)=%q want %q", name, got, want)
		}
	}
}

func TestInstallScriptRejectsUnsupportedExt(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("JIYING_RES", dir)

	if err := InstallScript("x.txt", "hello"); err == nil {
		t.Fatal("非 .js/.py 扩展名应被拒绝")
	}
	if err := InstallScript("x.ts", "console.log(1)"); err == nil {
		t.Fatal(".ts 应被拒绝")
	}
	if _, err := os.Stat(filepath.Join(dir, "scripts", "x.ts")); err == nil {
		t.Fatal("不支持的类型不应落盘")
	}
	assertNoTempScript(t, filepath.Join(dir, "scripts"))
}

func TestDeliverScriptOutput(t *testing.T) {
	var sent []string
	got := deliverScriptOutput("line1\nSEND:hello\nline2\n", func(s string) { sent = append(sent, s) })
	if got != "line1\nline2" {
		t.Fatalf("带会话上下文应剔除 SEND 行，实际 %q", got)
	}
	if len(sent) != 1 || sent[0] != "hello" {
		t.Fatalf("SEND 行应投递一次，实际 %v", sent)
	}
	if keep := deliverScriptOutput("\nSEND:only\n", nil); keep != "SEND:only" {
		t.Fatalf("无会话上下文应保留 SEND 行，实际 %q", keep)
	}
}

func TestPythonScriptInstallAndRun(t *testing.T) {
	requirePython(t)
	dir := t.TempDir()
	t.Setenv("JIYING_RES", dir)

	code := "import os\n" +
		"args = os.environ.get('JIYING_ARGS', '')\n" +
		"print('hi ' + args)\n"
	if err := InstallScript("hello.py", code); err != nil {
		t.Fatalf("安装 Python 脚本失败: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "scripts", "hello.py")); err != nil {
		t.Fatalf("脚本未落盘: %v", err)
	}
	assertNoTempScript(t, filepath.Join(dir, "scripts"))

	got, err := runScriptFile("hello.py", "world")
	if err != nil {
		t.Fatalf("运行 Python 脚本失败: %v", err)
	}
	if got != "hi world" {
		t.Fatalf("输出=%q，期望 %q", got, "hi world")
	}
}

func TestPythonSyntaxErrorLeavesNoFile(t *testing.T) {
	requirePython(t)
	dir := t.TempDir()
	t.Setenv("JIYING_RES", dir)

	if err := InstallScript("bad.py", "def broken(:\n"); err == nil {
		t.Fatal("语法错误应导致安装失败")
	}
	if _, err := os.Stat(filepath.Join(dir, "scripts", "bad.py")); err == nil {
		t.Fatal("语法错误不应落盘")
	}
	assertNoTempScript(t, filepath.Join(dir, "scripts"))
}

func TestPythonRuntimeErrorFailsTrialRun(t *testing.T) {
	requirePython(t)
	dir := t.TempDir()
	t.Setenv("JIYING_RES", dir)

	if err := InstallScript("boom.py", "raise SystemExit(3)\n"); err == nil {
		t.Fatal("试跑非零退出应导致安装失败")
	}
	if _, err := os.Stat(filepath.Join(dir, "scripts", "boom.py")); err == nil {
		t.Fatal("试跑失败不应落盘")
	}
}

func requirePython(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath(pythonBin()); err != nil {
		t.Skip("未找到 python3，跳过 Python 脚本测试")
	}
}
