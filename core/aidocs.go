package core

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// apiDoc 描述一个可供 AI 调用的控制台接口。Fields 为表单或查询参数名，
// 全部按字符串传输，布尔值使用 "true"/"false"。
type apiDoc struct {
	Method  string
	Path    string
	Summary string
	Scope   string
	Fields  []string
	Notes   string
}

var aiEndpoints = []apiDoc{
	{"GET", "/jiying/api/overview", "总览：连接、账号、应用、功能、定时、任务、插件与就绪状态", "readonly", nil, ""},
	{"GET", "/jiying/api/features", "功能列表（名称、类型、目标、权限、启用状态、应用范围）", "readonly", nil, "apps 为空或 [\"*\"] 表示全部应用；apps_locked=true 表示范围由控制台手动指定"},
	{"POST", "/jiying/api/feature/toggle", "启用或停用功能", "ops", []string{"name", "enabled"}, `enabled 取 "true"/"false"`},
	{"POST", "/jiying/api/feature/install", "安装 HTTP 功能（转发到外部地址）", "ops", []string{"name", "url"}, ""},
	{"POST", "/jiying/api/feature/delete", "删除功能；脚本型功能会同步删除插件文件", "ops", []string{"name"}, "破坏性操作，执行前确认"},
	{"POST", "/jiying/api/feature/update", "调整功能的权限范围、身份、确认与应用范围", "ops", []string{"name", "target", "scope", "identity", "confirm", "apps", "reset_apps"}, "仅 http 功能可改 target；apps 支持 JSON 数组或逗号/空格分隔，* 表示全部应用；手动保存后 apps_locked=true，插件重装/重扫保留；reset_apps=true 恢复插件清单声明并解锁"},
	{"POST", "/jiying/api/feature/run", "执行功能并返回输出", "ops", []string{"name", "args"}, "args 为字符串，按功能自行解析；按当前用户权限与会话归属应用校验"},
	{"GET", "/jiying/api/scripts", "插件列表与清单（title/class/service/apps/description/rules/config_schema）", "readonly", nil, "apps 为插件声明的应用范围；功能上的 apps_locked 表示控制台手动覆盖"},
	{"POST", "/jiying/api/script/get", "读取插件源码与语言", "ops", []string{"name"}, ""},
	{"POST", "/jiying/api/script/install", "安装或覆盖插件源码", "ops", []string{"name", "code", "confirm"}, "name 扩展名 .js/.py 决定运行方式；文件头可声明 class/service/apps 与 network，已有手动应用范围不会被覆盖；高风险代码需 confirm=true"},
	{"POST", "/jiying/api/script/delete", "删除插件文件与配置", "ops", []string{"name"}, "破坏性操作，执行前确认"},
	{"POST", "/jiying/api/script/run", "直接运行插件", "ops", []string{"name", "args"}, ""},
	{"POST", "/jiying/api/script/config", "读取插件配置 schema 与当前值", "ops", []string{"name"}, "secret 字段只返回是否已设置"},
	{"POST", "/jiying/api/script/config/save", "保存插件配置", "ops", []string{"name", "values"}, "values 为 JSON 对象字符串"},
	{"GET", "/jiying/api/crons", "定时任务列表（含下次运行时间与最近结果）", "readonly", nil, "主控返回全部，其他身份只返回本人创建的任务"},
	{"POST", "/jiying/api/cron/add", "新建定时任务", "ops", []string{"name", "cron", "command", "chat_id"}, "command 支持功能 JSON 或系统命令；chat_id 可省略"},
	{"POST", "/jiying/api/cron/update", "修改并重新调度定时任务", "ops", []string{"id", "name", "cron", "command", "chat_id"}, "字段可选，只更新传入项；非法表达式或命令返回错误且保留原任务"},
	{"POST", "/jiying/api/cron/toggle", "启用或停用定时任务", "ops", []string{"id", "enabled"}, "仅本人任务或主控可操作"},
	{"POST", "/jiying/api/cron/delete", "删除定时任务", "ops", []string{"id"}, "破坏性操作，执行前确认；仅本人任务或主控可操作"},
	{"POST", "/jiying/api/cron/run", "立即执行一次定时任务", "ops", []string{"id"}, "仅本人任务或主控可操作"},
	{"GET", "/jiying/api/executions", "执行账本", "readonly", []string{"kind", "name", "status", "limit"}, "kind: cron/feature/task/message"},
	{"GET", "/jiying/api/tasks", "任务队列（含 app_id 应用归属）", "readonly", []string{"status", "feature", "user_id", "limit", "silent"}, "主控返回全部，其他身份只返回本人发起的任务；silent=1 返回最近 50 条 Agent 工具调用（静默任务）"},
	{"POST", "/jiying/api/task/retry", "重试任务", "ops", []string{"id"}, "仅本人任务或主控可操作，并写入审计"},
	{"POST", "/jiying/api/task/delete", "删除任务", "ops", []string{"id"}, "仅本人任务或主控可操作，并写入审计"},
	{"GET", "/jiying/api/conversations", "会话列表（个人账号与在线应用合并，含 relay_app_* 归属字段）", "readonly", nil, "账号未登录或查询失败时仍返回应用会话；relay_app_known=false 且 relay_account=true 表示账号会话，relay_app_configured=false 表示创建应用尚未接入"},
	{"GET", "/jiying/api/messages", "会话消息", "readonly", []string{"conversation_id"}, "账号通道不可用时按会话归属应用回退读取最近 50 条，data.source=app"},
	{"GET", "/jiying/api/conversation/capabilities", "会话能力（发送身份与应用归属）", "readonly", []string{"conversation_id"}, "返回 app_id/app_name/app_online/app_known/account_email/reason"},
	{"POST", "/jiying/api/send", "发送纯文本（旧接口）", "ops", []string{"conversation_id", "text"}, "建议改用 /send/unified；实际通道由会话归属应用决定"},
	{"POST", "/jiying/api/send/unified", "统一发送：自动选择身份与消息类型", "ops", []string{"conversation_id", "text", "identity", "reply_to", "dry_run"}, "identity 支持 auto/app/user，auto 优先应用身份；通道按会话归属应用选择，不接受 app_id；dry_run=true 只预演"},
	{"POST", "/jiying/api/send/image", "发送图片", "ops", []string{"conversation_id", "url", "file_id", "caption", "caption_type", "reply_to"}, "url 与 file_id 二选一；账号会话优先用账号身份，应用归属会话走应用通道"},
	{"POST", "/jiying/api/send/file", "发送文件", "ops", []string{"conversation_id", "url", "file_id", "name", "size", "reply_to"}, "url 与 file_id 二选一；size 为字节数；账号会话优先用账号身份，应用归属会话走应用通道"},
	{"POST", "/jiying/api/upload", "上传临时文件并获取 file_id", "ops", []string{"name", "content_type", "data_base64"}, "也支持 multipart 的 file 字段；最大 20 MB"},
	{"POST", "/jiying/api/send/card", "发送卡片", "ops", []string{"conversation_id", "title", "description", "url", "reply_to"}, "通道按会话归属应用选择"},
	{"POST", "/jiying/api/send/markdown", "发送 Markdown", "ops", []string{"conversation_id", "content", "text", "reply_to"}, "content 与 text 等价；通道按会话归属应用选择"},
	{"POST", "/jiying/api/send/link", "发送链接卡片", "ops", []string{"conversation_id", "url", "title", "reply_to"}, "通道按会话归属应用选择"},
	{"POST", "/jiying/api/send/chart", "发送图表", "ops", []string{"conversation_id", "chart_type", "title", "description", "data", "reply_to"}, "data 为 JSON 字符串；通道按会话归属应用选择"},
	{"GET", "/jiying/api/logs", "读取运行日志", "ops", []string{"tail", "day"}, "tail 最大 2000；day 形如 20261017"},
	{"GET", "/jiying/api/connect/status", "接入状态：应用列表、账号列表与资料、运行时就绪度", "readonly", nil, ""},
	{"GET", "/jiying/api/connect/readiness", "接入就绪度检查", "readonly", nil, ""},
	{"POST", "/jiying/api/connect/app", "配置即应应用（WebSocket 地址等）", "ops", []string{"ws_url", "app_id", "secret"}, "敏感字段用 **** 占位表示不修改"},
	{"POST", "/jiying/api/connect/test-app", "测试应用连接", "ops", []string{"ws_url", "app_id", "secret"}, ""},
	{"POST", "/jiying/api/connect/account", "配置账号", "ops", []string{"base_url", "email", "password", "code_login"}, ""},
	{"POST", "/jiying/api/connect/test-account", "测试账号连接", "ops", []string{"base_url", "email", "password"}, ""},
	{"POST", "/jiying/api/connect/account/logout", "退出即应账号", "ops", nil, ""},
	{"GET", "/jiying/api/connect/account/health", "账号健康快照", "readonly", nil, ""},
	{"POST", "/jiying/api/connect/account/health/check", "立即检查账号健康", "ops", nil, ""},
	{"POST", "/jiying/api/connect/request-code", "请求邮箱验证码", "ops", []string{"email"}, ""},
	{"POST", "/jiying/api/connect/login-code", "使用验证码登录", "ops", []string{"email", "code"}, ""},
	{"GET", "/jiying/api/connections", "连接池快照（应用与账号，含昵称与头像）", "readonly", nil, ""},
	{"POST", "/jiying/api/connections/app", "新增或更新连接池应用", "ops", []string{"app_id", "ws_url", "app_secret"}, "app_secret 传 **** 表示沿用已存密钥"},
	{"POST", "/jiying/api/connections/app/delete", "从连接池删除应用", "ops", []string{"app_id"}, "破坏性操作，执行前确认"},
	{"POST", "/jiying/api/connections/app/default", "设置默认应用", "ops", []string{"app_id"}, ""},
	{"POST", "/jiying/api/connections/account/select", "切换当前账号（必要时自动登录）", "ops", []string{"email"}, ""},
	{"POST", "/jiying/api/connections/account/delete", "从连接池删除账号", "ops", []string{"email"}, "破坏性操作，执行前确认"},
	{"POST", "/jiying/api/connections/account/profile", "刷新当前账号资料（昵称/头像）", "readonly", nil, "幂等刷新，无业务写入"},
	{"GET", "/jiying/api/apps/profile", "刷新应用资料（名称/头像/描述）", "readonly", []string{"app_id"}, "省略 app_id 时取默认应用；会发起上游调用"},
	{"GET", "/jiying/api/connections/app/matrix", "应用矩阵：功能、角色、会话与定时隔离情况", "ops", nil, "角色明细仅 master 可见"},
	{"POST", "/jiying/api/app-roles", "设置应用级角色覆盖", "master", []string{"app_id", "user", "role"}, "role: readonly/ops/master；应用必须已接入"},
	{"POST", "/jiying/api/app-roles/delete", "移除应用级角色覆盖", "master", []string{"app_id", "user"}, ""},
	{"GET", "/jiying/api/roles", "角色列表（master 返回成员与主人列表，其他角色只返回自身）", "master", nil, "非 master 仅返回 me/me_id"},
	{"GET", "/jiying/api/dingtalk/status", "钉钉连接器状态与最近事件", "ops", nil, ""},
	{"GET", "/jiying/api/dingtalk/events", "钉钉事件摘要", "ops", []string{"limit"}, "最多 200 条"},
	{"POST", "/jiying/api/dingtalk/config", "保存钉钉连接器配置并启停", "master", []string{"enabled", "bin", "auto_push", "push_app_id", "push_conversation_id"}, "认证由本机 dws 持有"},
	{"POST", "/jiying/api/dingtalk/start", "启动钉钉事件订阅", "master", nil, ""},
	{"POST", "/jiying/api/dingtalk/stop", "停止钉钉事件订阅", "master", nil, ""},
	{"POST", "/jiying/api/dingtalk/probe", "检查 dws 登录状态", "master", nil, ""},
	{"POST", "/jiying/api/dingtalk/tool", "调用钉钉工具（排障用）", "master", []string{"name", "args"}, "args 为 JSON 对象"},
	{"POST", "/jiying/api/roles", "设置用户角色", "master", []string{"user", "role"}, "role: readonly/ops/master"},
	{"POST", "/jiying/api/roles/delete", "移除角色覆盖", "master", []string{"user"}, ""},
	{"POST", "/jiying/api/masters", "添加 owner 用户", "master", []string{"user"}, ""},
	{"POST", "/jiying/api/masters/delete", "移除 owner 用户", "master", []string{"user"}, ""},
	{"GET", "/jiying/api/audit", "审计日志", "readonly", []string{"limit"}, ""},
	{"GET", "/jiying/api/market", "插件市场列表", "readonly", nil, ""},
	{"GET", "/jiying/api/market/publishers", "受信任插件发布者公钥列表", "master", nil, "公钥用于安装时校验发布者签名"},
	{"POST", "/jiying/api/market/publishers", "登记受信任发布者公钥", "master", []string{"name", "public_key"}, "public_key 为 32 字节 Ed25519 公钥的 Base64；名称登记后不可换键"},
	{"POST", "/jiying/api/market/publishers/delete", "删除受信任发布者公钥", "master", []string{"name"}, "删除后其已上架插件将无法通过签名校验"},
	{"GET", "/jiying/api/market/export", "导出插件包内容", "ops", []string{"name"}, ""},
	{"POST", "/jiying/api/market/install", "从市场安装插件", "ops", []string{"name", "force", "confirm"}, "安装前校验内容摘要与发布者签名；force 只允许降级，不能绕过校验；脚本插件需 confirm=true"},
	{"POST", "/jiying/api/market/publish", "发布插件到市场", "ops", []string{"name", "version", "author", "description", "identity", "scope", "kind", "target", "content", "publisher", "signature", "force"}, "未指定 publisher 时用本机密钥自动签名；外部发布者需先登记公钥并提供签名"},
	{"POST", "/jiying/api/market/delete", "从市场移除插件", "ops", []string{"name"}, ""},
	{"GET", "/jiying/api/skills", "技能列表", "readonly", nil, ""},
	{"POST", "/jiying/api/skill/run", "运行技能", "ops", []string{"name", "args"}, ""},
	{"POST", "/jiying/api/skill/install", "安装技能", "ops", []string{"name", "code"}, ""},
	{"POST", "/jiying/api/skill/delete", "删除技能", "ops", []string{"name"}, ""},
	{"GET", "/jiying/api/agent/tools", "AI 工具能力列表", "ops", nil, ""},
	{"GET", "/jiying/api/agent/config", "读取 Agent 模型配置（含 app_systems，密钥仅返回是否已设置）", "master", nil, ""},
	{"POST", "/jiying/api/agent/config", "保存 Agent 模型配置", "master", []string{"enabled", "base_url", "model", "fallback_models", "system", "temperature", "api_key", "max_turns", "max_history", "max_tools", "max_context_kb"}, "api_key 传 **** 表示不修改；max_context_kb 限制 32-1024，max_tools 限制 1-32"},
	{"POST", "/jiying/api/agent/app-system", "设置或清除单个应用的系统提示", "master", []string{"app_id", "system"}, "app_id 必填；system 最多 20000 字符，留空删除覆盖并继承全局提示"},
	{"POST", "/jiying/api/agent/app-override", "设置应用级模型/轮次/温度", "master", []string{"app_id", "model", "max_turns", "temperature"}, "空值恢复继承全局配置"},
	{"POST", "/jiying/api/agent/run", "调用内置 AI 代理", "ops", []string{"prompt", "conversation_id", "reset"}, "需要先在控制台配置模型；按会话归属应用选择系统提示与工具范围；reset=true 清空上下文"},
	{"GET", "/jiying/api/icon", "生成并缓存图标（需 msg 关键词）", "readonly", []string{"msg", "n"}, "n 非空时返回 PNG"},
	{"GET", "/jiying/api/keys", "列出 API Key（仅浏览器会话）", "master", nil, "密钥不能管理自身"},
	{"POST", "/jiying/api/keys", "创建 API Key（仅浏览器会话）", "master", []string{"label", "scope", "expires_days"}, "创建后明文只返回一次"},
	{"POST", "/jiying/api/keys/rotate", "轮换 API Key（仅浏览器会话）", "master", []string{"id", "expires_days"}, "expires_days 留空则沿用旧密钥剩余有效期"},
	{"POST", "/jiying/api/keys/delete", "撤销 API Key（仅浏览器会话）", "master", []string{"id"}, ""},
	{"GET", "/jiying/api/ai/manifest", "AI 能力清单：功能、定时、端点、多应用约定与插件规范", "readonly", nil, ""},
}

