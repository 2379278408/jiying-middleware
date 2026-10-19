package core

import (
	"regexp"
	"strconv"
	"strings"
)

// PluginManifest 是插件头部注释里声明的元数据。
// 语法与 AutMan 兼容：`// [title: xxx]`、`# [rule: ^命令$]`、`# [cron: ...]` 等。
// 未声明任何已知字段时 HasManifest 为 false，调用方据此回退到文件名等默认值。
type PluginManifest struct {
	Title       string              `json:"title"`
	Author      string              `json:"author"`
	Language    string              `json:"language"`
	Class       string              `json:"class"`
	Service     string              `json:"service"`
	Apps        []string            `json:"apps,omitempty"`
	Disable     *bool               `json:"disable,omitempty"`
	Admin       *bool               `json:"admin,omitempty"`
	Rules       []string            `json:"rules"`
	Cron        []string            `json:"cron"`
	Priority    *int                `json:"priority,omitempty"`
	Platform    []string            `json:"platform"`
	OpenSource  *bool               `json:"open_source,omitempty"`
	Network     *bool               `json:"network,omitempty"`
	Icon        string              `json:"icon"`
	Version     string              `json:"version"`
	Public      *bool               `json:"public,omitempty"`
	Price       *int                `json:"price,omitempty"`
	Description string              `json:"description"`
	Params      []string            `json:"params"`
	Config      []PluginConfigField `json:"config"`
	HasManifest bool                `json:"has_manifest"`
}

// manifestScanLimit 限制只扫描文件头部若干行，避免把正文里的字面量误当清单。
const manifestScanLimit = 200

// manifestLineRe 匹配去掉注释符后的 [key: value] 头部声明；value 允许包含 ']'。
var manifestLineRe = regexp.MustCompile(`^\[([A-Za-z_][A-Za-z0-9_]*)\s*:\s*(.*)\]$`)

// ParsePluginManifest 从插件源码里解析头部注释清单。
func ParsePluginManifest(content string) PluginManifest {
	m := PluginManifest{}
	lines := strings.Split(content, "\n")
	if len(lines) > manifestScanLimit {
		lines = lines[:manifestScanLimit]
	}
	found := false
	for _, line := range lines {
		key, value, ok := parseManifestLine(line)
		if !ok {
			continue
		}
		switch key {
		case "title":
			m.Title = strings.TrimSpace(value)
		case "author":
			m.Author = strings.TrimSpace(value)
		case "language":
			m.Language = strings.TrimSpace(value)
		case "class":
			m.Class = strings.TrimSpace(value)
		case "service":
			m.Service = strings.TrimSpace(value)
		case "app", "apps":
			m.Apps = append(m.Apps, splitManifestList(value)...)
		case "disable":
			if b, ok := parseManifestBool(value); ok {
				m.Disable = &b
			}
		case "admin":
			if b, ok := parseManifestBool(value); ok {
				m.Admin = &b
			}
		case "rule":
			if v := strings.TrimSpace(value); v != "" {
				m.Rules = append(m.Rules, v)
			}
		case "cron":
			if v := strings.TrimSpace(value); v != "" {
				m.Cron = append(m.Cron, v)
			}
		case "priority":
			if n, ok := parseManifestInt(value); ok {
				m.Priority = &n
			}
		case "platform":
			m.Platform = append(m.Platform, splitManifestList(value)...)
		case "open_source":
			if b, ok := parseManifestBool(value); ok {
				m.OpenSource = &b
			}
		case "network":
			if b, ok := parseManifestBool(value); ok {
				m.Network = &b
			}
		case "icon":
			m.Icon = strings.TrimSpace(value)
		case "version":
			m.Version = strings.TrimSpace(value)
		case "public":
			if b, ok := parseManifestBool(value); ok {
				m.Public = &b
			}
		case "price":
			if n, ok := parseManifestInt(value); ok {
				m.Price = &n
			}
		case "description":
			m.Description = strings.TrimSpace(value)
		case "param":
			if v := strings.TrimSpace(value); v != "" {
				m.Params = append(m.Params, v)
			}
		case "params":
			m.Params = append(m.Params, splitManifestList(value)...)
		case "config":
			if f, ok := parseConfigField(value); ok {
				m.Config = append(m.Config, f)
			}
		default:
			continue
		}
		found = true
	}
	m.HasManifest = found
	return m
}

