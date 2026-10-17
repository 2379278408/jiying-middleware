package core

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"jiying/middleware/utils"
)

var jiyingMsgStats = struct {
	sync.Mutex
	day     string
	inbound int
	out     int
}{}

// traceIDContextKey 在请求上下文中缓存追踪 ID，供响应体与访问日志共用同一值。
const traceIDContextKey = "jiying_trace_id"

func jiyingCountInbound() {
	jiyingMsgStats.Lock()
	defer jiyingMsgStats.Unlock()
	day := time.Now().Format("2006-01-02")
	if jiyingMsgStats.day != day {
		jiyingMsgStats.day = day
		jiyingMsgStats.inbound = 0
		jiyingMsgStats.out = 0
	}
	jiyingMsgStats.inbound++
}

func jiyingCountOutbound() {
	jiyingMsgStats.Lock()
	defer jiyingMsgStats.Unlock()
	day := time.Now().Format("2006-01-02")
	if jiyingMsgStats.day != day {
		jiyingMsgStats.day = day
		jiyingMsgStats.inbound = 0
		jiyingMsgStats.out = 0
	}
	jiyingMsgStats.out++
}

func jiyingMsgCounts() (int, int) {
	jiyingMsgStats.Lock()
	defer jiyingMsgStats.Unlock()
	if jiyingMsgStats.day != time.Now().Format("2006-01-02") {
		return 0, 0
	}
	return jiyingMsgStats.inbound, jiyingMsgStats.out
}

// jiyingRateLimitedTotal 累计应用通道被服务端限流（rate_limited）的调用次数。
var jiyingRateLimitedTotal atomic.Int64

func jiyingCountRateLimited() { jiyingRateLimitedTotal.Add(1) }

// JiyingRateLimitedTotal 返回累计被限流次数，供指标与排障读取。
func JiyingRateLimitedTotal() int64 { return jiyingRateLimitedTotal.Load() }

// jiyingCallStat 按 RPC 方法聚合调用结果，用于 /metrics 输出与运行观测。
type jiyingCallStat struct {
	OK          int64
	Failed      int64
	TotalMicros int64
	LastCode    string
	LastAt      time.Time
	Codes       map[string]int64
}

var jiyingCallStats = struct {
	sync.Mutex
	m map[string]*jiyingCallStat
}{m: map[string]*jiyingCallStat{}}

// JiyingCallSnapshot 是某个 RPC 方法的只读统计快照。
type JiyingCallSnapshot struct {
	Method   string
	OK       int64
	Failed   int64
	AvgMs    float64
	LastCode string
	LastAt   time.Time
	Codes    map[string]int64
}

// jiyingCallErrorCode 把调用错误归一成稳定标签，避免指标因错误文本变化而发散。
func jiyingCallErrorCode(err error) string {
	if err == nil {
		return ""
	}
	var ce *jiyingCallError
	if errors.As(err, &ce) && ce.Code != "" {
		return ce.Code
	}
	if errors.Is(err, errJiyingNotConnected) {
		return "not_connected"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}
	if errors.Is(err, context.Canceled) {
		return "canceled"
	}
	return "transport_error"
}

// jiyingObserveCall 记录一次 RPC 调用的耗时与结果。
func jiyingObserveCall(method string, d time.Duration, err error) {
	if method == "" {
		return
	}
	jiyingCallStats.Lock()
	defer jiyingCallStats.Unlock()
	st := jiyingCallStats.m[method]
	if st == nil {
		st = &jiyingCallStat{Codes: map[string]int64{}}
		jiyingCallStats.m[method] = st
	}
	st.TotalMicros += d.Microseconds()
	st.LastAt = time.Now()
	if err == nil {
		st.OK++
		return
	}
	code := jiyingCallErrorCode(err)
	st.Failed++
	st.LastCode = code
	st.Codes[code]++
}

// JiyingCallStats 返回按方法名排序的调用统计快照。
func JiyingCallStats() []JiyingCallSnapshot {
	jiyingCallStats.Lock()
	defer jiyingCallStats.Unlock()
	out := make([]JiyingCallSnapshot, 0, len(jiyingCallStats.m))
	for method, st := range jiyingCallStats.m {
		total := st.OK + st.Failed
		avg := float64(0)
		if total > 0 {
			avg = float64(st.TotalMicros) / float64(total) / 1000
		}
		codes := make(map[string]int64, len(st.Codes))
		for code, n := range st.Codes {
			codes[code] = n
		}
		out = append(out, JiyingCallSnapshot{
			Method: method, OK: st.OK, Failed: st.Failed,
			AvgMs: avg, LastCode: st.LastCode, LastAt: st.LastAt,
			Codes: codes,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Method < out[j].Method })
	return out
}

func jiyingJSON(c *gin.Context, data any) {
	c.JSON(http.StatusOK, gin.H{
		"ok":       true,
		"code":     200,
		"message":  "",
		"trace_id": requestTraceID(c),
		"data":     data,
	})
}

func jiyingErr(c *gin.Context, msg string) {
	c.JSON(http.StatusOK, gin.H{
		"ok":         false,
		"code":       1,
		"error_code": "REQUEST_FAILED",
		"msg":        msg,
		"message":    msg,
		"trace_id":   requestTraceID(c),
	})
}

func requestTraceID(c *gin.Context) string {
	if v, ok := c.Get(traceIDContextKey); ok {
		if s, _ := v.(string); s != "" {
			return s
		}
	}
	id := strings.TrimSpace(c.GetHeader("X-Trace-ID"))
	if !validTraceID(id) {
		id = newTraceID()
	}
	c.Set(traceIDContextKey, id)
	return id
}

// validTraceID 限制调用方传入的追踪 ID 只含安全字符：该值会回写响应头并进入访问日志，
// 放任控制字符会污染日志行。非法时由服务端重新生成。
func validTraceID(id string) bool {
	if id == "" || len(id) > 128 {
		return false
	}
	for _, r := range id {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case r == '-', r == '_', r == '.':
		default:
			return false
		}
	}
	return true
}

