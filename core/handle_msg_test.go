package core

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"jiying/middleware/utils"
)

func TestDispatchSysCommand(t *testing.T) {
	var called atomic.Bool
	cmdRouter.Store("hello", func() { called.Store(true) })
	defer cmdRouter.Delete("hello")

	snapshot := Functions
	Functions = append([]*Function(nil), snapshot...)
	Functions = append(Functions, &Function{
		Title:  "问候",
		Handle: "hello",
		Origin: true,
	})
	defer func() { Functions = snapshot }()

	s := &Faker{Message: "hello 小明", Type: 2, UserID: "u1", ChatID: "c1"}
	if !HandleMessage(s) {
		t.Fatal("HandleMessage 应命中 hello 命令")
	}
	waitTrue(&called)
}

// finishFake 在异步处理结束时关闭 done，供测试等待后台 goroutine 完全退出。
type finishFake struct {
	*Faker
	done chan struct{}
	once sync.Once
}

func (f *finishFake) FinishAck() {
	f.once.Do(func() { close(f.done) })
}

func TestDispatchScript(t *testing.T) {
	if ScriptEngine == nil {
		return
	}
	snapshot := Functions
	Functions = append([]*Function(nil), snapshot...)
	Functions = append(Functions, &Function{
		Title:         "demo",
		Handle:        "",
		PluginPath:    "demo.js",
		PluginContent: `log("dispatch ok");`,
		PluginLang:    "js",
		Origin:        false,
	})
	defer func() { Functions = snapshot }()

	done := make(chan struct{})
	s := &finishFake{Faker: &Faker{Message: "demo test", Type: 2, UserID: "u1", ChatID: "c1"}, done: done}
	if !HandleMessage(s) {
		t.Fatal("HandleMessage 应命中 demo 脚本")
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("脚本执行未在超时内结束")
	}
}

func TestDispatchDisabledSkip(t *testing.T) {
	snapshot := Functions
	Functions = append([]*Function(nil), snapshot...)
	Functions = append(Functions, &Function{
		Title:   "demo",
		Handle:  "demo",
		Origin:  true,
		Disable: true,
	})
	defer func() { Functions = snapshot }()

	s := &Faker{Message: "demo", Type: 2, UserID: "u1", ChatID: "c1"}
	if HandleMessage(s) {
		t.Fatal("禁用插件不应被分发")
	}
}

func TestDispatchNoMatch(t *testing.T) {
	snapshot := Functions
	defer func() { Functions = snapshot }()

	s := &Faker{Message: "zzz_nothing_match", Type: 2, UserID: "u1", ChatID: "c1"}
	if HandleMessage(s) {
		t.Fatal("无匹配命令时应返回 false")
	}
}

