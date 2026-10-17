package core

import (
	"strings"
	"testing"
)

func TestAgentSystemPromptInjectsIdentity(t *testing.T) {
	got := agentSystemPrompt("你是助手", "admin")
	if !strings.Contains(got, "你是助手") || !strings.Contains(got, "当前用户ID：admin") {
		t.Fatalf("缺少基础提示或身份信息: %q", got)
	}
	if !strings.Contains(got, "主控权限") {
		t.Fatalf("主控用户未标注权限: %q", got)
	}
	if !strings.Contains(got, agentGroundRule) {
		t.Fatalf("缺少可信度约束: %q", got)
	}
	if got := agentSystemPrompt("", "user-1"); !strings.Contains(got, "当前用户ID：user-1（普通用户）") {
		t.Fatalf("空系统提示应追加身份: %q", got)
	}
	if got := agentSystemPrompt("仅基础", ""); !strings.Contains(got, "仅基础") || strings.Contains(got, "当前用户ID") {
		t.Fatalf("无用户时不应追加身份: %q", got)
	}
}

func TestAgentToolChoiceForRealtimeIntent(t *testing.T) {
	defs := []map[string]any{
		{"type": "function", "function": map[string]any{
			"name":        "tool_0",
			"description": "执行中间件功能「天气」（类型 script）",
		}},
	}
	if got := agentToolChoice("今天北京天气怎么样", defs); got != "required" {
		t.Fatalf("实时意图应强制工具: %q", got)
	}
	if got := agentToolChoice("你好", defs); got != "" {
		t.Fatalf("普通闲聊不应强制工具: %q", got)
	}
	if got := agentToolChoice("今天天气如何", nil); got != "" {
		t.Fatalf("无工具时不应强制: %q", got)
	}
}

func TestSanitizeAgentReplyDowngradesMarkdownImage(t *testing.T) {
	in := "已生成图片：![二维码](https://example.com/a.png)\n其余内容"
	got := sanitizeAgentReply(in)
	if strings.Contains(got, "![") {
		t.Fatalf("Markdown 图片未降级: %q", got)
	}
	if !strings.Contains(got, "[图片：二维码](https://example.com/a.png)") {
		t.Fatalf("降级链接格式不符: %q", got)
	}
	if got := sanitizeAgentReply("普通文本"); got != "普通文本" {
		t.Fatalf("普通文本不应变化: %q", got)
	}
}

func TestScriptFailureDetailKeepsStdout(t *testing.T) {
	got := scriptFailureDetail(`{"error":"鉴权失败"}`, "", errTest("exit status 1"))
	if !strings.Contains(got, "stdout") || !strings.Contains(got, "鉴权失败") {
		t.Fatalf("失败详情应包含 stdout: %q", got)
	}
	got = scriptFailureDetail("", "traceback", errTest("exit status 1"))
	if !strings.Contains(got, "stderr") || !strings.Contains(got, "traceback") {
		t.Fatalf("失败详情应包含 stderr: %q", got)
	}
	if got := scriptFailureDetail("", "", errTest("boom")); got != "boom" {
		t.Fatalf("无输出时应回退错误文本: %q", got)
	}
}

type errTest string

func (e errTest) Error() string { return string(e) }

func TestRecentSilentTasksRingBuffer(t *testing.T) {
	q := &taskQueue{}
	for i := 0; i < silentTaskHistory+10; i++ {
		q.appendSilentLocked(&Task{ID: "t" + string(rune('0'+i%10))})
	}
	if len(q.silentRecent) != silentTaskHistory {
		t.Fatalf("环形缓冲长度 = %d, want %d", len(q.silentRecent), silentTaskHistory)
	}
}
