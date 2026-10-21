package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizeFeatureApps(t *testing.T) {
	got := normalizeFeatureApps([]string{" app-a ", "APP-A", "", "app-b", "app-b"})
	if len(got) != 2 || got[0] != "app-a" || got[1] != "app-b" {
		t.Fatalf("去空与忽略大小写去重失败: %#v", got)
	}
	if star := normalizeFeatureApps([]string{"app-a", "*", "app-b"}); len(star) != 1 || star[0] != "*" {
		t.Fatalf("包含 * 时应归一为全部应用: %#v", star)
	}
}

func TestUpdateFeatureSettingsLocksApps(t *testing.T) {
	setupAgentBucket(t, "feature-settings")
	if err := UpsertFeature(Feature{Name: "设置范围-xyz", Kind: "http", Target: "https://example.com/a", Scope: "public", Identity: "app", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	scope, identity := "master", "user"
	confirm := true
	err := UpdateFeatureSettings("设置范围-xyz", featurePatch{
		Scope:    &scope,
		Identity: &identity,
		Confirm:  &confirm,
		Apps:     []string{" app-a ", "APP-A", "app-b"},
		SetApps:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	f, ok := GetFeature("设置范围-xyz")
	if !ok {
		t.Fatal("功能应存在")
	}
	if f.Scope != "master" || f.Identity != "user" || !f.Confirm {
		t.Fatalf("权限字段未更新: %+v", f)
	}
	if !f.AppsLocked || len(f.Apps) != 2 || f.Apps[0] != "app-a" || f.Apps[1] != "app-b" {
		t.Fatalf("应用范围未锁定或未归一: %+v", f)
	}
	// 插件重新扫描只能更新分类与服务，不能覆盖手动范围。
	features.setScriptMeta("设置范围-xyz", "工具类", "svc", []string{"plugin-app"})
	f, _ = GetFeature("设置范围-xyz")
	if len(f.Apps) != 2 || f.Apps[0] != "app-a" {
		t.Fatalf("手动范围应抵御清单覆盖: %#v", f.Apps)
	}
	if f.Class != "工具类" || f.Service != "svc" {
		t.Fatalf("分类与服务仍应随清单更新: %+v", f)
	}
}

func TestResetFeatureAppsReadsManifest(t *testing.T) {
	setupAgentBucket(t, "feature-reset")
	if err := os.MkdirAll(scriptsDir(), 0o755); err != nil {
		t.Fatal(err)
	}
	code := "// [title: 清单范围-xyz]\n// [apps: app-a, app-b]\nresult = 1;\n"
	if err := os.WriteFile(filepath.Join(scriptsDir(), "manifest-xyz.js"), []byte(code), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := UpsertFeature(Feature{Name: "清单范围-xyz", Kind: "script", Target: "manifest-xyz.js", Scope: "master", Identity: "app", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	if err := UpdateFeatureSettings("清单范围-xyz", featurePatch{Apps: []string{"only-manual"}, SetApps: true}); err != nil {
		t.Fatal(err)
	}
	if f, _ := GetFeature("清单范围-xyz"); !f.AppsLocked {
		t.Fatal("保存后应处于手动范围")
	}
	if err := ResetFeatureApps("清单范围-xyz"); err != nil {
		t.Fatal(err)
	}
	f, _ := GetFeature("清单范围-xyz")
	if f.AppsLocked {
		t.Fatal("恢复后应解除范围锁定")
	}
	if len(f.Apps) != 2 || f.Apps[0] != "app-a" || f.Apps[1] != "app-b" {
		t.Fatalf("应恢复插件清单声明的范围: %#v", f.Apps)
	}
}

func TestSetAppSystemAndSystemForApp(t *testing.T) {
	setupAgentBucket(t, "agent-app-system")
	cfg := LoadAgentConfig()
	cfg.System = "全局提示"
	SaveAgentConfig(cfg)
	if err := SetAppSystem(" app-a ", " 应用提示 "); err != nil {
		t.Fatal(err)
	}
	got := LoadAgentConfig()
	if got.SystemForApp("app-a") != "应用提示" {
		t.Fatalf("应命中应用覆盖: %q", got.SystemForApp("app-a"))
	}
	if got.SystemForApp("app-b") != "全局提示" {
		t.Fatalf("未配置的应用应继承全局: %q", got.SystemForApp("app-b"))
	}
	if got.SystemForApp("") != "全局提示" {
		t.Fatalf("空应用应继承全局: %q", got.SystemForApp(""))
	}
	if err := SetAppSystem("app-a", "   "); err != nil {
		t.Fatal(err)
	}
	if got := LoadAgentConfig(); got.SystemForApp("app-a") != "全局提示" {
		t.Fatalf("清空覆盖后应继承全局: %q", got.SystemForApp("app-a"))
	}
	if err := SetAppSystem("", "x"); err == nil {
		t.Fatal("缺少应用 ID 应报错")
	}
}

func TestAppIDForConversationUsesOwner(t *testing.T) {
	setupAgentBucket(t, "conv-owner")
	rememberConvApp("conv-owner-xyz", "app-owner")
	if got := appIDForConversation("conv-owner-xyz"); got != "app-owner" {
		t.Fatalf("应返回会话归属应用，实际 %q", got)
	}
}

func TestBuildToolContractsForAppFiltersByScope(t *testing.T) {
	setupAgentBucket(t, "tools-app-scope")
	if err := UpsertFeature(Feature{Name: "范围工具-甲", Kind: "http", Target: "https://example.com/a", Scope: "public", Identity: "app", Enabled: true, Apps: []string{"app-a"}}); err != nil {
		t.Fatal(err)
	}
	if err := UpsertFeature(Feature{Name: "范围工具-全体", Kind: "http", Target: "https://example.com/b", Scope: "public", Identity: "app", Enabled: true, Apps: []string{"*"}}); err != nil {
		t.Fatal(err)
	}
	names := func(appID string) map[string]bool {
		out := map[string]bool{}
		for _, c := range BuildToolContractsForApp("system", appID) {
			out[c.Feature] = true
		}
		return out
	}
	a := names("app-a")
	if !a["范围工具-甲"] || !a["范围工具-全体"] {
		t.Fatalf("app-a 应看到两个功能: %#v", a)
	}
	b := names("app-b")
	if b["范围工具-甲"] || !b["范围工具-全体"] {
		t.Fatalf("app-b 不应看到被限定的功能: %#v", b)
	}
}

func TestSenderAppIDFallsBackToConversation(t *testing.T) {
	setupAgentBucket(t, "sender-app")
	rememberConvApp("conv-sender-xyz", "app-sender")
	if got := senderAppID(nil); got != "" {
		t.Fatalf("nil sender 应返回空，实际 %q", got)
	}
}
