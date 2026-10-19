package core

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"jiying/middleware/utils"
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
	Publisher   string `json:"publisher,omitempty"`
	Signature   string `json:"signature,omitempty"`
	Signed      bool   `json:"signed"`
	UpdatedAt   string `json:"updated_at"`
}

// MarketPublisher 是受信任的发布者公钥。
type MarketPublisher struct {
	Name      string `json:"name"`
	PublicKey string `json:"public_key"`
	CreatedAt string `json:"created_at"`
}

// MarketEntry 是市场条目：元数据加上脚本源码（http 插件为空）。
type MarketEntry struct {
	PluginMeta
	Content string `json:"content,omitempty"`
}

const (
	marketKey            = "plugin_market"
	marketPublishersKey  = "market_publishers"
	marketLocalPublisher = "local"
)

var marketMu sync.Mutex
var marketKeyMu sync.Mutex
var marketLocalKeyCache struct {
	sync.Mutex
	key ed25519.PrivateKey
}

// pluginSignaturePayload 是签名覆盖的规范化字段；摘要已包含内容与目标信息。
type pluginSignaturePayload struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Author      string `json:"author"`
	Description string `json:"description"`
	Identity    string `json:"identity"`
	Scope       string `json:"scope"`
	Kind        string `json:"kind"`
	Target      string `json:"target"`
	Digest      string `json:"digest"`
	Publisher   string `json:"publisher"`
}

func pluginSigningPayload(m PluginMeta) []byte {
	raw, _ := json.Marshal(pluginSignaturePayload{
		Name: m.Name, Version: m.Version, Author: m.Author, Description: m.Description,
		Identity: m.Identity, Scope: m.Scope, Kind: m.Kind, Target: m.Target,
		Digest: m.Digest, Publisher: m.Publisher,
	})
	return raw
}

func loadPublishers() map[string]MarketPublisher {
	out := map[string]MarketPublisher{}
	if BucketApp == nil {
		return out
	}
	if raw := BucketApp.GetString(marketPublishersKey); raw != "" {
		_ = json.Unmarshal([]byte(raw), &out)
	}
	return out
}

func savePublishers(items map[string]MarketPublisher) {
	if BucketApp == nil {
		return
	}
	raw, _ := json.Marshal(items)
	BucketApp.Set(marketPublishersKey, string(raw))
}

// normalizePublisherKey 接受标准或 URL-safe Base64 的 32 字节 Ed25519 公钥。
func normalizePublisherKey(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("公钥不能为空")
	}
	for _, enc := range []*base64.Encoding{base64.StdEncoding, base64.RawStdEncoding, base64.URLEncoding, base64.RawURLEncoding} {
		if data, err := enc.DecodeString(raw); err == nil && len(data) == ed25519.PublicKeySize {
			return base64.StdEncoding.EncodeToString(data), nil
		}
	}
	return "", fmt.Errorf("公钥需为 32 字节 Ed25519 公钥的 Base64 编码")
}

// RegisterMarketPublisher 登记或更新受信任发布者；名称固定后不允许复用其他公钥。
func RegisterMarketPublisher(name, publicKey string) (MarketPublisher, error) {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 64 || strings.ContainsAny(name, `/\..:*?"<>| `) {
		return MarketPublisher{}, fmt.Errorf("发布者名称不合法")
	}
	key, err := normalizePublisherKey(publicKey)
	if err != nil {
		return MarketPublisher{}, err
	}
	if name == marketLocalPublisher {
		priv, err := localMarketSigningKey()
		if err != nil {
			return MarketPublisher{}, err
		}
		if base64.StdEncoding.EncodeToString(priv.Public().(ed25519.PublicKey)) != key {
			return MarketPublisher{}, fmt.Errorf("local 是本机发布者保留名称，不能绑定外部公钥")
		}
	}
	marketKeyMu.Lock()
	defer marketKeyMu.Unlock()
	items := loadPublishers()
	if prev, ok := items[name]; ok && prev.PublicKey != key {
		return MarketPublisher{}, fmt.Errorf("发布者 %s 已登记其他公钥，如需轮换请先删除", name)
	}
	item := MarketPublisher{Name: name, PublicKey: key, CreatedAt: time.Now().Format(time.RFC3339)}
	if prev, ok := items[name]; ok {
		item.CreatedAt = prev.CreatedAt
	}
	items[name] = item
	savePublishers(items)
	return item, nil
}

func DeleteMarketPublisher(name string) error {
	marketKeyMu.Lock()
	defer marketKeyMu.Unlock()
	items := loadPublishers()
	if _, ok := items[name]; !ok {
		return fmt.Errorf("发布者不存在")
	}
	delete(items, name)
	savePublishers(items)
	return nil
}

