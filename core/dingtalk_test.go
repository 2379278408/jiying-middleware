package core

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDingTalkParseAndClassify(t *testing.T) {
	cases := []struct {
		raw    string
		reason string
	}{
		{`{"type":"user_im_message_receive_o2o_all","sender":"张三","content":"你好","conversation_id":"c1","message_id":"m1"}`, "dm"},
		{`{"type":"user_im_message_receive_at","sender":"李四","content":"在吗","conversation_id":"c2","message_id":"m2"}`, "mention"},
		{`{"type":"user_im_message_receive_at","sender":"王五","content":"@所有人 开会","conversation_id":"c3","message_id":"m3"}`, "at-all"},
		{`{"type":"user_im_message_receive_group_all","sender":"赵六","content":"普通群消息","conversation_id":"c4","message_id":"m4"}`, ""},
	}
	for _, tc := range cases {
		ev, ok := parseDingTalkEvent([]byte(tc.raw))
		if tc.reason == "" {
			if ok {
				t.Fatalf("未命中的群消息不应入库: %+v", ev)
			}
			continue
		}
		if !ok || ev.Reason != tc.reason {
			t.Fatalf("分类错误: got %+v want %s", ev, tc.reason)
		}
	}
}

func TestDingTalkToolArgsValidation(t *testing.T) {
	if _, err := dingTalkToolArgs("send_message", map[string]any{"text": "hi"}); err == nil {
		t.Fatal("缺少目标应拒绝发送")
	}
	args, err := dingTalkToolArgs("send_message", map[string]any{"text": "hi", "conversation_id": "cid"})
	if err != nil || !strings.Contains(strings.Join(args, " "), "--group cid") {
		t.Fatalf("群发送参数不正确: %v %v", args, err)
	}
	if _, err := dingTalkToolArgs("reply_message", map[string]any{"text": "hi"}); err == nil {
		t.Fatal("回复缺少 message_id 应拒绝")
	}
	if _, err := dingTalkToolArgs("search_messages", map[string]any{}); err == nil {
		t.Fatal("空搜索条件应拒绝")
	}
}

func TestAgentConfigForAppOverride(t *testing.T) {
	cfg := AgentConfig{Model: "base", MaxTurns: 4, Temperature: 0.2, AppOverrides: map[string]AppAgentOverride{
		"app-a": {Model: "special", MaxTurns: 9, Temperature: 1.1},
		"app-b": {Model: "model-b"},
	}}
	a := cfg.ForApp("app-a")
	if a.Model != "special" || a.MaxTurns != 9 || a.Temperature != 1.1 {
		t.Fatalf("应用覆盖未生效: %+v", a)
	}
	b := cfg.ForApp("app-b")
	if b.Model != "model-b" || b.MaxTurns != 4 || b.Temperature != 0.2 {
		t.Fatalf("未覆盖字段应继承全局: %+v", b)
	}
	if c := cfg.ForApp(""); c.Model != "base" {
		t.Fatalf("空应用应使用全局: %+v", c)
	}
}

func TestAgentToolNameStable(t *testing.T) {
	a := agentToolName("功能甲")
	b := agentToolName("功能甲")
	c := agentToolName("功能乙")
	if a != b || a == c {
		t.Fatalf("工具名应由功能名稳定派生: %s %s %s", a, b, c)
	}
	if !strings.HasPrefix(a, "tool_") || len(a) != len("tool_")+10 {
		t.Fatalf("工具名格式不正确: %s", a)
	}
}

func TestAppIDUnknownDoesNotFallbackDefault(t *testing.T) {
	appConnPool.mu.Lock()
	oldItems, oldDefault := appConnPool.items, appConnPool.defaultID
	appConnPool.items = map[string]*appRuntime{"def": {cfg: AppConn{AppID: "def"}}}
	appConnPool.defaultID = "def"
	appConnPool.mu.Unlock()
	defer func() {
		appConnPool.mu.Lock()
		appConnPool.items, appConnPool.defaultID = oldItems, oldDefault
		appConnPool.mu.Unlock()
	}()
	if got := appIDForConversation("unknown-conv"); got != "" {
		t.Fatalf("未知会话不得回退默认应用，got %q", got)
	}
}

