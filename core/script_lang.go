package core

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"jiying/middleware/utils"
)

const (
	pythonScriptTimeout  = 15 * time.Second
	maxScriptOutputBytes = 256 * 1024
	scriptMemoryLimit    = 256 * 1024 * 1024
	scriptCPUQuota       = 15
	scriptMaxProcesses   = 64
	scriptMaxOpenFiles   = 128
)

// scriptLangOf 依据文件扩展名判定脚本语言，返回 "js" / "python"；不支持的类型返回 ""。
func scriptLangOf(name string) string {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".js", ".mjs":
		return "js"
	case ".py":
		return "python"
	default:
		return ""
	}
}

func isPythonLang(lang string) bool {
	switch strings.ToLower(strings.TrimSpace(lang)) {
	case "python", "py":
		return true
	}
	return false
}

// scriptTitle 去掉脚本扩展名，作为功能名与展示名。
func scriptTitle(name string) string {
	base := filepath.Base(name)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

func pythonBin() string {
	if v := strings.TrimSpace(os.Getenv("JIYING_PYTHON")); v != "" {
		return v
	}
	return "python3"
}

// scriptEnv 只放行运行脚本必需的环境变量，避免把进程内的平台凭据泄漏给用户脚本。
// extra 中的值覆盖同名白名单变量，调用方可用 sandbox 的 HOME/TMPDIR 替换宿主路径。
func scriptEnv(extra map[string]string) []string {
	allow := []string{"PATH", "LANG", "LC_ALL", "LC_CTYPE", "TZ", "PYTHONIOENCODING"}
	merged := map[string]string{}
	for _, k := range allow {
		if v, ok := os.LookupEnv(k); ok {
			merged[k] = v
		}
	}
	for k, v := range extra {
		merged[k] = v
	}
	keys := make([]string, 0, len(merged))
	for k := range merged {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	env := make([]string, 0, len(keys))
	for _, k := range keys {
		env = append(env, k+"="+merged[k])
	}
	return env
}

// scriptSandbox 描述一次脚本执行的工作目录、插件目录与网络策略。
type scriptSandbox struct {
	Dir       string
	PluginDir string
	Network   bool
	cleanup   func()
}

// newScriptSandbox 为单次执行创建独立工作目录，并读取插件清单里的网络策略。
// 工作目录隔离 HOME/TMPDIR 与相对路径写入，插件目录只通过环境变量暴露。
func newScriptSandbox(pluginPath string) (*scriptSandbox, error) {
	dir, err := os.MkdirTemp("", "jiying-script-")
	if err != nil {
		return nil, fmt.Errorf("创建脚本工作目录失败: %w", err)
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		_ = os.RemoveAll(dir)
		return nil, fmt.Errorf("设置脚本工作目录权限失败: %w", err)
	}
	network := true
	if data, err := os.ReadFile(pluginPath); err == nil {
		if m := ParsePluginManifest(string(data)); m.Network != nil {
			network = *m.Network
		}
	}
	return &scriptSandbox{
		Dir:       dir,
		PluginDir: filepath.Dir(pluginPath),
		Network:   network,
		cleanup:   func() { _ = os.RemoveAll(dir) },
	}, nil
}

// sandboxCommand 组合无网络命名空间、rlimit 与解释器命令。
// prlimit 不可用时退化为普通子进程，仍保留超时与进程组终止。
func sandboxCommand(ctx context.Context, sandbox *scriptSandbox, bin string, argv []string) *exec.Cmd {
	cmdArgs := append([]string(nil), argv...)
	if prlimit, err := exec.LookPath("prlimit"); err == nil {
		limit := []string{
			fmt.Sprintf("--as=%d", scriptMemoryLimit),
			fmt.Sprintf("--cpu=%d", scriptCPUQuota),
			fmt.Sprintf("--nproc=%d", scriptMaxProcesses),
			fmt.Sprintf("--nofile=%d", scriptMaxOpenFiles),
			"--",
			bin,
		}
		cmdArgs = append(limit, cmdArgs...)
		bin = prlimit
	}
	if !sandbox.Network {
		if unshare, err := exec.LookPath("unshare"); err == nil {
			flags := []string{"-n", "--"}
			if os.Geteuid() != 0 {
				// 非 root 运行需要同时创建用户命名空间，才能获得 CAP_SYS_ADMIN 建立网络命名空间。
				flags = []string{"-Urn", "--"}
			}
			cmdArgs = append(append(flags, bin), cmdArgs...)
			bin = unshare
		}
	}
	return exec.CommandContext(ctx, bin, cmdArgs...)
}

// limitWriter 限制写入总量，超出部分丢弃，避免脚本输出撑爆内存。
type limitWriter struct {
	buf   bytes.Buffer
	limit int
}

func (w *limitWriter) Write(p []byte) (int, error) {
	if w.buf.Len() < w.limit {
		remain := w.limit - w.buf.Len()
		if len(p) <= remain {
			w.buf.Write(p)
		} else {
			w.buf.Write(p[:remain])
		}
	}
	return len(p), nil
}

func (w *limitWriter) String() string { return w.buf.String() }

// validatePythonScript 只做语法解析，不执行脚本，用于安装前的静态校验。
func validatePythonScript(path string) error {
	const check = "import ast, sys\n" +
		"with open(sys.argv[1], 'r', encoding='utf-8') as fh:\n" +
		"    ast.parse(fh.read(), sys.argv[1])\n"
	ctx, cancel := context.WithTimeout(context.Background(), pythonScriptTimeout)
	defer cancel()
	sandbox, err := newScriptSandbox(path)
	if err != nil {
		return err
	}
	defer sandbox.cleanup()
	cmd := sandboxCommand(ctx, sandbox, pythonBin(), []string{"-I", "-c", check, path})
	configureProcessGroup(cmd)
	cmd.Dir = sandbox.Dir
	cmd.Env = scriptEnv(map[string]string{"HOME": sandbox.Dir, "TMPDIR": sandbox.Dir})
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("语法校验超时")
		}
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("语法错误: %s", msg)
	}
	return nil
}

