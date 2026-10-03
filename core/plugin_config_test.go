package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestParseConfigField(t *testing.T) {
	content := "# [title: 示例]\n" +
		"# [config: apiKey:secret::接口密钥]\n" +
		"# [config: mode:select:fast|slow:运行模式]\n" +
		"# [config: retries:number:3:重试次数]\n" +
		"# [config: verbose:bool:yes:详细日志]\n" +
		"# [config: note:说明文字]\n" +
		"# [config: prompt:text::提示词]\n" +
		"# [config: 9bad:ignored]\n"
	m := ParsePluginManifest(content)
	if len(m.Config) != 6 {
		t.Fatalf("want 6 config fields, got %d: %+v", len(m.Config), m.Config)
	}
	byKey := map[string]PluginConfigField{}
	for _, f := range m.Config {
		byKey[f.Key] = f
	}
	if f := byKey["apiKey"]; f.Type != "secret" || f.Desc != "接口密钥" {
		t.Fatalf("apiKey 解析错误: %+v", f)
	}
	if f := byKey["mode"]; f.Type != "select" || len(f.Options) != 2 || f.Default != "fast" {
		t.Fatalf("mode 解析错误: %+v", f)
	}
	if f := byKey["retries"]; f.Type != "number" || f.Default != "3" {
		t.Fatalf("retries 解析错误: %+v", f)
	}
	if f := byKey["verbose"]; f.Type != "bool" || f.Default != "true" {
		t.Fatalf("verbose 解析错误: %+v", f)
	}
	if f := byKey["note"]; f.Type != "string" || f.Desc != "说明文字" {
		t.Fatalf("note 解析错误: %+v", f)
	}
	if f := byKey["prompt"]; f.Type != "text" {
		t.Fatalf("prompt 解析错误: %+v", f)
	}
}

func TestMergePluginConfig(t *testing.T) {
	schema := []PluginConfigField{
		{Key: "mode", Type: "select", Default: "fast", Options: []string{"fast", "slow"}},
		{Key: "retries", Type: "number", Default: "3"},
		{Key: "verbose", Type: "bool", Default: "false"},
		{Key: "apiKey", Type: "secret"},
		{Key: "name", Type: "string"},
	}
	prev := map[string]string{"apiKey": "old-secret", "name": "keep"}

	// secret 留空保持旧值；未知字段被忽略；number/bool 归一化。
	out, err := mergePluginConfig(schema, prev, map[string]string{
		"apiKey": "", "name": "new", "retries": "5", "verbose": "yes", "unknown": "x",
	})
	if err != nil {
		t.Fatalf("merge 失败: %v", err)
	}
	if out["apiKey"] != "old-secret" {
		t.Fatalf("secret 空值应保持旧值, got %q", out["apiKey"])
	}
	if out["name"] != "new" || out["retries"] != "5" || out["verbose"] != "true" {
		t.Fatalf("合并结果错误: %+v", out)
	}
	if _, ok := out["unknown"]; ok {
		t.Fatalf("未知字段不应写入: %+v", out)
	}

	// secret 提供新值时更新。
	out, err = mergePluginConfig(schema, prev, map[string]string{"apiKey": "new-secret"})
	if err != nil || out["apiKey"] != "new-secret" {
		t.Fatalf("secret 更新失败: %v %+v", err, out)
	}

	// 非法取值被拒绝。
	if _, err := mergePluginConfig(schema, prev, map[string]string{"retries": "abc"}); err == nil {
		t.Fatal("非数字 retries 应被拒绝")
	}
	if _, err := mergePluginConfig(schema, prev, map[string]string{"mode": "unknown"}); err == nil {
		t.Fatal("select 非法取值应被拒绝")
	}
}

func TestPluginConfigForAPIMasksSecrets(t *testing.T) {
	schema := []PluginConfigField{
		{Key: "token", Type: "secret"},
		{Key: "host", Type: "string"},
	}
	values, set := pluginConfigForAPI(schema, map[string]string{"token": "s3cr3t", "host": "example.com"})
	if values["token"] != "" {
		t.Fatalf("secret 不应回显: %q", values["token"])
	}
	if values["host"] != "example.com" {
		t.Fatalf("普通字段应回显: %q", values["host"])
	}
	if len(set) != 1 || set[0] != "token" {
		t.Fatalf("set 应标注已配置密钥: %+v", set)
	}
}

