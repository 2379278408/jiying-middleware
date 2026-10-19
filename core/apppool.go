package core

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// apppoolKey 是应用连接池在本地桶中的持久化键。
const apppoolKey = "relay_apps"

// convAppsKey 持久化会话到应用的归属映射，重启后仍能把出站消息送回原通道。
const convAppsKey = "relay_conv_apps"

// convAppsLimit 限制归属映射规模，超出后按最近使用顺序淘汰最旧记录。
const convAppsLimit = 2000

type appRuntime struct {
	cfg    AppConn
	client *JiyingClient
	cancel context.CancelFunc
	// profile 是应用资料缓存，由后台循环在连接可用后刷新；读取时需持有 appConnPool.mu。
	profile   JiyingApp
	profileAt time.Time
}

// appPool 管理多个应用通道连接，按 app_id 隔离租户。
type appPool struct {
	mu        sync.Mutex
	items     map[string]*appRuntime
	defaultID string
	// convApp 记录会话归属的应用，用于把出站消息路由回正确通道。
	convApp map[string]string
	// convOrder 按最近使用顺序保存会话 ID，用于有界淘汰。
	convOrder []string
	// convBucket 标记当前归属映射来自哪个存储桶，测试或热切换时自动重载。
	convBucket string
	convLoaded bool
}

var appConnPool = &appPool{items: map[string]*appRuntime{}, convApp: map[string]string{}}

// ensureConvAppsLocked 在持有 appConnPool.mu 时按需加载归属映射；存储桶变化会清空旧缓存。
func (p *appPool) ensureConvAppsLocked() {
	key := ""
	if BucketApp != nil {
		key = BucketApp.String()
	}
	if p.convLoaded && p.convBucket == key {
		return
	}
	p.convLoaded = true
	p.convBucket = key
	p.convApp = map[string]string{}
	p.convOrder = nil
	if BucketApp == nil {
		return
	}
	raw := BucketApp.GetString(convAppsKey)
	if raw == "" {
		return
	}
	var stored struct {
		Items map[string]string `json:"items"`
		Order []string          `json:"order"`
	}
	if json.Unmarshal([]byte(raw), &stored) != nil {
		return
	}
	seen := make(map[string]bool, len(stored.Order))
	for _, id := range stored.Order {
		id = strings.TrimSpace(id)
		appID := strings.TrimSpace(stored.Items[id])
		if id == "" || appID == "" || seen[id] {
			continue
		}
		seen[id] = true
		p.convApp[id] = appID
		p.convOrder = append(p.convOrder, id)
	}
	for id, appID := range stored.Items {
		if id == "" || appID == "" || seen[id] || len(p.convApp) >= convAppsLimit {
			continue
		}
		p.convApp[id] = appID
		p.convOrder = append(p.convOrder, id)
	}
}

// persistConvAppsLocked 把当前归属映射写回存储；调用方必须持有 appConnPool.mu。
func (p *appPool) persistConvAppsLocked() {
	if BucketApp == nil {
		return
	}
	raw, err := json.Marshal(struct {
		Items map[string]string `json:"items"`
		Order []string          `json:"order"`
	}{Items: p.convApp, Order: p.convOrder})
	if err != nil {
		return
	}
	BucketApp.Set(convAppsKey, string(raw))
}

// forgetConvAppsLocked 清理已移除应用的归属记录；调用方必须持有 appConnPool.mu。
func (p *appPool) forgetConvAppsLocked(alive map[string]bool) bool {
	changed := false
	for convID, appID := range p.convApp {
		if alive[appID] {
			continue
		}
		delete(p.convApp, convID)
		changed = true
	}
	if !changed {
		return false
	}
	kept := p.convOrder[:0]
	for _, convID := range p.convOrder {
		if _, ok := p.convApp[convID]; ok {
			kept = append(kept, convID)
		}
	}
	p.convOrder = kept
	p.persistConvAppsLocked()
	return true
}

// poolMu 串行化连接池配置的「读-改-写」：loadAppConns / loadAccountConns 都基于
// 整份列表覆写，若两个请求并发修改会发生丢失更新。所有会写配置的入口都要持有它。
var poolMu sync.Mutex

