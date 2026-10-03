package core

import (
	"strings"
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

	s := &Faker{Message: "demo test", Type: 2, UserID: "u1", ChatID: "c1"}
	if !HandleMessage(s) {
		t.Fatal("HandleMessage 应命中 demo 脚本")
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
			sender: &JiyingSender{ConvType: "group", SenderType: "user", Text: "{(@app/" + appID + ")} 请介绍自己"},
			want:   "请介绍自己",
			ok:     true,
		},
		{
			name:   "group mention only",
			sender: &JiyingSender{ConvType: "group", SenderType: "user", Text: "{(@app/" + appID + ")}"},
			ok:     false,
		},
		{
			name:   "app sender",
			sender: &JiyingSender{ConvType: "app", SenderType: "app", Text: "自动消息"},
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