// scriptConsoleEnv 为脚本注入本机控制台地址与专用脚本账号，插件可直接调用
// /jiying/api/* 发送消息或上传文件，无需自行猜测端口。
// 控制台口令是随机生成的 script 账号令牌，不向脚本暴露即应账号密码；
// 需要以即应账号身份调用接口时使用 JIYING_ACCOUNT_*，密码始终留在服务端。
func scriptConsoleEnv() map[string]string {
	port := 8080
	if BucketApp != nil {
		if p := BucketApp.GetInt("port"); p != 0 {
			port = p
		}
	}
	env := map[string]string{
		"JIYING_PORT":     strconv.Itoa(port),
		"JIYING_BASE_URL": fmt.Sprintf("http://127.0.0.1:%d", port),
	}
	if token := scriptConsoleToken(); token != "" {
		env["JIYING_CONSOLE_USER"] = ScriptConsoleUser
		env["JIYING_CONSOLE_PASS"] = token
	}
	cfg := CurrentAccountConfig()
	if cfg.BaseURL != "" {
		env["JIYING_ACCOUNT_BASE_URL"] = cfg.BaseURL
	}
	if cfg.Email != "" {
		env["JIYING_ACCOUNT_EMAIL"] = cfg.Email
	}
	return env
}

// scriptFailureDetail 合并 stderr 与 stdout，便于插件失败时定位真实原因。
// 非零退出时 stdout 常含插件自己打印的错误信息，不能被丢弃。
func scriptFailureDetail(stdout, stderr string, err error) string {
	parts := make([]string, 0, 2)
	if s := strings.TrimSpace(stderr); s != "" {
		parts = append(parts, "stderr:\n"+s)
	}
	if s := strings.TrimSpace(stdout); s != "" {
		parts = append(parts, "stdout:\n"+s)
	}
	if len(parts) == 0 {
		return err.Error()
	}
	return truncateRunes(strings.Join(parts, "\n"), 2000)
}