// requireLogin 校验控制台会话或 API Key；未认证直接以 401 中止，前端据此跳转登录页。
func requireLogin(c *gin.Context) {
	if authenticateAPIKey(c) {
		if c.IsAborted() {
			return
		}
		c.Next()
		return
	}
	if !CheckCookie(c) {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "请先登录"})
		return
	}
	c.Next()
}

func registerJiyingAPI() {
	g := Server.Group("/jiying/api", requireLogin)
	g.GET("/overview", jiyingAPIOverview)
	g.GET("/features", jiyingAPIFeatures)
	g.POST("/feature/toggle", requireRole(RoleOps), jiyingAPIFeatureToggle)
	g.POST("/feature/install", requireRole(RoleOps), jiyingAPIFeatureInstall)
	g.POST("/feature/delete", requireRole(RoleOps), jiyingAPIFeatureDelete)
	g.POST("/feature/run", requireRole(RoleOps), jiyingAPIFeatureRun)
	g.GET("/scripts", jiyingAPIScripts)
	g.POST("/script/get", requireRole(RoleOps), jiyingAPIScriptGet)
	g.POST("/script/install", requireRole(RoleOps), jiyingAPIScriptInstall)
	g.POST("/script/delete", requireRole(RoleOps), jiyingAPIScriptDelete)
	g.POST("/script/run", requireRole(RoleOps), jiyingAPIScriptRun)
	g.POST("/script/config", requireRole(RoleOps), jiyingAPIScriptConfigGet)
	g.POST("/script/config/save", requireRole(RoleOps), jiyingAPIScriptConfigSave)
	g.GET("/crons", jiyingAPICrons)
	g.POST("/cron/add", requireRole(RoleOps), jiyingAPICronAdd)
	g.POST("/cron/toggle", requireRole(RoleOps), jiyingAPICronToggle)
	g.POST("/cron/delete", requireRole(RoleOps), jiyingAPICronDelete)
	g.POST("/cron/run", requireRole(RoleOps), jiyingAPICronRun)
	g.GET("/executions", jiyingAPIExecutions)
	g.GET("/tasks", jiyingAPITasks)
	g.POST("/task/retry", requireRole(RoleOps), jiyingAPITaskRetry)
	g.POST("/task/delete", requireRole(RoleOps), jiyingAPITaskDelete)
	g.GET("/conversations", jiyingAPIConversations)
	g.GET("/messages", jiyingAPIMessages)
	g.POST("/send", requireRole(RoleOps), jiyingAPISend)
	g.POST("/send/unified", requireRole(RoleOps), jiyingAPISendUnified)
	g.GET("/conversation/capabilities", jiyingAPIConversationCapabilities)
	g.POST("/send/image", requireRole(RoleOps), jiyingAPISendImage)
	g.POST("/send/file", requireRole(RoleOps), jiyingAPISendFile)
	g.POST("/upload", requireRole(RoleOps), jiyingAPIUpload)
	g.POST("/send/card", requireRole(RoleOps), jiyingAPISendCard)
	g.POST("/send/markdown", requireRole(RoleOps), jiyingAPISendMarkdown)
	g.POST("/send/link", requireRole(RoleOps), jiyingAPISendLink)
	g.POST("/send/chart", requireRole(RoleOps), jiyingAPISendChart)
	g.GET("/logs", requireRole(RoleOps), jiyingAPILogs)
	Server.GET("/jiying", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/")
	})
}

func jiyingAPIOverview(c *gin.Context) {
	in, out := jiyingMsgCounts()
	plugins := 0
	for _, f := range FunctionsSnapshot() {
		if f != nil && !f.Origin {
			plugins++
		}
	}
	convs := 0
	ctx, cancel := context.WithTimeout(c.Request.Context(), 8*time.Second)
	defer cancel()
	if raw, err := UserCall(ctx, http.MethodGet, "/api/client/conversations", nil); err == nil {
		var parsed struct {
			Data struct {
				Conversations []json.RawMessage `json:"conversations"`
			} `json:"data"`
		}
		if json.Unmarshal(raw, &parsed) == nil {
			convs = len(parsed.Data.Conversations)
		}
	}
	appID := CurrentAppConfig().AppID
	jiyingJSON(c, gin.H{
		"ws":             JiyingState(),
		"account":        JiyingAccountStatus(),
		"uptime":         JiyingUptime(),
		"uptime_seconds": JiyingUptimeSeconds(),
		"app_id":         appID,
		"features":       len(ListFeatureItems()),
		"crons":          len(CronCmdsSnapshot()),
		"tasks":          len(ListTaskItems(200)),
		"plugins":        plugins,
		"conv_cnt":       convs,
		"msg_in":         in,
		"msg_out":        out,
		"readiness":      connectionReadiness(),
		"time":           time.Now().Format("2006-01-02 15:04:05"),
	})
}

