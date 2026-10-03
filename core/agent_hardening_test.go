package core

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestBuildArgsForTool(t *testing.T) {
	cases := []struct {
		name   string
		raw    string
		params []string
		want   string
	}{
		{name: "msg envelope", raw: `{"args":"hello"}`, params: nil, want: "hello"},
		{name: "single named", raw: `{"city":"北京"}`, params: []string{"city:城市名称"}, want: "北京"},
		{name: "multi named ordered", raw: `{"b":"2","a":"1"}`, params: []string{"a:一", "b:二"}, want: "1 2"},
		{name: "named numeric", raw: `{"count":5}`, params: []string{"count:数量"}, want: "5"},
		{name: "empty named", raw: `{"city":""}`, params: []string{"city:城市"}, want: ""},
		{name: "no params plain json", raw: `{"city":"北京"}`, params: nil, want: ""},
		{name: "non json passthrough", raw: "北京 晴", params: nil, want: "北京 晴"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := buildArgsForTool(c.raw, c.params); got != c.want {
				t.Fatalf("buildArgsForTool(%q, %v) = %q, want %q", c.raw, c.params, got, c.want)
			}
		})
	}
}

func TestAgentHistoryPersistsToBucket(t *testing.T) {
	setupAgentBucket(t, "agent-hist-persist")
	resetAgentHistory("ph-1")
	appendAgentHistory("ph-1", chatMessage{Role: "user", Content: "你好"})
	appendAgentHistory("ph-1", chatMessage{Role: "assistant", Content: "有什么可以帮你"})
	raw := BucketApp.GetString(agentHistKey("ph-1"))
	if raw == "" {
		t.Fatal("history should be persisted to bucket")
	}
	var list []chatMessage
	if err := json.Unmarshal([]byte(raw), &list); err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 || list[1].Content != "有什么可以帮你" {
		t.Fatalf("unexpected persisted history: %+v", list)
	}
	resetAgentHistory("ph-1")
	if BucketApp.GetString(agentHistKey("ph-1")) != "" {
		t.Fatal("reset should delete persisted history")
	}
}

func TestAgentHistoryTrimsOverlongMessage(t *testing.T) {
	setupAgentBucket(t, "agent-hist-trim")
	resetAgentHistory("trim-1")
	long := strings.Repeat("x", agentMaxMsgLen+100)
	appendAgentHistory("trim-1", chatMessage{Role: "user", Content: long})
	got := getAgentHistory("trim-1")
	if len(got) != 1 {
		t.Fatalf("want 1 message, got %d", len(got))
	}
	if len(got[0].Content) > agentMaxMsgLen {
		t.Fatalf("message should be trimmed to %d, got %d", agentMaxMsgLen, len(got[0].Content))
	}
}

func TestAgentHistoryEvictsOldestConvs(t *testing.T) {
	setupAgentBucket(t, "agent-hist-evict")
	for i := 0; i < agentMaxConvs+8; i++ {
		appendAgentHistory("evc-"+string(rune('A'+i%26))+string(rune('a'+i/26%26)), chatMessage{Role: "user", Content: "m"})
	}
	agentHistory.Lock()
	order := append([]string(nil), agentHistory.order...)
	agentHistory.Unlock()
	if len(order) > agentMaxConvs {
		t.Fatalf("history should cap at %d convs, got %d", agentMaxConvs, len(order))
	}
	if len(order) == 0 {
		t.Fatal("history should keep recent convs")
	}
	oldest := order[0]
	if BucketApp.GetString(agentHistKey(oldest)) == "" {
		t.Fatalf("most recent conv %q should persist", oldest)
	}
}

func TestRunAgentToolRoutesThroughQueue(t *testing.T) {
	setupAgentBucket(t, "agent-tool-queue")
	resetTaskQueue()
	initTaskQueue()
	out := runAgentTool(context.Background(), "帮助", `{"args":""}`, "system", "conv-tq-1")
	if !strings.Contains(out, "帮助") {
		t.Fatalf("tool should return 帮助 output, got %q", out)
	}
	// 静默任务完成后应被清除，不残留内存与持久层。
	for _, it := range ListTaskItems(50) {
		if it != nil && it.Silent {
			t.Fatalf("silent task should not be listed: %+v", it)
		}
	}
	if raw := BucketApp.GetString("jiying_tasks"); strings.Contains(raw, "conv-tq-1") || strings.Contains(raw, "agent.tool") {
		t.Fatalf("silent tasks should not be persisted: %s", raw)
	}
}

func TestCompleteWithFallbackAuditsAllFailedModels(t *testing.T) {
	setupAgentBucket(t, "agent-fallback-audit")
	BucketApp.Delete("jiying_audit")
	// a 与 b 均失败，c 成功：审计目标应为实际使用的 c，明细需列出全部不可用模型。
	call := func(_ context.Context, model string) (*chatResponse, error) {
		if model == "c" {
			return &chatResponse{}, nil
		}
		return nil, fmt.Errorf("模型 %s 拒绝", model)
	}
	resp, err := completeWithFallback(context.Background(), []string{"a", "b", "c"}, call)
	if err != nil || resp == nil {
		t.Fatalf("降级应成功命中 c，resp=%+v err=%v", resp, err)
	}
	list := ListAudit(10)
	if len(list) != 1 {
		t.Fatalf("应记录一条降级审计，got %d", len(list))
	}
	e := list[0]
	if e.Target != "c" || !strings.Contains(e.Detail, "a") || !strings.Contains(e.Detail, "b") {
		t.Fatalf("降级审计应指向 c 且列出 a/b：%+v", e)
	}
	if e.Result != "降级成功" {
		t.Fatalf("降级成功不应判为失败：%q", e.Result)
	}
}

func TestCompleteWithFallbackAllFail(t *testing.T) {
	setupAgentBucket(t, "agent-fallback-allfail")
	BucketApp.Delete("jiying_audit")
	call := func(_ context.Context, model string) (*chatResponse, error) {
		return nil, fmt.Errorf("模型 %s 不可用", model)
	}
	if _, err := completeWithFallback(context.Background(), []string{"a", "b"}, call); err == nil {
		t.Fatal("全部失败时应返回错误")
	}
	list := ListAudit(10)
	if len(list) != 1 {
		t.Fatalf("应记录一条降级审计，got %d", len(list))
	}
	if list[0].Target != "a" || !strings.Contains(list[0].Detail, "b") {
		t.Fatalf("全部失败审计应指向主模型并列出后备：%+v", list[0])
	}
	if !strings.Contains(list[0].Result, "全部失败") {
		t.Fatalf("全部失败结果应判为失败：%q", list[0].Result)
	}
}

func TestCompleteWithFallbackPrimarySuccessNoAudit(t *testing.T) {
	setupAgentBucket(t, "agent-fallback-primary")
	BucketApp.Delete("jiying_audit")
	call := func(_ context.Context, _ string) (*chatResponse, error) {
		return &chatResponse{}, nil
	}
	if _, err := completeWithFallback(context.Background(), []string{"a", "b"}, call); err != nil {
		t.Fatalf("主模型成功不应报错：%v", err)
	}
	if list := ListAudit(10); len(list) != 0 {
		t.Fatalf("主模型成功不应产生降级审计：%+v", list)
	}
	if _, err := completeWithFallback(context.Background(), nil, call); err == nil {
		t.Fatal("空模型列表应报错")
	}
}