// currentJiying 在连接池锁内读取默认客户端指针，避免与 applyAppConns 的写并发。
func currentJiying() *JiyingClient {
	appConnPool.mu.Lock()
	defer appConnPool.mu.Unlock()
	return activeJiying
}

// loadAppConns 读取连接池配置；池为空时回退到旧的单应用配置。
func loadAppConns() []AppConn {
	if BucketApp == nil {
		return nil
	}
	var list []AppConn
	if raw := BucketApp.GetString(apppoolKey); raw != "" {
		_ = json.Unmarshal([]byte(raw), &list)
	}
	for i := range list {
		list[i].AppSecret = openSecret(list[i].AppSecret)
	}
	if len(list) == 0 {
		legacy := loadRelayConn().App
		if legacy.WSURL != "" && legacy.AppID != "" && legacy.AppSecret != "" {
			list = []AppConn{legacy}
		}
	}
	return list
}

func saveAppConns(list []AppConn) {
	if BucketApp == nil {
		return
	}
	sealed := make([]AppConn, 0, len(list))
	for _, c := range list {
		value, err := sealSecret(c.AppSecret)
		if err != nil {
			GetMyLog().Error("应用密钥加密失败，已取消保存: %v", err)
			return
		}
		c.AppSecret = value
		sealed = append(sealed, c)
	}
	raw, _ := json.Marshal(sealed)
	BucketApp.Set(apppoolKey, string(raw))
}

// applyAppConns 用给定配置同步运行中的连接池：保留未变的连接，重启有变化的，停止已删除的。
func applyAppConns(list []AppConn) {
	saveAppConns(list)
	appConnPool.mu.Lock()
	defer appConnPool.mu.Unlock()
	want := map[string]AppConn{}
	order := make([]string, 0, len(list))
	for _, c := range list {
		if c.AppID == "" || c.WSURL == "" || c.AppSecret == "" {
			continue
		}
		want[c.AppID] = c
		order = append(order, c.AppID)
	}
	// 停止已删除或配置变更的连接。
	for id, rt := range appConnPool.items {
		next, ok := want[id]
		if ok && rt.cfg == next {
			continue
		}
		rt.cancel()
		delete(appConnPool.items, id)
	}
	// 启动新增或变更的连接。
	for id, cfg := range want {
		if _, ok := appConnPool.items[id]; ok {
			continue
		}
		client := NewJiyingClient(JiyingConfig{WSURL: cfg.WSURL, AppID: cfg.AppID, Secret: cfg.AppSecret}, nil)
		ctx, cancel := context.WithCancel(context.Background())
		appConnPool.items[id] = &appRuntime{cfg: cfg, client: client, cancel: cancel}
		go func(c *JiyingClient, cancel context.CancelFunc) {
			if err := c.Run(ctx); err != nil && ctx.Err() == nil {
				GetMyLog().Warn("即应应用通道退出: %v", err)
			}
			cancel()
		}(client, cancel)
		go refreshAppProfile(ctx, client, id)
	}
	if len(order) > 0 {
		appConnPool.defaultID = order[0]
	} else {
		appConnPool.defaultID = ""
	}
	alive := make(map[string]bool, len(want))
	for id := range want {
		alive[id] = true
	}
	appConnPool.ensureConvAppsLocked()
	appConnPool.forgetConvAppsLocked(alive)
	appConnPool.syncDefaultLocked()
}

// syncDefaultLocked 把默认应用镜像到 activeJiying，供既有单连接代码路径使用。
func (p *appPool) syncDefaultLocked() {
	if rt, ok := p.items[p.defaultID]; ok {
		activeJiying = rt.client
		return
	}
	activeJiying = nil
}

func stopAppPool() {
	appConnPool.mu.Lock()
	defer appConnPool.mu.Unlock()
	for id, rt := range appConnPool.items {
		rt.cancel()
		delete(appConnPool.items, id)
	}
	activeJiying = nil
}

