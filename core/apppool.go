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
}

var appConnPool = &appPool{items: map[string]*appRuntime{}, convApp: map[string]string{}}

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
func rememberConvApp(conversationID, appID string) {
	if conversationID == "" || appID == "" {
		return
	}
	appConnPool.mu.Lock()
	appConnPool.convApp[conversationID] = appID
	if len(appConnPool.convApp) > 2000 {
		// 简单上限，避免长期运行无限增长。
		for k := range appConnPool.convApp {
			if k != conversationID {
				delete(appConnPool.convApp, k)
			}
		}
	}
	appConnPool.mu.Unlock()
}

// clientForConv 返回处理该会话的客户端：优先按会话归属路由，否则回退默认/单连接。
func clientForConv(conversationID string) *JiyingClient {
	appConnPool.mu.Lock()
	defer appConnPool.mu.Unlock()
	if appID, ok := appConnPool.convApp[conversationID]; ok {
		if rt, ok := appConnPool.items[appID]; ok {
			return rt.client
		}
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
}

// AppPoolSnapshot 返回当前所有应用通道状态，密钥仅以掩码呈现。
func AppPoolSnapshot() []AppPoolItem {
	appConnPool.mu.Lock()
	defer appConnPool.mu.Unlock()
	out := make([]AppPoolItem, 0, len(appConnPool.items))
	for id, rt := range appConnPool.items {
		out = append(out, AppPoolItem{
			AppID:       id,
			WSURL:       maskURL(rt.cfg.WSURL),
			Online:      rt.client.Connected(),
			IsDefault:   id == appConnPool.defaultID,
			Name:        rt.profile.Name,
			Description: rt.profile.Description,
			AvatarURL:   rt.profile.Avatar,
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