func requestBaseURL(c *gin.Context) string {
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	if proto := strings.TrimSpace(c.GetHeader("X-Forwarded-Proto")); proto != "" {
		scheme = strings.Split(proto, ",")[0]
	}
	host := c.Request.Host
	if host == "" {
		host = "127.0.0.1:8080"
	}
	return scheme + "://" + host
}

// normalizeScope 把空值、未知角色与大小写差异统一成只读。
func normalizeScope(scope string) string {
	scope = strings.ToLower(strings.TrimSpace(scope))
	if _, ok := roleRank[scope]; !ok {
		return RoleReadonly
	}
	return scope
}

func scopeRank(scope string) int {
	return roleRank[normalizeScope(scope)]
}

// filterEndpointsByScope 按角色等级裁剪端点；调用方拿不到高于自身 scope 的条目。
func filterEndpointsByScope(items []apiDoc, scope string) []apiDoc {
	allowed := scopeRank(scope)
	out := make([]apiDoc, 0, len(items))
	for _, ep := range items {
		need := roleRank[ep.Scope]
		if need == 0 {
			need = roleRank[RoleReadonly]
		}
		if allowed >= need {
			out = append(out, ep)
		}
	}
	return out
}

// requestPublic 判断是否为匿名公开访问；带有效密钥或会话的请求返回 false。
func requestPublic(c *gin.Context) bool {
	if v, ok := c.Get(ctxAPIKeyScope); ok {
		if scope, ok := v.(string); ok && strings.TrimSpace(scope) != "" {
			return false
		}
	}
	return !CheckCookie(c)
}