func TestAgentWakePrompt(t *testing.T) {
	const appID = "app-wake-test"
	setupAgentBucket(t, "agent-wake")
	resetAppReplyBudgetForTest()
	appConnPool.mu.Lock()
	oldItems, oldDefault := appConnPool.items, appConnPool.defaultID
	appConnPool.items = map[string]*appRuntime{appID: {cfg: AppConn{AppID: appID}}}
	appConnPool.defaultID = appID
	appConnPool.mu.Unlock()
	defer func() {
		appConnPool.mu.Lock()
		appConnPool.items, appConnPool.defaultID = oldItems, oldDefault
		appConnPool.mu.Unlock()
	}()
	self := &JiyingClient{cfg: JiyingConfig{AppID: appID}}
	mention := "{(@app/" + appID + ")}"

	tests := []struct {
		name   string
		sender *JiyingSender
		want   string
		ok     bool
	}{
		{
			name:   "private ordinary text",
			sender: &JiyingSender{ConvType: "app", SenderType: "user", Text: "你好"},
			want:   "你好",
			ok:     true,
		},
		{
			name:   "group without mention",
			sender: &JiyingSender{ConvType: "group", SenderType: "user", Text: "你好"},
			ok:     false,
		},
		{
			name:   "group mention",
			sender: &JiyingSender{ConvType: "group", SenderType: "user", Text: mention + " 请介绍自己"},
			want:   "请介绍自己",
			ok:     true,
		},
		{
			name:   "group mention only",
			sender: &JiyingSender{ConvType: "group", SenderType: "user", Text: mention},
			want:   "对方@了你，请简短回应",
			ok:     true,
		},
		{
			name:   "app sender without id",
			sender: &JiyingSender{ConvType: "app", SenderType: "app", Text: "自动消息"},
			ok:     false,
		},
		{
			name:   "app private peer",
			sender: &JiyingSender{client: self, ChatID: "c1", ConvType: "app", SenderType: "app", UserID: "app-peer", Text: "自动消息"},
			want:   "自动消息",
			ok:     true,
		},
		{
			name:   "app group mention",
			sender: &JiyingSender{client: self, ConvType: "group", SenderType: "app", UserID: "app-peer", ChatID: "g1", Text: mention + " 请回复"},
			want:   "请回复",
			ok:     true,
		},
		{
			name:   "app group without mention",
			sender: &JiyingSender{client: self, ConvType: "group", SenderType: "app", UserID: "app-peer", ChatID: "g1", Text: "自动消息"},
			ok:     false,
		},
		{
			name:   "app cannot wake self",
			sender: &JiyingSender{client: self, ConvType: "group", SenderType: "app", UserID: appID, ChatID: "g1", Text: mention + " 自问自答"},
			ok:     false,
		},
		{
			name:   "hidden characters",
			sender: &JiyingSender{ConvType: "app", SenderType: "user", Text: "你\u200b好"},
			want:   "你好",
			ok:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := agentWakePrompt(tt.sender)
			if ok != tt.ok || got != tt.want {
				t.Fatalf("agentWakePrompt() = (%q, %v), want (%q, %v)", got, ok, tt.want, tt.ok)
			}
		})
	}
}

func resetAppReplyBudgetForTest() {
	appReplyBudget.Lock()
	appReplyBudget.loaded = false
	appReplyBudget.bucket = ""
	appReplyBudget.day = ""
	appReplyBudget.counts = map[string]int{}
	appReplyBudget.Unlock()
}

func TestAppReplyBudgetPersistsAndResets(t *testing.T) {
	setupAgentBucket(t, "app-reply-budget")
	resetAppReplyBudgetForTest()
	for i := 0; i < appReplyBudgetPerConv; i++ {
		if !allowAppReplyBudget("conv-1") {
			t.Fatalf("第 %d 条应用回复不应被拒绝", i+1)
		}
	}
	if allowAppReplyBudget("conv-1") {
		t.Fatal("达到每日上限后应拒绝")
	}
	if !allowAppReplyBudget("conv-2") {
		t.Fatal("不同会话应独立计数")
	}

	// 模拟进程重启：内存清空后从存储恢复当日计数。
	resetAppReplyBudgetForTest()
	if allowAppReplyBudget("conv-1") {
		t.Fatal("重启后不应重置当日配额")
	}
	if !allowAppReplyBudget("conv-2") {
		t.Fatal("重启后应继续保留其他会话计数")
	}

	// 跨天自动清零。
	appReplyBudget.Lock()
	appReplyBudget.loaded = false
	appReplyBudget.bucket = ""
	appReplyBudget.day = ""
	appReplyBudget.counts = map[string]int{}
	appReplyBudget.Unlock()
	raw, err := json.Marshal(appReplyBudgetStore{
		Day:    time.Now().AddDate(0, 0, -1).Format("2006-01-02"),
		Counts: map[string]int{"conv-1": appReplyBudgetPerConv},
	})
	if err != nil {
		t.Fatal(err)
	}
	BucketApp.Set(appReplyBudgetKey, string(raw))
	if !allowAppReplyBudget("conv-1") {
		t.Fatal("跨天后配额应重置")
	}
}

