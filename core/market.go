package core

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// PluginMeta 描述一个可在市场上架的插件。
type PluginMeta struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Author      string `json:"author"`
	Description string `json:"description"`
	Identity    string `json:"identity"` // app 或 user
	Scope       string `json:"scope"`    // public 或 master
	Kind        string `json:"kind"`     // script 或 http
	Target      string `json:"target"`   // http 插件的地址
	Digest      string `json:"digest"`   // 内容 SHA-256，安装时校验完整性
	UpdatedAt   string `json:"updated_at"`
}

// MarketEntry 是市场条目：元数据加上脚本源码（http 插件为空）。
type MarketEntry struct {
	PluginMeta
	Content string `json:"content,omitempty"`
}

const marketKey = "plugin_market"

var marketMu sync.Mutex

func loadMarket() map[string]MarketEntry {
	out := map[string]MarketEntry{}
	if BucketApp == nil {
		return out
	}
	raw := BucketApp.GetString(marketKey)
	if raw == "" {
		return out
	}
	_ = json.Unmarshal([]byte(raw), &out)
	return out
}

func saveMarket(items map[string]MarketEntry) {
	if BucketApp == nil {
		return
	}
	raw, _ := json.Marshal(items)
	BucketApp.Set(marketKey, string(raw))
}

func validPluginName(name string) bool {
	if name == "" || len(name) > 64 {
		return false
	}
	return !strings.ContainsAny(name, `/\..:*?"<>|`)
}

func normalizePluginMeta(m PluginMeta) (PluginMeta, error) {
	m.Name = strings.TrimSpace(m.Name)
	if !validPluginName(m.Name) {
		return m, fmt.Errorf("插件名不合法")
	}
	if m.Version == "" {
		m.Version = "0.1.0"
	}
	if m.Identity == "" {
		m.Identity = "app"
	}
	if m.Identity != "app" && m.Identity != "user" {
		return m, fmt.Errorf("插件身份只能是 app 或 user")
	}
	if m.Scope == "" {
		m.Scope = "public"
	}
	if m.Scope != "public" && m.Scope != "master" {
		return m, fmt.Errorf("插件范围只能是 public 或 master")
	}
	if m.Kind == "" {
		m.Kind = "script"
	}
	if m.Kind != "script" && m.Kind != "http" {
		return m, fmt.Errorf("插件类型只能是 script 或 http")
	}
	if m.Kind == "http" {
		if !strings.HasPrefix(m.Target, "http://") && !strings.HasPrefix(m.Target, "https://") {
			return m, fmt.Errorf("http 插件需要 http(s) 地址")
		}
	} else if strings.TrimSpace(m.Target) == "" {
		return m, fmt.Errorf("script 插件的目标应为脚本文件名")
	}
	return m, nil
}

// CompareVersion 比较两个点分版本号：a>b 返回 1，a<b 返回 -1，相等返回 0。
// 数字段按数值比较，非数字段按字典序比较；缺失段视为 0；预发布标签（- 之后）小于同版本正式版。
func CompareVersion(a, b string) int {
	coreA, preA := splitVersion(a)
	coreB, preB := splitVersion(b)
	if c := compareCore(coreA, coreB); c != 0 {
		return c
	}
	switch {
	case preA == preB:
		return 0
	case preA == "":
		return 1
	case preB == "":
		return -1
	case preA > preB:
		return 1
	default:
		return -1
	}
}

// splitVersion 把版本号拆成核心段与预发布标签（首个 - 之前为core）。
func splitVersion(v string) ([]string, string) {
	v = strings.TrimSpace(v)
	pre := ""
	if i := strings.IndexByte(v, '-'); i >= 0 {
		pre = v[i+1:]
		v = v[:i]
	}
	return strings.Split(v, "."), pre
}

func compareCore(pa, pb []string) int {
	n := len(pa)
	if len(pb) > n {
		n = len(pb)
	}
	for i := 0; i < n; i++ {
		x, y := "0", "0"
		if i < len(pa) {
			x = strings.TrimSpace(pa[i])
		}
		if i < len(pb) {
			y = strings.TrimSpace(pb[i])
		}
		if x == "" {
			x = "0"
		}
		if y == "" {
			y = "0"
		}
		xa, xErr := strconv.Atoi(x)
		ya, yErr := strconv.Atoi(y)
		if xErr == nil && yErr == nil {
			if xa > ya {
				return 1
			}
			if xa < ya {
				return -1
			}
			continue
		}
		if x > y {
			return 1
		}
		if x < y {
			return -1
		}
	}
	return 0
}