// parseManifestLine 去掉行首的注释符（//、#、*、--、;）后匹配 [key: value]。
func parseManifestLine(line string) (key, value string, ok bool) {
	s := strings.TrimSpace(strings.TrimPrefix(line, "\uFEFF"))
	for {
		before := s
		for _, p := range []string{"//", "#", "*", "--", ";"} {
			if strings.HasPrefix(s, p) {
				s = strings.TrimSpace(strings.TrimPrefix(s, p))
				break
			}
		}
		if s == before {
			break
		}
	}
	m := manifestLineRe.FindStringSubmatch(s)
	if m == nil {
		return "", "", false
	}
	return strings.ToLower(m[1]), m[2], true
}

func parseManifestBool(v string) (bool, bool) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "true", "1", "yes", "on", "y":
		return true, true
	case "false", "0", "no", "off", "n":
		return false, true
	}
	return false, false
}

func parseManifestInt(v string) (int, bool) {
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil {
		return 0, false
	}
	return n, true
}

// splitManifestList 按逗号/空格/分号/竖线切分列表，过滤空项。
func splitManifestList(v string) []string {
	fields := strings.FieldsFunc(v, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t' || r == ';' || r == '|'
	})
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		if s := strings.TrimSpace(f); s != "" {
			out = append(out, s)
		}
	}
	return out
}

// pluginTitle 返回插件展示名：优先使用清单 title，否则回退文件名。
func pluginTitle(name, content string) string {
	if t := strings.TrimSpace(ParsePluginManifest(content).Title); t != "" {
		return t
	}
	return scriptTitle(name)
}

// scriptFeatureNameScope 由脚本文件名与内容推导功能名与范围。
// 声明 [admin: true] 的插件映射到 master 范围，与主控权限模型对齐。
func scriptFeatureNameScope(name, content string) (string, string) {
	title := scriptTitle(name)
	scope := "public"
	if m := ParsePluginManifest(content); m.HasManifest {
		if t := strings.TrimSpace(m.Title); t != "" {
			title = t
		}
		if m.Admin != nil && *m.Admin {
			scope = "master"
		}
	}
	return title, scope
}

// applyFunctionManifest 用头部清单补全 Function 元数据；未声明的字段保持调用方给的默认值。
func applyFunctionManifest(f *Function, content string) {
	if f == nil {
		return
	}
	m := ParsePluginManifest(content)
	f.Manifest = m
	if m.Title != "" {
		f.Title = m.Title
	}
	if m.Author != "" {
		f.Author = m.Author
	}
	if m.Version != "" {
		f.Version = m.Version
	}
	if m.Class != "" {
		f.Class = m.Class
	}
	if m.Service != "" {
		f.Service = m.Service
	}
	if m.Icon != "" {
		f.Icon = m.Icon
	}
	if m.Description != "" {
		f.Description = m.Description
	}
	if len(m.Rules) > 0 {
		f.Rules = m.Rules
	}
	if len(m.Cron) > 0 {
		f.Cron = m.Cron[0]
	}
	if m.Priority != nil {
		f.Priority = *m.Priority
	}
	if m.Admin != nil {
		f.Admin = *m.Admin
	}
	if m.Disable != nil {
		f.Disable = *m.Disable
	}
	if m.OpenSource != nil {
		f.OpenSource = *m.OpenSource
	}
	if m.Public != nil {
		f.Public = *m.Public
	}
	if m.Price != nil {
		f.Price = *m.Price
	}
	if len(m.Params) > 0 {
		f.Params = m.Params
	}
	if len(m.Platform) > 0 {
		f.ImTypeWhite = m.Platform
		f.Platform = true
	}
	f.rulesRe = compileManifestRules(m.Rules)
}

// compileManifestRules 预编译规则正则；非法正则跳过并记日志，不影响其余规则。
func compileManifestRules(rules []string) []*regexp.Regexp {
	if len(rules) == 0 {
		return nil
	}
	out := make([]*regexp.Regexp, 0, len(rules))
	for _, r := range rules {
		re, err := regexp.Compile(r)
		if err != nil {
			GetMyLog().Warn("插件规则 %q 不是合法正则，已跳过: %v", r, err)
			continue
		}
		out = append(out, re)
	}
	return out
}

func matchManifestRules(f *Function, msg string) bool {
	if f == nil {
		return false
	}
	for _, re := range f.rulesRe {
		if re.MatchString(msg) {
			return true
		}
	}
	return false
}

// splitParam 解析 [param: 名称:说明] 形式的参数声明，返回名称与说明。
func splitParam(p string) (name, desc string) {
	p = strings.TrimSpace(p)
	for _, sep := range []string{":", "=", "|"} {
		if idx := strings.Index(p, sep); idx > 0 {
			return strings.TrimSpace(p[:idx]), strings.TrimSpace(p[idx+len(sep):])
		}
	}
	return p, ""
}
