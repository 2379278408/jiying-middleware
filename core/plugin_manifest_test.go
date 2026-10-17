package core

import (
	"os"
	"path/filepath"
	"testing"
)

const samplePluginJS = `// [title: 天气查询]
// [author: demo]
// [language: nodejs]
// [class: 工具类]
// [apps: app-a, app-b]
// [service: 10086]
// [disable: false]
// [admin: true]
// [rule: ^天气\s+(\S+)$]
// [rule: ^查天气]
// [cron: 0 8 * * *]
// [priority: 50]
// [platform: qq,wx,web]
// [open_source: true]
// [icon: https://example.com/i.png]
// [version: 1.2.3]
// [public: false]
// [price: 9]
// [description: 查询指定城市天气]
// [param: city:城市名称]
result = "ok";
`

func TestParsePluginManifestJS(t *testing.T) {
	m := ParsePluginManifest(samplePluginJS)
	if !m.HasManifest {
		t.Fatal("应识别出头部清单")
	}
	if m.Title != "天气查询" || m.Author != "demo" || m.Version != "1.2.3" {
		t.Fatalf("基础字段解析错误: %+v", m)
	}
	if m.Class != "工具类" || m.Service != "10086" || m.Icon != "https://example.com/i.png" {
		t.Fatalf("扩展字段解析错误: %+v", m)
	}
	if len(m.Apps) != 2 || m.Apps[0] != "app-a" || m.Apps[1] != "app-b" {
		t.Fatalf("apps 解析错误: %v", m.Apps)
	}
	if len(m.Rules) != 2 || m.Rules[0] != `^天气\s+(\S+)$` || m.Rules[1] != "^查天气" {
		t.Fatalf("规则应为多行累积: %v", m.Rules)
	}
	if len(m.Cron) != 1 || m.Cron[0] != "0 8 * * *" {
		t.Fatalf("cron 解析错误: %v", m.Cron)
	}
	if m.Priority == nil || *m.Priority != 50 {
		t.Fatalf("priority 解析错误: %v", m.Priority)
	}
	if len(m.Platform) != 3 || m.Platform[0] != "qq" || m.Platform[2] != "web" {
		t.Fatalf("platform 解析错误: %v", m.Platform)
	}
	if m.Admin == nil || !*m.Admin || m.Disable == nil || *m.Disable {
		t.Fatalf("布尔字段解析错误: admin=%v disable=%v", m.Admin, m.Disable)
	}
	if m.OpenSource == nil || !*m.OpenSource || m.Public == nil || *m.Public {
		t.Fatalf("布尔字段解析错误: open=%v public=%v", m.OpenSource, m.Public)
	}
	if m.Price == nil || *m.Price != 9 {
		t.Fatalf("price 解析错误: %v", m.Price)
	}
	if m.Description != "查询指定城市天气" {
		t.Fatalf("description 解析错误: %q", m.Description)
	}
	if len(m.Params) != 1 || m.Params[0] != "city:城市名称" {
		t.Fatalf("param 解析错误: %v", m.Params)
	}
}

func TestParsePluginManifestPythonHash(t *testing.T) {
	code := "# [title: py插件]\n" +
		"# [rule: ^py$]\n" +
		"# [params: a, b;c]\n" +
		"print('hi')\n"
	m := ParsePluginManifest(code)
	if !m.HasManifest || m.Title != "py插件" {
		t.Fatalf("Python 注释清单解析失败: %+v", m)
	}
	if len(m.Rules) != 1 || m.Rules[0] != "^py$" {
		t.Fatalf("Python rule 解析错误: %v", m.Rules)
	}
	if len(m.Params) != 3 || m.Params[2] != "c" {
		t.Fatalf("Python params 解析错误: %v", m.Params)
	}
}

func TestParsePluginManifestNoHeader(t *testing.T) {
	m := ParsePluginManifest("result = 1 + 1;\n")
	if m.HasManifest {
		t.Fatalf("无头部清单时不应标记 HasManifest: %+v", m)
	}
	if m.Title != "" || len(m.Rules) != 0 {
		t.Fatalf("无头部清单不应有字段: %+v", m)
	}
}

func TestApplyFunctionManifestFallback(t *testing.T) {
	f := &Function{Title: "demo"}
	applyFunctionManifest(f, "result = 1;")
	if f.Title != "demo" {
		t.Fatalf("无清单应保留默认标题，实际 %q", f.Title)
	}
	if f.Manifest.HasManifest {
		t.Fatal("无清单不应标记 HasManifest")
	}
}