// requestScope 解析文档请求方权限：API Key 优先，其次登录会话，最后只读。
func requestScope(c *gin.Context) string {
	if v, ok := c.Get(ctxAPIKeyScope); ok {
		if scope, ok := v.(string); ok && strings.TrimSpace(scope) != "" {
			return normalizeScope(scope)
		}
	}
	if CheckCookie(c) {
		return normalizeScope(UserRole(consoleUser(c)))
	}
	return RoleReadonly
}

// visibleEndpoints 先按权限裁剪；匿名访问只暴露可安全公开的 GET 端点。
func visibleEndpoints(items []apiDoc, scope string, public bool) []apiDoc {
	out := filterEndpointsByScope(items, scope)
	if !public {
		return out
	}
	safe := out[:0]
	for _, ep := range out {
		if ep.Method == http.MethodGet {
			safe = append(safe, ep)
		}
	}
	return safe
}

// optionalAPIKeyAuth 让公开文档也能识别 Bearer；无效密钥仍按 401 拒绝。
func optionalAPIKeyAuth(c *gin.Context) {
	if authenticateAPIKey(c) && c.IsAborted() {
		return
	}
	c.Next()
}

func registerAIDocs() {
	Server.GET("/jiying/ai.txt", optionalAPIKeyAuth, func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")
		c.Header("Vary", "Authorization, Cookie")
		c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(buildAIGuide(requestBaseURL(c), requestScope(c), requestPublic(c))))
	})
	Server.GET("/jiying/openapi.json", optionalAPIKeyAuth, func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")
		c.Header("Vary", "Authorization, Cookie")
		c.JSON(http.StatusOK, buildOpenAPI(requestBaseURL(c), requestScope(c), requestPublic(c)))
	})
	Server.GET("/jiying/api/ai/manifest", optionalAPIKeyAuth, func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")
		c.Header("Vary", "Authorization, Cookie")
		aiManifest(c)
	})
}

