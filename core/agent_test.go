package core

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func setupAgentBucket(t *testing.T, name string) {
	t.Helper()
	t.Setenv("JIYING_RES", t.TempDir())
	t.Setenv("JIYING_DATA", t.TempDir())
	t.Setenv("JIYING_SECRET_KEY", "00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff")
	b, err := MakeBucket(name, 6)
	if err != nil {
		t.Fatal(err)
	}
	BucketApp = b
	BucketApp.Delete(agentConfigKey)
	features.items = map[string]*Feature{}
	initFeatures()
	resetTaskQueue()
	initTaskQueue()
	t.Cleanup(func() { BucketApp.Delete(agentConfigKey) })
}

func TestBuildAgentToolsMapsEnabledFeatures(t *testing.T) {
	setupAgentBucket(t, "agent-tools")
	defs, index, _ := buildAgentTools("system")
	items := ListFeatureItems()
	if len(defs) != len(items)+1 {
		t.Fatalf("want %d tool defs (features + ask_choice), got %d", len(items)+1, len(defs))
	}
	if len(index) == 0 {
		t.Fatal("index must map tool names back to features")
	}
	for _, name := range index {
		if _, ok := GetFeature(name); !ok {
			t.Fatalf("indexed feature %q not found", name)
		}
	}
}

func TestBuildAgentToolsFiltersByPermission(t *testing.T) {
	setupAgentBucket(t, "agent-tools-perm")
	// system 拥有全部权限，普通用户只能看到非主控、非 Admin 的功能。
	all, _, _ := buildAgentTools("system")
	limited, index, _ := buildAgentTools("user-normal")
	if len(limited) >= len(all) {
		t.Fatalf("normal user should see fewer tools: all=%d limited=%d", len(all), len(limited))
	}
	if len(limited) == 0 {
		t.Fatal("normal user should still see public tools")
	}
	for _, name := range index {
		f, ok := GetFeature(name)
		if !ok {
			t.Fatalf("indexed feature %q not found", name)
		}
		if !featureAllowedFor("user-normal", f, "") {
			t.Fatalf("feature %q leaked into normal user tool list", name)
		}
	}
}

func TestAgentHistoryCap(t *testing.T) {
	setupAgentBucket(t, "agent-history")
	SaveAgentConfig(AgentConfig{MaxHistory: 3})
	resetAgentHistory("c1")
	for i := 0; i < 10; i++ {
		appendAgentHistory("c1", chatMessage{Role: "user", Content: "m"})
	}
	agentHistory.Lock()
	got := len(agentHistory.items["c1"])
	agentHistory.Unlock()
	if got != 3 {
		t.Fatalf("history should cap at 3, got %d", got)
	}
}