func jiyingAPIFeatures(c *gin.Context) {
	jiyingJSON(c, ListFeatureItems())
}

func jiyingAPIFeatureToggle(c *gin.Context) {
	name := c.PostForm("name")
	enabled := c.PostForm("enabled") == "true" || c.PostForm("enabled") == "1"
	if err := SetFeatureEnabled(name, enabled); err != nil {
		RecordAudit(consoleUser(c), "feature.toggle", name, fmt.Sprintf("enabled=%v", enabled), "失败: "+err.Error())
		jiyingErr(c, err.Error())
		return
	}
	RecordAudit(consoleUser(c), "feature.toggle", name, fmt.Sprintf("enabled=%v", enabled), "成功")
	jiyingJSON(c, "ok")
}

func jiyingAPIFeatureInstall(c *gin.Context) {
	name := strings.TrimSpace(c.PostForm("name"))
	url := strings.TrimSpace(c.PostForm("url"))
	if name == "" || url == "" {
		jiyingErr(c, "缺少功能名或地址")
		return
	}
	if err := UpsertFeature(Feature{Name: name, Kind: "http", Target: url, Scope: "master", Identity: "app", Enabled: true, Confirm: true}); err != nil {
		RecordAudit(consoleUser(c), "feature.install", name, auditURL(url), "失败: "+err.Error())
		jiyingErr(c, err.Error())
		return
	}
	RecordAudit(consoleUser(c), "feature.install", name, auditURL(url), "成功")
	jiyingJSON(c, "已安装功能 "+name)
}

func jiyingAPIFeatureDelete(c *gin.Context) {
	name := strings.TrimSpace(c.PostForm("name"))
	if err := deleteFeatureWithPlugin(name); err != nil {
		RecordAudit(consoleUser(c), "feature.delete", name, "", "失败: "+err.Error())
		jiyingErr(c, err.Error())
		return
	}
	RecordAudit(consoleUser(c), "feature.delete", name, "", "成功")
	jiyingJSON(c, "ok")
}

// deleteFeatureWithPlugin 删除功能；插件型功能与脚本文件一一对应，同步移除插件文件，避免残留。
// 若仍有其他功能引用同一脚本文件，则保留文件。
func deleteFeatureWithPlugin(name string) error {
	f, has := getFeatureRaw(name)
	if err := DeleteFeature(name); err != nil {
		return err
	}
	if has && f != nil && f.Kind == "script" && strings.TrimSpace(f.Target) != "" {
		if !featureTargetInUse(f.Target) {
			if full := safeScriptPath(f.Target); full != "" {
				_ = os.Remove(full)
			}
			// 文件删除后同步卸载内存中的插件，避免命令表继续暴露已删除的脚本。
			UnloadScript(f.Target)
		}
	}
	return nil
}

func jiyingAPIFeatureRun(c *gin.Context) {
	name := strings.TrimSpace(c.PostForm("name"))
	args := c.PostForm("args")
	if name == "" {
		jiyingErr(c, "缺少功能名")
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()
	out, err := executeFeature(ctx, name, args, "system", "")
	if err != nil {
		RecordAudit(consoleUser(c), "feature.run", name, args, "失败: "+err.Error())
		jiyingErr(c, err.Error())
		return
	}
	RecordAudit(consoleUser(c), "feature.run", name, args, "成功")
	jiyingJSON(c, out)
}

func jiyingAPIScripts(c *gin.Context) {
	type item struct {
		Name        string              `json:"name"`
		Size        int64               `json:"size"`
		Lang        string              `json:"lang"`
		Title       string              `json:"title"`
		Version     string              `json:"version,omitempty"`
		Author      string              `json:"author,omitempty"`
		Class       string              `json:"class,omitempty"`
		Service     string              `json:"service,omitempty"`
		Apps        []string            `json:"apps,omitempty"`
		Description string              `json:"description,omitempty"`
		Rules       []string            `json:"rules,omitempty"`
		Cron        []string            `json:"cron,omitempty"`
		Platform    []string            `json:"platform,omitempty"`
		Priority    *int                `json:"priority,omitempty"`
		Admin       bool                `json:"admin"`
		Icon        string              `json:"icon,omitempty"`
		Config      []PluginConfigField `json:"config_schema,omitempty"`
		HasManifest bool                `json:"has_manifest"`
	}
	out := []item{}
	entries, err := os.ReadDir(scriptsDir())
	if err == nil {
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			lang := scriptLangOf(e.Name())
			if lang == "" {
				continue
			}
			fi, _ := e.Info()
			size := int64(0)
			if fi != nil {
				size = fi.Size()
			}
			data, _ := os.ReadFile(filepath.Join(scriptsDir(), e.Name()))
			content := string(data)
			m := ParsePluginManifest(content)
			out = append(out, item{
				Name:        e.Name(),
				Size:        size,
				Lang:        lang,
				Title:       pluginTitle(e.Name(), content),
				Version:     m.Version,
				Author:      m.Author,
				Class:       m.Class,
				Service:     m.Service,
				Apps:        m.Apps,
				Description: m.Description,
				Rules:       m.Rules,
				Cron:        m.Cron,
				Platform:    m.Platform,
				Priority:    m.Priority,
				Admin:       m.Admin != nil && *m.Admin,
				Icon:        m.Icon,
				Config:      redactConfigSchema(m.Config),
				HasManifest: m.HasManifest,
			})
		}
	}
	jiyingJSON(c, out)
}