func aiManifest(c *gin.Context) {
	apps := AppPoolSnapshot()
	connected := 0
	for _, a := range apps {
		if a.Online {
			connected++
		}
	}
	scope := requestScope(c)
	public := requestPublic(c)
	plugin := gin.H{
		"extensions": []string{".js 在 JS 运行时执行，.py 在 Python 执行"},
		"manifest":   "文件头注释块，每行 Key: Value；支持 title/class/service/apps/author/version/description/rule/cron/platform/priority/admin/icon/config/network",
		"apps":       "[apps: app-id-1, app-id-2] 声明插件默认应用范围，* 或省略表示全部应用；控制台手动设置后以 apps_locked=true 的记录为准",
		"output":     "功能调用返回 stdout / 返回值；多段结果用 SEND: 前缀可直接回发会话",
		"security":   "插件以服务进程权限运行；[network: false] 的 Python 插件在无网络命名空间中执行，只安装可信来源",
	}
	if scopeRank(scope) >= roleRank[RoleOps] {
		plugin["install"] = "POST /jiying/api/script/install，name 决定类型，code 为完整源码；高风险代码需同时传 confirm=true"
	}
	features := ListFeatureItems()
	cronActor, cronRole := ConsoleActor(c)
	crons := CronCmdsSnapshotFor(cronActor, cronRole)
	if public {
		features = []*Feature{}
		crons = []*CronCmd{}
	}
	jiyingJSON(c, gin.H{
		"name":     "Relay 即应控制台",
		"base_url": requestBaseURL(c),
		"auth": gin.H{
			"header": "Authorization: Bearer <API Key>",
			"note":   "匿名访问只返回只读 GET 文档；在控制台「AI 接入」页创建密钥后按 scope 读取写操作与管理接口。",
		},
		"conventions": gin.H{
			"success":       `{"code":200,"data":...}，部分接口 data 为业务对象或字符串`,
			"error":         `{"code":1,"message":"原因"}`,
			"content_type":  "application/x-www-form-urlencoded；参数一律按字符串传递",
			"bool":          `使用 "true"/"false"`,
			"http_errors":   "401 未认证、403 权限不足、429 触发限流（默认 180 次/分钟/密钥）",
			"before_write":  "先读后写；删除、停用、退出、改角色属于破坏性操作，执行前向用户确认",
			"side_effects":  "发送消息、执行功能、运行插件都会产生真实外部效果",
			"multi_app":     "会话归属应用决定发送通道、功能可见性、工具契约与系统提示；发送类接口不接受 app_id。未记录归属时回退默认应用，已记录但应用离线时失败关闭。",
			"app_scope":     "功能 apps 为空或 [\"*\"] 表示全部应用；控制台手动保存后 apps_locked=true，重装/重扫保留手动范围；传 reset_apps=true 恢复插件清单声明。",
			"idempotency":   "发送类接口不支持幂等键，重试前先查 /messages",
			"no_simulation": "除 send/unified 的 dry_run=true 外，没有演练模式",
		},
		"runtime": gin.H{
			"role":           scope,
			"public":         public,
			"app_count":      len(apps),
			"app_connected":  connected,
			"account_logged": userLoggedIn(),
			"readiness":      connectionReadiness(),
			"uptime_seconds": JiyingUptimeSeconds(),
		},
		"features":  features,
		"crons":     crons,
		"endpoints": visibleEndpoints(aiEndpoints, scope, public),
		"cron": gin.H{
			"format":   "5 段（分 时 日 月 周）或 6 段（秒 分 时 日 月 周），支持 @daily 等描述符",
			"examples": []string{"*/30 * * * * * 每 30 秒", "0 */5 * * * * 每 5 分钟", "0 0 9 * * 1-5 工作日 9 点", "0 0 18 * * * 每天 18 点"},
			"command":  `{"type":"feature","feature":"状态","args":"","conversation_id":"<可选>"}`,
			"chat_id":  "可选，指定结果回发的会话 ID",
		},
		"plugin": plugin,
		"links": gin.H{
			"guide":   requestBaseURL(c) + "/jiying/ai.txt",
			"openapi": requestBaseURL(c) + "/jiying/openapi.json",
		},
	})
}

