package core

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"jiying/middleware/utils"
)

const (
	pythonScriptTimeout  = 15 * time.Second
	maxScriptOutputBytes = 256 * 1024
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
func scriptEnv(extra map[string]string) []string {
	allow := []string{"PATH", "HOME", "LANG", "LC_ALL", "LC_CTYPE", "TZ", "TMPDIR", "TEMP", "TMP", "PYTHONIOENCODING"}
	env := make([]string, 0, len(allow)+len(extra))
	for _, k := range allow {
		if v, ok := os.LookupEnv(k); ok {
			env = append(env, k+"="+v)
		}
	}
	for k, v := range extra {
		env = append(env, k+"="+v)
	}
	return env
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
	cmd := exec.CommandContext(ctx, pythonBin(), "-I", "-c", check, path)
	cmd.Env = scriptEnv(nil)
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

// scriptConsoleEnv 为脚本注入本机控制台地址与控制台账号，插件可直接调用
// /jiying/api/* 发送消息或上传文件，无需自行猜测端口。
// 仅管理员安装的脚本会拿到这些变量；普通用户脚本由功能权限控制。
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
	cfg := CurrentAccountConfig()
	if cfg.Email != "" {
		env["JIYING_CONSOLE_USER"] = cfg.Email
	}
	if cfg.Password != "" {
		env["JIYING_CONSOLE_PASS"] = cfg.Password
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
	argv := []string{"-I", path}
	if fields := strings.Fields(args); len(fields) > 0 {
		argv = append(argv, fields...)
	}
	cmd := exec.CommandContext(ctx, pythonBin(), argv...)
	extra := map[string]string{
		"JIYING_ARGS":     args,
		"JIYING_CONV_ID":  convID,
		"JIYING_REPLY_TO": replyTo,
		"JIYING_RES_HOME": utils.GetResHome(),
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
	result := deliverScriptOutput(out.String(), emit)
	if runErr != nil {
		return "", fmt.Errorf("%s", scriptFailureDetail(out.String(), errBuf.String(), runErr))
	}
	return result, nil
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