func TestConfigRuntimeValuesAndEnv(t *testing.T) {
	schema := []PluginConfigField{
		{Key: "count", Type: "number"},
		{Key: "debug", Type: "bool"},
		{Key: "label", Type: "string"},
	}
	raw := map[string]string{"count": "7", "debug": "yes", "label": "hi"}
	obj := configRuntimeValues(raw, schema)
	if n, ok := obj["count"].(float64); !ok || n != 7 {
		t.Fatalf("number 应转为 float64: %#v", obj["count"])
	}
	if b, ok := obj["debug"].(bool); !ok || !b {
		t.Fatalf("bool 应转为 bool: %#v", obj["debug"])
	}
	env := configEnvFor(raw)
	var parsed map[string]string
	if err := json.Unmarshal([]byte(env["JIYING_CONFIG"]), &parsed); err != nil {
		t.Fatalf("JIYING_CONFIG 应为 JSON: %v", err)
	}
	if parsed["count"] != "7" {
		t.Fatalf("JIYING_CONFIG 内容错误: %+v", parsed)
	}
	if env["JIYING_CFG_LABEL"] != "hi" {
		t.Fatalf("逐项环境变量缺失: %+v", env)
	}
}

func TestPluginConfigStorageRoundTrip(t *testing.T) {
	setupAccountPoolBucket(t, "plugin-config-roundtrip")
	name := "demo.js"
	BucketApp.Delete(pluginConfigBucketKey(name))
	t.Cleanup(func() { BucketApp.Delete(pluginConfigBucketKey(name)) })

	if err := savePluginConfig(name, map[string]string{"host": "h", "token": "t"}); err != nil {
		t.Fatalf("保存失败: %v", err)
	}
	got := LoadPluginConfig(name)
	if got["host"] != "h" || got["token"] != "t" {
		t.Fatalf("读回不一致: %+v", got)
	}
	if bucketKey := pluginConfigBucketKey("sub/demo.js"); bucketKey != pluginConfigBucketKey(name) {
		t.Fatalf("存储键应按文件名归一: %q", bucketKey)
	}
}

func TestEffectivePluginConfigAppliesDefaults(t *testing.T) {
	setupAccountPoolBucket(t, "plugin-config-defaults")
	dir := t.TempDir()
	t.Setenv("JIYING_RES", dir)
	if err := os.MkdirAll(filepath.Join(dir, "scripts"), 0o755); err != nil {
		t.Fatal(err)
	}
	name := "confd.js"
	code := "// [title: 配置默认值]\n// [config: host:string:example.com:主机]\n// [config: token:secret::密钥]\nresult = \"ok\";\n"
	if err := os.WriteFile(filepath.Join(dir, "scripts", name), []byte(code), 0o644); err != nil {
		t.Fatal(err)
	}
	BucketApp.Delete(pluginConfigBucketKey(name))
	t.Cleanup(func() { BucketApp.Delete(pluginConfigBucketKey(name)) })

	_, eff := effectivePluginConfig(name)
	if eff["host"] != "example.com" {
		t.Fatalf("默认值未应用: %+v", eff)
	}
	if _, ok := eff["token"]; ok {
		t.Fatalf("密钥默认值不应伪造: %+v", eff)
	}
	// 保存后覆盖默认值。
	if err := savePluginConfig(name, map[string]string{"host": "h2", "token": "s"}); err != nil {
		t.Fatal(err)
	}
	_, eff = effectivePluginConfig(name)
	if eff["host"] != "h2" || eff["token"] != "s" {
		t.Fatalf("保存值未生效: %+v", eff)
	}
}

func TestRedactConfigSchemaHidesSecretDefaults(t *testing.T) {
	schema := []PluginConfigField{
		{Key: "host", Type: "string", Default: "example.com"},
		{Key: "token", Type: "secret", Default: "hard-coded"},
	}
	out := redactConfigSchema(schema)
	if out[0].Default != "example.com" {
		t.Fatalf("非密钥默认值不应被清除: %+v", out[0])
	}
	if out[1].Default != "" {
		t.Fatalf("密钥默认值应被清除: %+v", out[1])
	}
	// 不应修改原 schema。
	if schema[1].Default != "hard-coded" {
		t.Fatalf("原 schema 被意外修改: %+v", schema[1])
	}
}