var aiSessionOnly = map[string]bool{
	"/jiying/api/keys":        true,
	"/jiying/api/keys/rotate": true,
	"/jiying/api/keys/delete": true,
}

// fieldExample 为常见参数给出可直接替换的示例值，避免文档里只有类型没有形状。
func fieldExample(name string) string {
	switch name {
	case "id", "file_id", "message_id", "reply_to":
		return "abc123"
	case "conversation_id":
		return "conv_abc123"
	case "app_id":
		return "app_abc123"
	case "app_secret", "api_key":
		return "****"
	case "apps":
		return `["app-id-1","app-id-2"]`
	case "reset_apps", "confirm", "dry_run":
		return "false"
	case "enabled":
		return "true"
	case "identity":
		return "auto"
	case "url":
		return "https://example.com/file.png"
	case "name", "label":
		return "巡检机器人"
	case "title":
		return "运行结果"
	case "content", "text", "code", "system", "prompt":
		return "示例内容"
	case "data_base64", "content_base64":
		return "aGVsbG8="
	case "content_type":
		return "text/plain"
	case "scope":
		return "ops"
	case "expires_days":
		return "30"
	case "data":
		return `{"labels":["一月","二月"],"values":[12,18]}`
	case "values":
		return `{"token":"****"}`
	case "fallback_models":
		return "model-a,model-b"
	case "max_turns":
		return "8"
	case "max_history":
		return "20"
	case "msg":
		return "状态"
	case "tail", "day":
		return "200"
	case "cron":
		return "0 */30 * * * *"
	case "args":
		return ""
	default:
		return "<" + name + ">"
	}
}