func TestRunAgentToolLoop(t *testing.T) {
	setupAgentBucket(t, "agent-loop")
	const feature = "循环测试工具"
	if err := UpsertFeature(Feature{Name: feature, Kind: "builtin", Target: "help", Scope: "public", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		w.Header().Set("Content-Type", "application/json")
		if n == 1 {
			// 第一轮：模型请求调用第一个工具。
			toolName := agentToolName(feature)
			_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","tool_calls":[{"id":"call_1","type":"function","function":{"name":"` + toolName + `","arguments":"{\"args\":\"\"}"}}]}}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"最终答复"}}]}`))
	}))
	defer srv.Close()

	SaveAgentConfig(AgentConfig{Enabled: true, BaseURL: srv.URL, APIKey: "test-key", Model: "fake", MaxTurns: 4})
	resetAgentHistory("conv-1")
	out, err := RunAgent(context.Background(), "conv-1", "system", "帮我看看", 0)
	if err != nil {
		t.Fatal(err)
	}
	if out != "最终答复" {
		t.Fatalf("unexpected agent output: %q", out)
	}
	if atomic.LoadInt32(&calls) < 2 {
		t.Fatal("agent should have performed at least two model calls")
	}
}

func TestAgentConfigSealsKey(t *testing.T) {
	setupAgentBucket(t, "agent-secret")
	SaveAgentConfig(AgentConfig{Enabled: true, BaseURL: "https://x", APIKey: "plain-key", Model: "m"})
	raw := BucketApp.GetString(agentConfigKey)
	if bytes.Contains([]byte(raw), []byte("plain-key")) {
		t.Fatalf("api key must be sealed at rest: %s", raw)
	}
	if got := LoadAgentConfig().APIKey; got != "plain-key" {
		t.Fatalf("key round trip mismatch: %q", got)
	}
}

func TestParseModelList(t *testing.T) {
	got := parseModelList("a, b\nc；d、e; a ,, ")
	want := []string{"a", "b", "c", "d", "e"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("parseModelList() = %v, want %v", got, want)
	}
}

func TestCandidateModelsKeepsPrimaryFirst(t *testing.T) {
	cfg := AgentConfig{Model: "a", Fallback: []string{" b ", "a", "", "c", "b"}}
	got := candidateModels(cfg)
	if strings.Join(got, ",") != "a,b,c" {
		t.Fatalf("candidateModels() = %v, want [a b c]", got)
	}
}

func TestAgentConfigFallbackRoundTrip(t *testing.T) {
	setupAgentBucket(t, "agent-fb-cfg")
	SaveAgentConfig(AgentConfig{Enabled: true, BaseURL: "https://x", APIKey: "k", Model: "m", Fallback: []string{"f1", "f2"}})
	got := LoadAgentConfig()
	if strings.Join(got.Fallback, ",") != "f1,f2" {
		t.Fatalf("fallback round trip mismatch: %v", got.Fallback)
	}
	// 保存空列表后应清空，不被环境变量或旧值回填。
	SaveAgentConfig(AgentConfig{Enabled: true, BaseURL: "https://x", APIKey: "k", Model: "m"})
	if len(LoadAgentConfig().Fallback) != 0 {
		t.Fatal("清空后备模型后不应保留旧值")
	}
}

func TestAgentFallsBackWhenPrimaryUnavailable(t *testing.T) {
	setupAgentBucket(t, "agent-fallback")
	var primaryCalls, backupCalls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Model string `json:"model"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		switch body.Model {
		case "primary":
			atomic.AddInt32(&primaryCalls, 1)
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"error":{"message":"No available channel for model primary"}}`))
		case "backup":
			atomic.AddInt32(&backupCalls, 1)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"降级成功"}}]}`))
		default:
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	defer srv.Close()

	SaveAgentConfig(AgentConfig{Enabled: true, BaseURL: srv.URL, APIKey: "test-key", Model: "primary", Fallback: []string{"backup"}, MaxTurns: 2})
	resetAgentHistory("conv-fb")
	out, err := RunAgent(context.Background(), "conv-fb", "system", "你好", 0)
	if err != nil {
		t.Fatal(err)
	}
	if out != "降级成功" {
		t.Fatalf("unexpected agent output: %q", out)
	}
	if atomic.LoadInt32(&primaryCalls) == 0 {
		t.Fatal("主模型应被调用一次")
	}
	if atomic.LoadInt32(&backupCalls) == 0 {
		t.Fatal("主模型不可用后应调用后备模型")
	}
}

// TestAgentToolCallKeepsContentField 回归：assistant 带 tool_calls 时 content 为空也必须序列化出来，
// 否则严格网关会以 missing field `content` 拒绝第二轮请求。
func TestAgentToolCallKeepsContentField(t *testing.T) {
	msg := chatMessage{Role: "assistant", ToolCalls: []chatToolCall{{ID: "call_1", Type: "function"}}}
	raw, err := json.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"content":""`) {
		t.Fatalf("assistant tool_call 消息必须保留 content 字段，got %s", raw)
	}
	user, err := json.Marshal(chatMessage{Role: "user", Content: "你好"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(user), "tool_calls") || strings.Contains(string(user), "tool_call_id") {
		t.Fatalf("普通消息不应携带工具字段，got %s", user)
	}
}

func TestBuildAgentToolsIncludesAskChoice(t *testing.T) {
	defs, _, _ := buildAgentTools("nobody")
	found := false
	for _, d := range defs {
		fn, _ := d["function"].(map[string]any)
		if fn != nil && fn["name"] == askChoiceTool {
			found = true
		}
	}
	if !found {
		t.Fatal("ask_choice tool missing from agent tool definitions")
	}
}

func TestResumeAgentChoiceUnknownReturnsFalse(t *testing.T) {
	if resumeAgentChoice("missing", "conv", "u1", []string{"yes"}) {
		t.Fatal("unknown choice must not resume")
	}
}