func jiyingAPIScriptGet(c *gin.Context) {
	name := strings.TrimSpace(c.PostForm("name"))
	full := safeScriptPath(name)
	if full == "" {
		jiyingErr(c, "非法脚本名")
		return
	}
	data, err := os.ReadFile(full)
	if err != nil {
		jiyingErr(c, "读取失败: "+err.Error())
		return
	}
	jiyingJSON(c, gin.H{"name": filepathBase(name), "lang": scriptLangOf(name), "content": string(data)})
}

func jiyingAPIScriptInstall(c *gin.Context) {
	name := strings.TrimSpace(c.PostForm("name"))
	code := c.PostForm("code")
	if name == "" || code == "" {
		jiyingErr(c, "缺少文件名或代码")
		return
	}
	if err := InstallScript(name, code); err != nil {
		RecordAudit(consoleUser(c), "script.install", name, "", "失败: "+err.Error())
		jiyingErr(c, "安装失败: "+err.Error())
		return
	}
	base := filepathBase(name)
	title, scope := scriptFeatureNameScope(base, code)
	m := ParsePluginManifest(code)
	_ = UpsertFeature(Feature{Name: title, Kind: "script", Target: base, Scope: scope, Identity: "app", Enabled: true, Class: m.Class, Apps: m.Apps})
	RecordAudit(consoleUser(c), "script.install", base, fmt.Sprintf("%d 字节", len(code)), "成功")
	jiyingJSON(c, "脚本已安装 "+base)
}

func jiyingAPIScriptDelete(c *gin.Context) {
	name := strings.TrimSpace(c.PostForm("name"))
	full := safeScriptPath(name)
	if full == "" {
		jiyingErr(c, "非法脚本名")
		return
	}
	data, _ := os.ReadFile(full)
	title, _ := scriptFeatureNameScope(name, string(data))
	if err := os.Remove(full); err != nil && !os.IsNotExist(err) {
		RecordAudit(consoleUser(c), "script.delete", name, "", "失败: "+err.Error())
		jiyingErr(c, err.Error())
		return
	}
	// 文件可能已被移除，仍需清理内存插件与功能记录，保证删除幂等。
	UnloadScript(name)
	if BucketApp != nil {
		BucketApp.Delete(pluginConfigBucketKey(name))
	}
	if title != "" {
		_ = DeleteFeature(title)
	}
	RecordAudit(consoleUser(c), "script.delete", name, "", "成功")
	jiyingJSON(c, "ok")
}

func jiyingAPIScriptRun(c *gin.Context) {
	name := strings.TrimSpace(c.PostForm("name"))
	args := c.PostForm("args")
	out, err := runScriptFile(name, args)
	if err != nil {
		RecordAudit(consoleUser(c), "script.run", name, args, "失败: "+err.Error())
		jiyingErr(c, err.Error())
		return
	}
	RecordAudit(consoleUser(c), "script.run", name, args, "成功")
	jiyingJSON(c, out)
}

// jiyingAPIScriptConfigGet 返回插件配置 schema 与当前值；secret 字段只回传是否已设置。
func jiyingAPIScriptConfigGet(c *gin.Context) {
	name := strings.TrimSpace(c.PostForm("name"))
	full := safeScriptPath(name)
	if full == "" {
		jiyingErr(c, "非法脚本名")
		return
	}
	data, err := os.ReadFile(full)
	if err != nil {
		jiyingErr(c, "读取失败: "+err.Error())
		return
	}
	schema := ParsePluginManifest(string(data)).Config
	values, set := pluginConfigForAPI(schema, LoadPluginConfig(name))
	jiyingJSON(c, gin.H{"schema": redactConfigSchema(schema), "values": values, "set": set})
}