// runPythonScript 在子进程里执行 Python 脚本，捕获标准输出作为结果。
// 标准输出中以 "SEND:" 开头的行会被当作会话消息回发；无会话上下文时忽略。
func runPythonScript(path, args, convID, replyTo string, cfgEnv map[string]string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), pythonScriptTimeout)
	defer cancel()
	sandbox, err := newScriptSandbox(path)
	if err != nil {
		return "", err
	}
	defer sandbox.cleanup()
	argv := []string{"-I", path}
	if fields := strings.Fields(args); len(fields) > 0 {
		argv = append(argv, fields...)
	}
	cmd := sandboxCommand(ctx, sandbox, pythonBin(), argv)
	configureProcessGroup(cmd)
	cmd.Dir = sandbox.Dir
	extra := map[string]string{
		"JIYING_ARGS":       args,
		"JIYING_CONV_ID":    convID,
		"JIYING_REPLY_TO":   replyTo,
		"JIYING_RES_HOME":   utils.GetResHome(),
		"JIYING_WORK_DIR":   sandbox.Dir,
		"JIYING_PLUGIN_DIR": sandbox.PluginDir,
		"HOME":              sandbox.Dir,
		"TMPDIR":            sandbox.Dir,
		"TEMP":              sandbox.Dir,
		"TMP":               sandbox.Dir,
	}
	for k, v := range scriptConsoleEnv() {
		extra[k] = v
	}
	for k, v := range cfgEnv {
		extra[k] = v
	}
	cmd.Env = scriptEnv(extra)
	out := &limitWriter{limit: maxScriptOutputBytes}
	errBuf := &limitWriter{limit: maxScriptOutputBytes}
	cmd.Stdout = out
	cmd.Stderr = errBuf
	runErr := cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		return "", fmt.Errorf("脚本执行超时")
	}
	var emit func(string)
	if convID != "" {
		emit = func(text string) { sendScriptText(convID, replyTo, text) }
	}
	result := deliverScriptOutput(redactSecretText(out.String()), emit)
	if runErr != nil {
		return "", fmt.Errorf("%s", redactSecretText(scriptFailureDetail(out.String(), errBuf.String(), runErr)))
	}
	return result, nil
}

// scriptRiskReasons 识别需要二次确认的高风险脚本能力。
func scriptRiskReasons(name, code string) []string {
	lower := strings.ToLower(code)
	lang := scriptLangOf(name)
	checks := []struct {
		lang   string
		needle string
		reason string
	}{
		{"python", "os.system", "执行系统命令"},
		{"python", "os.popen", "执行系统命令"},
		{"python", "subprocess", "执行子进程"},
		{"python", "shutil.rmtree", "删除目录"},
		{"python", "socket.", "网络连接"},
		{"python", "requests.", "网络请求"},
		{"python", "urllib.", "网络请求"},
		{"python", "httpx.", "网络请求"},
		{"python", "ftplib", "网络连接"},
		{"python", "paramiko", "网络连接"},
		{"python", "eval(", "动态执行代码"},
		{"python", "exec(", "动态执行代码"},
		{"js", "child_process", "执行子进程"},
		{"js", "require('fs')", "读写文件"},
		{`js`, `require("fs")`, "读写文件"},
		{"js", "readfile", "读写文件"},
		{"js", "writefile", "读写文件"},
		{"js", "fetch(", "网络请求"},
		{"js", "xmlhttprequest", "网络请求"},
		{"js", "eval(", "动态执行代码"},
		{"js", "new function", "动态执行代码"},
	}
	seen := map[string]bool{}
	var out []string
	for _, c := range checks {
		if c.lang != lang || seen[c.reason] || !strings.Contains(lower, c.needle) {
			continue
		}
		seen[c.reason] = true
		out = append(out, c.reason)
	}
	return out
}

// scriptRiskMessage 生成统一的安装确认提示。
func scriptRiskMessage(name, code string) string {
	reasons := scriptRiskReasons(name, code)
	if len(reasons) == 0 {
		return ""
	}
	return "脚本包含高风险能力（" + strings.Join(reasons, "、") + "），安装前需要确认；确认后请带 confirm=true 重试"
}

// deliverScriptOutput 解析标准输出：有 emit 时把 SEND: 行投递并剔除，
// 无会话上下文（emit 为 nil）时保留原样，返回其余内容作为结果。
func deliverScriptOutput(stdout string, emit func(string)) string {
	lines := strings.Split(stdout, "\n")
	kept := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimRight(line, "\r")
		if emit != nil && strings.HasPrefix(trimmed, "SEND:") {
			emit(strings.TrimPrefix(trimmed, "SEND:"))
			continue
		}
		kept = append(kept, trimmed)
	}
	return strings.TrimSpace(strings.Join(kept, "\n"))
}

func sendScriptText(convID, replyTo, text string) {
	if convID == "" || strings.TrimSpace(text) == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if _, err := SendMediaToConv(ctx, convID, replyTo, map[string]any{"type": "text", "content": text}); err != nil {
		GetMyLog().Warn("Python 脚本回发消息失败：%v", err)
	}
}