// rememberConvApp 记录某会话所属应用，用于出站消息路由。
// 只有归属变化时才落盘，避免每条入站消息都重写整份映射。
func rememberConvApp(conversationID, appID string) {
	conversationID = strings.TrimSpace(conversationID)
	appID = strings.TrimSpace(appID)
	if conversationID == "" || appID == "" {
		return
	}
	appConnPool.mu.Lock()
	defer appConnPool.mu.Unlock()
	appConnPool.ensureConvAppsLocked()
	if old, ok := appConnPool.convApp[conversationID]; ok && old == appID {
		return
	}
	appConnPool.convApp[conversationID] = appID
	kept := appConnPool.convOrder[:0]
	for _, id := range appConnPool.convOrder {
		if id != conversationID {
			kept = append(kept, id)
		}
	}
	appConnPool.convOrder = append(kept, conversationID)
	for len(appConnPool.convOrder) > convAppsLimit {
		oldest := appConnPool.convOrder[0]
		appConnPool.convOrder = appConnPool.convOrder[1:]
		delete(appConnPool.convApp, oldest)
	}
	appConnPool.persistConvAppsLocked()
}

// ConvAppOwner 返回会话归属的应用资料。known 为 false 表示尚未记录归属。
func ConvAppOwner(conversationID string) (appID, name string, online, known bool) {
	appConnPool.mu.Lock()
	defer appConnPool.mu.Unlock()
	appConnPool.ensureConvAppsLocked()
	appID, known = appConnPool.convApp[conversationID]
	if !known {
		return "", "", false, false
	}
	rt, ok := appConnPool.items[appID]
	if !ok || rt == nil {
		return appID, "", false, true
	}
	return appID, rt.profile.Name, rt.client.Connected(), true
}

// appIDForConversation 返回会话所属应用；未记录归属时回退默认应用。
func appIDForConversation(conversationID string) string {
	if appID, _, _, known := ConvAppOwner(conversationID); known && appID != "" {
		return appID
	}
	return strings.TrimSpace(CurrentAppConfig().AppID)
}

// clientForConv 返回处理该会话的客户端：已知归属但通道缺失时失败关闭，避免误发到默认应用。
func clientForConv(conversationID string) *JiyingClient {
	appConnPool.mu.Lock()
	defer appConnPool.mu.Unlock()
	appConnPool.ensureConvAppsLocked()
	if appID, ok := appConnPool.convApp[conversationID]; ok {
		if rt, ok := appConnPool.items[appID]; ok && rt != nil {
			return rt.client
		}
		return nil
	}
	if rt, ok := appConnPool.items[appConnPool.defaultID]; ok {
		return rt.client
	}
	return activeJiying
}

// clientForApp 返回指定应用的客户端；appID 为空时使用默认应用。
func clientForApp(appID string) *JiyingClient {
	appConnPool.mu.Lock()
	defer appConnPool.mu.Unlock()
	if appID == "" {
		appID = appConnPool.defaultID
	}
	if rt, ok := appConnPool.items[appID]; ok {
		return rt.client
	}
	if appID == "" {
		return activeJiying
	}
	return nil
}

// refreshAppProfile 在连接可用后拉取应用资料并缓存，失败按短间隔重试，成功后每 10 分钟刷新。
func refreshAppProfile(ctx context.Context, client *JiyingClient, appID string) {
	delay := 3 * time.Second
	for {
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
		if !client.Connected() {
			delay = 5 * time.Second
			continue
		}
		callCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
		app, err := client.AppsGet(callCtx, appID)
		cancel()
		if err != nil || app.ID == "" {
			delay = 10 * time.Second
			continue
		}
		appConnPool.mu.Lock()
		if rt, ok := appConnPool.items[appID]; ok {
			rt.profile = app
			rt.profileAt = time.Now()
		}
		appConnPool.mu.Unlock()
		delay = 10 * time.Minute
	}
}

// SendTextToConv 按会话归属选择应用通道发送文本。
func SendTextToConv(ctx context.Context, conversationID, replyTo, text string) error {
	c := clientForConv(conversationID)
	if c == nil {
		return errJiyingNotConnected
	}
	return c.SendText(ctx, conversationID, replyTo, text)
}