// jiyingAPIScriptConfigSave 按 schema 校验后保存插件配置；密钥值不写入审计。
func jiyingAPIScriptConfigSave(c *gin.Context) {
	name := strings.TrimSpace(c.PostForm("name"))
	full := safeScriptPath(name)
	if full == "" {
		jiyingErr(c, "非法脚本名")
		return
	}
	data, err := os.ReadFile(full)
	if err != nil {
		jiyingErr(c, "读取失败: "+err.Error())
		return
	}
	schema := ParsePluginManifest(string(data)).Config
	if len(schema) == 0 {
		jiyingErr(c, "该插件未声明可配置项")
		return
	}
	in := map[string]string{}
	if raw := c.PostForm("values"); raw != "" {
		if err := json.Unmarshal([]byte(raw), &in); err != nil {
			jiyingErr(c, "配置格式错误")
			return
		}
	}
	merged, err := mergePluginConfig(schema, LoadPluginConfig(name), in)
	if err != nil {
		RecordAudit(consoleUser(c), "script.config", name, "", "失败: "+err.Error())
		jiyingErr(c, err.Error())
		return
	}
	if err := savePluginConfig(name, merged); err != nil {
		RecordAudit(consoleUser(c), "script.config", name, "", "失败: "+err.Error())
		jiyingErr(c, "保存失败: "+err.Error())
		return
	}
	RecordAudit(consoleUser(c), "script.config", name, strings.Join(configKeys(schema), ","), "成功")
	jiyingJSON(c, "配置已保存")
}

func jiyingAPICrons(c *gin.Context) {
	jiyingJSON(c, CronCmdsSnapshot())
}

func jiyingAPICronAdd(c *gin.Context) {
	name := strings.TrimSpace(c.PostForm("name"))
	spec := strings.TrimSpace(c.PostForm("cron"))
	command := strings.TrimSpace(c.PostForm("command"))
	chat := strings.TrimSpace(c.PostForm("chat_id"))
	if name == "" || spec == "" || command == "" {
		jiyingErr(c, "缺少名称/表达式/命令")
		return
	}
	if !isValidCronSpec(spec) {
		RecordAudit(consoleUser(c), "cron.add", name, spec+" "+command, "失败: 表达式无效")
		jiyingErr(c, "Cron 表达式无效，支持 5 段或 6 段（含秒）写法，如 0 */30 * * * *（每 30 分钟）")
		return
	}
	if parsed, isJSON, err := parseCronFeatureCommand(command); isJSON {
		if err != nil {
			jiyingErr(c, err.Error())
			return
		}
		if parsed.Type == "feature" {
			if _, ok := GetFeature(strings.TrimSpace(parsed.Feature)); !ok {
				jiyingErr(c, "功能不存在: "+parsed.Feature)
				return
			}
			if conv := strings.TrimSpace(parsed.ConversationID); conv != "" && !validConversationID(conv) {
				jiyingErr(c, "会话 ID 不合法")
				return
			}
		}
	}
	actor, role := ConsoleActor(c)
	cmd := &CronCmd{Name: name, Cron: spec, Command: command, Status: true, Type: []string{"web"}, Owner: actor, OwnerRole: role}
	if chat != "" {
		cmd.GroupManager = []string{chat}
	}
	id := ccs.add(cmd)
	RecordAudit(consoleUser(c), "cron.add", name, spec+" "+command, "成功")
	jiyingJSON(c, gin.H{"id": id})
}

func jiyingAPICronToggle(c *gin.Context) {
	id := strings.TrimSpace(c.PostForm("id"))
	on := c.PostForm("enabled") == "true" || c.PostForm("enabled") == "1"
	actor, role := ConsoleActor(c)
	if !SetCronCmdStatus(id, on, actor, role) {
		RecordAudit(consoleUser(c), "cron.toggle", id, fmt.Sprintf("enabled=%v", on), "失败: 定时不存在")
		jiyingErr(c, "定时不存在")
		return
	}
	RecordAudit(consoleUser(c), "cron.toggle", id, fmt.Sprintf("enabled=%v", on), "成功")
	jiyingJSON(c, "ok")
}

func jiyingAPICronRun(c *gin.Context) {
	id := strings.TrimSpace(c.PostForm("id"))
	actor, role := ConsoleActor(c)
	if RunCronCmdsByIds([]string{id}, actor, role) == 0 {
		jiyingErr(c, "定时不存在")
		return
	}
	RecordAudit(consoleUser(c), "cron.run", id, "", "成功")
	jiyingJSON(c, "ok")
}

func jiyingAPIExecutions(c *gin.Context) {
	limit := 100
	if raw := strings.TrimSpace(c.Query("limit")); raw != "" {
		if _, err := fmt.Sscanf(raw, "%d", &limit); err != nil || limit <= 0 {
			limit = 100
		}
	}
	jiyingJSON(c, listExecutions(ExecutionFilter{
		Kind:   strings.TrimSpace(c.Query("kind")),
		Name:   strings.TrimSpace(c.Query("name")),
		Status: strings.TrimSpace(c.Query("status")),
		Limit:  limit,
	}))
}

func jiyingAPICronDelete(c *gin.Context) {
	id := strings.TrimSpace(c.PostForm("id"))
	actor, role := ConsoleActor(c)
	if DeleteCronCmdsByIds([]string{id}, actor, role) == 0 {
		RecordAudit(consoleUser(c), "cron.delete", id, "", "失败: 定时不存在")
		jiyingErr(c, "定时不存在")
		return
	}
	RecordAudit(consoleUser(c), "cron.delete", id, "", "成功")
	jiyingJSON(c, "ok")
}