// sha256Hex 返回输入内容的 SHA-256 十六进制摘要。
func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// pluginDigestSource 决定摘要的原始内容：脚本取源码，http 取地址与描述。
func pluginDigestSource(m PluginMeta, content string) string {
	if m.Kind == "http" {
		return m.Target + "\n" + m.Description
	}
	return content
}

// installedScriptVersion 读取已安装脚本清单里的版本；读取失败或未声明时返回空串。
func installedScriptVersion(target string) string {
	full := safeScriptPath(target)
	if full == "" {
		return ""
	}
	data, err := os.ReadFile(full)
	if err != nil {
		return ""
	}
	return ParsePluginManifest(string(data)).Version
}

// PublishPlugin 把插件元数据与内容登记到市场。
// script 插件若未显式填写，会从源码头部清单补齐名称、版本、作者与描述。
// 版本低于已发布版本时默认拒绝（防降级），显式传 force 才允许回滚覆盖。
func PublishPlugin(m PluginMeta, content string, force bool) error {
	if m.Kind == "script" && strings.TrimSpace(content) != "" {
		pm := ParsePluginManifest(content)
		if strings.TrimSpace(m.Name) == "" {
			m.Name = pm.Title
		}
		if strings.TrimSpace(m.Version) == "" {
			m.Version = pm.Version
		}
		if strings.TrimSpace(m.Author) == "" {
			m.Author = pm.Author
		}
		if strings.TrimSpace(m.Description) == "" {
			m.Description = pm.Description
		}
	}
	m, err := normalizePluginMeta(m)
	if err != nil {
		return err
	}
	if m.Version == "" {
		m.Version = "0.1.0"
	}
	m.Digest = sha256Hex(pluginDigestSource(m, content))
	m.UpdatedAt = time.Now().Format(time.RFC3339)
	marketMu.Lock()
	defer marketMu.Unlock()
	items := loadMarket()
	if prev, ok := items[m.Name]; ok {
		switch cmp := CompareVersion(m.Version, prev.Version); {
		case cmp < 0 && !force:
			return fmt.Errorf("版本 %s 低于已发布版本 %s，拒绝降级；如需回滚请显式指定 force", m.Version, prev.Version)
		case cmp == 0 && prev.Digest == m.Digest:
			return nil
		}
	}
	items[m.Name] = MarketEntry{PluginMeta: m, Content: content}
	saveMarket(items)
	return nil
}

func DeleteMarketPlugin(name string) error {
	marketMu.Lock()
	defer marketMu.Unlock()
	items := loadMarket()
	if _, ok := items[name]; !ok {
		return fmt.Errorf("插件不存在")
	}
	delete(items, name)
	saveMarket(items)
	return nil
}

// MarketPluginItem 是市场列表项，标注是否已安装。
type MarketPluginItem struct {
	PluginMeta
	Installed bool `json:"installed"`
}

