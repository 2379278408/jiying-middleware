package core

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// AuditEntry 记录主控敏感操作，便于事后追溯。
type AuditEntry struct {
	Time   string `json:"time"`
	Actor  string `json:"actor"`
	AppID  string `json:"app_id,omitempty"`
	Action string `json:"action"`
	Target string `json:"target"`
	Detail string `json:"detail"`
	Result string `json:"result"`
}

const auditKeep = 500

// auditFlushDelay 合并短时间内的多条审计写入，避免高频操作反复序列化 500 条记录。
const auditFlushDelay = time.Second

var auditMu sync.Mutex

var auditCache = struct {
	key    string
	loaded bool
	list   []AuditEntry
	dirty  bool
	timer  *time.Timer
}{}

// loadAuditLocked 按当前存储桶加载审计到内存；桶切换时自动重载，测试之间不串数据。
func loadAuditLocked() []AuditEntry {
	if BucketApp == nil {
		auditCache.key = ""
		auditCache.loaded = true
		auditCache.list = nil
		return nil
	}
	key := BucketApp.String()
	if auditCache.loaded && auditCache.key == key {
		return auditCache.list
	}
	auditCache.key = key
	auditCache.loaded = true
	auditCache.list = nil
	raw := BucketApp.GetString("jiying_audit")
	if raw == "" {
		return nil
	}
	var list []AuditEntry
	if json.Unmarshal([]byte(raw), &list) == nil {
		auditCache.list = list
	}
	return auditCache.list
}

// flushAudit 把内存中的审计列表写入桶；失败只记警告，下次写入会再次触发。
func flushAudit() { flushAuditTo(BucketApp) }

// flushAuditTo 将待写审计写入指定桶。定时器在调度时就固定目标桶，
// 避免回调触发时再次读取全局 BucketApp（测试替换桶时会构成数据竞争）。
func flushAuditTo(bucket Bucket) {
	auditMu.Lock()
	defer auditMu.Unlock()
	auditCache.timer = nil
	if !auditCache.dirty || bucket == nil {
		return
	}
	// 存储桶已切换（测试或热切换）时丢弃旧桶的待写数据，避免写到新桶。
	if !auditCache.loaded || auditCache.key != bucket.String() {
		auditCache.dirty = false
		return
	}
	raw, err := json.Marshal(auditCache.list)
	if err != nil {
		GetMyLog().Warn("审计日志序列化失败: %v", err)
		return
	}
	if bucket.Set("jiying_audit", string(raw)) {
		auditCache.dirty = false
	}
}

// FlushAudit 在进程退出前补写尚未落盘的审计记录。
func FlushAudit() { flushAudit() }

// auditURL 只保留协议、主机与路径，剥离 userinfo、查询串与片段，
// 避免把 URL 中可能携带的 token、密钥写入审计留痕。
func auditURL(raw string) string {
	raw = strings.TrimSpace(raw)
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return raw
	}
	u.User = nil
	u.RawQuery = ""
	u.Fragment = ""
	return u.String()
}

// RecordAudit 落审计日志；失败只记警告，不影响主流程。
func RecordAudit(actor, action, target, detail, result string) {
	RecordAuditApp("", actor, action, target, detail, result)
}

// RecordAuditApp 在审计记录中附带应用归属，便于按应用过滤与追溯越权尝试。
func RecordAuditApp(appID, actor, action, target, detail, result string) {
	entry := AuditEntry{
		Time:   time.Now().Format(time.RFC3339),
		Actor:  sanitizeLogText(strings.TrimSpace(actor)),
		AppID:  sanitizeLogText(strings.TrimSpace(appID)),
		Action: sanitizeLogText(strings.TrimSpace(action)),
		Target: sanitizeLogText(strings.TrimSpace(target)),
		Detail: sanitizeLogText(truncateRunes(strings.TrimSpace(detail), 200)),
		Result: sanitizeLogText(strings.TrimSpace(result)),
	}
	if entry.Actor == "" {
		entry.Actor = "unknown"
	}
	auditMu.Lock()
	defer auditMu.Unlock()
	list := append(loadAuditLocked(), entry)
	if len(list) > auditKeep {
		list = list[len(list)-auditKeep:]
	}
	auditCache.list = list
	auditCache.dirty = true
	if auditCache.timer == nil {
		bucket := BucketApp
		auditCache.timer = time.AfterFunc(auditFlushDelay, func() { flushAuditTo(bucket) })
	}
}

// ListAudit 按时间顺序返回最近 limit 条审计记录。
func ListAudit(limit int) []AuditEntry {
	auditMu.Lock()
	defer auditMu.Unlock()
	list := append([]AuditEntry(nil), loadAuditLocked()...)
	if limit > 0 && len(list) > limit {
		list = list[len(list)-limit:]
	}
	return list
}

func truncateRunes(s string, n int) string {
	if n <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "..."
}

// consoleUser 优先读取 API Key 身份，其次从控制台会话读取操作者。
func consoleUser(c *gin.Context) string {
	if v, ok := c.Get(ctxAPIKeyUser); ok {
		if user, ok := v.(string); ok && user != "" {
			return user
		}
	}
	if SessionStore == nil {
		return "unknown"
	}
	s, err := SessionStore.Get(c.Request, "jiying_session")
	if err != nil {
		return "unknown"
	}
	if id, ok := s.Values["user"].(string); ok && id != "" {
		return id
	}
	return "unknown"
}

// ConsoleActor 返回当前操作者及其有效角色：API Key 请求以 Key 作用域为准，
// 会话请求以用户角色为准。定时任务归属与权限判定统一使用该结果。
func ConsoleActor(c *gin.Context) (string, string) {
	actor := consoleUser(c)
	role := UserRole(actor)
	if v, ok := c.Get(ctxAPIKeyScope); ok {
		if scope, _ := v.(string); roleRank[scope] > 0 {
			role = scope
		}
	}
	return actor, role
}

func registerAuditAPI() {
	g := Server.Group("/jiying/api", requireLogin, requireRole(RoleOps))
	g.GET("/audit", func(c *gin.Context) {
		limit := 100
		if v, err := strconv.Atoi(c.Query("limit")); err == nil && v > 0 && v <= auditKeep {
			limit = v
		}
		jiyingJSON(c, ListAudit(limit))
	})
}

// AuditLogSummary 供功能与命令回显，避免调用方自己拼格式。
func AuditLogSummary(limit int) string {
	list := ListAudit(limit)
	if len(list) == 0 {
		return "没有审计记录"
	}
	var b strings.Builder
	for _, e := range list {
		fmt.Fprintf(&b, "%s %s %s %s %s\n", e.Time, e.Actor, e.Action, e.Target, e.Result)
	}
	return strings.TrimSpace(b.String())
}