func TestApplyFunctionManifestOverrides(t *testing.T) {
	f := &Function{Title: "old"}
	applyFunctionManifest(f, samplePluginJS)
	if f.Title != "天气查询" {
		t.Fatalf("清单标题应覆盖默认标题，实际 %q", f.Title)
	}
	if !f.Admin || f.Priority != 50 || f.Version != "1.2.3" {
		t.Fatalf("清单字段未落到 Function: %+v", f)
	}
	if len(f.rulesRe) != 2 {
		t.Fatalf("规则正则应预编译 2 条，实际 %d", len(f.rulesRe))
	}
	if !matchManifestRules(f, "天气 北京") {
		t.Fatal("规则应匹配『天气 北京』")
	}
	if matchManifestRules(f, "无关消息") {
		t.Fatal("规则不应匹配『无关消息』")
	}
	if len(f.Params) != 1 || f.Params[0] != "city:城市名称" {
		t.Fatalf("参数未落到 Function: %v", f.Params)
	}
}

func TestScriptFeatureNameScope(t *testing.T) {
	name, scope := scriptFeatureNameScope("demo.js", samplePluginJS)
	if name != "天气查询" || scope != "master" {
		t.Fatalf("admin 插件应映射到 master，实际 name=%q scope=%q", name, scope)
	}
	name2, scope2 := scriptFeatureNameScope("plain.js", "result = 1")
	if name2 != "plain" || scope2 != "public" {
		t.Fatalf("无清单应回退文件名与 public，实际 name=%q scope=%q", name2, scope2)
	}
	if got := pluginTitle("plain.js", ""); got != "plain" {
		t.Fatalf("pluginTitle 回退错误: %q", got)
	}
}

func TestSplitParam(t *testing.T) {
	name, desc := splitParam("city:城市名称")
	if name != "city" || desc != "城市名称" {
		t.Fatalf("splitParam 解析错误: %q %q", name, desc)
	}
	name, desc = splitParam("plain")
	if name != "plain" || desc != "" {
		t.Fatalf("splitParam 无说明时解析错误: %q %q", name, desc)
	}
}

func TestBuildToolContractFallbackArgs(t *testing.T) {
	c := buildToolContract("tool_0", &Feature{Name: "不存在-xyz", Kind: "http", Identity: "app", Scope: "public", Enabled: true})
	if c.Feature != "不存在-xyz" || c.Tool != "tool_0" {
		t.Fatalf("契约基础字段错误: %+v", c)
	}
	props, _ := c.Schema["properties"].(map[string]any)
	if _, ok := props["args"]; !ok {
		t.Fatalf("无参数声明时应回退 args，实际 %+v", props)
	}
}

func TestParsePluginManifestSkipsCodeLookalikes(t *testing.T) {
	code := "const xs = [1, 2, 3];\n" +
		"// [标题: 中文key应忽略]\n" +
		"result = xs[0];\n"
	if m := ParsePluginManifest(code); m.HasManifest {
		t.Fatalf("非清单行不应被识别: %+v", m)
	}
}

func TestPublishPluginFillsMetaFromManifest(t *testing.T) {
	setupAgentBucket(t, "market-manifest")
	if err := PublishPlugin(PluginMeta{Kind: "script", Target: "demo.js"}, samplePluginJS, false); err != nil {
		t.Fatal(err)
	}
	e, ok := loadMarket()["天气查询"]
	if !ok {
		t.Fatalf("应按清单标题登记市场条目: %+v", loadMarket())
	}
	if e.Version != "1.2.3" || e.Author != "demo" || e.Description == "" {
		t.Fatalf("清单元数据未补齐: %+v", e)
	}
}

func TestInstallMarketPluginKeepsPythonExtension(t *testing.T) {
	requirePython(t)
	setupAgentBucket(t, "market-python")
	if err := PublishPlugin(PluginMeta{Name: "py去重", Kind: "script", Target: "py_target.py"}, "print('ok')\n", false); err != nil {
		t.Fatal(err)
	}
	if err := InstallMarketPlugin("py去重", false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(scriptsDir(), "py_target.py")); err != nil {
		t.Fatalf("Python 插件应保留 .py 扩展名: %v", err)
	}
	if _, err := os.Stat(filepath.Join(scriptsDir(), "py_target.py.js")); err == nil {
		t.Fatal("不应为 Python 插件错误追加 .js")
	}
}
