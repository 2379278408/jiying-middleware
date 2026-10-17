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
	Action string `json:"action"`
	Target string `json:"target"`
	Detail string `json:"detail"`
	Result string `json:"result"`
}

const auditKeep = 500

var auditMu sync.Mutex

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
	entry := AuditEntry{
		Time:   time.Now().Format(time.RFC3339),
		Actor:  strings.TrimSpace(actor),
		Action: strings.TrimSpace(action),
		Target: strings.TrimSpace(target),
		Detail: truncateRunes(strings.TrimSpace(detail), 200),
		Result: strings.TrimSpace(result),
	}
	if entry.Actor == "" {
		entry.Actor = "unknown"
	}
	auditMu.Lock()
	defer auditMu.Unlock()
	list := ListAudit(auditKeep)
	list = append(list, entry)
	if len(list) > auditKeep {
		list = list[len(list)-auditKeep:]
	}
	if BucketApp == nil {
		return
	}
	raw, err := json.Marshal(list)
	if err != nil {
		GetMyLog().Warn("审计日志序列化失败: %v", err)
		return
	}
	BucketApp.Set("jiying_audit", string(raw))
}

// ListAudit 按时间顺序返回最近 limit 条审计记录。
func ListAudit(limit int) []AuditEntry {
	if BucketApp == nil {
		return nil
	}
	raw := BucketApp.GetString("jiying_audit")
	if raw == "" {
		return nil
	}
	var list []AuditEntry
	if err := json.Unmarshal([]byte(raw), &list); err != nil {
		return nil
	}
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

func registerAuditAPI() {
	g := Server.Group("/jiying/api", requireLogin)
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