// openAPIResponse 返回统一的成功响应结构，业务错误仍以 HTTP 200 + code=1 返回。
func openAPIResponse() gin.H {
	return gin.H{
		"description": "统一响应；code=200 表示成功，code=1 表示业务失败",
		"content": gin.H{
			"application/json": gin.H{
				"schema": gin.H{
					"type":     "object",
					"required": []string{"code"},
					"properties": gin.H{
						"ok":       gin.H{"type": "boolean"},
						"code":     gin.H{"type": "integer", "example": 200},
						"message":  gin.H{"type": "string"},
						"trace_id": gin.H{"type": "string"},
						"data":     gin.H{},
					},
				},
				"example": gin.H{"ok": true, "code": 200, "message": "", "data": gin.H{}},
			},
		},
	}
}

func buildOpenAPI(base, scope string, public bool) gin.H {
	scope = normalizeScope(scope)
	paths := gin.H{}
	for _, ep := range visibleEndpoints(aiEndpoints, scope, public) {
		path := ep.Path
		security := []gin.H{{"bearerAuth": []string{}}}
		if aiSessionOnly[path] {
			security = []gin.H{{"cookieAuth": []string{}}}
		}
		op := gin.H{
			"summary":         ep.Summary,
			"tags":            []string{strings.Split(strings.TrimPrefix(path, "/jiying/api/"), "/")[0]},
			"security":        security,
			"x-required-role": ep.Scope,
			"responses": gin.H{
				"200": openAPIResponse(),
				"401": gin.H{"description": "未登录、密钥无效或已过期"},
				"403": gin.H{"description": "角色不足，或密钥无权访问该接口"},
				"429": gin.H{"description": "API Key 请求超过每分钟 180 次"},
			},
		}
		if ep.Notes != "" {
			op["description"] = ep.Notes + "。所需角色：" + ep.Scope
		} else {
			op["description"] = "所需角色：" + ep.Scope
		}
		if aiSessionOnly[path] {
			op["description"] = op["description"].(string) + "。仅限浏览器会话，不支持 API Key。"
		}
		if ep.Method == "GET" {
			if len(ep.Fields) > 0 {
				params := make([]gin.H, 0, len(ep.Fields))
				for _, f := range ep.Fields {
					params = append(params, gin.H{"name": f, "in": "query", "schema": gin.H{"type": "string"}})
				}
				op["parameters"] = params
			}
		} else {
			props := gin.H{}
			example := gin.H{}
			for _, f := range ep.Fields {
				props[f] = gin.H{"type": "string"}
				example[f] = fieldExample(f)
			}
			op["requestBody"] = gin.H{
				"required": true,
				"content": gin.H{
					"application/x-www-form-urlencoded": gin.H{
						"schema":  gin.H{"type": "object", "properties": props},
						"example": example,
					},
				},
			}
		}
		if _, ok := paths[path]; !ok {
			paths[path] = gin.H{}
		}
		paths[path].(gin.H)[strings.ToLower(ep.Method)] = op
	}
	infoDescription := "控制台 REST 接口。统一响应 {code:200,data}；认证使用 Authorization: Bearer <API Key>。文档按请求方权限裁剪。" + base
	if scope == RoleReadonly {
		if public {
			infoDescription = "只读公开版：仅包含只读 GET 端点。写操作与管理能力需在控制台「AI 接入」创建密钥后读取。统一响应 {code:200,data}；认证使用 Authorization: Bearer <API Key>。" + base
		} else {
			infoDescription = "只读版：仅包含 readonly 端点。写操作需使用 ops 及以上密钥。统一响应 {code:200,data}；认证使用 Authorization: Bearer <API Key>。" + base
		}
	}
	return gin.H{
		"openapi": "3.0.3",
		"info": gin.H{
			"title":       "Relay 即应控制台 API",
			"version":     "1.0.0",
			"description": infoDescription,
		},
		"servers": []gin.H{{"url": base}},
		"components": gin.H{
			"securitySchemes": gin.H{
				"bearerAuth": gin.H{"type": "http", "scheme": "bearer", "bearerFormat": "jy_xxx"},
				"cookieAuth": gin.H{"type": "apiKey", "in": "cookie", "name": "jiying_session"},
			},
		},
		"paths": paths,
	}
}

