package core

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestNormalizeAgentLimitsDefaultsAndCaps(t *testing.T) {
	cfg := AgentConfig{}
	normalizeAgentLimits(&cfg)
	if cfg.MaxTurns != 6 || cfg.MaxHistory != 12 || cfg.MaxContextBytes != agentDefaultContext || cfg.MaxTools != agentDefaultTools {
		t.Fatalf("默认预算不正确: %+v", cfg)
	}
	cfg = AgentConfig{MaxTurns: 99, MaxHistory: 999, MaxContextBytes: 1, MaxTools: 999}
	normalizeAgentLimits(&cfg)
	if cfg.MaxTurns != agentMaxTurns || cfg.MaxHistory != agentMaxHistory || cfg.MaxContextBytes != agentMinContextBytes || cfg.MaxTools != agentMaxTools {
		t.Fatalf("预算上限裁剪不正确: %+v", cfg)
	}
}

func TestTrimAgentHistoryDropsOldest(t *testing.T) {
	msgs := []chatMessage{
		{Role: "user", Content: strings.Repeat("a", 100)},
		{Role: "assistant", Content: strings.Repeat("b", 100)},
		{Role: "user", Content: strings.Repeat("c", 100)},
	}
	got := trimAgentHistory(msgs, 220)
	if len(got) != 2 || !strings.HasPrefix(got[0].Content, "b") || !strings.HasPrefix(got[1].Content, "c") {
		t.Fatalf("应丢弃最旧历史: %+v", got)
	}
	if len(trimAgentHistory(msgs, 0)) != 0 {
		t.Fatal("零预算应清空历史")
	}
}

func TestRunAgentStopsAtToolBudget(t *testing.T) {
	setupAgentBucket(t, "agent-tool-budget")
	const feature = "预算测试工具"
	if err := UpsertFeature(Feature{Name: feature, Kind: "builtin", Target: "help", Scope: "public", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	var calls int32
	var secondTools int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		var req struct {
			Tools    []any         `json:"tools"`
			Messages []chatMessage `json:"messages"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		if n == 1 {
			tool := agentToolName(feature)
			_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","tool_calls":[` +
				`{"id":"c1","type":"function","function":{"name":"` + tool + `","arguments":"{}"}},` +
				`{"id":"c2","type":"function","function":{"name":"` + tool + `","arguments":"{}"}}]}}]}`))
			return
		}
		atomic.StoreInt32(&secondTools, int32(len(req.Tools)))
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"已按预算作答"}}]}`))
	}))
	defer srv.Close()

	SaveAgentConfig(AgentConfig{Enabled: true, BaseURL: srv.URL, APIKey: "test-key", Model: "fake", MaxTurns: 4, MaxTools: 1})
	resetAgentHistory("conv-budget")
	out, err := RunAgent(context.Background(), "conv-budget", "system", "帮我看看", 0)
	if err != nil {
		t.Fatal(err)
	}
	if out != "已按预算作答" {
		t.Fatalf("unexpected output: %q", out)
	}
	if atomic.LoadInt32(&calls) != 2 {
		t.Fatalf("预算用尽后应只再调用一次模型，got %d", calls)
	}
	if atomic.LoadInt32(&secondTools) != 0 {
		t.Fatalf("预算用尽后不应再下发工具定义，got %d", secondTools)
	}
}
