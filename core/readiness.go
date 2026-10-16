package core

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
)

type readinessIssue struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Hint    string `json:"hint,omitempty"`
}

type readinessCheck struct {
	Status string           `json:"status"`
	Issues []readinessIssue `json:"issues,omitempty"`
}

func checkURL(raw string, schemes ...string) bool {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return false
	}
	for _, scheme := range schemes {
		if strings.EqualFold(u.Scheme, scheme) {
			return true
		}
	}
	return false
}

// uuidRe 匹配标准 8-4-4-4-12 十六进制 UUID。
var uuidRe = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// isUUID 校验应用 ID。MagicChat 要求应用通道的 App ID 为 UUID，
// 格式不符时握手会直接返回 400，所以保存前就要拦住。
func isUUID(s string) bool {
	return uuidRe.MatchString(strings.TrimSpace(s))
}

func appReadiness(cfg AppConn) readinessCheck {
	issues := make([]readinessIssue, 0, 2)
	if !checkURL(cfg.WSURL, "ws", "wss") {
		issues = append(issues, readinessIssue{Code: "invalid_ws_url", Message: "应用地址必须是 ws:// 或 wss:// 地址", Hint: "检查应用通道地址和端口"})
	}
	if id := strings.TrimSpace(cfg.AppID); id == "" {
		issues = append(issues, readinessIssue{Code: "missing_app_id", Message: "缺少应用 ID", Hint: "填写 MagicChat 应用 ID"})
	} else if !isUUID(id) {
		issues = append(issues, readinessIssue{Code: "invalid_app_id", Message: "应用 ID 不是合法 UUID", Hint: "从 MagicChat 应用详情复制 UUID（8-4-4-4-12），格式不符会导致应用握手 400"})
	}
	if strings.TrimSpace(cfg.AppSecret) == "" {
		issues = append(issues, readinessIssue{Code: "missing_app_secret", Message: "缺少应用密钥", Hint: "填写应用密钥后保存"})
	}
	if len(issues) > 0 {
		return readinessCheck{Status: "invalid", Issues: issues}
	}
	return readinessCheck{Status: "configured"}
}

func accountReadiness(cfg UserLogin) readinessCheck {
	issues := make([]readinessIssue, 0, 2)
	if !checkURL(cfg.BaseURL, "http", "https") {
		issues = append(issues, readinessIssue{Code: "invalid_base_url", Message: "服务地址必须是 http:// 或 https:// 地址", Hint: "检查账号服务地址和端口"})
	}
	if strings.TrimSpace(cfg.Email) == "" {
		issues = append(issues, readinessIssue{Code: "missing_email", Message: "缺少账号邮箱", Hint: "填写用于登录 MagicChat 的邮箱"})
	}
	if strings.TrimSpace(cfg.Password) == "" {
		issues = append(issues, readinessIssue{Code: "missing_secret", Message: "缺少登录凭据", Hint: "填写密码或使用验证码登录"})
	}
	if len(issues) > 0 {
		return readinessCheck{Status: "invalid", Issues: issues}
	}
	return readinessCheck{Status: "configured"}
}

// agentReadiness 汇报自然语言 Agent 是否可用；未启用时按“未配置”处理。
func agentReadiness(cfg AgentConfig) readinessCheck {
	if !cfg.Enabled {
		return readinessCheck{Status: "missing", Issues: []readinessIssue{{Code: "agent_disabled", Message: "Agent 未启用", Hint: "启用后可让用户用自然语言调用功能"}}}
	}
	issues := make([]readinessIssue, 0, 3)
	if !checkURL(cfg.BaseURL, "http", "https") {
		issues = append(issues, readinessIssue{Code: "invalid_agent_base_url", Message: "模型地址必须是 http:// 或 https:// 地址", Hint: "填写 OpenAI 兼容接口地址"})
	}
	if strings.TrimSpace(cfg.Model) == "" {
		issues = append(issues, readinessIssue{Code: "missing_agent_model", Message: "缺少模型名称", Hint: "填写服务商支持的模型名"})
	}
	if strings.TrimSpace(cfg.APIKey) == "" {
		issues = append(issues, readinessIssue{Code: "missing_agent_key", Message: "缺少模型密钥", Hint: "填写 API Key 后保存"})
	}
	if len(issues) > 0 {
		return readinessCheck{Status: "invalid", Issues: issues}
	}
	return readinessCheck{Status: "ready"}
}

func connectionReadiness() map[string]any {
	apps := loadAppConns()
	app := readinessCheck{Status: "missing", Issues: []readinessIssue{{Code: "missing_app", Message: "尚未配置应用通道", Hint: "先添加一个 MagicChat 应用连接"}}}
	if len(apps) > 0 {
		app = appReadiness(apps[0])
		if app.Status == "configured" {
			if AppChannelConnected() {
				app.Status = "connected"
			} else if msg := AppChannelLastError(); msg != "" {
				app.Issues = append(app.Issues, readinessIssue{
					Code:    "app_connect_failed",
					Message: msg,
					Hint:    "核对应用地址、ID、连接密钥，并确认应用已启用且未重置密钥",
				})
			}
		}
	}
	account := accountReadiness(CurrentAccountConfig())
	if account.Status == "configured" && userLoggedIn() {
		account.Status = "connected"
	}
	return map[string]any{
		"storage": map[string]any{"status": map[bool]string{true: "ready", false: "blocked"}[BucketApp != nil]},
		"app":     app,
		"account": account,
		"agent":   agentReadiness(LoadAgentConfig()),
	}
}

func testAppConnection(ctx context.Context, cfg AppConn) error {
	check := appReadiness(cfg)
	if check.Status != "configured" {
		if len(check.Issues) > 0 {
			return fmt.Errorf("%s", check.Issues[0].Message)
		}
		return fmt.Errorf("应用配置无效")
	}
	testCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	client := NewJiyingClient(JiyingConfig{WSURL: cfg.WSURL, AppID: cfg.AppID, Secret: cfg.AppSecret}, nil)
	sock, err := client.dial(testCtx)
	if err != nil {
		return fmt.Errorf("应用握手失败: %w", err)
	}
	return sock.Close()
}