func TestAgentFailureMessage(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want string
	}{
		{name: "timeout", err: context.DeadlineExceeded, want: "Agent 处理超时，请缩短问题或稍后重试。"},
		{name: "canceled", err: context.Canceled, want: "Agent 执行已取消，请重试。"},
		{name: "not configured", err: errors.New("Agent 未配置：请设置 USER_LLM_BASE_URL"), want: "Agent 尚未配置，请联系管理员。"},
		{name: "auth", err: errors.New("模型 HTTP 401"), want: "Agent 模型鉴权失败，请联系管理员。"},
		{name: "rate limit", err: errors.New("模型 HTTP 429"), want: "Agent 模型限流，请稍后重试。"},
		{name: "turns", err: errors.New("Agent 超过最大轮次"), want: "Agent 轮次已用尽，请拆分为更小的问题。"},
		{name: "unknown", err: errors.New("boom"), want: "Agent 执行失败，请稍后重试。"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := agentFailureMessage(tt.err); got != tt.want {
				t.Fatalf("agentFailureMessage() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBoundary(t *testing.T) {
	if matchFunctionCommand(&Function{Title: "demo"}, "demonstrate x") {
		t.Fatal("demo 不应匹配 demonstrate")
	}
	if !matchFunctionCommand(&Function{Title: "demo"}, "demo x") {
		t.Fatal("demo 应匹配 demo x")
	}
	if !matchFunctionCommand(&Function{Title: "demo"}, "demo") {
		t.Fatal("精确标题应匹配")
	}
}

func TestHandleMsgLoopBroadcast(t *testing.T) {
	Senders = make(chan Sender, 4)
	resetImChannels()
	defer func() {
		Senders = make(chan Sender, 100)
		resetImChannels()
	}()

	ch := RegisterImChannel("test-ch")
	go handleMsgLoop()

	snapshot := Functions
	defer func() { Functions = snapshot }()

	Senders <- &Faker{Message: "no_such_cmd_xyz", Type: 2, UserID: "u1", ChatID: "c1"}
	select {
	case got := <-ch:
		if !strings.Contains(got.GetMsg(), "no_such_cmd_xyz") {
			t.Fatalf("广播内容错误: %q", got.GetMsg())
		}
	case <-time.After(1 * time.Second):
		t.Fatal("未命中的消息应广播到 im 通道")
	}
}

func TestTrimHiddenCharacter(t *testing.T) {
	in := "he\u200bllo\u2060\uFEFF 命令"
	got := utils.TrimHiddenCharacter(in)
	if got != "hello 命令" {
		t.Fatalf("TrimHiddenCharacter got %q", got)
	}
}

func TestExecuteFunctionAdminGate(t *testing.T) {
	setupRolesBucket(t, "exec-admin-gate")
	var called atomic.Bool
	cmdRouter.Store("adminprobe", func() { called.Store(true) })
	defer cmdRouter.Delete("adminprobe")

	fn := &Function{Title: "管理探针", Handle: "adminprobe", Origin: true, Admin: true}

	s := &Faker{UserID: "u1", ChatID: "c1"}
	withSender(s, func() { executeFunction(fn) })
	if called.Load() {
		t.Fatal("非主控不应执行管理命令")
	}
	if s.Message != "无权限" {
		t.Fatalf("非主控应收到无权限提示, got %q", s.Message)
	}

	BucketApp.Set("masters", "u1")
	withSender(s, func() { executeFunction(fn) })
	if !called.Load() {
		t.Fatal("主控应能执行管理命令")
	}
}

func TestFunctionAdminOnly(t *testing.T) {
	tru := true
	cases := []struct {
		name string
		fn   *Function
		want bool
	}{
		{"nil", nil, false},
		{"plain", &Function{}, false},
		{"flag", &Function{Admin: true}, true},
		{"manifest", &Function{Manifest: PluginManifest{Admin: &tru}}, true},
		{"manifest false", &Function{Manifest: PluginManifest{Admin: new(bool)}}, false},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.fn.adminOnly(); got != tt.want {
				t.Fatalf("adminOnly() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSplit(t *testing.T) {
	got := utils.Split("status 参数")
	if len(got) != 2 || got[0] != "status" || got[1] != "参数" {
		t.Fatalf("Split got %v", got)
	}
	if e := utils.ExpandChineseRegexSpace("a\u3000b"); e != "a[ \\t\u3000]b" {
		t.Fatalf("ExpandChineseRegexSpace got %q", e)
	}
}

func waitTrue(b *atomic.Bool) {
	for i := 0; i < 50; i++ {
		if b.Load() {
			return
		}
	}
}
