package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// PluginConfigField 描述插件头部 [config: ...] 声明的一项配置。
// 语法：[config: key:type:default:desc]，也支持简写 [config: key:说明]（等价 string）。
// type 取 string|number|bool|select|text|secret，缺省 string；
// select 的 default 用 a|b|c 给出候选，第一项为默认值。
type PluginConfigField struct {
	Key     string   `json:"key"`
	Type    string   `json:"type"`
	Default string   `json:"default"`
	Desc    string   `json:"desc,omitempty"`
	Options []string `json:"options,omitempty"`
}

var configKeyRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_-]*$`)

func validConfigType(t string) bool {
	switch t {
	case "string", "number", "bool", "select", "text", "secret":
		return true
	}
	return false
}

// parseConfigField 解析单条 [config: ...] 声明；非法字段名返回 false 并跳过。
func parseConfigField(value string) (PluginConfigField, bool) {
	parts := strings.SplitN(strings.TrimSpace(value), ":", 4)
	key := strings.TrimSpace(parts[0])
	if !configKeyRe.MatchString(key) {
		return PluginConfigField{}, false
	}
	f := PluginConfigField{Key: key, Type: "string"}
	switch len(parts) {
	case 1:
		// [config: key]
	case 2:
		f.Desc = strings.TrimSpace(parts[1])
	default:
		t := strings.ToLower(strings.TrimSpace(parts[1]))
		if !validConfigType(t) {
			// 第二个字段不是合法类型时，整段按说明处理，避免误判字段含义。
			f.Desc = strings.TrimSpace(strings.Join(parts[1:], ":"))
			return f, true
		}
		f.Type = t
		f.Default = strings.TrimSpace(parts[2])
		if len(parts) == 4 {
			f.Desc = strings.TrimSpace(parts[3])
		}
	}
	normalizeConfigField(&f)
	return f, true
}

func normalizeConfigField(f *PluginConfigField) {
	switch f.Type {
	case "select":
		opts := strings.FieldsFunc(f.Default, func(r rune) bool { return r == '|' || r == ',' })
		out := make([]string, 0, len(opts))
		for _, o := range opts {
			if s := strings.TrimSpace(o); s != "" {
				out = append(out, s)
			}
		}
		f.Options = out
		f.Default = ""
		if len(out) > 0 {
			f.Default = out[0]
		}
	case "bool":
		if b, ok := parseManifestBool(f.Default); ok {
			f.Default = strconv.FormatBool(b)
		} else {
			f.Default = "false"
		}
	case "number":
		if _, err := strconv.ParseFloat(f.Default, 64); err != nil {
			f.Default = "0"
		}
	case "string", "text", "secret":
		// 保持原样
	default:
		f.Type = "string"
	}
}

const pluginConfigPrefix = "plugincfg:"

func pluginConfigBucketKey(name string) string {
	return pluginConfigPrefix + filepath.Base(name)
}

// LoadPluginConfig 读取插件已保存的配置值；未配置或解析失败时返回空表。
func LoadPluginConfig(name string) map[string]string {
	out := map[string]string{}
	if BucketApp == nil {
		return out
	}
	raw := BucketApp.GetString(pluginConfigBucketKey(name))
	if raw == "" {
		return out
	}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		GetMyLog().Warn("插件配置解析失败 %s: %v", name, err)
		return map[string]string{}
	}
	return out
}

func savePluginConfig(name string, values map[string]string) error {
	if BucketApp == nil {
		return fmt.Errorf("存储未初始化")
	}
	b, err := json.Marshal(values)
	if err != nil {
		return err
	}
	BucketApp.Set(pluginConfigBucketKey(name), string(b))
	return nil
}

// pluginConfigSchemaFor 读取脚本文件并解析其配置声明。
func pluginConfigSchemaFor(name string) []PluginConfigField {
	full := safeScriptPath(filepath.Base(name))
	if full == "" {
		return nil
	}
	data, err := os.ReadFile(full)
	if err != nil {
		return nil
	}
	return ParsePluginManifest(string(data)).Config
}

// effectivePluginConfig 返回应用默认值后的生效配置及对应 schema。
func effectivePluginConfig(name string) ([]PluginConfigField, map[string]string) {
	schema := pluginConfigSchemaFor(name)
	raw := LoadPluginConfig(name)
	eff := map[string]string{}
	for _, f := range schema {
		if v, ok := raw[f.Key]; ok {
			eff[f.Key] = v
		} else if f.Default != "" {
			eff[f.Key] = f.Default
		}
	}
	return schema, eff
}

// pluginRuntimeConfig 生成注入运行时的类型化配置对象与 Python 环境变量。
func pluginRuntimeConfig(name string) (map[string]any, map[string]string) {
	schema, eff := effectivePluginConfig(name)
	return configRuntimeValues(eff, schema), configEnvFor(eff)
}

// mergePluginConfig 按 schema 校验并合并提交值；secret 字段留空表示保持旧值不变。
func mergePluginConfig(schema []PluginConfigField, prev, in map[string]string) (map[string]string, error) {
	fields := make(map[string]PluginConfigField, len(schema))
	for _, f := range schema {
		fields[f.Key] = f
	}
	out := map[string]string{}
	for k, v := range prev {
		if _, ok := fields[k]; ok {
			out[k] = v
		}
	}
	for k, raw := range in {
		f, ok := fields[k]
		if !ok {
			continue
		}
		v := strings.TrimSpace(raw)
		switch f.Type {
		case "secret":
			if v == "" {
				continue
			}
		case "number":
			if v == "" {
				out[k] = f.Default
				continue
			}
			if _, err := strconv.ParseFloat(v, 64); err != nil {
				return nil, fmt.Errorf("配置 %s 需要数字", f.Key)
			}
		case "bool":
			if v == "" {
				v = f.Default
			} else if b, ok := parseManifestBool(v); ok {
				v = strconv.FormatBool(b)
			} else {
				return nil, fmt.Errorf("配置 %s 需要布尔值", f.Key)
			}
		case "select":
			if v == "" {
				v = f.Default
			}
			if !configContains(f.Options, v) {
				return nil, fmt.Errorf("配置 %s 取值不在候选中", f.Key)
			}
		}
		out[k] = v
	}
	return out, nil
}

// pluginConfigForAPI 返回前端可见的配置：secret 值清空，另用 set 标注已配置的密钥项。
func pluginConfigForAPI(schema []PluginConfigField, raw map[string]string) (map[string]string, []string) {
	secret := map[string]bool{}
	for _, f := range schema {
		if f.Type == "secret" {
			secret[f.Key] = true
		}
	}
	values := map[string]string{}
	set := make([]string, 0)
	for k, v := range raw {
		if secret[k] {
			if v != "" {
				set = append(set, k)
			}
			values[k] = ""
			continue
		}
		values[k] = v
	}
	return values, set
}

// configRuntimeValues 把字符串配置转成类型化值，供 goja 运行时以 config 全局注入。
func configRuntimeValues(raw map[string]string, schema []PluginConfigField) map[string]any {
	types := make(map[string]string, len(schema))
	for _, f := range schema {
		types[f.Key] = f.Type
	}
	out := make(map[string]any, len(raw))
	for k, v := range raw {
		switch types[k] {
		case "number":
			if n, err := strconv.ParseFloat(v, 64); err == nil {
				out[k] = n
			} else {
				out[k] = v
			}
		case "bool":
			if b, ok := parseManifestBool(v); ok {
				out[k] = b
			} else {
				out[k] = v
			}
		default:
			out[k] = v
		}
	}
	return out
}

// configEnvFor 生成 Python 脚本可见的配置环境变量：整体 JSON 加逐项 JIYING_CFG_*。
func configEnvFor(raw map[string]string) map[string]string {
	env := map[string]string{}
	if len(raw) == 0 {
		return env
	}
	if b, err := json.Marshal(raw); err == nil {
		env["JIYING_CONFIG"] = string(b)
	}
	for k, v := range raw {
		env["JIYING_CFG_"+configEnvKey(k)] = v
	}
	return env
}

func configEnvKey(k string) string {
	var b strings.Builder
	for _, r := range strings.ToUpper(k) {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	return b.String()
}

// configDefaults 返回 schema 中非密钥项的默认值，用于安装试跑时提供可运行的配置。
func configDefaults(schema []PluginConfigField) map[string]string {
	out := map[string]string{}
	for _, f := range schema {
		if f.Type == "secret" {
			continue
		}
		if f.Default != "" {
			out[f.Key] = f.Default
		}
	}
	return out
}

func configContains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

// configKeys 返回 schema 中的字段名列表，用于审计留痕（只记键，不记值）。
func configKeys(schema []PluginConfigField) []string {
	out := make([]string, 0, len(schema))
	for _, f := range schema {
		out = append(out, f.Key)
	}
	return out
}

// redactConfigSchema 返回可下发给前端的 schema：清空 secret 字段的默认值，
// 避免作者写在头部清单里的密钥通过列表或配置接口回显。
func redactConfigSchema(schema []PluginConfigField) []PluginConfigField {
	if len(schema) == 0 {
		return schema
	}
	out := make([]PluginConfigField, len(schema))
	copy(out, schema)
	for i := range out {
		if out[i].Type == "secret" {
			out[i].Default = ""
		}
	}
	return out
}