func jiyingAPITasks(c *gin.Context) {
	limit := 100
	if raw := strings.TrimSpace(c.Query("limit")); raw != "" {
		if _, err := fmt.Sscanf(raw, "%d", &limit); err != nil || limit <= 0 {
			limit = 100
		}
	}
	jiyingJSON(c, listTaskItems(TaskFilter{
		Status:  strings.TrimSpace(c.Query("status")),
		Feature: strings.TrimSpace(c.Query("feature")),
		UserID:  strings.TrimSpace(c.Query("user_id")),
		Limit:   limit,
	}))
}

func jiyingAPITaskRetry(c *gin.Context) {
	id := strings.TrimSpace(c.PostForm("id"))
	if err := RetryTask(id); err != nil {
		jiyingErr(c, err.Error())
		return
	}
	jiyingJSON(c, "ok")
}

func jiyingAPITaskDelete(c *gin.Context) {
	id := strings.TrimSpace(c.PostForm("id"))
	if !DeleteTask(id) {
		jiyingErr(c, "任务不存在")
		return
	}
	jiyingJSON(c, "ok")
}

func jiyingAPIConversations(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()
	raw, err := UserCall(ctx, http.MethodGet, "/api/client/conversations", nil)
	if err != nil {
		jiyingErr(c, err.Error())
		return
	}
	var parsed struct {
		Data json.RawMessage `json:"data"`
	}
	if json.Unmarshal(raw, &parsed) != nil || len(parsed.Data) == 0 {
		jiyingErr(c, "会话列表解析失败")
		return
	}
	c.Data(http.StatusOK, "application/json; charset=utf-8", wrapJiyingData(parsed.Data))
}

func wrapJiyingData(data json.RawMessage) []byte {
	out, _ := json.Marshal(gin.H{"code": 200, "data": json.RawMessage(data)})
	return out
}

// validConversationID 限制会话 ID 只含安全字符。该值会被拼进上游请求路径
// （/api/client/conversations/<id>/messages），若含 /、.、? 等字符可改变请求目标。
func validConversationID(id string) bool {
	if id == "" || id == "." || id == ".." || len(id) > 128 {
		return false
	}
	for _, r := range id {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case r == '-', r == '_', r == '.':
		default:
			return false
		}
	}
	return true
}