func TestFeatureAllowedForAppDefaultDeny(t *testing.T) {
	appConnPool.mu.Lock()
	oldItems, oldDefault := appConnPool.items, appConnPool.defaultID
	appConnPool.items = map[string]*appRuntime{"def": {cfg: AppConn{AppID: "def"}}}
	appConnPool.defaultID = "def"
	appConnPool.mu.Unlock()
	defer func() {
		appConnPool.mu.Lock()
		appConnPool.items, appConnPool.defaultID = oldItems, oldDefault
		appConnPool.mu.Unlock()
	}()
	f := &Feature{Name: "noapps"}
	if !featureAllowedForApp(f, "def") {
		t.Fatal("默认应用应可见未声明范围的插件")
	}
	if featureAllowedForApp(f, "other") {
		t.Fatal("非默认应用不得自动继承未声明范围的插件")
	}
	star := &Feature{Name: "star", Apps: []string{"*"}}
	if !featureAllowedForApp(star, "other") {
		t.Fatal("* 应允许所有应用")
	}
}

func TestAppScopedRoleGrantsMasterWithinApp(t *testing.T) {
	setupAgentBucket(t, "app-role")
	appConnPool.mu.Lock()
	oldItems, oldDefault := appConnPool.items, appConnPool.defaultID
	appConnPool.items = map[string]*appRuntime{"def": {cfg: AppConn{AppID: "def"}}}
	appConnPool.defaultID = "def"
	appConnPool.mu.Unlock()
	defer func() {
		appConnPool.mu.Lock()
		appConnPool.items, appConnPool.defaultID = oldItems, oldDefault
		appConnPool.mu.Unlock()
	}()
	if err := SetAppUserRole("def", "u-app", RoleMaster); err != nil {
		t.Fatal(err)
	}
	masterFeature := &Feature{Name: "仅主控", Scope: "master", Enabled: true}
	if !featureAllowedFor("u-app", masterFeature, "def") {
		t.Fatal("应用级 master 应可执行本应用主控功能")
	}
	if featureAllowedFor("u-app", masterFeature, "other") {
		t.Fatal("应用级角色不应外溢到其他应用")
	}
	if role := UserRoleForApp("u-app", "def"); role != RoleMaster {
		t.Fatalf("应用级角色应生效，got %s", role)
	}
	if err := DeleteAppUserRole("def", "u-app"); err != nil {
		t.Fatal(err)
	}
}

func TestRecordAuditAppField(t *testing.T) {
	setupAgentBucket(t, "audit-app")
	RecordAuditApp("app-a", "tester", "demo", "target", "detail", "成功")
	list := ListAudit(1)
	if len(list) != 1 || list[0].AppID != "app-a" {
		raw, _ := json.Marshal(list)
		t.Fatalf("审计应带应用归属: %s", raw)
	}
}

func TestDingTalkToolsBoundToSingleApp(t *testing.T) {
	setupAppPoolBucket(t, "dingtalk-tool-binding")
	dingTalk.mu.Lock()
	oldCfg := dingTalk.cfg
	dingTalk.mu.Unlock()
	t.Cleanup(func() {
		dingTalk.mu.Lock()
		dingTalk.cfg = oldCfg
		dingTalk.mu.Unlock()
	})
	if err := saveDingTalkConfig(DingTalkConfig{Enabled: true, PushAppID: "app-a"}); err != nil {
		t.Fatal(err)
	}
	BucketApp.Set(appRolesKey, `{"app-a":{"ops-a":"ops"},"app-b":{"ops-b":"ops"}}`)
	if dingTalkToolAppID() != "app-a" {
		t.Fatalf("dingtalk tools must bind to push app, got %q", dingTalkToolAppID())
	}
	if !dingTalkToolAllowed("ops-a", "app-a") {
		t.Fatal("ops of bound app should use dingtalk tools")
	}
	if dingTalkToolAllowed("ops-b", "app-b") {
		t.Fatal("ops of another app must not use the shared dingtalk login")
	}
	if dingTalkToolAllowed("nobody", "app-a") {
		t.Fatal("readonly user must not use dingtalk tools")
	}
}
