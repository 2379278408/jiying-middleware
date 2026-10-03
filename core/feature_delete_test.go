package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDeleteFeatureRemovesPluginFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("JIYING_RES", dir)
	if err := os.MkdirAll(scriptsDir(), 0o755); err != nil {
		t.Fatal(err)
	}
	full := filepath.Join(scriptsDir(), "gone.js")
	if err := os.WriteFile(full, []byte("result='x'"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := UpsertFeature(Feature{Name: "gone", Kind: "script", Target: "gone.js", Scope: "master", Identity: "app", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	if err := deleteFeatureWithPlugin("gone"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(full); err == nil {
		t.Fatal("删除插件型功能应同时移除插件文件")
	}
	if _, ok := GetFeature("gone"); ok {
		t.Fatal("功能应已删除")
	}
}

func TestDeleteHTTPFeatureRemovesEntryOnly(t *testing.T) {
	if err := UpsertFeature(Feature{Name: "天气-xyz", Kind: "http", Target: "https://example.com/api", Scope: "public", Identity: "app", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	if err := deleteFeatureWithPlugin("天气-xyz"); err != nil {
		t.Fatal(err)
	}
	if _, ok := GetFeature("天气-xyz"); ok {
		t.Fatal("功能应已删除")
	}
}

func TestDeleteFeatureMissingErrors(t *testing.T) {
	if err := deleteFeatureWithPlugin("不存在的功能-xyz"); err == nil {
		t.Fatal("删除不存在的功能应返回错误")
	}
}

func TestDeleteScriptFeatureIgnoresUnsafeTarget(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("JIYING_RES", dir)
	outside := filepath.Join(dir, "victim.txt")
	if err := os.WriteFile(outside, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := UpsertFeature(Feature{Name: "逃逸-xyz", Kind: "script", Target: "../victim.txt", Scope: "master", Identity: "app", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	if err := deleteFeatureWithPlugin("逃逸-xyz"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(outside); err != nil {
		t.Fatal("越权目标不应被删除")
	}
}

func TestDeleteFeatureKeepsSharedPluginFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("JIYING_RES", dir)
	if err := os.MkdirAll(scriptsDir(), 0o755); err != nil {
		t.Fatal(err)
	}
	full := filepath.Join(scriptsDir(), "shared.js")
	if err := os.WriteFile(full, []byte("result='x'"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, n := range []string{"共享甲-xyz", "共享乙-xyz"} {
		if err := UpsertFeature(Feature{Name: n, Kind: "script", Target: "shared.js", Scope: "master", Identity: "app", Enabled: true}); err != nil {
			t.Fatal(err)
		}
	}
	if err := deleteFeatureWithPlugin("共享甲-xyz"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(full); err != nil {
		t.Fatal("仍有功能引用时不应删除插件文件")
	}
	if err := deleteFeatureWithPlugin("共享乙-xyz"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(full); err == nil {
		t.Fatal("最后一个引用删除后应移除文件")
	}
}

func TestDeleteDisabledScriptFeatureRemovesFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("JIYING_RES", dir)
	if err := os.MkdirAll(scriptsDir(), 0o755); err != nil {
		t.Fatal(err)
	}
	full := filepath.Join(scriptsDir(), "off.js")
	if err := os.WriteFile(full, []byte("result='x'"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := UpsertFeature(Feature{Name: "停用插件-xyz", Kind: "script", Target: "off.js", Scope: "master", Identity: "app", Enabled: false}); err != nil {
		t.Fatal(err)
	}
	if err := deleteFeatureWithPlugin("停用插件-xyz"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(full); err == nil {
		t.Fatal("停用的插件功能删除时也应移除文件")
	}
}

func TestDeleteScriptFeatureUnloadsPluginFromMemory(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("JIYING_RES", dir)
	if err := os.MkdirAll(scriptsDir(), 0o755); err != nil {
		t.Fatal(err)
	}
	full := filepath.Join(scriptsDir(), "unload.js")
	if err := os.WriteFile(full, []byte("result='x'"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := UpsertFeature(Feature{Name: "卸载插件-xyz", Kind: "script", Target: "unload.js", Scope: "master", Identity: "app", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	backup := Functions
	appendFunction(&Function{Title: "卸载插件-xyz", PluginPath: full})
	t.Cleanup(func() { Functions = backup })

	if err := deleteFeatureWithPlugin("卸载插件-xyz"); err != nil {
		t.Fatal(err)
	}
	for _, f := range FunctionsSnapshot() {
		if f != nil && f.PluginPath == full {
			t.Fatal("删除后内存命令表仍残留已删除的插件")
		}
	}
}

func TestGetFeatureAcceptsUniqueTargetAlias(t *testing.T) {
	backup := features.items
	features.items = map[string]*Feature{}
	t.Cleanup(func() { features.items = backup })

	features.items["状态"] = &Feature{Name: "状态", Kind: "builtin", Target: "status", Enabled: true}
	if f, ok := GetFeature("status"); !ok || f.Name != "状态" {
		t.Fatalf("应能按唯一 target 取到功能: %+v %v", f, ok)
	}
	if _, ok := GetFeature("missing"); ok {
		t.Fatal("不存在的名称不应命中")
	}

	// 同一 target 出现多次时视为歧义，不再按别名解析。
	features.items["状态2"] = &Feature{Name: "状态2", Kind: "builtin", Target: "status", Enabled: true}
	if _, ok := GetFeature("status"); ok {
		t.Fatal("target 重复时不应按别名命中")
	}
}