func ListMarket() []MarketPluginItem {
	marketMu.Lock()
	items := loadMarket()
	marketMu.Unlock()
	out := make([]MarketPluginItem, 0, len(items))
	for name, e := range items {
		_, installed := GetFeature(name)
		meta := e.PluginMeta
		out = append(out, MarketPluginItem{PluginMeta: meta, Installed: installed})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// InstallMarketPlugin 从市场安装插件，安装后写入功能表并区分身份。
// 安装前校验内容摘要；已安装版本高于市场版本时拒绝降级，force 可强制覆盖。
func InstallMarketPlugin(name string, force bool) error {
	marketMu.Lock()
	e, ok := loadMarket()[name]
	marketMu.Unlock()
	if !ok {
		return fmt.Errorf("插件不存在")
	}
	switch e.Kind {
	case "script":
		if strings.TrimSpace(e.Content) == "" {
			return fmt.Errorf("插件缺少脚本内容")
		}
		if e.Digest != "" && sha256Hex(e.Content) != e.Digest {
			return fmt.Errorf("内容摘要不匹配，拒绝安装")
		}
		if !force {
			if installed, ok := GetFeature(name); ok && installed.Target != "" {
				if cur := installedScriptVersion(installed.Target); cur != "" && CompareVersion(cur, e.Version) > 0 {
					return fmt.Errorf("已安装版本 %s 高于市场版本 %s，拒绝降级安装", cur, e.Version)
				}
			}
		}
		file := e.Target
		if scriptLangOf(file) == "" {
			file += ".js"
		}
		if err := InstallScript(file, e.Content); err != nil {
			return err
		}
		return UpsertFeature(Feature{
			Name: e.Name, Kind: "script", Target: file,
			Scope: e.Scope, Identity: e.Identity, Enabled: true,
		})
	case "http":
		if e.Digest != "" && sha256Hex(e.Target+"\n"+e.Description) != e.Digest {
			return fmt.Errorf("元数据摘要不匹配，拒绝安装")
		}
		return UpsertFeature(Feature{
			Name: e.Name, Kind: "http", Target: e.Target,
			Scope: e.Scope, Identity: e.Identity, Enabled: true, Confirm: true,
		})
	default:
		return fmt.Errorf("未知插件类型")
	}
}

// ExportPlugin 导出插件包，便于分发或再次上架。
func ExportPlugin(name string) (MarketEntry, bool) {
	marketMu.Lock()
	defer marketMu.Unlock()
	e, ok := loadMarket()[name]
	return e, ok
}

func registerMarketAPI() {
	g := Server.Group("/jiying/api", requireLogin)
	g.GET("/market", func(c *gin.Context) {
		jiyingJSON(c, ListMarket())
	})
	g.POST("/market/publish", requireRole(RoleOps), func(c *gin.Context) {
		meta := PluginMeta{
			Name:        strings.TrimSpace(c.PostForm("name")),
			Version:     strings.TrimSpace(c.PostForm("version")),
			Author:      strings.TrimSpace(c.PostForm("author")),
			Description: c.PostForm("description"),
			Identity:    strings.TrimSpace(c.PostForm("identity")),
			Scope:       strings.TrimSpace(c.PostForm("scope")),
			Kind:        strings.TrimSpace(c.PostForm("kind")),
			Target:      strings.TrimSpace(c.PostForm("target")),
		}
		if err := PublishPlugin(meta, c.PostForm("content"), c.PostForm("force") != ""); err != nil {
			RecordAudit(consoleUser(c), "market.publish", meta.Name, "", "失败: "+err.Error())
			jiyingErr(c, err.Error())
			return
		}
		RecordAudit(consoleUser(c), "market.publish", meta.Name, meta.Kind, "成功")
		jiyingJSON(c, "ok")
	})
	g.POST("/market/install", requireRole(RoleOps), func(c *gin.Context) {
		name := strings.TrimSpace(c.PostForm("name"))
		if err := InstallMarketPlugin(name, c.PostForm("force") != ""); err != nil {
			RecordAudit(consoleUser(c), "market.install", name, "", "失败: "+err.Error())
			jiyingErr(c, err.Error())
			return
		}
		RecordAudit(consoleUser(c), "market.install", name, "", "成功")
		jiyingJSON(c, "已安装 "+name)
	})
	g.POST("/market/delete", requireRole(RoleOps), func(c *gin.Context) {
		name := strings.TrimSpace(c.PostForm("name"))
		if err := DeleteMarketPlugin(name); err != nil {
			RecordAudit(consoleUser(c), "market.delete", name, "", "失败: "+err.Error())
			jiyingErr(c, err.Error())
			return
		}
		RecordAudit(consoleUser(c), "market.delete", name, "", "成功")
		jiyingJSON(c, "ok")
	})
	g.GET("/market/export", requireRole(RoleOps), func(c *gin.Context) {
		name := strings.TrimSpace(c.Query("name"))
		e, ok := ExportPlugin(name)
		if !ok {
			jiyingErr(c, "插件不存在")
			return
		}
		jiyingJSON(c, e)
	})
}