// AppPoolItem 是连接池对外展示的只读快照。
type AppPoolItem struct {
	AppID       string `json:"app_id"`
	WSURL       string `json:"ws_url"`
	Online      bool   `json:"online"`
	IsDefault   bool   `json:"is_default"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	AvatarURL   string `json:"avatar_url,omitempty"`
	// Conversations 是该应用当前记录的会话归属数量，便于控制台区分活跃应用。
	Conversations int `json:"conversation_count"`
}

// AppPoolSnapshot 返回当前所有应用通道状态，密钥仅以掩码呈现。
func AppPoolSnapshot() []AppPoolItem {
	appConnPool.mu.Lock()
	defer appConnPool.mu.Unlock()
	appConnPool.ensureConvAppsLocked()
	convCount := make(map[string]int, len(appConnPool.items))
	for _, appID := range appConnPool.convApp {
		convCount[appID]++
	}
	out := make([]AppPoolItem, 0, len(appConnPool.items))
	for id, rt := range appConnPool.items {
		out = append(out, AppPoolItem{
			AppID:         id,
			WSURL:         maskURL(rt.cfg.WSURL),
			Online:        rt.client.Connected(),
			IsDefault:     id == appConnPool.defaultID,
			Name:          rt.profile.Name,
			Description:   rt.profile.Description,
			AvatarURL:     rt.profile.Avatar,
			Conversations: convCount[id],
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].IsDefault != out[j].IsDefault {
			return out[i].IsDefault
		}
		return out[i].AppID < out[j].AppID
	})
	return out
}

// DefaultAppConn 返回默认应用连接配置；连接池为空时返回零值。
func DefaultAppConn() AppConn {
	appConnPool.mu.Lock()
	defer appConnPool.mu.Unlock()
	if rt, ok := appConnPool.items[appConnPool.defaultID]; ok {
		return rt.cfg
	}
	return AppConn{}
}

// UpsertAppConn 新增或更新一个应用连接，并立即同步到连接池。
func UpsertAppConn(cfg AppConn) error {
	if strings.TrimSpace(cfg.WSURL) == "" || strings.TrimSpace(cfg.AppID) == "" || strings.TrimSpace(cfg.AppSecret) == "" {
		return fmt.Errorf("缺少应用地址 / ID / 密钥")
	}
	poolMu.Lock()
	defer poolMu.Unlock()
	list := loadAppConns()
	replaced := false
	for i := range list {
		if list[i].AppID == cfg.AppID {
			list[i] = cfg
			replaced = true
			break
		}
	}
	if !replaced {
		list = append(list, cfg)
	}
	applyAppConns(list)
	return nil
}

// RemoveAppConn 从连接池移除指定应用并停止其连接。
func RemoveAppConn(appID string) error {
	appID = strings.TrimSpace(appID)
	if appID == "" {
		return fmt.Errorf("缺少应用 ID")
	}
	poolMu.Lock()
	defer poolMu.Unlock()
	list := loadAppConns()
	kept := list[:0]
	found := false
	for _, c := range list {
		if c.AppID == appID {
			found = true
			continue
		}
		kept = append(kept, c)
	}
	if !found {
		return fmt.Errorf("应用不存在")
	}
	applyAppConns(kept)
	return nil
}

// SetDefaultAppConn 将指定应用排到连接池首位并作为新会话默认出口。
func SetDefaultAppConn(appID string) error {
	appID = strings.TrimSpace(appID)
	if appID == "" {
		return fmt.Errorf("缺少应用 ID")
	}
	poolMu.Lock()
	defer poolMu.Unlock()
	list := loadAppConns()
	var selected AppConn
	found := false
	kept := make([]AppConn, 0, len(list))
	for _, cfg := range list {
		if cfg.AppID == appID {
			selected = cfg
			found = true
			continue
		}
		kept = append(kept, cfg)
	}
	if !found {
		return fmt.Errorf("应用不存在")
	}
	applyAppConns(append([]AppConn{selected}, kept...))
	return nil
}

// Connected 报告该客户端当前是否持有活动连接。
func (c *JiyingClient) Connected() bool {
	if c == nil {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn != nil
}