func jiyingAPIMessages(c *gin.Context) {
	convID := strings.TrimSpace(c.Query("conversation_id"))
	if convID == "" {
		jiyingErr(c, "缺少 conversation_id")
		return
	}
	if !validConversationID(convID) {
		jiyingErr(c, "会话 ID 不合法")
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()
	raw, err := UserCall(ctx, http.MethodGet, "/api/client/conversations/"+convID+"/messages", nil)
	if err != nil {
		jiyingErr(c, err.Error())
		return
	}
	var parsed struct {
		Data json.RawMessage `json:"data"`
	}
	if json.Unmarshal(raw, &parsed) != nil || len(parsed.Data) == 0 {
		jiyingErr(c, "消息解析失败")
		return
	}
	c.Data(http.StatusOK, "application/json; charset=utf-8", wrapJiyingData(parsed.Data))
}

func jiyingAPIConversationCapabilities(c *gin.Context) {
	convID := strings.TrimSpace(c.Query("conversation_id"))
	if !validConversationID(convID) {
		jiyingErr(c, "会话 ID 不合法")
		return
	}
	jiyingJSON(c, conversationCapabilities(convID))
}

func jiyingAPISendUnified(c *gin.Context) {
	convID := strings.TrimSpace(c.PostForm("conversation_id"))
	text := strings.TrimSpace(c.PostForm("text"))
	identity := strings.TrimSpace(c.PostForm("identity"))
	dryRun := c.PostForm("dry_run") == "true" || c.PostForm("dry_run") == "1"
	result, err := SendMessage(c.Request.Context(), SendOptions{
		ConversationID: convID,
		Text:           text,
		Identity:       identity,
		ReplyTo:        strings.TrimSpace(c.PostForm("reply_to")),
		DryRun:         dryRun,
	})
	if err != nil {
		RecordAudit(consoleUser(c), "message.send", convID, identity, "失败: "+err.Error())
		jiyingErr(c, err.Error())
		return
	}
	summary := truncateText(text, 50)
	if dryRun {
		RecordAudit(consoleUser(c), "message.dry_run", convID, result.Identity, summary)
	} else {
		jiyingCountOutbound()
		RecordAudit(consoleUser(c), "message.send", convID, result.Identity, summary)
	}
	jiyingJSON(c, result)
}

func jiyingAPISend(c *gin.Context) {
	convID := strings.TrimSpace(c.PostForm("conversation_id"))
	text := strings.TrimSpace(c.PostForm("text"))
	if convID == "" || text == "" {
		jiyingErr(c, "缺少会话或内容")
		return
	}
	if clientForConv(convID) == nil {
		jiyingErr(c, "即应未连接")
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()
	if err := SendTextToConv(ctx, convID, "", text); err != nil {
		jiyingErr(c, err.Error())
		return
	}
	jiyingCountOutbound()
	jiyingJSON(c, "ok")
}

// jiyingAPISendMedia 统一校验媒体发送所需会话并落审计。
func jiyingAPISendMedia(c *gin.Context, action string, message map[string]any) {
	convID := strings.TrimSpace(c.PostForm("conversation_id"))
	if convID == "" {
		jiyingErr(c, "缺少会话")
		return
	}
	if clientForConv(convID) == nil {
		jiyingErr(c, "即应未连接")
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()
	if _, err := SendMediaToConv(ctx, convID, strings.TrimSpace(c.PostForm("reply_to")), message); err != nil {
		RecordAudit(consoleUser(c), action, convID, "", "失败: "+err.Error())
		jiyingErr(c, err.Error())
		return
	}
	jiyingCountOutbound()
	RecordAudit(consoleUser(c), action, convID, "", "成功")
	jiyingJSON(c, "ok")
}

// jiyingAPISendResolvedMedia 用于 media 消息体需先解析可拉取 URL 的场景。
func jiyingAPISendResolvedMedia(c *gin.Context, action string, build func(ctx context.Context) (map[string]any, error)) {
	convID := strings.TrimSpace(c.PostForm("conversation_id"))
	if convID == "" {
		jiyingErr(c, "缺少会话")
		return
	}
	if clientForConv(convID) == nil {
		jiyingErr(c, "即应未连接")
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 20*time.Second)
	defer cancel()
	message, err := build(ctx)
	if err != nil {
		RecordAudit(consoleUser(c), action, convID, "", "失败: "+err.Error())
		jiyingErr(c, err.Error())
		return
	}
	if _, err := SendMediaToConv(ctx, convID, strings.TrimSpace(c.PostForm("reply_to")), message); err != nil {
		RecordAudit(consoleUser(c), action, convID, "", "失败: "+err.Error())
		jiyingErr(c, err.Error())
		return
	}
	jiyingCountOutbound()
	RecordAudit(consoleUser(c), action, convID, "", "成功")
	jiyingJSON(c, "ok")
}

func jiyingAPISendImage(c *gin.Context) {
	src := strings.TrimSpace(c.PostForm("url"))
	if src == "" {
		src = strings.TrimSpace(c.PostForm("file_id"))
	}
	if src == "" {
		jiyingErr(c, "缺少 url 或 file_id")
		return
	}
	jiyingAPISendResolvedMedia(c, "media.image", func(ctx context.Context) (map[string]any, error) {
		return buildImageMessage(ctx, strings.TrimSpace(c.PostForm("conversation_id")), src, c.PostForm("caption"), c.PostForm("caption_type"))
	})
}

const (
	maxUploadBytes        = 20 << 20
	maxUploadRequestBytes = 30 << 20
)

// uploadSlots 限制并发上传数，避免多个大文件同时解码与缓冲放大内存峰值。
var uploadSlots = make(chan struct{}, 2)

// cleanUploadContentType 限制类型长度并校验语法，非法值回退为二进制流。
func cleanUploadContentType(v string) string {
	v = strings.TrimSpace(v)
	if v == "" || len(v) > 100 || strings.ContainsAny(v, "\r\n") {
		return "application/octet-stream"
	}
	if _, _, err := mime.ParseMediaType(v); err != nil {
		return "application/octet-stream"
	}
	return v
}

func decodeUploadBase64(v string) ([]byte, error) {
	v = strings.TrimSpace(v)
	v = strings.NewReplacer("\r", "", "\n", "").Replace(v)
	if v == "" {
		return nil, fmt.Errorf("缺少文件内容")
	}
	if data, err := base64.StdEncoding.DecodeString(v); err == nil {
		return data, nil
	}
	if data, err := base64.RawStdEncoding.DecodeString(v); err == nil {
		return data, nil
	}
	return nil, fmt.Errorf("文件内容不是合法的 base64")
}

// jiyingAPIUpload 上传临时文件并返回 file_id，供 send/file 等接口引用。
// 支持 multipart 的 file 字段，或表单字段 name/content_type/data_base64。
func jiyingAPIUpload(c *gin.Context) {
	select {
	case uploadSlots <- struct{}{}:
		defer func() { <-uploadSlots }()
	default:
		jiyingErr(c, "上传任务繁忙，请稍后重试")
		return
	}
	if c.Request.ContentLength > maxUploadRequestBytes {
		jiyingErr(c, fmt.Sprintf("上传内容过大，最大 %d MB", maxUploadBytes>>20))
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadRequestBytes)
	name := strings.TrimSpace(c.PostForm("name"))
	contentType := strings.TrimSpace(c.PostForm("content_type"))
	var (
		data []byte
		err  error
	)
	if strings.HasPrefix(c.ContentType(), "multipart/form-data") {
		fh, ferr := c.FormFile("file")
		if ferr != nil {
			jiyingErr(c, "读取上传文件失败: "+ferr.Error())
			return
		}
		if fh.Size > maxUploadBytes {
			jiyingErr(c, fmt.Sprintf("上传内容过大，最大 %d MB", maxUploadBytes>>20))
			return
		}
		f, oerr := fh.Open()
		if oerr != nil {
			jiyingErr(c, "读取上传文件失败: "+oerr.Error())
			return
		}
		data, err = io.ReadAll(io.LimitReader(f, maxUploadBytes+1))
		f.Close()
		if err != nil {
			jiyingErr(c, "读取上传文件失败: "+err.Error())
			return
		}
		if len(data) > maxUploadBytes {
			jiyingErr(c, fmt.Sprintf("上传内容过大，最大 %d MB", maxUploadBytes>>20))
			return
		}
		if name == "" {
			name = fh.Filename
		}
		if contentType == "" {
			contentType = fh.Header.Get("Content-Type")
		}
	} else {
		raw := c.PostForm("data_base64")
		if raw == "" {
			raw = c.PostForm("content_base64")
		}
		data, err = decodeUploadBase64(raw)
		if err != nil {
			jiyingErr(c, err.Error())
			return
		}
		if len(data) > maxUploadBytes {
			jiyingErr(c, fmt.Sprintf("上传内容过大，最大 %d MB", maxUploadBytes>>20))
			return
		}
	}
	if len(data) == 0 {
		jiyingErr(c, "上传内容为空")
		return
	}
	if strings.TrimSpace(name) == "" {
		jiyingErr(c, "请填写文件名")
		return
	}
	name = sanitizeFilename(name)
	contentType = cleanUploadContentType(contentType)
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()
	fileID, err := UserUploadFile(ctx, name, data, contentType)
	if err != nil {
		RecordAudit(consoleUser(c), "file.upload", name, fmt.Sprintf("%d bytes", len(data)), "失败: "+err.Error())
		jiyingErr(c, err.Error())
		return
	}
	RecordAudit(consoleUser(c), "file.upload", name, fmt.Sprintf("%d bytes · %s", len(data), contentType), "成功")
	jiyingJSON(c, gin.H{"file_id": fileID, "name": name, "size": len(data), "content_type": contentType})
}

func jiyingAPISendFile(c *gin.Context) {
	src := strings.TrimSpace(c.PostForm("url"))
	if src == "" {
		src = strings.TrimSpace(c.PostForm("file_id"))
	}
	if src == "" {
		jiyingErr(c, "缺少 url 或 file_id")
		return
	}
	var size int64
	fmt.Sscan(c.PostForm("size"), &size)
	jiyingAPISendResolvedMedia(c, "media.file", func(ctx context.Context) (map[string]any, error) {
		return buildFileMessage(ctx, strings.TrimSpace(c.PostForm("conversation_id")), src, c.PostForm("name"), size)
	})
}

func jiyingAPISendCard(c *gin.Context) {
	title := strings.TrimSpace(c.PostForm("title"))
	if title == "" {
		jiyingErr(c, "缺少标题")
		return
	}
	jiyingAPISendMedia(c, "media.card", CardMessage(title, c.PostForm("description"), c.PostForm("url")))
}

func jiyingAPISendMarkdown(c *gin.Context) {
	content := strings.TrimSpace(c.PostForm("content"))
	if content == "" {
		content = strings.TrimSpace(c.PostForm("text"))
	}
	if content == "" {
		jiyingErr(c, "缺少内容")
		return
	}
	jiyingAPISendMedia(c, "media.markdown", MarkdownMessage(content))
}

func jiyingAPISendLink(c *gin.Context) {
	url := strings.TrimSpace(c.PostForm("url"))
	if url == "" {
		jiyingErr(c, "缺少链接")
		return
	}
	jiyingAPISendMedia(c, "media.link", LinkMessage(url, c.PostForm("title")))
}

func jiyingAPISendChart(c *gin.Context) {
	title := strings.TrimSpace(c.PostForm("title"))
	if title == "" {
		jiyingErr(c, "缺少标题")
		return
	}
	data := map[string]any{}
	if raw := strings.TrimSpace(c.PostForm("data")); raw != "" {
		if err := json.Unmarshal([]byte(raw), &data); err != nil {
			jiyingErr(c, "图表数据不是合法 JSON")
			return
		}
	}
	jiyingAPISendMedia(c, "media.chart", ChartMessage(c.PostForm("chart_type"), title, c.PostForm("description"), data))
}

func jiyingAPILogs(c *gin.Context) {
	tail := 300
	if v := c.Query("tail"); v != "" {
		fmt.Sscan(v, &tail)
	}
	if tail <= 0 || tail > 2000 {
		tail = 300
	}
	day := c.Query("day")
	if day == "" {
		day = time.Now().Format("20060102")
	}
	day = strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, day)
	if len(day) != 8 {
		jiyingErr(c, "日期格式错误")
		return
	}
	path := filepath.Join(utils.GetResHome(), "logs", day+".log")
	lines, err := readTailLines(path, tail, maxLogTailBytes)
	if err != nil {
		jiyingJSON(c, gin.H{"day": day, "lines": []string{}})
		return
	}
	jiyingJSON(c, gin.H{"day": day, "lines": lines})
}
