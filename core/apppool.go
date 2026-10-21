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

// convAccountsKey 持久化账号可见会话；这类会话没有创建应用，可回退默认应用。
const convAccountsKey = "relay_conv_accounts"

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
	// accountConv 记录账号可见但没有创建应用的会话，仅用于区分「账号会话」与未知归属。
	accountConv   map[string]bool
	accountOrder  []string
	accountScope  string
	accountBucket string
	accountLoaded bool
}

var appConnPool = &appPool{items: map[string]*appRuntime{}, convApp: map[string]string{}, accountConv: map[string]bool{}}

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

// accountConversationScope 是账号会话标记的归属范围。
// 标记必须随活动账号切换而失效，避免旧账号可见的会话在新账号下回退默认应用。
func accountConversationScope() string {
	return strings.TrimSpace(activeAccountEmail())
}

// ensureAccountConvsLocked 按需加载账号会话标记；存储桶或活动账号变化时清空旧缓存。
// 旧版无账号范围的标记直接丢弃，宁可暂时失败关闭，也不跨账号沿用。
func (p *appPool) ensureAccountConvsLocked(scope string) {
	key := ""
	if BucketApp != nil {
		key = BucketApp.String()
	}
	if p.accountLoaded && p.accountBucket == key && p.accountScope == scope {
		return
	}
	p.accountLoaded = true
	p.accountBucket = key
	p.accountScope = scope
	p.accountConv = map[string]bool{}
	p.accountOrder = nil
	if BucketApp == nil {
		return
	}
	raw := BucketApp.GetString(convAccountsKey)
	if raw == "" {
		return
	}
	var stored struct {
		Scope string   `json:"scope"`
		Order []string `json:"order"`
	}
	if json.Unmarshal([]byte(raw), &stored) != nil || strings.TrimSpace(stored.Scope) != scope {
		return
	}
	for _, id := range stored.Order {
		id = strings.TrimSpace(id)
		if id == "" || p.accountConv[id] || len(p.accountConv) >= convAppsLimit {
			continue
		}
		p.accountConv[id] = true
		p.accountOrder = append(p.accountOrder, id)
	}
}

// persistAccountConvsLocked 把账号会话标记写回存储；调用方必须持有 appConnPool.mu。
func (p *appPool) persistAccountConvsLocked() {
	if BucketApp == nil {
		return
	}
	raw, err := json.Marshal(struct {
		Scope string   `json:"scope"`
		Order []string `json:"order"`
	}{Scope: p.accountScope, Order: p.accountOrder})
	if err != nil {
		return
	}
	BucketApp.Set(convAccountsKey, string(raw))
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
		if rt.cancel != nil {
			rt.cancel()
		}
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
		if rt.cancel != nil {
			rt.cancel()
		}
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

// rememberAccountConvs 批量记录账号可见但没有创建应用的会话，只落盘一次。
// 这类会话允许回退默认应用执行工具，未知来源的跨应用会话仍然失败关闭。
func rememberAccountConvs(conversationIDs []string) {
	scope := accountConversationScope()
	appConnPool.mu.Lock()
	defer appConnPool.mu.Unlock()
	appConnPool.ensureAccountConvsLocked(scope)
	changed := false
	for _, conversationID := range conversationIDs {
		conversationID = strings.TrimSpace(conversationID)
		if conversationID == "" || appConnPool.accountConv[conversationID] {
			continue
		}
		appConnPool.accountConv[conversationID] = true
		appConnPool.accountOrder = append(appConnPool.accountOrder, conversationID)
		changed = true
	}
	for len(appConnPool.accountOrder) > convAppsLimit {
		oldest := appConnPool.accountOrder[0]
		appConnPool.accountOrder = appConnPool.accountOrder[1:]
		delete(appConnPool.accountConv, oldest)
		changed = true
	}
	if changed {
		appConnPool.persistAccountConvsLocked()
	}
}

// rememberAccountConv 记录单个账号会话，供单条补登记路径使用。
func rememberAccountConv(conversationID string) {
	rememberAccountConvs([]string{conversationID})
}

// IsAccountConversation 报告会话是否来自账号可见列表且没有创建应用。
func IsAccountConversation(conversationID string) bool {
	conversationID = strings.TrimSpace(conversationID)
	if conversationID == "" {
		return false
	}
	scope := accountConversationScope()
	appConnPool.mu.Lock()
	defer appConnPool.mu.Unlock()
	appConnPool.ensureAccountConvsLocked(scope)
	return appConnPool.accountConv[conversationID]
}

// KnownConvApps 返回会话到所属应用的快照，供轮询等只读场景使用。
func KnownConvApps() map[string]string {
	appConnPool.mu.Lock()
	defer appConnPool.mu.Unlock()
	appConnPool.ensureConvAppsLocked()
	out := make(map[string]string, len(appConnPool.convApp))
	for k, v := range appConnPool.convApp {
		out[k] = v
	}
	return out
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

// appIDForConversation 返回会话所属应用；未记录归属时返回空字符串。
// 只有账号可见且没有创建应用的会话才回退默认应用；跨应用未知会话仍失败关闭。
func appIDForConversation(conversationID string) string {
	if conversationID == "" {
		return strings.TrimSpace(CurrentAppConfig().AppID)
	}
	if appID, _, _, known := ConvAppOwner(conversationID); known && appID != "" {
		return appID
	}
	if !IsAccountConversation(conversationID) {
		return ""
	}
	if appID := strings.TrimSpace(CurrentAppConfig().AppID); appID != "" {
		return appID
	}
	appConnPool.mu.Lock()
	defer appConnPool.mu.Unlock()
	return strings.TrimSpace(appConnPool.defaultID)
}

// clientForConv 返回处理该会话的客户端。
// 已知归属但应用缺失时失败关闭；账号会话回退默认应用；未知会话只在单应用兼容模式下回退，
// 多应用部署必须已有归属，避免把消息、历史或脚本回执发到错误应用。
func clientForConv(conversationID string) *JiyingClient {
	scope := accountConversationScope()
	appConnPool.mu.Lock()
	defer appConnPool.mu.Unlock()
	appConnPool.ensureConvAppsLocked()
	appConnPool.ensureAccountConvsLocked(scope)
	if strings.TrimSpace(conversationID) == "" {
		return appConnPool.defaultClientLocked()
	}
	if appID, ok := appConnPool.convApp[conversationID]; ok {
		if rt, ok := appConnPool.items[appID]; ok && rt != nil {
			return rt.client
		}
		return nil
	}
	if appConnPool.accountConv[conversationID] {
		return appConnPool.defaultClientLocked()
	}
	if len(appConnPool.items) <= 1 {
		return appConnPool.defaultClientLocked()
	}
	return nil
}

// defaultClientLocked 返回默认应用客户端；调用方必须持有 appConnPool.mu。
func (p *appPool) defaultClientLocked() *JiyingClient {
	if rt, ok := p.items[p.defaultID]; ok && rt != nil {
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
	// LastError 是最近一次连接失败原因；在线时为空。
	LastError string `json:"last_error,omitempty"`
	// RetryAt 是下一次重连时间；在线或未计划时为空。
	RetryAt string `json:"retry_at,omitempty"`
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
		item := AppPoolItem{
			AppID:         id,
			WSURL:         maskURL(rt.cfg.WSURL),
			Online:        rt.client.Connected(),
			IsDefault:     id == appConnPool.defaultID,
			Name:          rt.profile.Name,
			Description:   rt.profile.Description,
			AvatarURL:     rt.profile.Avatar,
			Conversations: convCount[id],
		}
		if !item.Online {
			item.LastError = truncateText(rt.client.LastError(), 200)
			if at := rt.client.RetryAt(); !at.IsZero() {
				item.RetryAt = at.Format(time.RFC3339)
			}
		}
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].IsDefault != out[j].IsDefault {
			return out[i].IsDefault
		}
		return out[i].AppID < out[j].AppID
	})
	return out
}

