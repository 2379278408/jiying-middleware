package core

import (
	"strings"
	"testing"
)

func TestAgentReadiness(t *testing.T) {
	cases := []struct {
		name   string
		cfg    AgentConfig
		status string
	}{
		{"disabled", AgentConfig{Enabled: false}, "missing"},
		{"missing_url", AgentConfig{Enabled: true, Model: "m", APIKey: "k"}, "invalid"},
		{"missing_model", AgentConfig{Enabled: true, BaseURL: "http://x/v1", APIKey: "k"}, "invalid"},
		{"missing_key", AgentConfig{Enabled: true, BaseURL: "http://x/v1", Model: "m"}, "invalid"},
		{"ready", AgentConfig{Enabled: true, BaseURL: "http://x/v1", Model: "m", APIKey: "k"}, "ready"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := agentReadiness(tc.cfg); got.Status != tc.status {
				t.Fatalf("status = %q, want %q", got.Status, tc.status)
			}
		})
	}
}

func TestAccountReadinessCodeLogin(t *testing.T) {
	code := accountReadiness(UserLogin{BaseURL: "https://h", Email: "a@b.c", CodeLogin: true})
	if code.Status != "configured" {
		t.Fatalf("验证码账号应 configured，got %q (%+v)", code.Status, code.Issues)
	}
	missing := accountReadiness(UserLogin{BaseURL: "https://h", Email: "a@b.c"})
	if missing.Status != "invalid" || len(missing.Issues) == 0 || missing.Issues[0].Code != "missing_secret" {
		t.Fatalf("无密码且非验证码应 invalid，got %q (%+v)", missing.Status, missing.Issues)
	}
	noEmail := accountReadiness(UserLogin{BaseURL: "https://h", CodeLogin: true})
	if noEmail.Status != "invalid" {
		t.Fatalf("缺邮箱仍应 invalid，got %q", noEmail.Status)
	}
}

func TestIsUUID(t *testing.T) {
	cases := map[string]bool{
		"a6567dc0-9bef-872a-4c3e-8f0a1b2c3d4e":     true,
		"A6567DC0-9BEF-872A-4C3E-8F0A1B2C3D4E":     true,
		"  a6567dc0-9bef-872a-4c3e-8f0a1b2c3d4e  ": true,
		"":                                     false,
		"not-a-uuid":                           false,
		"a6567dc09bef872a4c3e8f0a1b2c3d4e":     false,
		"a6567dc0-9bef-872a-4c3e":              false,
		"g6567dc0-9bef-872a-4c3e-8f0a1b2c3d4e": false,
	}
	for in, want := range cases {
		if got := isUUID(in); got != want {
			t.Fatalf("isUUID(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestAppReadinessValidatesUUID(t *testing.T) {
	bad := appReadiness(AppConn{WSURL: "wss://h/api/app/ws", AppID: "not-a-uuid", AppSecret: "s"})
	if bad.Status != "invalid" {
		t.Fatalf("非法 UUID 应判 invalid，got %q", bad.Status)
	}
	if len(bad.Issues) == 0 || bad.Issues[0].Code != "invalid_app_id" {
		t.Fatalf("应给出 invalid_app_id，got %+v", bad.Issues)
	}
	if !strings.Contains(bad.Issues[0].Hint, "400") {
		t.Fatalf("提示应说明握手会 400，got %q", bad.Issues[0].Hint)
	}
	ok := appReadiness(AppConn{WSURL: "wss://h/api/app/ws", AppID: "a6567dc0-9bef-872a-4c3e-8f0a1b2c3d4e", AppSecret: "s"})
	if ok.Status != "configured" {
		t.Fatalf("合法 UUID 应 configured，got %q", ok.Status)
	}
}