func buildAIGuide(base, scope string, public bool) string {
	scope = normalizeScope(scope)
	ops := scopeRank(scope) >= roleRank[RoleOps]
	master := scope == RoleMaster
	var b strings.Builder
	b.WriteString("Relay 即应控制台 API 接入说明\n")
	b.WriteString("Base URL: " + base + "\n")
	b.WriteString("OpenAPI: " + base + "/jiying/openapi.json\n\n")
	b.WriteString(`一、认证
在控制台「AI 接入」页创建 API Key，然后每个请求带上：
  Authorization: Bearer jy_xxxxxxxx
密钥权限分为 readonly / ops / master，分别对应只读、日常操作与角色管理。
密钥只在创建时显示一次；管理密钥本身仍要求浏览器登录。
`)
	if public {
		b.WriteString("当前为只读公开文档，只包含只读 GET 端点。写操作、管理能力与完整清单需要 API Key。\n\n")
	} else if !ops {
		b.WriteString("当前为只读文档，只包含 readonly 端点。写操作需要 ops 及以上密钥。\n\n")
	} else if master {
		b.WriteString("当前为 master 文档，包含只读、日常操作与管理接口。\n\n")
	} else {
		b.WriteString("当前为 ops 文档，包含只读与日常操作接口。\n\n")
	}
	b.WriteString(`二、响应约定
- 成功：{"code":200,"data":...}
- 业务失败：{"code":1,"message":"原因"}
- HTTP 失败：401 未认证、403 权限不足、429 请求过于频繁（每把钥匙 180 次/分钟）
`)
	if !public {
		b.WriteString(`- 写接口使用 application/x-www-form-urlencoded，所有参数按字符串传递
- 先读后写；删除、停用、退出、改角色前必须先向用户确认
- 发送消息、执行功能、运行插件都会产生真实外部效果
`)
	}
	b.WriteString(`
三、读懂系统
1. GET /jiying/api/ai/manifest：当前权限、就绪度、多应用约定、端点清单与插件规范
2. GET /jiying/api/overview：连接、账号、应用、任务与运行状态
3. GET /jiying/api/features：功能名称、启用状态、应用范围与应用锁定状态
4. GET /jiying/api/scripts：插件清单、分类、服务分组、应用白名单、说明、规则与 config_schema
5. GET /jiying/api/connect/status：应用是否连接、账号是否登录
6. GET /jiying/api/conversations：账号与在线应用会话合并列表，含 relay_app_id/name/online/known
7. GET /jiying/api/conversation/capabilities：指定会话能否以应用或账号身份发送
8. GET /jiying/api/tasks：任务队列；GET /jiying/api/executions：执行账本

四、多应用与会话归属
- 发送、功能执行、Agent 工具与系统提示都按会话归属应用选择通道、权限与提示
- 发送类接口只传 conversation_id，不接收 app_id；未记录归属时回退默认应用，已记录但应用离线时拒绝发送
- 账号未登录或账号看不到会话时，消息接口按归属应用回退，响应 data.source=app
- 功能 apps 为空或 ["*"] 表示全部应用；手工保存后 apps_locked=true，插件重装/重扫保留；传 reset_apps=true 恢复插件清单
- Agent 的 app_systems 按应用覆盖系统提示；system 留空即恢复全局提示

五、常用流程
- 查看连接：GET /jiying/api/connect/status
- 查看会话：GET /jiying/api/conversations，再用 /conversation/capabilities 判断发送身份
- 读取消息：GET /jiying/api/messages?conversation_id=<id>
- 查看任务：GET /jiying/api/tasks
- 查看执行结果：GET /jiying/api/executions?limit=50
`)
	if ops {
		b.WriteString(`- 执行功能：POST /jiying/api/feature/run  name,args
- 上传文件：POST /jiying/api/upload  name,content_type,data_base64（或 multipart file），最大 20 MB
- 发送文件：POST /jiying/api/send/file  conversation_id,file_id,name,size
- 预演发送：POST /jiying/api/send/unified  conversation_id,text,dry_run=true
- 正式发送：同上去掉 dry_run；身份不确定时先 dry_run
- 调整范围：POST /jiying/api/feature/update  name,apps（JSON 数组）或 reset_apps=true
- 排障：GET /jiying/api/logs?tail=200&day=YYYYMMDD
`)
	}
	if master {
		b.WriteString(`- Agent 配置：GET/POST /jiying/api/agent/config
- API Key 管理（仅浏览器会话）：/jiying/api/keys、/keys/rotate、/keys/delete
`)
	}
	b.WriteString(`
六、插件
- 支持 .js（JS 运行时）与 .py（Python）
- 文件头注释块声明元数据，每行 Key: Value，常用：Title、Class、Service、Apps、Description、Author、Version、Rule、Cron、Priority、Admin、Icon、Config、Network
- Apps 声明默认应用范围，例如 [apps: app-id-1, app-id-2]；* 或省略表示全部应用
- 控制台手动设置应用范围后写入 apps_locked=true，重装或重扫保留；需要回到清单声明时传 reset_apps=true
- Config 声明可配置项，控制台会生成表单；密钥字段不会回显
- Network 默认允许；[network: false] 的 Python 插件在无网络命名空间中执行，看不到外部地址
`)
	if ops {
		b.WriteString(`- 安装：POST /jiying/api/script/install  name,code；高风险代码需同时传 confirm=true
- 调试：POST /jiying/api/script/run  name,args；配置：/script/config、/script/config/save
`)
	}
	b.WriteString(`- 插件在独立工作目录执行，只读插件目录通过 JIYING_PLUGIN_DIR 暴露；只安装可信来源

七、定时任务
- 查看列表：GET /jiying/api/crons
`)
	if ops {
		b.WriteString(`表达式支持 5 段（分 时 日 月 周）或 6 段（秒 分 时 日 月 周），也支持 @daily 等描述符。
示例：
  */30 * * * * *   每 30 秒
  0 */5 * * * *    每 5 分钟
  0 0 9 * * 1-5    工作日 9 点
command 推荐写成功能 JSON：
  {"type":"feature","feature":"状态","args":"","conversation_id":"<可选>"}
chat_id 可选，用于把结果发回指定会话。
新建后默认启用；调整用 /cron/update，停用、删除、立即执行分别用 /cron/toggle、/cron/delete、/cron/run。
非主控身份只能查看和操作自己创建的任务。
`)
	}
	b.WriteString(`
八、安全边界
- 不要尝试读取或转储密钥、密码、Cookie
- 不要绕过权限；403 表示当前 scope 不足
`)
	if !public {
		b.WriteString("- 不要在未确认时批量发送、删除或停用\n- 日志与审计只读，修改类操作都会记录操作者\n")
	}
	return b.String()
}