// appConnExists 报告连接池中是否存在指定应用。
func appConnExists(appID string) bool {
	appID = strings.TrimSpace(appID)
	if appID == "" {
		return false
	}
	appConnPool.mu.Lock()
	defer appConnPool.mu.Unlock()
	_, ok := appConnPool.items[appID]
	return ok
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
		poolMu.Unlock()
		return fmt.Errorf("应用不存在")
	}
	appConnPool.ensureConvAppsLocked()
	removedConvs := make([]string, 0)
	for convID, owner := range appConnPool.convApp {
		if owner == appID {
			removedConvs = append(removedConvs, convID)
		}
	}
	applyAppConns(kept)
	poolMu.Unlock()
	cleanupRemovedApp(appID, removedConvs)
	return nil
}

// cleanupRemovedApp 清理被删除应用的残留引用，避免配置、历史与任务继续指向失效应用。
func cleanupRemovedApp(appID string, convIDs []string) {
	if strings.TrimSpace(appID) == "" {
		return
	}
	for _, convID := range convIDs {
		deleteAgentHistory(convID)
	}
	cfg := LoadAgentConfig()
	changed := false
	if _, ok := cfg.AppSystems[appID]; ok {
		delete(cfg.AppSystems, appID)
		changed = true
	}
	if _, ok := cfg.AppOverrides[appID]; ok {
		delete(cfg.AppOverrides, appID)
		changed = true
	}
	if changed {
		SaveAgentConfig(cfg)
	}
	rolesMu.Lock()
	appRoles := loadAppRoles()
	if _, ok := appRoles[appID]; ok {
		delete(appRoles, appID)
		saveAppRoles(appRoles)
	}
	rolesMu.Unlock()
	features.mu.Lock()
	for _, f := range features.items {
		if f == nil || len(f.Apps) == 0 {
			continue
		}
		kept := make([]string, 0, len(f.Apps))
		for _, id := range f.Apps {
			if !strings.EqualFold(id, appID) {
				kept = append(kept, id)
			}
		}
		if len(kept) == len(f.Apps) {
			continue
		}
		f.Apps = kept
		f.AppsLocked = true
		if len(kept) == 0 {
			f.Enabled = false
		}
	}
	features.saveLocked()
	features.mu.Unlock()
	ccs.Lock()
	for _, cmd := range ccs.Pts {
		if cmd == nil || !strings.EqualFold(cmd.AppID, appID) || !cmd.Status {
			continue
		}
		cmd.Status = false
		cmd.LastError = "所属应用已删除，任务已暂停"
		if ccs.cronSched != nil && cmd.CronEntityId > 0 {
			ccs.cronSched.Remove(cmd.CronEntityId)
			cmd.CronEntityId = 0
		}
		ccs.save(cmd)
	}
	ccs.Unlock()
	removeDingTalkTopicsForApp(appID)
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