func ListMarketPublishers() []MarketPublisher {
	marketKeyMu.Lock()
	items := loadPublishers()
	marketKeyMu.Unlock()
	out := make([]MarketPublisher, 0, len(items))
	for _, p := range items {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func marketPublisherKey(name string) (ed25519.PublicKey, bool) {
	marketKeyMu.Lock()
	item, ok := loadPublishers()[name]
	marketKeyMu.Unlock()
	if !ok {
		return nil, false
	}
	raw, err := base64.StdEncoding.DecodeString(item.PublicKey)
	if err != nil || len(raw) != ed25519.PublicKeySize {
		return nil, false
	}
	return ed25519.PublicKey(raw), true
}

func marketDataDir() string {
	if v := strings.TrimSpace(os.Getenv("JIYING_DATA")); v != "" {
		return v
	}
	return utils.GetDataHome()
}

// localMarketSigningKey 加载或生成本机市场签名密钥；私钥只落在数据目录，权限 0600。
func localMarketSigningKey() (ed25519.PrivateKey, error) {
	marketLocalKeyCache.Lock()
	defer marketLocalKeyCache.Unlock()
	if len(marketLocalKeyCache.key) == ed25519.PrivateKeySize {
		return marketLocalKeyCache.key, nil
	}
	dir := marketDataDir()
	path := filepath.Join(dir, ".market_signing_key")
	if raw, err := os.ReadFile(path); err == nil {
		if data, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(raw))); err == nil && len(data) == ed25519.PrivateKeySize {
			marketLocalKeyCache.key = ed25519.PrivateKey(data)
			return marketLocalKeyCache.key, nil
		}
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("创建数据目录失败: %w", err)
	}
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("生成市场签名密钥失败: %w", err)
	}
	if err := os.WriteFile(path, []byte(base64.StdEncoding.EncodeToString(priv)), 0o600); err != nil {
		return nil, fmt.Errorf("保存市场签名密钥失败: %w", err)
	}
	_ = os.Chmod(path, 0o600)
	marketLocalKeyCache.key = priv
	return priv, nil
}

// localMarketPublicKey 返回本机发布者公钥，并确保已登记到发布者表。
func localMarketPublicKey() (string, error) {
	priv, err := localMarketSigningKey()
	if err != nil {
		return "", err
	}
	pub := base64.StdEncoding.EncodeToString(priv.Public().(ed25519.PublicKey))
	if _, ok := marketPublisherKey(marketLocalPublisher); !ok {
		if _, err := RegisterMarketPublisher(marketLocalPublisher, pub); err != nil {
			return "", err
		}
	}
	return pub, nil
}

// signMarketMeta 用本机密钥签名；外部发布者必须自带签名。
func signMarketMeta(m PluginMeta) (PluginMeta, error) {
	if m.Publisher == "" {
		m.Publisher = marketLocalPublisher
	}
	if m.Publisher == marketLocalPublisher {
		if _, err := localMarketPublicKey(); err != nil {
			return m, err
		}
		priv, err := localMarketSigningKey()
		if err != nil {
			return m, err
		}
		m.Signature = base64.StdEncoding.EncodeToString(ed25519.Sign(priv, pluginSigningPayload(m)))
		m.Signed = true
		return m, nil
	}
	if m.Signature == "" {
		return m, fmt.Errorf("外部发布者 %s 必须提供签名", m.Publisher)
	}
	if err := verifyMarketSignature(m); err != nil {
		return m, err
	}
	m.Signed = true
	return m, nil
}

func verifyMarketSignature(m PluginMeta) error {
	if m.Publisher == "" || m.Signature == "" {
		return fmt.Errorf("插件缺少发布者或签名")
	}
	pub, ok := marketPublisherKey(m.Publisher)
	if !ok {
		return fmt.Errorf("发布者 %s 不在受信任公钥表中", m.Publisher)
	}
	sig, err := base64.StdEncoding.DecodeString(m.Signature)
	if err != nil || len(sig) != ed25519.SignatureSize {
		return fmt.Errorf("签名格式不正确")
	}
	if !ed25519.Verify(pub, pluginSigningPayload(m), sig) {
		return fmt.Errorf("发布者签名校验失败")
	}
	return nil
}

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
	var errSign error
	m, errSign = signMarketMeta(m)
	if errSign != nil {
		return errSign
	}
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

// marketScriptRiskMessage 返回市场脚本安装前的二次确认提示；非脚本或低风险返回空。
func marketScriptRiskMessage(name string) string {
	marketMu.Lock()
	e, ok := loadMarket()[name]
	marketMu.Unlock()
	if !ok || e.Kind != "script" {
		return ""
	}
	return scriptRiskMessage(e.Target, e.Content)
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
	// 新发布的条目必须带发布者签名；历史无签名条目继续按摘要校验，重新发布后自动升级。
	if e.Publisher != "" || e.Signature != "" {
		if err := verifyMarketSignature(e.PluginMeta); err != nil {
			return fmt.Errorf("签名校验失败: %w", err)
		}
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
	g.GET("/market/publishers", requireRole(RoleMaster), func(c *gin.Context) {
		jiyingJSON(c, ListMarketPublishers())
	})
	g.POST("/market/publishers", requireRole(RoleMaster), func(c *gin.Context) {
		name := strings.TrimSpace(c.PostForm("name"))
		item, err := RegisterMarketPublisher(name, c.PostForm("public_key"))
		if err != nil {
			RecordAudit(consoleUser(c), "market.publisher.add", name, "", "失败: "+err.Error())
			jiyingErr(c, err.Error())
			return
		}
		RecordAudit(consoleUser(c), "market.publisher.add", item.Name, item.PublicKey, "成功")
		jiyingJSON(c, item)
	})
	g.POST("/market/publishers/delete", requireRole(RoleMaster), func(c *gin.Context) {
		name := strings.TrimSpace(c.PostForm("name"))
		if err := DeleteMarketPublisher(name); err != nil {
			RecordAudit(consoleUser(c), "market.publisher.delete", name, "", "失败: "+err.Error())
			jiyingErr(c, err.Error())
			return
		}
		RecordAudit(consoleUser(c), "market.publisher.delete", name, "", "成功")
		jiyingJSON(c, "ok")
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
			Publisher:   strings.TrimSpace(c.PostForm("publisher")),
			Signature:   strings.TrimSpace(c.PostForm("signature")),
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
		if msg := marketScriptRiskMessage(name); msg != "" && c.PostForm("confirm") != "true" {
			RecordAudit(consoleUser(c), "market.install", name, "", "待确认: "+msg)
			jiyingErr(c, msg)
			return
		}
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
