package core

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

func TestScriptEnvDropsPlatformSecrets(t *testing.T) {
	t.Setenv("JIYING_SECRET_TOKEN", "leak")
	t.Setenv("MCAI_LLM_API_KEY", "leak")
	env := scriptEnv(map[string]string{"HOME": "/sandbox"})
	joined := strings.Join(env, "\n")
	for _, forbidden := range []string{"JIYING_SECRET_TOKEN", "MCAI_LLM_API_KEY"} {
		if strings.Contains(joined, forbidden) {
			t.Fatalf("环境白名单泄漏 %s: %v", forbidden, env)
		}
	}
	if !strings.Contains(joined, "HOME=/sandbox") {
		t.Fatalf("显式覆盖的 HOME 应生效: %v", env)
	}
	if _, ok := os.LookupEnv("PATH"); ok && !strings.Contains(joined, "PATH=") {
		t.Fatal("PATH 应保留")
	}
}

func TestScriptRiskReasons(t *testing.T) {
	cases := []struct {
		name string
		code string
		want string
	}{
		{"a.py", "import subprocess\nsubprocess.run(['ls'])", "执行子进程"},
		{"a.py", "import requests\nrequests.get('https://example.com')", "网络请求"},
		{"a.js", "const cp = require('child_process')", "执行子进程"},
		{"a.js", "fetch('https://example.com')", "网络请求"},
		{"a.js", "eval('1+1')", "动态执行代码"},
	}
	for _, c := range cases {
		got := strings.Join(scriptRiskReasons(c.name, c.code), ",")
		if !strings.Contains(got, c.want) {
			t.Fatalf("%s 未识别 %s，实际 %q", c.name, c.want, got)
		}
		if msg := scriptRiskMessage(c.name, c.code); !strings.Contains(msg, "confirm=true") {
			t.Fatalf("确认提示缺少 confirm=true: %q", msg)
		}
	}
	if got := scriptRiskReasons("a.js", "result = 1 + 1"); len(got) != 0 {
		t.Fatalf("普通脚本不应触发确认: %v", got)
	}
}

func TestPythonScriptSandboxWorkdir(t *testing.T) {
	requirePython(t)
	dir := t.TempDir()
	t.Setenv("JIYING_RES", dir)
	code := "import os\n" +
		"print(os.getcwd())\n" +
		"print(os.environ.get('JIYING_WORK_DIR'))\n" +
		"print(os.environ.get('JIYING_PLUGIN_DIR'))\n"
	if err := InstallScript("wd.py", code); err != nil {
		t.Fatalf("安装失败: %v", err)
	}
	out, err := runScriptFile("wd.py", "")
	if err != nil {
		t.Fatalf("运行失败: %v", err)
	}
	lines := strings.Split(out, "\n")
	if len(lines) != 3 {
		t.Fatalf("输出行数=%d: %q", len(lines), out)
	}
	if lines[0] != lines[1] || !strings.Contains(filepath.Base(lines[0]), "jiying-script-") {
		t.Fatalf("应在独立工作目录执行，cwd=%q work=%q", lines[0], lines[1])
	}
	if lines[2] != filepath.Join(dir, "scripts") {
		t.Fatalf("JIYING_PLUGIN_DIR=%q，期望 %q", lines[2], filepath.Join(dir, "scripts"))
	}
}

func TestPythonScriptSandboxNetworkDisabled(t *testing.T) {
	requirePython(t)
	if _, err := exec.LookPath("unshare"); err != nil {
		t.Skip("未找到 unshare，跳过网络命名空间测试")
	}
	dir := t.TempDir()
	t.Setenv("JIYING_RES", dir)
	if _, err := exec.Command("unshare", "-n", "true").CombinedOutput(); err != nil {
		t.Skip("当前环境不允许创建网络命名空间")
	}
	code := "# [network: false]\nimport socket\nprint(','.join(sorted(n for _, n in socket.if_nameindex())))\n"
	if err := InstallScript("offline.py", code); err != nil {
		t.Fatalf("安装失败: %v", err)
	}
	out, err := runScriptFile("offline.py", "")
	if err != nil {
		t.Fatalf("运行失败: %v", err)
	}
	if out != "lo" {
		t.Fatalf("[network: false] 应只看到 lo，实际 %q", out)
	}
}

func TestPythonScriptResourceLimits(t *testing.T) {
	requirePython(t)
	if _, err := exec.LookPath("prlimit"); err != nil {
		t.Skip("未找到 prlimit，跳过资源限制测试")
	}
	dir := t.TempDir()
	t.Setenv("JIYING_RES", dir)
	code := "import resource\nprint(resource.getrlimit(resource.RLIMIT_AS)[0])\n"
	if err := InstallScript("limits.py", code); err != nil {
		t.Fatalf("安装失败: %v", err)
	}
	out, err := runScriptFile("limits.py", "")
	if err != nil {
		t.Fatalf("运行失败: %v", err)
	}
	if out != "268435456" {
		t.Fatalf("RLIMIT_AS=%q，期望 268435456", out)
	}
}
