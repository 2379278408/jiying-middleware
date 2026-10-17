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
	{"GET", "/jiying/api/overview", "总览：连接、账号、功能、定时、任务、插件与就绪状态", "readonly", nil, ""},
	{"GET", "/jiying/api/features", "功能列表（名称、类型、目标、权限、启用状态）", "readonly", nil, ""},
	{"POST", "/jiying/api/feature/toggle", "启用或停用功能", "ops", []string{"name", "enabled"}, `enabled 取 "true"/"false"`},
	{"POST", "/jiying/api/feature/install", "安装 HTTP 功能（转发到外部地址）", "ops", []string{"name", "url"}, ""},
	{"POST", "/jiying/api/feature/delete", "删除功能；脚本型功能会同步删除插件文件", "ops", []string{"name"}, "破坏性操作，执行前确认"},
	{"POST", "/jiying/api/feature/run", "执行功能并返回输出", "ops", []string{"name", "args"}, "args 为字符串，按功能自行解析"},
	{"GET", "/jiying/api/scripts", "插件列表与清单（title/description/rules/config_schema）", "readonly", nil, ""},
	{"POST", "/jiying/api/script/get", "读取插件源码与语言", "ops", []string{"name"}, ""},
	{"POST", "/jiying/api/script/install", "安装或覆盖插件源码", "ops", []string{"name", "code"}, "name 扩展名 .js/.py 决定运行方式"},
	{"POST", "/jiying/api/script/delete", "删除插件文件与配置", "ops", []string{"name"}, "破坏性操作，执行前确认"},
	{"POST", "/jiying/api/script/run", "直接运行插件", "ops", []string{"name", "args"}, ""},
	{"POST", "/jiying/api/script/config", "读取插件配置 schema 与当前值", "ops", []string{"name"}, "secret 字段只返回是否已设置"},
	{"POST", "/jiying/api/script/config/save", "保存插件配置", "ops", []string{"name", "values"}, "values 为 JSON 对象字符串"},
	{"GET", "/jiying/api/crons", "定时任务列表（含下次运行时间与最近结果）", "readonly", nil, ""},
	{"POST", "/jiying/api/cron/add", "新建定时任务", "ops", []string{"name", "cron", "command", "chat_id"}, "command 支持功能 JSON 或系统命令"},
	{"POST", "/jiying/api/cron/toggle", "启用或停用定时任务", "ops", []string{"id", "enabled"}, ""},
	{"POST", "/jiying/api/cron/delete", "删除定时任务", "ops", []string{"id"}, "破坏性操作，执行前确认"},
	{"POST", "/jiying/api/cron/run", "立即执行一次定时任务", "ops", []string{"id"}, ""},
	{"GET", "/jiying/api/executions", "执行账本", "readonly", []string{"kind", "name", "status", "limit"}, "kind: cron/feature/task/message"},
	{"GET", "/jiying/api/tasks", "任务队列", "readonly", []string{"status", "feature", "user_id", "limit"}, ""},
	{"POST", "/jiying/api/task/retry", "重试任务", "ops", []string{"id"}, ""},
	{"POST", "/jiying/api/task/delete", "删除任务", "ops", []string{"id"}, ""},
	{"GET", "/jiying/api/conversations", "会话列表（来自即应账号）", "readonly", nil, "需要账号已登录"},
	{"GET", "/jiying/api/messages", "会话消息", "readonly", []string{"conversation_id"}, ""},
	{"GET", "/jiying/api/conversation/capabilities", "会话能力（可发送的消息类型）", "readonly", []string{"conversation_id"}, ""},
	{"POST", "/jiying/api/send", "发送纯文本（旧接口）", "ops", []string{"conversation_id", "text"}, "建议改用 /send/unified"},
	{"POST", "/jiying/api/send/unified", "统一发送：自动选择身份与消息类型", "ops", []string{"conversation_id", "text", "identity", "reply_to", "dry_run"}, "dry_run=true 只预演"},
	{"POST", "/jiying/api/send/image", "发送图片", "ops", []string{"conversation_id", "url", "file_id", "caption", "caption_type", "reply_to"}, "url 与 file_id 二选一"},
	{"POST", "/jiying/api/send/file", "发送文件", "ops", []string{"conversation_id", "url", "file_id", "name", "size", "reply_to"}, "url 与 file_id 二选一；size 为字节数"},
	{"POST", "/jiying/api/send/card", "发送卡片", "ops", []string{"conversation_id", "title", "description", "url", "reply_to"}, ""},
	{"POST", "/jiying/api/send/markdown", "发送 Markdown", "ops", []string{"conversation_id", "content", "text", "reply_to"}, "content 与 text 等价"},
	{"POST", "/jiying/api/send/link", "发送链接卡片", "ops", []string{"conversation_id", "url", "title", "reply_to"}, ""},
	{"POST", "/jiying/api/send/chart", "发送图表", "ops", []string{"conversation_id", "chart_type", "title", "description", "data", "reply_to"}, "data 为 JSON 字符串"},
	{"GET", "/jiying/api/logs", "读取运行日志", "readonly", []string{"tail", "day"}, "tail 最大 2000；day 形如 20261017"},
	{"GET", "/jiying/api/connect/status", "接入状态：应用、账号、运行时就绪度", "readonly", nil, ""},
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
	{"GET", "/jiying/api/connections", "连接池快照（应用与账号）", "readonly", nil, ""},
	{"POST", "/jiying/api/connections/app", "新增或更新连接池应用", "ops", []string{"app_id", "ws_url", "app_secret"}, "app_secret 传 **** 表示沿用已存密钥"},
	{"POST", "/jiying/api/connections/app/delete", "从连接池删除应用", "ops", []string{"app_id"}, "破坏性操作，执行前确认"},
	{"POST", "/jiying/api/connections/app/default", "设置默认应用", "ops", []string{"app_id"}, ""},
	{"POST", "/jiying/api/connections/account/select", "切换当前账号（必要时自动登录）", "ops", []string{"email"}, ""},
	{"POST", "/jiying/api/connections/account/delete", "从连接池删除账号", "ops", []string{"email"}, "破坏性操作，执行前确认"},
	{"POST", "/jiying/api/connections/account/profile", "刷新当前账号资料（昵称/头像）", "readonly", nil, ""},
	{"GET", "/jiying/api/apps/profile", "刷新应用资料（名称/头像/描述）", "readonly", []string{"app_id"}, "省略 app_id 时取默认应用；会发起上游调用"},
	{"GET", "/jiying/api/roles", "角色列表", "readonly", nil, ""},
	{"POST", "/jiying/api/roles", "设置用户角色", "master", []string{"user", "role"}, "role: readonly/ops/master"},
	{"POST", "/jiying/api/roles/delete", "移除角色覆盖", "master", []string{"user"}, ""},
	{"POST", "/jiying/api/masters", "添加 owner 用户", "master", []string{"user"}, ""},
	{"POST", "/jiying/api/masters/delete", "移除 owner 用户", "master", []string{"user"}, ""},
	{"GET", "/jiying/api/audit", "审计日志", "readonly", []string{"limit"}, ""},
	{"GET", "/jiying/api/market", "插件市场列表", "readonly", nil, ""},
	{"GET", "/jiying/api/market/export", "导出插件包内容", "ops", []string{"name"}, ""},
	{"POST", "/jiying/api/market/install", "从市场安装插件", "ops", []string{"name"}, ""},
	{"POST", "/jiying/api/market/publish", "发布插件到市场", "ops", []string{"name", "version", "author", "description", "identity", "scope", "kind", "target", "content", "force"}, ""},
	{"POST", "/jiying/api/market/delete", "从市场移除插件", "ops", []string{"name"}, ""},
	{"GET", "/jiying/api/skills", "技能列表", "readonly", nil, ""},
	{"POST", "/jiying/api/skill/run", "运行技能", "ops", []string{"name", "args"}, ""},
	{"POST", "/jiying/api/skill/install", "安装技能", "ops", []string{"name", "code"}, ""},
	{"POST", "/jiying/api/skill/delete", "删除技能", "ops", []string{"name"}, ""},
	{"GET", "/jiying/api/agent/tools", "AI 工具能力列表", "ops", nil, ""},
	{"GET", "/jiying/api/agent/config", "读取 Agent 模型配置（密钥仅返回是否已设置）", "master", nil, ""},
	{"POST", "/jiying/api/agent/config", "保存 Agent 模型配置", "master", []string{"enabled", "base_url", "model", "fallback_models", "system", "api_key", "max_turns", "max_history"}, "api_key 传 **** 表示不修改"},
	{"POST", "/jiying/api/agent/run", "调用内置 AI 代理", "ops", []string{"prompt", "conversation_id", "reset"}, "需要先在控制台配置模型；reset=true 清空上下文"},
	{"GET", "/jiying/api/icon", "生成并缓存图标（需 msg 关键词）", "readonly", []string{"msg", "n"}, "n 非空时返回 PNG"},
	{"GET", "/jiying/api/keys", "列出 API Key（仅浏览器会话）", "master", nil, "密钥不能管理自身"},
	{"POST", "/jiying/api/keys", "创建 API Key（仅浏览器会话）", "master", []string{"label", "scope", "expires_days"}, "创建后明文只返回一次"},
	{"POST", "/jiying/api/keys/delete", "撤销 API Key（仅浏览器会话）", "master", []string{"id"}, ""},
	{"GET", "/jiying/api/ai/manifest", "AI 能力清单：功能、定时、端点、约定", "readonly", nil, ""},
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

func registerAIDocs() {
	Server.GET("/jiying/ai.txt", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(buildAIGuide(requestBaseURL(c))))
	})
	Server.GET("/jiying/openapi.json", func(c *gin.Context) {
		c.JSON(http.StatusOK, buildOpenAPI(requestBaseURL(c)))
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
	role := UserRole(consoleUser(c))
	if v, ok := c.Get(ctxAPIKeyScope); ok {
		if scope, ok := v.(string); ok && scope != "" {
			role = scope
		}
	}
	jiyingJSON(c, gin.H{
		"name":     "Relay 即应控制台",
		"base_url": requestBaseURL(c),
		"auth": gin.H{
			"header": "Authorization: Bearer <API Key>",
			"note":   "在控制台「AI 接入」页创建密钥；密钥按 scope 映射角色，管理密钥本身仍需浏览器登录。",
		},
		"conventions": gin.H{
			"success":       `{"code":200,"data":...}，部分接口 data 为业务对象或字符串`,
			"error":         `{"code":1,"message":"原因"}`,
			"content_type":  "application/x-www-form-urlencoded；参数一律按字符串传递",
			"bool":          `使用 "true"/"false"`,
			"http_errors":   "401 未认证、403 权限不足、429 触发限流（默认 180 次/分钟/密钥）",
			"before_write":  "先读后写；删除、停用、退出、改角色属于破坏性操作，执行前向用户确认",
			"side_effects":  "发送消息、执行功能、运行插件都会产生真实外部效果",
			"idempotency":   "发送类接口不支持幂等键，重试前先查 /messages",
			"no_simulation": "除 send/unified 的 dry_run=true 外，没有演练模式",
		},
		"runtime": gin.H{
			"role":           role,
			"app_count":      len(apps),
			"app_connected":  connected,
			"account_logged": userLoggedIn(),
			"readiness":      connectionReadiness(),
			"uptime_seconds": JiyingUptimeSeconds(),
		},
		"features":  ListFeatureItems(),
		"crons":     CronCmdsSnapshot(),
		"endpoints": aiEndpoints,
		"cron": gin.H{
			"format":   "5 段（分 时 日 月 周）或 6 段（秒 分 时 日 月 周），支持 @daily 等描述符",
			"examples": []string{"*/30 * * * * * 每 30 秒", "0 */5 * * * * 每 5 分钟", "0 0 9 * * 1-5 工作日 9 点", "0 0 18 * * * 每天 18 点"},
			"command":  `{"type":"feature","feature":"状态","args":"","conversation_id":"<可选>"}；也可直接写系统命令字符串`,
			"chat_id":  "可选，指定结果回发的会话 ID",
		},
		"plugin": gin.H{
			"extensions": []string{".js 在 JS 运行时执行，.py 在 Python 执行"},
			"manifest":   "文件头注释块，每行 Key: Value；支持 title/author/version/description/rule/cron/platform/priority/admin/icon/config",
			"output":     "功能调用返回 stdout / 返回值；多段结果用 SEND: 前缀可直接回发会话",
			"install":    "POST /jiying/api/script/install，name 决定类型，code 为完整源码",
			"security":   "插件以服务进程权限运行，只安装可信来源",
		},
		"links": gin.H{
			"guide":   requestBaseURL(c) + "/jiying/ai.txt",
			"openapi": requestBaseURL(c) + "/jiying/openapi.json",
		},
	})
}

var aiSessionOnly = map[string]bool{
	"/jiying/api/keys":        true,
	"/jiying/api/keys/delete": true,
}

func buildOpenAPI(base string) gin.H {
	paths := gin.H{}
	for _, ep := range aiEndpoints {
		path := ep.Path
		security := []gin.H{{"bearerAuth": []string{}}}
		if aiSessionOnly[path] {
			security = []gin.H{{"cookieAuth": []string{}}}
		}
		op := gin.H{
			"summary":  ep.Summary,
			"tags":     []string{strings.Split(strings.TrimPrefix(path, "/jiying/api/"), "/")[0]},
			"security": security,
			"responses": gin.H{
				"200": gin.H{"description": "统一响应 {code:200,data}"},
				"401": gin.H{"description": "未认证"},
				"403": gin.H{"description": "权限不足"},
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
			for _, f := range ep.Fields {
				props[f] = gin.H{"type": "string"}
			}
			op["requestBody"] = gin.H{
				"content": gin.H{"application/x-www-form-urlencoded": gin.H{"schema": gin.H{"type": "object", "properties": props}}},
			}
		}
		if _, ok := paths[path]; !ok {
			paths[path] = gin.H{}
		}
		paths[path].(gin.H)[strings.ToLower(ep.Method)] = op
	}
	return gin.H{
		"openapi": "3.0.3",
		"info": gin.H{
			"title":       "Relay 即应控制台 API",
			"version":     "1.0.0",
			"description": "控制台 REST 接口。统一响应 {code:200,data}；认证使用 Authorization: Bearer <API Key>。" + base,
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

func buildAIGuide(base string) string {
	var b strings.Builder
	b.WriteString("Relay 即应控制台 API 接入说明\n")
	b.WriteString("Base URL: " + base + "\n")
	b.WriteString("OpenAPI: " + base + "/jiying/openapi.json\n\n")
	b.WriteString(`一、认证
在控制台「AI 接入」页创建 API Key，然后每个请求带上：
  Authorization: Bearer jy_xxxxxxxx
密钥权限分为 readonly / ops / master，分别对应只读、日常操作与角色管理。
密钥只在创建时显示一次；管理密钥本身仍要求浏览器登录。

二、响应约定
- 成功：{"code":200,"data":...}
- 业务失败：{"code":1,"message":"原因"}
- HTTP 失败：401 未认证、403 权限不足、429 请求过于频繁（每把钥匙 180 次/分钟）
- 写接口使用 application/x-www-form-urlencoded，所有参数按字符串传递
- 先读后写；删除、停用、退出、改角色前必须先向用户确认
- 发送消息、执行功能、运行插件都会产生真实外部效果

三、读懂系统
1. GET /jiying/api/ai/manifest：当前功能、定时任务、权限、就绪度、端点清单与插件规范
2. GET /jiying/api/overview：连接、账号、任务与运行状态
3. GET /jiying/api/features：功能名称与启用状态；执行用 POST /jiying/api/feature/run
4. GET /jiying/api/scripts：插件清单、说明、规则与 config_schema
5. GET /jiying/api/connect/status：应用是否连接、账号是否登录

四、常用流程
- 查看连接：GET /jiying/api/connect/status
- 查看会话：GET /jiying/api/conversations
- 读取消息：GET /jiying/api/messages?conversation_id=<id>
- 预演发送：POST /jiying/api/send/unified  conversation_id,text,dry_run=true
- 正式发送：同上去掉 dry_run；身份不确定时先 dry_run
- 执行功能：POST /jiying/api/feature/run  name,args
- 新建定时：POST /jiying/api/cron/add  name,cron,command
- 查看执行结果：GET /jiying/api/executions?limit=50
- 排障：GET /jiying/api/logs?tail=200&day=YYYYMMDD

五、定时任务
表达式支持 5 段（分 时 日 月 周）或 6 段（秒 分 时 日 月 周），也支持 @daily 等描述符。
示例：
  */30 * * * * *   每 30 秒
  0 */5 * * * *    每 5 分钟
  0 0 9 * * 1-5    工作日 9 点
command 推荐写成功能 JSON：
  {"type":"feature","feature":"状态","args":"","conversation_id":"<可选>"}
chat_id 可选，用于把结果发回指定会话。
新建后默认启用；停用、删除、立即执行分别用 /cron/toggle、/cron/delete、/cron/run。

六、插件
- 支持 .js（JS 运行时）与 .py（Python）
- 文件头注释块声明元数据，每行 Key: Value，常用：Title、Description、Author、Version、Rule、Cron、Priority、Admin、Icon、Config
- Config 声明可配置项，控制台会生成表单；密钥字段不会回显
- 安装：POST /jiying/api/script/install  name,code（code 为完整源码）
- 调试：POST /jiying/api/script/run  name,args；配置：/script/config、/script/config/save
- 插件以服务进程权限运行，只安装可信来源

七、安全边界
- 不要尝试读取或转储密钥、密码、Cookie
- 不要绕过权限；403 表示当前 scope 不足
- 不要在未确认时批量发送、删除或停用
- 日志与审计只读，修改类操作都会记录操作者
`)
	return b.String()
}
