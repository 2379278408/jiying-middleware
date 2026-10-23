/* Relay 控制台 */
(function () {
  "use strict";

  // ---------- 国际化 ----------
  // 默认中文；语言选择存本机 localStorage，切换后刷新外壳文案（导航、顶栏、侧栏、登录页）。
  // 页面正文暂保持中文，新增文案统一走 t()/data-i18n 以便逐步扩展。
  var LANG_KEY = "relay-lang";
  var LANG = "zh";
  var DICT = {
    "brand.sub": { zh: "即应接入与调度", en: "Jiying relay & scheduling" },
    "nav.group.overview": { zh: "概览", en: "Overview" },
    "nav.group.access": { zh: "即应接入", en: "Connect" },
    "nav.group.run": { zh: "运行", en: "Runtime" },
    "nav.pinned": { zh: "置顶", en: "Pinned" },
    "nav.home": { zh: "工作台", en: "Dashboard" },
    "nav.chat": { zh: "消息收发", en: "Messages" },
    "chat.app": { zh: "应用", en: "App" },
    "chat.user": { zh: "用户", en: "User" },
    "chat.allApps": { zh: "全部应用", en: "All apps" },
    "chat.unassigned": { zh: "待识别", en: "Unidentified" },
    "chat.account": { zh: "账号会话", en: "Account session" },
    "chat.appMissing": { zh: "应用未接入", en: "App not connected" },
    "chat.filter.title": { zh: "按应用筛选会话", en: "Filter conversations by app" },
    "chat.noAssignment": { zh: "会话未记录应用归属", en: "No app assignment" },
    "chat.appOffline": { zh: "应用通道离线", en: "app channel offline" },
    "chat.group": { zh: "群", en: "Group" },
    "chat.dm": { zh: "私聊", en: "Direct" },
    "chat.noIdentity": { zh: "当前会话没有可用发送身份", en: "No send identity available" },
    "chat.canSend": { zh: "可发送身份：", en: "Send as: " },
    "chat.filter.empty": { zh: "该筛选下暂无会话", en: "No conversations match this filter" },
    "nav.connect": { zh: "接入系统", en: "Connect" },
    "nav.pool": { zh: "连接池", en: "Connection pool" },
    "nav.dingtalk": { zh: "钉钉连接器", en: "DingTalk connector" },
    "nav.roles": { zh: "权限管理", en: "Access control" },
    "nav.ai": { zh: "AI 接入", en: "AI access" },
    "nav.scripts": { zh: "插件", en: "Plugins" },
    "nav.market": { zh: "插件市场", en: "Plugin market" },
    "nav.features": { zh: "功能管理", en: "Features" },
    "nav.agent": { zh: "Agent 工作台", en: "Agent" },
    "nav.cron": { zh: "定时任务", en: "Schedules" },
    "nav.executions": { zh: "执行记录", en: "Runs" },
    "nav.tasks": { zh: "任务队列", en: "Task queue" },
    "nav.audit": { zh: "审计日志", en: "Audit log" },
    "nav.observe": { zh: "运行观测", en: "Observability" },
    "nav.logs": { zh: "运行日志", en: "Runtime logs" },
    "nav.filter.ph": { zh: "筛选菜单", en: "Filter menu" },
    "nav.custom.title": { zh: "自定义侧栏", en: "Customize sidebar" },
    "side.ws": { zh: "应用通道", en: "App channel" },
    "side.acct": { zh: "账号通道", en: "Account channel" },
    "side.connected": { zh: "已连接", en: "connected" },
    "side.disconnected": { zh: "未连接", en: "disconnected" },
    "side.notlogged": { zh: "未登录", en: "not signed in" },
    "chip.app.on": { zh: "应用已连接", en: "App connected" },
    "chip.app.off": { zh: "应用未连接", en: "App disconnected" },
    "chip.acct.off": { zh: "账号未登录", en: "Signed out" },
    "theme.toggle": { zh: "切换深浅色", en: "Toggle theme" },
    "lang.toggle": { zh: "切换语言", en: "Switch language" },
    "logout.title": { zh: "退出登录", en: "Sign out" },
    "nav.toggle.title": { zh: "收起菜单", en: "Toggle menu" },
    "nav.close.title": { zh: "关闭菜单", en: "Close menu" },
    "nav.custom.hint": { zh: "调整顺序、置顶或隐藏侧栏菜单，设置保存在本机浏览器；新增菜单会自动加入。", en: "Reorder, pin or hide sidebar items. Changes are stored in this browser; new items are added automatically." },
    "nav.custom.reset": { zh: "恢复默认", en: "Reset" },
    "nav.custom.done": { zh: "完成", en: "Done" },
    "nav.custom.pin": { zh: "置顶", en: "Pin" },
    "nav.custom.unpin": { zh: "取消置顶", en: "Unpin" },
    "nav.custom.show": { zh: "显示", en: "Show" },
    "nav.custom.hide": { zh: "隐藏", en: "Hide" },
    "login.brand.sub": { zh: "即应接入与任务调度控制台", en: "Jiying relay & scheduling console" },
    "login.tagline": { zh: "接入即应，编排一切", en: "Connect Jiying, orchestrate everything" },
    "login.desc": { zh: "Relay 面向即应应用的消息接入与任务调度中间件。通过应用密钥或账号授权一键接入，在聊天中完成功能调用、脚本执行与定时编排。", en: "Relay is the message-connectivity and task-scheduling middleware for Jiying apps. Connect with an app secret or account authorization, then invoke features, run scripts and schedule jobs right from chat." },
    "login.f1": { zh: "应用密钥接入", en: "App secret" },
    "login.f2": { zh: "账号 / 邮箱验证码", en: "Account / email code" },
    "login.f3": { zh: "功能目录", en: "Feature catalog" },
    "login.f4": { zh: "任务队列", en: "Task queue" },
    "login.f5": { zh: "定时调度", en: "Scheduling" },
    "login.heading": { zh: "登录控制台", en: "Sign in" },
    "login.sub": { zh: "登录后接入即应应用或账号通道", en: "Sign in to connect an app or account channel" },
    "login.username": { zh: "用户名", en: "Username" },
    "login.username.ph": { zh: "请输入用户名", en: "Enter username" },
    "login.password": { zh: "密码", en: "Password" },
    "login.password.ph": { zh: "请输入密码", en: "Enter password" },
    "login.submit": { zh: "登 录", en: "Sign in" },
    "login.empty": { zh: "请输入用户名与密码", en: "Enter username and password" },
    "login.busy": { zh: "登录中…", en: "Signing in…" },
    "login.fail": { zh: "登录失败", en: "Sign-in failed" },
    "login.network": { zh: "网络错误，请稍后重试", en: "Network error, please retry" },
    "login.parse": { zh: "服务返回了无法解析的响应", en: "Unreadable response from server" },
    "login.title": { zh: "登录 · Relay 控制台", en: "Sign in · Relay Console" }
  };

  // 英文模式下的整页文案表：渲染完成后按文本节点精确替换，用户数据与代码块不参与。
  var EN_TEXT = {
    "网络异常，请检查服务状态。": "Network error. Check the service status.",
    "AI 接入": "AI access",
    "创建 API Key，把控制台能力交给外部 AI 或脚本；接口文档无需登录即可阅读。": "Create API keys for external AI or scripts. API docs are public.",
    "接入说明": "Guide",
    "三步接入": "Three steps",
    "创建密钥": "Create a key",
    "为不同用途分别建钥匙，权限最小化。": "Issue separate keys per use case with least privilege.",
    "复制说明": "Copy brief",
    "下载说明": "Download brief",
    "说明包含地址、认证与常用流程，直接粘贴给 AI。": "The brief covers base URL, auth and common flows. Paste it into your AI.",
    "直接调用": "Call the API",
    "密钥放请求头，先读清单再执行操作。": "Send the key in the Authorization header. Read the manifest before acting.",
    "API Key": "API keys",
    "密钥只在创建时显示一次，妥善保存。": "A key is shown once at creation. Store it safely.",
    "备注": "Label",
    "前缀": "Prefix",
    "备注，例如 巡检机器人": "Label, e.g. patrol bot",
    "权限": "Scope",
    "有效期": "Expiry",
    "30 天": "30 days",
    "7 天": "7 days",
    "90 天": "90 days",
    "永久": "Never",
    "仅主人可管理密钥。": "Only owners can manage keys.",
    "给 AI 的说明书": "Brief for your AI",
    "下载 ai.txt": "Download ai.txt",
    "暂无 API Key": "No API keys yet",
    "新建一把密钥后即可让 AI 接入。": "Create a key to let an AI connect.",
    "密钥加载失败": "Failed to load keys",
    "撤销": "Revoke",
    "已撤销": "Revoked",
    "已创建": "Created",
    "已复制": "Copied",
    "已复制说明": "Brief copied",
    "复制": "Copy",
    "复制密钥": "Copy key",
    "复制失败，请手动选择文本": "Copy failed. Select the text manually.",
    "最后使用": "Last used",
    "创建于": "Created",
    "长期有效": "No expiry",
    "已过期": "Expired",
    "有效": "Active",
    "只读": "Read-only",
    "运营": "Operator",
    "主人": "Owner",
    "请输入备注": "Enter a label",
    "生成失败": "Generation failed",
    "新密钥只显示一次": "The new key is shown only once",
    "API Key 已创建": "API key created",
    "这是唯一一次显示明文，关闭后无法找回。": "This is the only time the key is shown. It cannot be recovered.",
    "我已保存": "I saved it",
    "复制接入说明": "Copy AI brief",
    "加载失败": "Load failed",
    "当前账号为只读或运营": "This account is read-only or operator",
    "执行频率": "Frequency",
    "每 5 分钟": "Every 5 min",
    "每 30 分钟": "Every 30 min",
    "每小时整点": "Hourly",
    "每天 09:00": "Daily 09:00",
    "每天 18:00": "Daily 18:00",
    "工作日 09:00": "Weekdays 09:00",
    "每周一 09:00": "Mondays 09:00",
    "自定义": "Custom",
    "每 5 分钟执行一次": "Runs every 5 minutes",
    "每 30 分钟执行一次": "Runs every 30 minutes",
    "每小时整点执行一次": "Runs hourly on the hour",
    "每天 09:00执行一次": "Runs daily at 09:00",
    "每天 18:00执行一次": "Runs daily at 18:00",
    "工作日 09:00执行一次": "Runs weekdays at 09:00",
    "每周一 09:00执行一次": "Runs Mondays at 09:00",
    "按秒 分 时 日 月 周填写": "Use sec min hour day month weekday",
    "每秒": "Every second",
    "每分钟": "Every minute",
    "每 10 分钟": "Every 10 min",
    "每 15 分钟": "Every 15 min",
    "模板": "Template",
    "选一个接近需求的模板，再改文件名和代码即可安装。": "Pick a close template, adjust the name and code, then install.",
    "JavaScript 空白模板": "Blank JavaScript",
    "Python 空白模板": "Blank Python",
    "回发消息（JavaScript）": "Echo message (JavaScript)",
    "回发消息（Python）": "Echo message (Python)",
    "权限不足": "Insufficient permission",
    "当前角色无权查看该页面": "Your role cannot open this page",
    "备注": "Label",
    "权限范围": "Scope",
    "有效期": "Expiry",
    "轮换": "Rotate",
    "已轮换": "Rotated",
    "轮换 API Key": "Rotate API key",
    "文件上传": "File upload",
    "最大 20 MB。上传后把 file_id 交给发送接口。": "Up to 20 MB. Pass the returned file_id to a send endpoint.",
    "选择文件": "Choose file",
    "文件名（可选）": "File name (optional)",
    "留空则使用原文件名": "Leave blank to keep the original name",
    "上传并获取 file_id": "Upload and get file_id",
    "请选择文件": "Choose a file",
    "文件不能超过 20 MB": "File must be 20 MB or smaller",
    "上传失败": "Upload failed",
    "上传成功": "Uploaded",
    "已复制 file_id": "file_id copied",
    "发送时调用 POST /jiying/api/send/file，传入 conversation_id、file_id、name、size。": "Then call POST /jiying/api/send/file with conversation_id, file_id, name and size.",
    "请先复制密钥，再点击「我已保存」": "Copy the key before closing",
    "运行失败": "Run failed",
    "已删除": "Deleted",
    "未登录": "Not signed in",
    "请稍后重试。": "Please try again later.",
    "已安装": "Installed",
    "删除": "Delete",
    "任务队列": "Task queue",
    "含 Agent 调用": "Include agent calls",
    "Agent 调用": "Agent call",
    "已连接": "Connected",
    "安装失败": "Install failed",
    "删除失败": "Delete failed",
    "插件": "Plugins",
    "登录失败": "Sign-in failed",
    "会话": "Conversation",
    "加载失败": "Load failed",
    "已登录": "Signed in",
    "参数": "Params",
    "运行中…": "Running…",
    "匹配": "matched",
    "置顶": "Pin",
    "消息收发": "Messages",
    "定时任务": "Schedules",
    "执行记录": "Runs",
    "按级别、追踪号与关键字检索；点击 trace 可反向筛选整条链路": "Filter by level, trace ID or keyword; click a trace to pivot across the request chain",
    "导出当前视图": "Export view",
    "日期": "Date",
    "行数": "Lines",
    "搜索": "Search",
    "关键字 / trace / 路径": "Keyword / trace / path",
    "自动换行": "Wrap lines",
    "全部": "All",
    "异常": "Issues",
    "错误": "Errors",
    "警告": "Warnings",
    "信息": "Info",
    "调试": "Debug",
    "回到底部": "Jump to latest",
    "双击日志行可复制 · 点击 trace 可按追踪号筛选": "Double-click a row to copy; click a trace to filter by it",
    "按此追踪号筛选": "Filter by this trace ID",
    "当日暂无日志": "No logs for this date",
    "无匹配日志": "No matching logs",
    "调整级别或关键字后重试。": "Adjust the level or keyword and try again.",
    "切换日期或刷新后查看。": "Switch the date or refresh to view logs.",
    "日志加载失败": "Failed to load logs",
    "加载失败": "Load failed",
    "加载中…": "Loading…",
    "重试": "Retry",
    "已复制该行日志": "Log line copied",
    "复制失败，请手动选择文本": "Copy failed. Select the text manually.",
    "已导出当前视图": "Exported current view",

    "记录": "Runs",
    "执行记录加载失败": "Failed to load runs",
    "暂无执行记录": "No runs yet",
    "定时任务执行或手动触发后会在这里留下结果。": "Scheduled or manual runs will appear here.",
    "新建定时后，中间件会按 cron 表达式自动执行，并记录每次结果。": "New schedules run automatically on their cron expression; every result is recorded.",
    "开始时间": "Started",
    "类型": "Type",
    "耗时": "Duration",
    "摘要": "Summary",
    "全部类型": "All types",
    "全部状态": "All statuses",
    "定时": "Scheduled",
    "手动": "Manual",
    "流程": "Flow",
    "成功": "Success",
    "失败": "Failed",
    "超时": "Timeout",
    "执行中": "Running",
    "跳过": "Skipped",
    "结束": "Finished",
    "执行者": "Owner",
    "追踪号": "Trace ID",
    "错误": "Error",
    "在日志中查找": "Find in logs",
    "执行详情": "Run details",
    "筛选名称 / 命令": "Filter name / command",
    "按类型筛选": "Filter by type",
    "按状态筛选": "Filter by status",
    "定时任务与手动触发的执行结果、耗时和追踪号": "Results, duration and trace IDs for scheduled and manual runs",
    "定时任务与手动触发后会在这里留下结果。": "Scheduled and manual runs leave their results here.",
    "运行一次": "Run now",
    "未运行": "Not run",
    "下次运行": "Next run",
    "上次结果": "Last result",
    "运行 / 失败": "Runs / fails",
    "调度": "Schedule",
    "按 cron 表达式执行系统命令或已启用功能，结果写入执行记录": "Run system commands or enabled features on a cron schedule; results are recorded under Runs",
    "连续失败 3 次会自动停用该定时，修正后可手动启用。": "Three consecutive failures pause the schedule automatically; re-enable it after fixing the cause.",
    "立即执行一次": "Run once now",
    "查看该任务执行记录": "View runs for this schedule",
    "每日额度重置": "Daily quota reset",
    "已启用功能": "Enabled feature",
    "系统命令": "System command",
    "功能名": "Feature name",
    "选择或输入已启用功能": "Pick or type an enabled feature",
    "功能参数（可选）": "Feature args (optional)",
    "会话 ID（可选）": "Conversation ID (optional)",
    "功能需要会话上下文时填写": "Fill when the feature needs conversation context",
    "管理员命令仅主人角色可创建与运行。": "Admin commands can only be created and run by the master role.",
    "请填写名称、表达式与执行内容": "Fill in the name, schedule and command",
    "已触发执行，稍后可在执行记录查看结果": "Run triggered. Check Runs for the result.",
    "触发失败": "Failed to trigger",
    "秒 分 时 日 月 周，例如 0 5 * * * * 表示每小时第 5 分钟。": "Second minute hour day month weekday, e.g. 0 5 * * * * means five minutes past every hour.",
    "该任务暂无执行记录": "No runs for this schedule yet",
    "账号会话": "Account session",
    "会话有效": "Session valid",
    "会话已失效": "Session expired",
    "状态未知": "Status unknown",
    "未配置账号": "No account configured",
    "立即检查": "Check now",
    "重新登录": "Sign in again",
    "退出账号": "Sign out account",
    "账号会话：": "Account session: ",
    "最近成功": "Last success",
    "检查于": "Checked",
    "该账号使用邮箱验证码登录": "This account signs in with an email code",
    "点击「立即检查」确认会话状态": "Click \"Check now\" to verify the session",
    "配置账号后可检查登录会话": "Configure an account to check its session",
    "退出后账号通道与自动发送将停止，确认退出？": "Signing out stops the account channel and automatic sending. Continue?",
    "已退出账号": "Account signed out",
    "退出失败": "Sign-out failed",
    "请重新获取验证码并登录": "Request a new code and sign in",
    "请重新输入密码并登录": "Enter the password again to sign in",
    "正在检查可用发送身份…": "Checking available send identities…",
    "选择会话后显示可用发送身份": "Pick a conversation to see available send identities",
    "身份 · 自动": "Identity · Auto",
    "身份 · 应用": "Identity · App",
    "身份 · 用户": "Identity · User",
    "预演": "Preview",
    "可发送身份：": "Send as: ",
    "当前会话没有可用发送身份 · ": "No send identity available for this conversation · ",
    "可用身份检查失败，发送时会再次校验": "Identity check failed; it will be rechecked on send",
    "发送预演": "Send preview",
    "解析身份": "Resolved identity",
    "知道了": "Got it",
    "未产生实际发送，也不会写入消息记录。": "Nothing was sent and no message was recorded.",
    "用户": "User",
    "网络异常，请稍后重试。": "Network error. Try again later.",
    "未连接": "Disconnected",
    "服务返回了无法解析的响应": "Unreadable response from server",
    "运营": "Operator",
    "只读": "Read-only",
    "功能": "Features",
    "正常": "Healthy",
    "未配置": "Not configured",
    "连接密钥": "Connection secret",
    "发送失败": "Send failed",
    "账号切换失败": "Account switch failed",
    "对方": "Peer",
    "应用": "App",
    "插件列表加载失败": "Failed to load plugins",
    "安装": "Install",
    "(无输出)": "(no output)",
    "读取配置失败": "Failed to read configuration",
    "配置已保存": "Configuration saved",
    "保存失败": "Save failed",
    "技能列表加载失败": "Failed to load skills",
    "账号": "Account",
    "功能列表加载失败": "Failed to load features",
    "已启用": "Enabled",
    "已停用": "Disabled",
    "操作失败": "Operation failed",
    "工具列表加载失败": "Failed to load tools",
    "定时任务加载失败": "Failed to load schedules",
    "状态": "Status",
    "任务列表加载失败": "Failed to load tasks",
    "网络异常，请稍后重试。": "Network error. Try again later.",
    "结果": "Result",
    "移除": "Remove",
    "移除失败": "Remove failed",
    "审计日志加载失败": "Failed to load audit log",
    "异常": "Unhealthy",
    "进程已运行": "Process uptime",
    "累计接收": "Total received",
    "授权失败": "Authorization failed",
    "即应接入与调度": "Jiying relay & scheduling",
    "概览": "Overview",
    "即应接入": "Connect",
    "运行": "Runtime",
    "工作台": "Dashboard",
    "接入系统": "Connect",
    "连接池": "Connection pool",
    "权限管理": "Access control",
    "插件市场": "Plugin market",
    "功能管理": "Features",
    "Agent 工作台": "Agent",
    "审计日志": "Audit log",
    "运行观测": "Observability",
    "运行日志": "Runtime logs",
    "筛选菜单": "Filter menu",
    "自定义侧栏": "Customize sidebar",
    "应用通道": "App channel",
    "账号通道": "Account channel",
    "应用已连接": "App connected",
    "应用未连接": "App disconnected",
    "账号未登录": "Signed out",
    "切换深浅色": "Toggle theme",
    "切换语言": "Switch language",
    "退出登录": "Sign out",
    "收起菜单": "Toggle menu",
    "调整顺序、置顶或隐藏侧栏菜单，设置保存在本机浏览器；新增菜单会自动加入。": "Reorder, pin or hide sidebar items. Changes are stored in this browser; new items are added automatically.",
    "恢复默认": "Reset",
    "完成": "Done",
    "取消置顶": "Unpin",
    "显示": "Show",
    "隐藏": "Hide",
    "即应接入与任务调度控制台": "Jiying relay & scheduling console",
    "接入即应，编排一切": "Connect Jiying, orchestrate everything",
    "Relay 面向即应应用的消息接入与任务调度中间件。通过应用密钥或账号授权一键接入，在聊天中完成功能调用、脚本执行与定时编排。": "Relay is the message-connectivity and task-scheduling middleware for Jiying apps. Connect with an app secret or account authorization, then invoke features, run scripts and schedule jobs right from chat.",
    "应用密钥接入": "App secret",
    "账号 / 邮箱验证码": "Account / email code",
    "功能目录": "Feature catalog",
    "定时调度": "Scheduling",
    "登录控制台": "Sign in",
    "登录后接入即应应用或账号通道": "Sign in to connect an app or account channel",
    "用户名": "Username",
    "请输入用户名": "Enter username",
    "密码": "Password",
    "请输入密码": "Enter password",
    "登 录": "Sign in",
    "请输入用户名与密码": "Enter username and password",
    "登录中…": "Signing in…",
    "网络错误，请稍后重试": "Network error, please retry",
    "登录 · Relay 控制台": "Sign in · Relay Console",
    "中文": "中文",
    "切换到浅色模式": "Switch to light mode",
    "切换到深色模式": "Switch to dark mode",
    "主人": "Owner",
    "关闭": "Close",
    "确认": "Confirm",
    "登录已过期，请重新登录": "Session expired. Please sign in again.",
    "（追踪": "(trace",
    "网络请求失败，请检查服务状态后重试": "Request failed. Check the service status and retry.",
    "网络请求失败": "Request failed",
    "已恢复默认侧栏": "Sidebar reset to defaults",
    "今日收": "Received today",
    "今日发": "Sent today",
    "运行时长": "Uptime",
    "定时": "Schedules",
    "任务": "Tasks",
    "会话列表与发送": "Conversations and sending",
    "安装与调试插件": "Install and debug plugins",
    "创建定时调度": "Create a schedule",
    "执行记录与重试": "Run history and retries",
    "已配置未连接": "Configured, offline",
    "配置有误": "Configuration error",
    "不可用": "Unavailable",
    "1. 本地存储": "1. Local storage",
    "2. 应用通道": "2. App channel",
    "3. 账号通道": "3. Account channel",
    "4. 模型 Agent": "4. Model agent",
    "未知状态": "Unknown state",
    "查看": "View",
    "去修复": "Fix",
    "全部就绪，可正常接收与回复消息": "All checks passed. Ready to receive and reply.",
    "项待处理": "items pending",
    "接口请求失败": "API request failed",
    "8-4-4-4-12 的 UUID，从应用详情复制": "UUID in 8-4-4-4-12 format, copied from the app details",
    "8 位邮箱验证码": "8-digit email code",
    "应用通道已更新": "App channel updated",
    "配置失败": "Configuration failed",
    "账号登录成功": "Signed in successfully",
    "账号验证并登录成功": "Verified and signed in",
    "应用握手成功": "App handshake succeeded",
    "连接测试失败": "Connection test failed",
    "请先填写服务地址与邮箱": "Enter the service URL and email first",
    "验证码已发送": "Code sent",
    "已验证码发送，有效期": "Code sent, valid for",
    "接入状态加载失败": "Failed to load connection status",
    "已连接 ·": "Connected ·",
    "选择一个会话": "Select a conversation",
    "从会话列表中选择，即可查看历史消息并发送内容。": "Pick a conversation to view history and send messages.",
    "输入消息，回车或点击发送": "Type a message, press Enter or click Send",
    "· 已登录": "· Signed in",
    "已切换账号": "Account switched",
    "会话加载失败": "Failed to load conversations",
    "暂无会话": "No conversations",
    "收到或发起会话后，会在这里列出。": "Conversations appear here after you receive or start one.",
    "会话列表加载失败": "Failed to load conversations",
    "消息加载失败": "Failed to load messages",
    "暂无消息": "No messages",
    "在下方输入框发送内容，中间件会按功能或 Agent 自动回复。": "Send a message below; the middleware replies via a feature or the agent.",
    "[图片]": "[Image]",
    "请选择会话并输入内容": "Select a conversation and enter a message",
    "已发送": "Sent",
    "系统": "System",
    "机器人": "Bot",
    "暂无插件": "No plugins",
    "插件放在 scripts 目录下，安装后会自动登记为可调用功能。": "Plugins live in the scripts directory and are registered as callable features after installation.",
    "按分类归档的插件资产，支持一个插件服务多个应用；安装后会自动登记为可调用功能": "Plugins filed by category. One plugin can serve multiple apps and becomes a callable feature after installation.",
    "分类": "Category",
    "服务": "Service",
    "目录": "Folders",
    "列表": "List",
    "全部应用": "All apps",
    "未归属": "Unidentified",
    "待识别": "Unidentified",
    "按应用筛选会话": "Filter conversations by app",
    "应用": "App",
    "会话未记录应用归属": "No app assignment",
    "应用已移除": "App removed",
    "应用通道可用": "app channel available",
    "应用通道离线": "app channel offline",
    "账号通道可用": "account available",
    "账号通道未登录": "account signed out",
    "该应用下暂无会话": "No conversations for this app",
    "该筛选下暂无会话": "No conversations match this filter",
    "未分类": "Uncategorized",
    "搜索插件 / 分类 / 服务 / 应用": "Search plugins / categories / services / apps",
    "搜索插件": "Search plugins",
    "规则": "Rule",
    "平台": "Platform",
    "编辑插件": "Edit plugin",
    "新建插件": "New plugin",
    "保存": "Save",
    "请填写文件名与代码": "Enter a file name and code",
    "读取失败": "Read failed",
    "该功能未关联插件文件": "This feature has no plugin file",
    "运行插件 ·": "Run plugin ·",
    "删除插件": "Delete plugin",
    "确认删除插件": "Delete plugin",
    "配置": "Configure",
    "需要数字": "Number required",
    "无配置项": "No configuration items",
    "该插件未声明 [config: ...]。": "This plugin does not declare [config: ...].",
    "插件配置 ·": "Plugin configuration ·",
    "暂无技能": "No skills",
    "技能可被插件通过 require 复用；安装 JS 技能时会先在沙箱内试跑。": "Skills can be reused by plugins via require; JS skills are test-run in a sandbox before installation.",
    "JS 模块": "JS module",
    "安装技能": "Install skill",
    "helper.js 或 note.md": "helper.js or note.md",
    "请填写文件名与内容": "Enter a file name and content",
    "运行技能 ·": "Run skill ·",
    "删除技能": "Delete skill",
    "确认删除技能": "Delete skill",
    "内置": "Built-in",
    "脚本": "Script",
    "公开": "Public",
    "主控": "Master",
    "需确认": "Needs confirmation",
    "手动范围": "Manual scope",
    "详情": "Details",
    "功能详情 ·": "Feature details ·",
    "HTTP 地址": "HTTP URL",
    "目标": "Target",
    "插件目标由插件文件决定，请通过「编辑插件」修改。": "The plugin target comes from its file. Use Edit plugin to change it.",
    "权限范围": "Permission scope",
    "public · 公开": "public · Public",
    "master · 主控": "master · Master",
    "调用身份": "Call identity",
    "账号": "Account",
    "调用前需要用户确认": "Require user confirmation before calls",
    "应用范围": "App scope",
    "尚未接入应用；接入后可为该功能指定应用范围。": "No apps connected yet. Once connected, you can scope this feature.",
    "取消「全部应用」后可勾选部分应用；保存后以手动设置为准，插件重启不会被清单覆盖。": "Clear All apps to select specific apps. Manual scope survives plugin rescans and reinstalls.",
    "功能不存在或列表未加载": "Feature not found or the list is not loaded",
    "插件配置": "Plugin config",
    "恢复插件声明范围": "Restore manifest scope",
    "恢复全部应用": "Restore all apps",
    "请选择至少一个应用，或勾选全部应用": "Select at least one app, or choose All apps",
    "功能设置已保存": "Feature settings saved",
    "应用范围已恢复": "App scope restored",
    "恢复失败": "Restore failed",
    "该插件未关联功能记录": "This plugin has no feature record",
    "该插件尚未登记为功能": "This plugin is not registered as a feature",
    "编辑应用 ·": "Edit app ·",
    "应用 ID": "App ID",
    "应用 ID 由 MagicChat 分配，不可修改。": "The app ID is assigned by MagicChat and cannot be changed.",
    "WebSocket 接入地址": "WebSocket endpoint",
    "应用密钥": "App secret",
    "保持 **** 表示沿用已保存密钥；更换接入地址时必须重新填写密钥。": "Keep **** to reuse the saved secret. Re-enter the secret if the endpoint changes.",
    "应用已更新": "App updated",
    "更新失败": "Update failed",
    "应用不在当前连接池中": "The app is not in the current pool",
    "编辑": "Edit",
    "全局系统提示": "Global system prompt",
    "按应用覆盖系统提示": "Per-app system prompt overrides",
    "该应用的系统提示": "System prompt for this app",
    "留空则继承全局系统提示": "Leave blank to inherit the global prompt",
    "在系统提示后追加该应用的约束与上下文。": "Adds app-specific constraints and context after the system prompt.",
    "保存应用提示": "Save app prompt",
    "暂无应用": "No apps",
    "接入应用后可为其设置独立系统提示。": "Connect an app to give it a dedicated system prompt.",
    "该应用当前使用独立系统提示；留空保存即恢复继承全局。": "This app uses its own prompt. Save an empty value to inherit the global prompt.",
    "留空保存即继承全局系统提示。": "Save an empty value to inherit the global prompt.",
    "请选择应用": "Select an app",
    "应用系统提示已保存": "App system prompt saved",
    "已清除覆盖，继承全局": "Override cleared; inheriting the global prompt",
    "暂无功能": "No features",
    "安装 HTTP 功能或添加插件后，会在这里统一管理。": "HTTP features and installed plugins are managed here.",
    "运行功能 ·": "Run feature ·",
    "对应插件文件也会一并删除。": "The plugin file will also be deleted.",
    "内置功能会在服务重启后自动恢复。": "Built-in features are restored after a restart.",
    "删除功能": "Delete feature",
    "确认删除功能": "Delete feature",
    "安装 HTTP 功能": "Install HTTP feature",
    "例如：天气": "Example: weather",
    "请填写功能名与地址": "Enter a feature name and URL",
    "留空则使用临时会话": "Leave blank to use a temporary conversation",
    "例如：现在系统状态怎么样": "Example: what is the system status now?",
    "无权限或未就绪": "No permission or not ready",
    "需要主控账号才能查看与修改 Agent 配置。": "Only master accounts can view or change agent configuration.",
    "填写服务商支持的模型名": "Enter a model name supported by the provider",
    "模型A, 模型B": "Model A, Model B",
    "Agent 配置加载失败": "Failed to load agent configuration",
    "暂无可调用功能": "No callable features",
    "启用功能或安装插件后，Agent 会自动获得对应工具。": "Enable features or install plugins to give the agent tools.",
    "请输入提问内容": "Enter a question",
    "暂无定时任务": "No schedules",
    "新建定时后，中间件会按 cron 表达式自动入队执行。": "Schedules enqueue jobs automatically based on the cron expression.",
    "停用": "Disable",
    "启用": "Enable",
    "新建定时任务": "New schedule",
    "示例任务": "Example task",
    "请填写名称、表达式与命令": "Enter a name, expression and command",
    "已创建": "Created",
    "创建失败": "Create failed",
    "删除定时任务": "Delete schedule",
    "确认删除该定时任务？": "Delete this schedule?",
    "按状态筛选": "Filter by status",
    "筛选功能 / 会话 / 结果": "Filter by feature / conversation / result",
    "暂无任务": "No tasks",
    "功能执行后会在这里留下记录。": "Feature runs are recorded here.",
    "（最近 100 条）": "(last 100)",
    "已重试": "Retried",
    "任务已不在列表中，请刷新": "The task is no longer in the list. Refresh.",
    "任务 ID": "Task ID",
    "用户": "User",
    "创建时间": "Created",
    "开始时间": "Started",
    "结束时间": "Finished",
    "耗时": "Duration",
    "重试次数": "Retries",
    "错误码": "Error code",
    "任务详情": "Task details",
    "删除任务": "Delete task",
    "确认删除任务？": "Delete this task?",
    "筛选关键字": "Filter keyword",
    "（无匹配行）": "(no matching rows)",
    "（当日暂无日志）": "(no logs today)",
    "暂无日志可导出": "No logs to export",
    "连接池状态加载失败": "Failed to load connection pool",
    "尚未接入应用": "No apps connected",
    "接入后可在此查看通道在线状态，并按应用隔离运行。": "Connect an app to see channel status and run apps in isolation.",
    "(未命名)": "(unnamed)",
    "暂无已授权账号": "No authorized accounts",
    "授权账号后可按账号身份调用功能，并与应用通道隔离运行。": "Authorize an account to call features with its identity, isolated from the app channel.",
    "账号已切换": "Account switched",
    "接入应用": "Connect app",
    "应用标识": "App ID",
    "请填写完整信息": "Fill in all fields",
    "已接入": "Connected",
    "接入失败": "Connect failed",
    "移除应用": "Remove app",
    "确认移除应用": "Remove app",
    "已移除": "Removed",
    "默认应用已切换": "Default app switched",
    "切换失败": "Switch failed",
    "管理应用 →": "Manage apps →",
    "正在切换到 ": "Switching to ",
    "已切换账号": "Account switched",
    "切换": "Switch",
    "当前": "Current",
    "个应用": "apps",
    "个账号": "accounts",
    "插件市场加载失败": "Failed to load plugin market",
    "市场暂无插件": "No plugins in the market",
    "上架插件后会在这里展示，供按应用或个人账号范围安装分发。": "Published plugins appear here for installation by app or account scope.",
    "未安装": "Not installed",
    "重装": "Reinstall",
    "上架插件": "Publish plugin",
    "插件用途": "Plugin usage",
    "// 脚本代码": "// script code",
    "请填写名称与目标": "Enter a name and target",
    "script 插件需要代码": "Script plugins require code",
    "已上架": "Published",
    "上架失败": "Publish failed",
    "下架插件": "Unpublish plugin",
    "确认从市场上架列表移除": "Remove from the market",
    "下架": "Unpublish",
    "筛选操作者 / 动作 / 目标 / 详情": "Filter by actor / action / target / details",
    "暂无审计记录": "No audit records",
    "敏感操作（接入、插件、角色等）会在这里留痕。": "Sensitive actions (connections, plugins, roles) are recorded here.",
    "在运行日志中查找": "Find in runtime logs",
    "暂无审计记录可导出": "No audit records to export",
    "时间": "Time",
    "操作者": "Actor",
    "动作": "Action",
    "目标": "Target",
    "详情": "Details",
    "条（最近 100 条）": "rows (last 100)",
    "失败": "Failed",
    "拒绝": "Rejected",
    "超时": "Timeout",
    "错误": "Error",
    "无效": "Invalid",
    "未授权": "Unauthorized",
    "健康探针": "Health probe",
    "运转时长": "Uptime",
    "已收消息": "Messages received",
    "执行中": "Running",
    "全部状态": "All statuses",
    "失败 / 超时": "Failed / timed out",
    "排队中": "Queued",
    "待确认": "Awaiting confirmation",
    "已取消": "Canceled",
    "应用插件": "App plugin",
    "个人账号插件": "Account plugin",
    "JavaScript 模块": "JavaScript module",
    "请先在接入系统登录账号": "Sign in to an account under Connect first",
    "账号状态加载失败，点「刷新」重试": "Failed to load account status. Click Refresh to retry.",
    "通道状态": "Channel status",
    "应用 · 账号": "App · Account",
    "限流拒绝": "Rate-limit rejections",
    "被拒绝任务": "Rejected tasks",
    "输入用户 ID": "Enter user ID",
    "主人列表加载失败": "Failed to load owners",
    "成员角色加载失败": "Failed to load member roles",
    "当前控制台账号": "Current console account",
    "暂无显式角色": "No explicit roles",
    "运营和只读成员会显示在这里。": "Operators and read-only members appear here.",
    "尚未设置主人": "No owners configured",
    "主人可以使用主控范围的功能和管理指令。": "Owners can use master-scope features and admin commands.",
    "请输入用户 ID": "Enter a user ID",
    "主人权限已授予": "Owner granted",
    "成员角色已更新": "Member role updated",
    "撤销角色": "Revoke role",
    "确认撤销该成员的显式角色？": "Revoke this member's explicit role?",
    "撤销": "Revoke",
    "角色已撤销": "Role revoked",
    "撤销失败": "Revoke failed",
    "移除主人": "Remove owner",
    "确认移除用户": "Remove user",
    "主人已移除": "Owner removed",
    "暂无 RPC 调用": "No RPC calls",
    "应用通道产生调用后，这里会按方法显示成功率与耗时。": "Once the app channel makes calls, success rate and latency appear here by method.",
    "（无指标）": "(no metrics)",
    "· 完成": "· done",
    "应用在线": "App online",
    "应用离线": "App offline",
    "累计拒绝 · 任务失败": "Rejected total · failed tasks",
    "中文": "中文",
    "主人": "Owner",
    "运营": "Operator",
    "只读": "Read-only",
    "即应接入与任务调度实时总览": "Live overview of Jiying connectivity and task scheduling",
    "刷新": "Refresh",
    "接入状态": "Connection status",
    "即应未连接": "Jiying not connected",
    "应用 ID": "App ID",
    "服务器时间": "Server time",
    "刷新周期": "Refresh interval",
    "10 秒自动刷新": "Auto-refresh every 10s",
    "快捷入口": "Quick actions",
    "接入检查": "Connection checklist",
    "首次使用按此顺序逐项自检": "Work through these checks in order on first use",
    "尚未配置应用通道 · 先添加一个 MagicChat 应用连接": "No app channel yet · add a MagicChat app connection first",
    "服务地址必须是 http:// 或 https:// 地址 · 检查账号服务地址和端口；缺少账号邮箱 · 填写用于登录 MagicChat 的邮箱；缺少登录凭据 · 填写密码或使用验证码登录": "Service address must be http:// or https:// · check the account service host and port; missing account email · enter the email used to sign in to MagicChat; missing credentials · enter a password or use an email code",
    "Agent 未启用 · 启用后可让用户用自然语言调用功能": "Agent disabled · enable it to let users invoke features in natural language",
    "管理员": "Administrator",
    "角色 ·": "Role ·",
    "· 参数": "· Params",
    "推荐顺序：先配「应用密钥」建立应用通道，再用「账号密码 / 邮箱验证码」授权账号；完成后在": "Recommended order: configure the app secret first to establish the app channel, then authorize the account with password or email code; afterwards verify each item in ",
    "逐项确认。": ".",
    "插件归属": "Plugin ownership",
    "或": "or",
    "，安装时按应用或个人账号隔离。": ", isolated by app or personal account at install time.",
    "帮助": "Help",
    "插件列表": "Plugin list",
    "定时列表": "Schedule list",
    "任务列表": "Task list",
    "日志": "Logs",
    "先清空上下文": "Clear context first",
    "自动刷新": "Auto-refresh",
    "尚未配置": "Not configured",
    "已配置，留空保持不变": "Configured; leave blank to keep",
    "自动换行": "Wrap lines",
    "选择会话查看消息并发送内容": "Select a conversation to read and send messages",
    "加载中…": "Loading…",
    "加载账号中…": "Loading accounts…",
    "重试": "Retry",
    "消息": "Messages",
    "发送": "Send",
    "工作台 · 接入检查": "Dashboard · Connection checklist",
    "1 应用密钥 → 2 账号授权 → 3 模型 Agent": "1 App secret → 2 Account authorization → 3 Model agent",
    "待处理项": "Pending items",
    "应用密钥": "App secret",
    "账号密码": "Account password",
    "邮箱验证码": "Email code",
    "WebSocket 接入地址": "WebSocket endpoint",
    "应用 ID 必须是 UUID，与账号通道指向同一个即应服务；不一致会导致应用握手 400。": "The app ID must be a UUID and point to the same Jiying service as the account channel; a mismatch causes a 400 during app handshake.",
    "保存并连接": "Save and connect",
    "测试连接": "Test connection",
    "多应用通道与个人账号状态，按应用隔离运行": "Multi-app channels and personal account status, isolated per app",
    "个人账号": "Personal account",
    "昵称": "Nickname",
    "未设置": "Not set",
    "已授权账号": "Authorized account",
    "去授权账号": "Authorize account",
    "接入指引": "Setup guide",
    "1. 准备应用": "1. Prepare the app",
    "在 MagicChat 创建应用，取得应用 ID 与应用密钥。": "Create an app in MagicChat and note its app ID and secret.",
    "2. 接入通道": "2. Connect the channel",
    "填写 WebSocket 地址、应用 ID 与密钥，保存后即建立通道。": "Enter the WebSocket endpoint, app ID and secret, then save to establish the channel.",
    "3. 按应用隔离": "3. Isolate by app",
    "多个应用各自运行与记账，默认应用用于未指定来源的任务。": "Each app runs and bills independently; the default app handles tasks without a specified source.",
    "主人权限面向即应用户 ID（在即应会话里发送「我的ID」获取）；运营与只读面向控制台登录账号。": "Owner permission targets Jiying user IDs (send \"my ID\" in a Jiying conversation to get yours); operator and read-only roles apply to console accounts.",
    "授权成员": "Authorized members",
    "授予权限": "Grant permission",
    "成员角色": "Member role",
    "主人列表": "Owner list",
    "已安装的插件资产，支持 JavaScript 与 Python；安装后会自动登记为可调用功能": "Installed plugin assets in JavaScript or Python; installed plugins are registered as callable features automatically",
    "功能管理 →": "Feature management →",
    "技能": "Skills",
    "可被插件 require 复用的 JS 模块，或 Markdown 说明；安装前会隔离试跑": "JS modules reusable via require, or Markdown notes; installed skills are trial-run in isolation first",
    "发现、上架与分发包，已安装的插件在「插件」页统一管理": "Discover, publish and distribute packages; installed plugins are managed on the Plugins page",
    "插件规范": "Plugin spec",
    "作用范围": "Scope",
    "类型": "Type",
    "支持脚本型与 http 型，脚本型随插件一同分发源码。": "Script and HTTP types are supported; script plugins ship their source with the package.",
    "版本": "Version",
    "同名插件按版本区分，已安装的会在列表中标注。": "Plugins with the same name differ by version; installed versions are marked in the list.",
    "面向即应会话的可调用功能：内置、HTTP 与插件功能的启停、试跑与配置": "Callable features for Jiying conversations: enable, trial-run and configure built-in, HTTP and plugin features",
    "安装功能": "Install feature",
    "薄 Agent 配置与在线试跑，模型只负责选功能与填参数": "Thin agent configuration and online trial runs; the model only picks features and fills parameters",
    "模型配置": "Model configuration",
    "仅主控可改": "Owner only",
    "模型": "Model",
    "后备模型": "Fallback models",
    "主模型不可用时按顺序降级，逗号分隔；留空表示不降级。": "Fallback in order when the primary model is unavailable, comma-separated; leave empty to disable fallback.",
    "系统提示": "System prompt",
    "上下文消息数": "Context messages",
    "每个会话保留的最近消息数量，数值越大越能延续上下文。": "Recent messages kept per conversation; larger values preserve more context.",
    "高级运行保护": "Advanced runtime guards",
    "单次工具调用上限": "Tool calls per run",
    "模型连续调用功能的最大次数，用于防止异常循环。": "Maximum consecutive feature calls per model run, to prevent runaway loops.",
    "保存配置": "Save configuration",
    "试跑": "Trial run",
    "不经过即应会话": "Without a Jiying conversation",
    "会话 ID（可选）": "Conversation ID (optional)",
    "提问": "Prompt",
    "等待运行…": "Waiting for run…",
    "工具契约": "Tool contract",
    "由功能与插件清单生成的 JSON Schema": "JSON Schema generated from features and plugin manifests",
    "按 cron 表达式定时入队执行功能": "Enqueue feature runs on a cron schedule",
    "新建定时": "New schedule",
    "名称": "Name",
    "表达式": "Expression",
    "命令": "Command",
    "操作": "Actions",
    "功能执行记录与状态": "Feature run history and status",
    "敏感操作留痕：操作者、动作、目标与结果": "Audit trail for sensitive actions: actor, action, target and result",
    "导出": "Export",
    "成功": "Success",
    "匿名探针与指标，不暴露任何凭据": "Anonymous probes and metrics, no credentials exposed",
    "RPC 调用": "RPC calls",
    "按方法聚合 · 来自 /metrics": "Aggregated by method · from /metrics",
    "原始指标": "Raw metrics",
    "Prometheus 文本格式": "Prometheus text format",
    "中间件运行日志（需登录控制台；访问日志仅记方法与路径）": "Middleware runtime logs (console sign-in required; access logs record method and path only)",
    "执行中间件功能「任务列表」（类型 builtin） · 参数 args": "Run middleware feature \"Task list\" (type builtin) · args",
    "执行中间件功能「定时列表」（类型 builtin） · 参数 args": "Run middleware feature \"Schedule list\" (type builtin) · args",
    "执行中间件功能「帮助」（类型 builtin） · 参数 args": "Run middleware feature \"Help\" (type builtin) · args",
    "执行中间件功能「插件列表」（类型 builtin） · 参数 args": "Run middleware feature \"Plugin list\" (type builtin) · args",
    "执行中间件功能「日志」（类型 builtin） · 参数 args": "Run middleware feature \"Logs\" (type builtin) · args",
    "执行中间件功能「状态」（类型 builtin） · 参数 args": "Run middleware feature \"Status\" (type builtin) · args",
    "___END___": ""
  };
  var I18N_SKIP = "code,script,style,textarea,[data-no-i18n],.conv-item,.msg-body,.log-line,.rl-codeblock";
  var I18N_RULES = [
    [/^(\d+) 项待处理$/, "$1 items pending"],
    [/^(\d+) 个常用动作$/, "$1 quick actions"],
    [/^(\d+) 条（最近 100 条）$/, "$1 rows (last 100)"],
    [/^(\d+) 会话$/, "$1 conversations"],
    [/^(\d+) 条$/, "$1 rows"],
    [/^共 (\d+) 个$/, "$1 total"],
    [/^(\d+) 个$/, "$1 items"],
    [/^(\d+) 行$/, "$1 lines"],
    [/^(\d+) 秒$/, "$1s"],
    [/^(.*)？其登记的可调用功能会一并移除。$/, "Delete $1? Its registered callable features will also be removed."],
    [/^(.*)？引用它的插件可能因此报错。$/, "Delete $1? Plugins that reference it may break."],
    [/^(.*)？已安装的插件与功能不受影响。$/, "Remove $1 from the market? Installed plugins and features are unaffected."],
    [/^功能 · (.*)$/, "Feature · $1"],
    [/^功能详情 · (.*)$/, "Feature details · $1"],
    [/^编辑应用 · (.*)$/, "Edit app · $1"],
    [/^角色 · 主人$/, "Role · Owner"],
    [/^角色 · 运营$/, "Role · Operator"],
    [/^角色 · 只读$/, "Role · Read-only"],
    [/^角色 · (\S+)$/, "Role · $1"],
    [/^账号通道未配置（追踪 (.+)）$/, "Account channel not configured (trace $1)"],
    [/^当前控制台账号 (\S+) · 主人$/, "Current console account $1 · Owner"],
    [/^当前控制台账号 (\S+) · 运营$/, "Current console account $1 · Operator"],
    [/^当前控制台账号 (\S+) · 只读$/, "Current console account $1 · Read-only"],
    [/^来源 (\S+)$/, "Source $1"],
    [/^共 (\d+) 条$/, "$1 rows"],
    [/^共 (\d+) 行$/, "$1 lines"],
    [/^匹配 (\d+) \/ 共 (\d+) 行$/, "$1 / $2 lines matched"],
    [/^筛选 (\d+) \/ 共 (\d+) 行$/, "$1 / $2 lines"],
    [/^共 (\d+) 行 · 最后刷新 (.+)$/, "$1 lines · updated $2"],
    [/^匹配 (\d+) \/ 共 (\d+) 行 · 最后刷新 (.+)$/, "$1 / $2 lines matched · updated $3"],
    [/^筛选 (\d+) \/ 共 (\d+) 行 · 最后刷新 (.+)$/, "$1 / $2 lines · updated $3"],
    [/^已导出当前视图 (\d+) 行$/, "Exported $1 lines"],
    [/^待 (\d+) · 完成 (\d+)$/, "Pending $1 · Done $2"],
    [/^累计拒绝 · 任务失败 (\d+)$/, "Rejected total · failed tasks $1"],
    [/^共 (\d+) 条（最多保留 500 条）$/, "$1 runs (up to 500 kept)"],
    [/^已通过(.+)身份发送$/, "Sent as $1"],
    [/^最近成功 (.+) · 检查于 (.+)$/, "Last success $1 · checked $2"],
    [/^检查于 (.+)$/, "Checked $1"],
    [/^可发送身份：应用 \/ 用户 · 应用通道可用；账号通道可用$/, "Send as: App / User · both channels available"],
    [/^可发送身份：应用 · 应用通道可用；账号通道未登录$/, "Send as: App · app channel available; account signed out"],
    [/^可发送身份：用户 · 会话不属于当前应用；账号通道可用$/, "Send as: User · conversation is not owned by this app; account available"],
    [/^可发送身份：用户 · 应用通道离线；账号通道可用$/, "Send as: User · app channel offline; account available"],
    [/^当前会话没有可用发送身份 · (.*)$/, "No send identity available · $1"],
    [/^命令已执行: (.+)$/, "Command executed: $1"],
    [/^命令未注册: (.+)$/, "Command not registered: $1"],
    [/^权限不足: (.+)$/, "Insufficient permission: $1"],
    [/^执行异常: (.*)$/, "Execution error: $1"],
    [/^连续失败 3 次，已暂停: (.*)$/, "Paused after 3 consecutive failures: $1"],
    [/^缺少功能名$/, "Missing feature name"],
    [/^会话 ID 不合法$/, "Invalid conversation ID"],
    [/^缺少命令$/, "Missing command"],
    [/^空命令$/, "Empty command"]
  ];
  var i18nObserver = null;
  function i18nTranslate(raw) {
    var text = String(raw || "");
    var s = text.trim();
    if (!s) return null;
    if (Object.prototype.hasOwnProperty.call(EN_TEXT, s)) return EN_TEXT[s];
    for (var i = 0; i < I18N_RULES.length; i++) {
      if (I18N_RULES[i][0].test(s)) return s.replace(I18N_RULES[i][0], I18N_RULES[i][1]);
    }
    return null;
  }
  function i18nLocalizeTextNode(node) {
    if (!node || LANG !== "en") return;
    var value = node.nodeValue || "";
    if (!value.trim()) return;
    if (node.__relayEn && node.__relayEn === value) return;
    var parent = node.parentElement;
    if (!parent || parent.closest(I18N_SKIP)) return;
    var hit = i18nTranslate(value);
    if (hit == null) return;
    if (node.__relayZh == null) node.__relayZh = value;
    var lead = value.match(/^\s*/)[0];
    var tail = value.match(/\s*$/)[0];
    node.__relayEn = lead + hit + tail;
    node.nodeValue = node.__relayEn;
  }
  function i18nLocalizeAttrs(el) {
    if (!el || el.nodeType !== 1 || LANG !== "en") return;
    ["placeholder", "title", "aria-label", "alt"].forEach(function (name) {
      var value = el.getAttribute(name);
      if (!value) return;
      var hit = i18nTranslate(value);
      if (hit == null) return;
      if (!el.__relayZhAttr) el.__relayZhAttr = {};
      if (el.__relayZhAttr[name] == null) el.__relayZhAttr[name] = value;
      el.setAttribute(name, hit);
    });
  }
  function i18nLocalize(root) {
    if (LANG !== "en" || !root) return;
    if (root.nodeType === 3) { i18nLocalizeTextNode(root); return; }
    if (root.nodeType !== 1) return;
    if (!root.closest(I18N_SKIP)) {
      var walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT, null);
      var nodes = [];
      while (walker.nextNode()) nodes.push(walker.currentNode);
      nodes.forEach(i18nLocalizeTextNode);
    }
    i18nLocalizeAttrs(root);
    Array.prototype.forEach.call(root.querySelectorAll("[placeholder],[title],[aria-label],[alt]"), i18nLocalizeAttrs);
  }
  function i18nRestore(root) {
    root = root || document.body;
    if (root.nodeType === 3) {
      if (root.__relayZh != null) root.nodeValue = root.__relayZh;
      return;
    }
    var walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT, null);
    var nodes = [];
    while (walker.nextNode()) nodes.push(walker.currentNode);
    nodes.forEach(function (n) { if (n.__relayZh != null) n.nodeValue = n.__relayZh; delete n.__relayEn; });
    var els = root.querySelectorAll ? root.querySelectorAll("*") : [];
    Array.prototype.forEach.call(els, function (el) {
      if (!el.__relayZhAttr) return;
      Object.keys(el.__relayZhAttr).forEach(function (name) { el.setAttribute(name, el.__relayZhAttr[name]); });
      delete el.__relayZhAttr;
    });
  }
  function i18nStartObserver() {
    if (i18nObserver || LANG !== "en") return;
    i18nObserver = new MutationObserver(function (list) {
      list.forEach(function (m) {
        if (m.type === "characterData") { i18nLocalizeTextNode(m.target); return; }
        Array.prototype.forEach.call(m.addedNodes, function (n) {
          if (n.nodeType === 1) i18nLocalize(n);
          else if (n.nodeType === 3) i18nLocalizeTextNode(n);
        });
      });
    });
    i18nObserver.observe(document.body, { childList: true, subtree: true, characterData: true });
  }
  function i18nStopObserver() {
    if (i18nObserver) { i18nObserver.disconnect(); i18nObserver = null; }
  }
  function readLang() {
    try {
      var v = localStorage.getItem(LANG_KEY);
      if (v === "en" || v === "zh") return v;
    } catch (e) { /* 隐私模式等场景下回退中文 */ }
    return "zh";
  }
  function t(key) {
    var e = DICT[key];
    if (!e) return key;
    return (LANG === "en" && e.en) ? e.en : e.zh;
  }
  function setLang(v) {
    LANG = (v === "en") ? "en" : "zh";
    try { localStorage.setItem(LANG_KEY, LANG); } catch (e) { /* 忽略存储失败 */ }
    document.documentElement.lang = (LANG === "en") ? "en" : "zh-CN";
    if (LANG === "en") i18nStartObserver();
    else i18nStopObserver();
  }
  // 翻译静态 HTML 里带 data-i18n* 标记的节点。
  function applyStaticI18n() {
    Array.prototype.forEach.call(document.querySelectorAll("[data-i18n]"), function (el) {
      el.textContent = t(el.getAttribute("data-i18n"));
    });
    Array.prototype.forEach.call(document.querySelectorAll("[data-i18n-ph]"), function (el) {
      el.setAttribute("placeholder", t(el.getAttribute("data-i18n-ph")));
    });
    Array.prototype.forEach.call(document.querySelectorAll("[data-i18n-title]"), function (el) {
      var v = t(el.getAttribute("data-i18n-title"));
      el.setAttribute("title", v);
      el.setAttribute("aria-label", v);
    });
  }
  function refreshLangButtons() {
    Array.prototype.forEach.call(document.querySelectorAll("[data-lang-label]"), function (el) {
      el.textContent = (LANG === "en") ? "中文" : "EN";
    });
  }
  // 在语言之间切换并重绘受影响的动态外壳；页面正文由各 render 函数按 t() 渲染。
  function toggleLang() {
    var next = LANG === "en" ? "zh" : "en";
    if (next === "zh") { i18nStopObserver(); i18nRestore(document.body); }
    setLang(next);
    applyStaticI18n();
    refreshLangButtons();
    // 登录页没有控制台外壳，只翻译静态文案后即可返回。
    if (!document.querySelector(".sidebar")) {
      if (LANG === "en") i18nLocalize(document.body);
      return;
    }
    renderNav();
    if (state.view) {
      var n = NAV.filter(function (x) { return x.id === state.view; })[0];
      var c = $("#crumbs");
      if (c) c.textContent = n ? "Relay · " + t(n.name) : "Relay";
    }
    loadConnectStatus();
    if (state.view && renderers[state.view]) { renderers[state.view](); decoratePage(state.view); }
    if (LANG === "en") i18nLocalize(document.body);
  }

  var NAV = [
    { id: "home", name: "nav.home", ik: "dashboard", group: "nav.group.overview" },
    { id: "chat", name: "nav.chat", ik: "message", group: "nav.group.access" },
    { id: "connect", name: "nav.connect", ik: "link", group: "nav.group.access" },
    { id: "pool", name: "nav.pool", ik: "layers", group: "nav.group.access" },
    { id: "dingtalk", name: "nav.dingtalk", ik: "chat-dots", group: "nav.group.access", minRole: "ops" },
    { id: "roles", name: "nav.roles", ik: "users", group: "nav.group.access" },
    { id: "ai", name: "nav.ai", ik: "key", group: "nav.group.access" },
    { id: "scripts", name: "nav.scripts", ik: "puzzle", group: "nav.group.run" },
    { id: "market", name: "nav.market", ik: "inbox", group: "nav.group.run" },
    { id: "features", name: "nav.features", ik: "toggles", group: "nav.group.run" },
    { id: "agent", name: "nav.agent", ik: "robot", group: "nav.group.run" },
    { id: "cron", name: "nav.cron", ik: "clock", group: "nav.group.run" },
    { id: "executions", name: "nav.executions", ik: "activity", group: "nav.group.run" },
    { id: "tasks", name: "nav.tasks", ik: "list", group: "nav.group.run" },
    { id: "audit", name: "nav.audit", ik: "shield", group: "nav.group.run" },
    { id: "observe", name: "nav.observe", ik: "activity", group: "nav.group.run" },
    { id: "logs", name: "nav.logs", ik: "file", group: "nav.group.run", minRole: "ops" }
  ];

  // 图标统一使用随包内置的 Bootstrap Icons（vendor/bootstrap），不再依赖外部图标服务。
  // 全局错误采集：保留最近 20 条，供自测脚本与排障读取（window.__jsErrors）。
  function installErrorSink() {
    window.__jsErrors = window.__jsErrors || [];
    function push(kind, msg, src) {
      try {
        window.__jsErrors.push({ kind: kind, msg: String(msg), src: String(src || ""), at: Date.now() });
        if (window.__jsErrors.length > 20) window.__jsErrors.shift();
      } catch (e) { /* 采集失败时静默，避免影响主流程 */ }
    }
    window.addEventListener("error", function (e) {
      push("error", e.message, (e.filename || "") + ":" + (e.lineno || 0) + ":" + (e.colno || 0));
    });
    window.addEventListener("unhandledrejection", function (e) {
      var r = e.reason;
      push("rejection", (r && (r.message || r.msg)) || r, "");
    });
  }
  var ICON = {
    dashboard: "bi-grid-1x2",
    message: "bi-chat-dots",
    link: "bi-link-45deg",
    code: "bi-code-slash",
    puzzle: "bi-puzzle",
    folder: "bi-folder2-open",
    "chevron-down": "bi-chevron-down",
    search: "bi-search",
    toggles: "bi-toggles",
    robot: "bi-robot",
    clock: "bi-clock-history",
    list: "bi-list-ul",
    file: "bi-file-earmark-text",
    inbox: "bi-box-seam",
    send: "bi-send",
    users: "bi-people",
    zap: "bi-lightning-charge",
    repeat: "bi-arrow-repeat",
    layers: "bi-layers",
    shield: "bi-shield-check",
    activity: "bi-activity",
    key: "bi-key",
    refresh: "bi-arrow-clockwise",
    user: "bi-person",
    logout: "bi-box-arrow-right",
    theme: "bi-moon-stars",
    themeLight: "bi-sun",
    settings: "bi-sliders",
    language: "bi-translate",
    close: "bi-x-lg"
  };
  function ico(kw) {
    return '<i class="bi ' + (ICON[kw] || ICON.zap) + '" aria-hidden="true"></i>';
  }
  // 插件图标：支持 Bootstrap 图标名（bi-*）、图片地址（http(s)/data:image/相对图片路径）与 emoji 文本。
  // 只放行可预期的安全形式，其余一律按文本转义输出，避免注入。
  function pluginIcon(raw) {
    var s = String(raw == null ? "" : raw).trim();
    if (!s) return ico("puzzle");
    if (/^bi-[a-z0-9-]+$/i.test(s)) return '<i class="bi ' + esc(s) + '" aria-hidden="true"></i>';
    if (/^https?:\/\//i.test(s) || /^data:image\//i.test(s)) return '<img src="' + esc(s) + '" alt="" loading="lazy">';
    if (s.indexOf("..") < 0 && s[0] !== "/" && /^[a-z0-9._/-]+\.(svg|png|jpe?g|gif|webp)$/i.test(s)) return '<img src="' + esc(s) + '" alt="" loading="lazy">';
    return '<span class="ico-emoji">' + esc(s) + "</span>";
  }
  // 仅补齐「声明了 data-ico 但没有内联图标」的静态节点（退出按钮、语言按钮）。
  // 顶栏头像是运行时渲染的图片/首字母，跳过它，避免切页面时被默认用户图标覆盖。
  function hydrateIcons() {
    $$("[data-ico]").forEach(function (el) {
      if (el.id === "avatar") return;
      if (el.querySelector("i, svg, img, .av-fb")) return;
      el.innerHTML = ico(el.getAttribute("data-ico"));
    });
  }
  // 深浅色主题：优先用户选择（localStorage），否则跟随系统；内联在 <head> 的脚本已先行写入属性避免闪白。
  var THEME_KEY = "relay-theme";
  function preferredTheme() {
    try { var t = localStorage.getItem(THEME_KEY); if (t === "dark" || t === "light") return t; } catch (e) {}
    return "dark";
  }
  function applyTheme(t, persist) {
    var dark = t === "dark";
    document.documentElement.setAttribute("data-bs-theme", dark ? "dark" : "light");
    if (persist) { try { localStorage.setItem(THEME_KEY, dark ? "dark" : "light"); } catch (e) {} }
    var b = $("#themeBtn");
    if (b) {
      b.innerHTML = ico(dark ? "themeLight" : "theme");
      var label = dark ? "切换到浅色模式" : "切换到深色模式";
      b.setAttribute("title", label);
      b.setAttribute("aria-label", label);
    }
  }
  function installTheme() {
    applyTheme(preferredTheme(), false);
    var b = $("#themeBtn");
    if (b) b.addEventListener("click", function () {
      applyTheme(document.documentElement.getAttribute("data-bs-theme") === "dark" ? "light" : "dark", true);
    });
  }
  // 统一的空状态：图标 + 主文案 + 说明 + 可选操作，替代单行灰字。
  function emptyState(kw, title, hint, actionHTML) {
    return '<div class="rl-empty" role="status"><div class="rl-empty-ico">' + ico(kw) + "</div>"
      + "<h4>" + esc(title) + "</h4>"
      + (hint ? "<p>" + esc(hint) + "</p>" : "")
      + (actionHTML ? '<div class="rl-empty-act">' + actionHTML + "</div>" : "")
      + "</div>";
  }
  // 统一的加载态：骨架条 + 无障碍播报，避免各页散落的单行灰字。
  function loadingState(label) {
    return '<div class="rl-loading" role="status" aria-live="polite" aria-busy="true">'
      + '<div class="rl-skel" aria-hidden="true"><span></span><span></span><span></span></div>'
      + '<span class="rl-sr">' + esc(label || "加载中…") + "</span>"
      + "</div>";
  }
  // 加载失败态：与空态区分开，并给一个重试入口，避免用户只能刷新整页。
  function errorState(box, kw, title, hint, retry) {
    if (!box) return;
    box.innerHTML = emptyState(kw, title, hint || "请稍后重试。", '<button class="btn btn-ghost btn-sm">重试</button>');
    box.setAttribute("aria-busy", "false");
    var st = box.querySelector(".rl-empty");
    if (st) st.setAttribute("role", "alert");
    var b = box.querySelector(".rl-empty-act button");
    if (b && retry) b.addEventListener("click", retry);
  }
  // 表格内的错误态必须包在 tr/td 中，否则浏览器会把块级元素移出 tbody 破坏表格布局。
  function tableError(box, cols, kw, title, hint, retry) {
    if (!box) return;
    box.innerHTML = '<tr><td colspan="' + cols + '">' + emptyState(kw, title, hint || "请稍后重试。", '<button class="btn btn-ghost btn-sm">重试</button>') + "</td></tr>";
    var b = box.querySelector(".rl-empty-act button");
    if (b && retry) b.addEventListener("click", retry);
  }

  var state = { view: "home", cerr: null, convs: [], conv: null, msgs: [], interval: null, baseUptime: null, uptimeAt: 0, uptimeTimer: null, chatEpoch: 0, chatRequest: 0, chatConvID: "", logLines: [], logShown: [], logPending: "", logTimer: null, logLevel: "all", logPinned: true, logFromBottom: 0, logLoadedAt: "", logDay: "", tasks: [], tasksErr: "", silentTasks: [], role: "", auditRows: [], auditErr: "", executions: [], executionsErr: "", execFilter: { kind: "", status: "", name: "" }, chatCaps: null, convFilter: "", account: null, app: null, apps: [], appProfiles: {}, appNames: {}, scriptView: "tree", scriptFolders: {} };
  var VIEW_KEY = "relay-view";
  var PROFILE_KEY = "relay-profile";
  var ROLE_LABEL = { master: "主人", ops: "运营", readonly: "只读" };

  function $(sel, root) { return (root || document).querySelector(sel); }
  function $$(sel, root) { return Array.prototype.slice.call((root || document).querySelectorAll(sel)); }
  function esc(s) {
    return String(s == null ? "" : s).replace(/[&<>"']/g, function (c) {
      return { "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c];
    });
  }
  // 受控 Markdown：先转义全部 HTML，再只放开行内代码、加粗与换行，
  // 不解析原始 HTML 或链接，避免消息内容成为 XSS 入口。
  function renderMarkdownSafe(text) {
    var html = esc(text)
      .replace(/`([^`\n]+)`/g, "<code>$1</code>")
      .replace(/\*\*([^*\n]+)\*\*/g, "<strong>$1</strong>")
      .replace(/\r?\n/g, "<br>");
    return html;
  }
  // Inline handlers are retained for generated legacy markup; encode every argument as a JS string literal.
  function jsArg(s) {
    return "'" + String(s == null ? "" : s)
      .replace(/\\/g, "\\\\").replace(/'/g, "\\'").replace(/"/g, "\\u0022")
      .replace(/&/g, "\\u0026").replace(/</g, "\\u003C").replace(/>/g, "\\u003E")
      .replace(/\r/g, "\\r").replace(/\n/g, "\\n").replace(/\u2028/g, "\\u2028").replace(/\u2029/g, "\\u2029") + "'";
  }
  function shortID(id) {
    id = String(id || "");
    return id.length > 18 ? id.slice(0, 8) + "…" + id.slice(-4) : id;
  }
  // 头像统一渲染：有图片时覆盖首字母；加载失败保留同色首字母，避免破图与空白。
  // 首字母底色由名称稳定散列，同名账号在不同页面保持同一颜色。
  function avatarHue(name) {
    var h = 0, s = String(name || "U");
    for (var i = 0; i < s.length; i++) h = (h * 31 + s.charCodeAt(i)) % 360;
    return h;
  }
  function avatarInner(url, name) {
    var label = String(name || "U").trim() || "U";
    var ch = esc(label.charAt(0).toUpperCase());
    return '<span class="av-fb" style="--av-h:' + avatarHue(label) + '">' + ch + "</span>"
      + (url ? '<img src="' + esc(url) + '" alt="" loading="lazy" decoding="async" referrerpolicy="no-referrer" onerror="this.onerror=null;this.style.display=\'none\';this.parentNode.classList.add(\'av-fallback\');console.warn(\'头像加载失败\',this.src)">' : "");
  }
  function avatarHTML(url, name, cls) {
    var klass = "avatar" + (cls ? " " + cls : "");
    return '<span class="' + klass + '">' + avatarInner(url, name) + "</span>";
  }
  // 上游不同版本把头像放在 avatar/avatar_url/icon 或 peer/user/target 等嵌套对象里，
  // 统一挑出第一个可用地址；缺失时返回空串，由首字母底色兜底，避免出现默认图标或空白。
  function pickAvatarURL(obj, depth) {
    if (!obj || typeof obj !== "object" || (depth || 0) > 3) return "";
    var keys = ["avatar_url", "avatar", "icon", "photo", "avatarUrl"];
    for (var i = 0; i < keys.length; i++) {
      var v = obj[keys[i]];
      if (typeof v === "string" && v.trim()) return v.trim();
    }
    var nested = ["peer", "user", "target", "contact", "sender", "profile", "owner", "from", "account"];
    for (var j = 0; j < nested.length; j++) {
      var got = pickAvatarURL(obj[nested[j]], (depth || 0) + 1);
      if (got) return got;
    }
    if (Object.prototype.toString.call(obj.members) === "[object Array]") {
      for (var k = 0; k < obj.members.length; k++) {
        var m = pickAvatarURL(obj.members[k], (depth || 0) + 1);
        if (m) return m;
      }
    }
    return "";
  }
  // 会话头像只读取会话或直接对端的头像。群会话没有头像时使用名称首字，
  // 不能从 members 里随便挑一个成员，否则会出现与会话无关的头像。
  function pickConversationAvatar(conv) {
    if (!conv || typeof conv !== "object") return "";
    var keys = ["avatar_url", "avatar", "icon", "photo", "avatarUrl"];
    for (var i = 0; i < keys.length; i++) {
      var v = conv[keys[i]];
      if (typeof v === "string" && v.trim()) return v.trim();
    }
    var peer = conv.peer || conv.user || conv.target || conv.contact || null;
    return peer ? pickAvatarURL(peer, 0) : "";
  }
  // 会话成员按 id 建索引，消息 sender 只带 id/type，昵称与头像需要从成员表补齐。
  function memberIndex(conv) {
    var out = {};
    var list = conv && conv.members;
    if (Object.prototype.toString.call(list) !== "[object Array]") return out;
    for (var i = 0; i < list.length; i++) {
      var m = list[i];
      if (m && m.id != null) out[String(m.id)] = m;
    }
    return out;
  }
  function appLookup(id) {
    if (!id) return null;
    var list = state.apps || [];
    for (var i = 0; i < list.length; i++) {
      if (String(list[i].app_id) === String(id)) return list[i];
    }
    return null;
  }
  function pickDisplayName(obj, fallback) {
    if (!obj || typeof obj !== "object") return fallback || "";
    return obj.nickname || obj.name || obj.title || obj.display_name || obj.email || fallback || "";
  }
  function avatarName(st) {
    var a = (st && st.account) || {};
    return a.nickname || a.email || "U";
  }
  // 提示条基于 Bootstrap Toast：自带淡入淡出、自动隐藏与无障碍播报。
  var TOAST_ICON = { ok: "bi-check-circle-fill", err: "bi-exclamation-triangle-fill", info: "bi-info-circle-fill" };
  function toast(msg, type) {
    var box = $("#toasts");
    if (!box) return;
    var kind = TOAST_ICON[type] ? type : "info";
    var el = document.createElement("div");
    el.className = "toast rl-toast rl-toast-" + kind;
    el.setAttribute("role", kind === "err" ? "alert" : "status");
    el.setAttribute("aria-live", kind === "err" ? "assertive" : "polite");
    el.setAttribute("aria-atomic", "true");
    // 错误信息里可能带短追踪号，附一个按钮直接按该号跳运行日志，省去手抄。
    var trace = (String(msg).match(/追踪 ([0-9a-zA-Z-]+)/) || [])[1] || "";
    el.innerHTML = '<div class="d-flex align-items-center">'
      + '<div class="toast-body">'
      + '<i class="bi ' + TOAST_ICON[kind] + ' rl-toast-ico" aria-hidden="true"></i>'
      + '<span class="rl-toast-msg">' + esc(msg) + "</span>"
      + (trace ? '<button type="button" class="btn btn-ghost btn-sm rl-toast-act">去日志</button>' : "")
      + "</div>"
      + '<button type="button" class="btn-close btn-close-white rl-toast-x" data-bs-dismiss="toast" aria-label="关闭"></button>'
      + "</div>";
    box.appendChild(el);
    if (LANG === "en") i18nLocalize(el);
    el.addEventListener("hidden.bs.toast", function () { el.remove(); });
    var act = el.querySelector(".rl-toast-act");
    if (act) act.addEventListener("click", function () {
      bootstrap.Toast.getOrCreateInstance(el).hide();
      searchLogs(trace);
    });
    bootstrap.Toast.getOrCreateInstance(el, { delay: trace ? 8000 : 3200 }).show();
  }
  // 弹窗基于 Bootstrap Modal：自带遮罩、焦点圈定、Esc 关闭与滚动锁定。
  function showModal(title, bodyHTML, actionsHTML) {
    $("#modalTitle").textContent = title;
    $("#modalBody").innerHTML = bodyHTML;
    $("#modalFoot").innerHTML = actionsHTML || '<button class="btn btn-ghost" data-bs-dismiss="modal">关闭</button>';
    if (LANG === "en") i18nLocalize($("#modal"));
    bootstrap.Modal.getOrCreateInstance($("#modal")).show();
  }
  function closeModal() {
    var inst = bootstrap.Modal.getInstance($("#modal"));
    if (inst) inst.hide();
  }
  // 危险操作确认：统一走 Bootstrap Modal，替代原生 confirm，保证样式、遮罩与可访问性一致。
  function confirmDialog(title, message, confirmLabel, onConfirm) {
    showModal(title,
      '<p class="rl-confirm-msg">' + esc(message) + "</p>",
      '<button class="btn btn-ghost" data-bs-dismiss="modal">取消</button>'
      + '<button class="btn btn-danger" id="rl_confirm_ok">' + esc(confirmLabel || "确认") + "</button>");
    var ok = $("#rl_confirm_ok");
    if (ok) ok.addEventListener("click", function () {
      closeModal();
      if (typeof onConfirm === "function") onConfirm();
    });
  }
  function api(url, opts) {
    opts = opts || {};
    var init = { method: opts.method || "GET", credentials: "same-origin", headers: {} };
    if (opts.form) {
      init.headers["Content-Type"] = "application/x-www-form-urlencoded";
      init.body = new URLSearchParams(opts.form).toString();
    } else if (opts.body) {
      init.body = opts.body;
    }
    // 统一超时，避免后端卡死时请求永久悬挂、按钮一直处于加载态。
    var timer = null, ctrl = null;
    if (typeof AbortController !== "undefined") {
      ctrl = new AbortController();
      init.signal = ctrl.signal;
      timer = setTimeout(function () { ctrl.abort(); }, opts.timeout || 30000);
    }
    var clearTimer = function () { if (timer) { clearTimeout(timer); timer = null; } };
    return fetch(url, init).then(function (r) {
      return r.text().then(function (raw) {
        var data;
        try { data = raw ? JSON.parse(raw) : {}; } catch (e) { data = { code: r.status, msg: "服务返回了无法解析的响应" }; }
        if (r.status === 401 || data.code === 401) {
          if (location.pathname.indexOf("login.html") < 0) location.href = "login.html";
          return { code: 401, msg: "登录已过期，请重新登录" };
        }
        if (r.status === 403 && (!data.code || data.code === 200)) data.code = 403;
        if (!data.code && !r.ok) data.code = r.status;
        // 错误响应附带短追踪号，方便在运行日志中按 trace 定位整体链路。
        if (data && data.code !== 200 && data.trace_id) {
          var shortTrace = "（追踪 " + String(data.trace_id).slice(0, 8) + "）";
          if (data.msg) data.msg += shortTrace;
          else if (data.message) data.message += shortTrace;
        }
        return data;
      });
    }).catch(function (err) {
      var timeout = !!(err && err.name === "AbortError");
      if (!opts.silent) toast(timeout ? "请求超时，请稍后重试" : "网络请求失败，请检查服务状态后重试", "err");
      return { code: timeout ? 408 : 0, msg: timeout ? "请求超时" : "网络请求失败" };
    }).then(function (v) { clearTimer(); return v; });
  }
  function badge(el, good, label) {
    el.className = "badge " + (good ? "badge-ok" : "badge-off");
    el.textContent = label + (good ? "" : "");
  }
  function statusChip(ok, yes, no) {
    return '<span class="chip ' + (ok ? "chip-success" : "chip-danger") + '">' + esc(ok ? yes : no) + "</span>";
  }

  /* ---------- 导航 ---------- */
  // 侧栏自定义：顺序、置顶、隐藏三项偏好存在本机浏览器，新增菜单项会自动补齐。
  var NAV_PREF_KEY = "relay-nav-prefs";
  function navPrefs() {
    var p = { order: [], hidden: [], pinned: [] };
    try {
      var raw = JSON.parse(localStorage.getItem(NAV_PREF_KEY) || "null");
      if (raw && typeof raw === "object") {
        if (Array.isArray(raw.order)) p.order = raw.order.filter(function (x) { return typeof x === "string"; });
        if (Array.isArray(raw.hidden)) p.hidden = raw.hidden.filter(function (x) { return typeof x === "string"; });
        if (Array.isArray(raw.pinned)) p.pinned = raw.pinned.filter(function (x) { return typeof x === "string"; });
      }
    } catch (e) { /* 隐私模式等场景下静默回退到默认布局 */ }
    var ids = NAV.map(function (x) { return x.id; });
    function valid(list) { return list.filter(function (id) { return ids.indexOf(id) >= 0; }); }
    p.order = valid(p.order);
    p.hidden = valid(p.hidden);
    p.pinned = valid(p.pinned);
    ids.forEach(function (id) { if (p.order.indexOf(id) < 0) p.order.push(id); });
    return p;
  }
  function saveNavPrefs(p) {
    try { localStorage.setItem(NAV_PREF_KEY, JSON.stringify(p)); } catch (e) { /* 忽略存储失败 */ }
  }
  function defaultNavPrefs() {
    return { order: NAV.map(function (x) { return x.id; }), hidden: [], pinned: [] };
  }
  // 按偏好顺序展开为实际可见菜单：置顶项在前，隐藏项被过滤。
  // 角色不足时隐藏对应菜单；角色尚未加载完成时先全部展示，由后端继续兜底。
  var ROLE_RANK = { master: 3, ops: 2, readonly: 1 };
  function roleAllowed(minRole) {
    if (!minRole) return true;
    var have = ROLE_RANK[state.role] || 0;
    return have === 0 || have >= ROLE_RANK[minRole];
  }
  function orderedNav(p) {
    var byId = {};
    NAV.forEach(function (x) { byId[x.id] = x; });
    var visible = p.order.map(function (id) { return byId[id]; }).filter(function (x) { return x && p.hidden.indexOf(x.id) < 0 && roleAllowed(x.minRole); });
    return {
      pinned: visible.filter(function (x) { return p.pinned.indexOf(x.id) >= 0; }),
      rest: visible.filter(function (x) { return p.pinned.indexOf(x.id) < 0; })
    };
  }
  function defaultViewId() {
    var o = orderedNav(navPrefs());
    var all = o.pinned.concat(o.rest);
    return all.length ? all[0].id : "home";
  }
  // 记住上次停留的页面：刷新或重新登录后回到原页面，而不是每次都跳回工作台。
  function savedViewId() {
    try {
      var v = localStorage.getItem(VIEW_KEY);
      if (v && NAV.some(function (x) { return x.id === v; })) return v;
    } catch (e) { /* 隐私模式回退默认页 */ }
    return "";
  }
  function rememberView(id) {
    try { localStorage.setItem(VIEW_KEY, id); } catch (e) { /* 忽略存储失败 */ }
  }
  function renderNav() {
    var nav = $("#nav");
    if (!nav) return;
    var prefs = navPrefs();
    var o = orderedNav(prefs);
    function itemHTML(it) {
      return '<button type="button" class="nav-item" data-view="' + it.id + '"><span class="ico" data-ico="' + it.ik + '">' + ico(it.ik) + '</span><span>' + esc(t(it.name)) + "</span></button>";
    }
    var html = "";
    var last = "";
    if (o.pinned.length) html += '<div class="nav-group">' + esc(t("nav.pinned")) + "</div>" + o.pinned.map(itemHTML).join("");
    o.rest.forEach(function (it) {
      if (it.group !== last) { html += '<div class="nav-group">' + esc(t(it.group)) + "</div>"; last = it.group; }
      html += itemHTML(it);
    });
    nav.innerHTML = html;
     $$("#nav [data-view]").forEach(function (a) {
      a.addEventListener("click", function () { switchView(a.getAttribute("data-view")); closeNav(); });
      a.classList.toggle("active", a.getAttribute("data-view") === state.view);
    });
    var filter = $("#navFilter");
    if (filter) {
      filter.addEventListener("input", function () { filterNav(filter.value); });
      filter.addEventListener("keydown", function (e) {
        if (e.key === "Escape") { filter.value = ""; filterNav(""); }
        if (e.key === "Enter") {
          var first = nav.querySelector('.nav-item:not([hidden])');
          if (first) { switchView(first.getAttribute("data-view")); closeNav(); }
        }
      });
    }
  }
  // 按名称筛选侧栏菜单，文案不匹配的项与空分组一并隐藏。
  function filterNav(kw) {
    kw = String(kw || "").trim().toLowerCase();
    var nav = $("#nav");
    if (!nav) return;
    var group = null, groupVisible = false;
    Array.prototype.forEach.call(nav.children, function (el) {
      if (el.classList.contains("nav-group")) {
        if (group && !groupVisible) group.hidden = true;
        group = el;
        groupVisible = false;
        el.hidden = false;
      } else if (el.classList.contains("nav-item")) {
        var hit = !kw || (el.textContent || "").toLowerCase().indexOf(kw) >= 0;
        el.hidden = !hit;
        if (hit) groupVisible = true;
      }
    });
    if (group && !groupVisible) group.hidden = true;
  }

  // 自定义侧栏弹窗：逐项调整顺序、置顶与显隐，改动即时预览并写入本机浏览器。
  function openNavCustomModal() {
    var prefs = navPrefs();
    var byId = {};
    NAV.forEach(function (x) { byId[x.id] = x; });
    function rowHTML(id, idx) {
      var it = byId[id];
      if (!it) return "";
      var pinned = prefs.pinned.indexOf(id) >= 0;
      var hidden = prefs.hidden.indexOf(id) >= 0;
      return '<div class="nav-custom-row' + (hidden ? " rl-dim" : "") + '">'
        + '<span class="ico">' + ico(it.ik) + '</span>'
        + '<span class="grow"><b>' + esc(t(it.name)) + '</b><small>' + esc(t(it.group)) + '</small></span>'
        + '<button type="button" class="btn btn-ghost btn-sm" data-act="pin" data-id="' + id + '">' + (pinned ? t("nav.custom.unpin") : t("nav.custom.pin")) + '</button>'
        + '<button type="button" class="btn btn-ghost btn-sm" data-act="up" data-id="' + id + '"' + (idx === 0 ? " disabled" : "") + ' aria-label="' + esc(t(it.name)) + '">↑</button>'
        + '<button type="button" class="btn btn-ghost btn-sm" data-act="down" data-id="' + id + '"' + (idx === prefs.order.length - 1 ? " disabled" : "") + ' aria-label="' + esc(t(it.name)) + '">↓</button>'
        + '<button type="button" class="btn btn-ghost btn-sm" data-act="hide" data-id="' + id + '">' + (hidden ? t("nav.custom.show") : t("nav.custom.hide")) + '</button>'
        + '</div>';
    }
    function draw() {
      var body = $("#modalBody");
      if (!body) return;
      body.innerHTML = '<p class="hint" style="margin:0 0 10px">' + esc(t("nav.custom.hint")) + "</p>"
        + '<div class="nav-custom-list">' + prefs.order.map(rowHTML).join("") + '</div>';
      $$("#modalBody [data-act]").forEach(function (b) {
        b.addEventListener("click", function () {
          applyNavAction(prefs, b.getAttribute("data-act"), b.getAttribute("data-id"));
          saveNavPrefs(prefs);
          renderNav();
          draw();
        });
      });
    }
    showModal(t("nav.custom.title"), "", '<button class="btn btn-ghost" id="navReset">' + esc(t("nav.custom.reset")) + '</button><button class="btn btn-primary" onclick="Relay.closeModal()">' + esc(t("nav.custom.done")) + "</button>");
    draw();
    var rb = $("#navReset");
    if (rb) rb.addEventListener("click", function () {
      prefs = defaultNavPrefs();
      saveNavPrefs(prefs);
      renderNav();
      draw();
      toast("已恢复默认侧栏", "ok");
    });
  }
  function applyNavAction(prefs, act, id) {
    var i = prefs.order.indexOf(id);
    if (i < 0) return;
    if (act === "up" && i > 0) {
      prefs.order.splice(i, 1);
      prefs.order.splice(i - 1, 0, id);
    } else if (act === "down" && i < prefs.order.length - 1) {
      prefs.order.splice(i, 1);
      prefs.order.splice(i + 1, 0, id);
    } else if (act === "pin") {
      var pi = prefs.pinned.indexOf(id);
      if (pi >= 0) prefs.pinned.splice(pi, 1); else prefs.pinned.push(id);
    } else if (act === "hide") {
      var hi = prefs.hidden.indexOf(id);
      if (hi >= 0) prefs.hidden.splice(hi, 1); else prefs.hidden.push(id);
    }
  }

  function isMobile() { return window.innerWidth <= 860; }
  function toggleNav() {
    var sb = $("#sidebar");
    if (!sb) return;
    if (isMobile()) {
      sb.classList.remove("collapsed");
      sb.classList.toggle("open");
      var bd = $("#navBackdrop");
      var open = sb.classList.contains("open");
      if (bd) bd.classList.toggle("show", open);
      document.body.classList.toggle("nav-open", open);
    } else {
      sb.classList.toggle("collapsed");
    }
    var toggle = $("#navToggle");
    if (toggle) toggle.setAttribute("aria-expanded", sb.classList.contains("open") || !sb.classList.contains("collapsed") ? "true" : "false");
  }
  function closeNav() {
    var sb = $("#sidebar");
    if (sb) sb.classList.remove("open");
    var bd = $("#navBackdrop");
    if (bd) bd.classList.remove("show");
    document.body.classList.remove("nav-open");
    var toggle = $("#navToggle");
    if (toggle && isMobile()) toggle.setAttribute("aria-expanded", "false");
  }

  var PAGE_CODE = {
    home: "OVERVIEW / 01", connect: "ACCESS / 02", chat: "MESSAGES / 03", pool: "POOL / 04", dingtalk: "DINGTALK / 04B",
    roles: "ROLES / 05", scripts: "PLUGINS / 06", market: "MARKET / 07", features: "FEATURES / 08",
    agent: "AGENT / 09", cron: "SCHEDULE / 10", executions: "RUNS / 11", tasks: "TASKS / 12",
    audit: "AUDIT / 13", observe: "METRICS / 14", logs: "LOGS / 15"
  };
  // 为没有显式眉标的页面补一条等宽编号，保证 14 个页面版式一致。
  function decoratePage(id) {
    var view = document.getElementById("view-" + id);
    if (!view) return;
    var head = view.querySelector(".page-head");
    if (!head || head.querySelector(".eyebrow")) return;
    var box = head.querySelector("div");
    if (!box) return;
    var span = document.createElement("span");
    span.className = "eyebrow";
    span.textContent = PAGE_CODE[id] || "RELAY";
    box.insertBefore(span, box.firstChild);
  }

  function switchView(id) {
    var target = NAV.filter(function (x) { return x.id === id; })[0];
    if (target && !roleAllowed(target.minRole)) {
      toast("当前角色无权查看该页面", "err");
      id = "home";
    }
    if (renderers[id]) renderers[id]();
    decoratePage(id);
    if (LANG === "en") i18nLocalize(document.getElementById("view-" + id) || document.body);
     $$("#nav [data-view]").forEach(function (a) { a.classList.toggle("active", a.getAttribute("data-view") === id); });
    $$(".page").forEach(function (p) { p.hidden = p.id !== "view-" + id; });
    state.view = id;
    rememberView(id);
    var n = NAV.filter(function (x) { return x.id === id; })[0];
    $("#crumbs").textContent = n ? "Relay · " + t(n.name) : "Relay";
    clearInterval(state.interval);
    state.interval = null;
    if (id !== "logs") stopLogAuto();
    if (id === "chat" && state.conv) startChatTimer();
    else if (POLL[id]) state.interval = setInterval(POLL[id], POLL_MS[id] || 10000);
    hydrateIcons();
    hintScroll();
  }

  // 窄屏可横向滚动的表格加右缘提示，避免用户误以为内容被裁掉
  function hintScroll() {
    $$(".table-wrap").forEach(function (w) {
      var more = w.scrollWidth - w.clientWidth - w.scrollLeft > 1;
      w.classList.toggle("scroll-x", more);
      if (!w.dataset.hinted) {
        w.dataset.hinted = "1";
        w.addEventListener("scroll", hintScroll);
      }
    });
  }

  function cacheProfile(st) {
    if (!st) return;
    try {
      var old = readProfileCache() || {};
      var email = (st.account && st.account.email) || old.email || "";
      var avatar = (st.account && st.account.avatar_url) || "";
      // 资料接口偶发返回空头像时沿用同账号的旧缓存，避免顶栏头像被空值覆盖。
      if (!avatar && old.email === email && old.avatar) avatar = old.avatar;
      localStorage.setItem(PROFILE_KEY, JSON.stringify({
        name: (st.account && (st.account.nickname || st.account.email)) || (old.email === email ? old.name : "") || "",
        email: email,
        avatar: avatar,
        app_name: (st.app && st.app.name) || "",
        app_avatar: (st.app && st.app.avatar_url) || "",
        at: Date.now()
      }));
    } catch (e) { /* 隐私模式下忽略 */ }
  }
  function readProfileCache() {
    try { return JSON.parse(localStorage.getItem(PROFILE_KEY) || "null"); } catch (e) { return null; }
  }
  // 先用本机缓存渲染顶栏，刷新或网络抖动时头像不会闪烁成默认图标。
  function applyCachedProfile() {
    var cached = readProfileCache();
    if (!cached) return;
    var av = $("#avatar");
    if (av && (cached.avatar || cached.name || cached.email)) {
      var label = cached.name || cached.email || "U";
      av.className = "avatar top-avatar";
      av.innerHTML = avatarInner(cached.avatar, label);
      av.dataset.sig = (cached.avatar || "") + "|" + label;
      av.title = label;
    }
    var un = $("#userName");
    if (un && cached.name && un.textContent === "管理员") un.textContent = cached.name;
  }
  function renderTopAvatar(url, name) {
    var av = $("#avatar");
    if (!av) return;
    var label = name || "U";
    var sig = (url || "") + "|" + label;
    if (av.dataset.sig === sig && av.querySelector("img, .av-fb")) return;
    av.dataset.sig = sig;
    av.className = "avatar top-avatar";
    av.innerHTML = avatarInner(url, label);
    av.title = label;
  }
  function refreshConnUI(st) {
    if (!st) return;
    state.account = st.account || null;
    state.app = st.app || null;
    state.apps = st.apps || [];
    cacheProfile(st);
    var wsOk = st.app && st.app.connected;
    var acctOk = st.account && st.account.logged_in;
    if (mascotState.ws === false && wsOk) mascotReact("ok", "应用通道已连接");
    if (mascotState.acct === false && acctOk) mascotReact("ok", "账号通道已登录");
    mascotState.ws = !!wsOk;
    mascotState.acct = !!acctOk;
    var acctName = acctOk ? (st.account.nickname || st.account.email) : "";
    var dWs = $("#dotWs"), dAcct = $("#dotAcct");
    if (dWs) { dWs.className = "dot " + (wsOk ? "on" : "off"); $("#sideWs").textContent = t("side.ws") + " " + (wsOk ? t("side.connected") : t("side.disconnected")); }
    if (dAcct) { dAcct.className = "dot " + (acctOk ? "on" : "off"); $("#sideAcct").textContent = t("side.acct") + " " + (acctOk ? acctName : t("side.notlogged")); }
    var cw = $("#chipWs"), ca = $("#chipAcct");
    if (cw) {
      cw.className = "chip " + (wsOk ? "chip-success" : "chip-danger");
      cw.textContent = wsOk ? (st.app && st.app.name ? st.app.name : t("chip.app.on")) : t("chip.app.off");
      cw.title = wsOk && st.app && st.app.name ? st.app.name : "";
    }
    if (ca) { ca.className = "chip " + (acctOk ? "chip-success" : "chip-danger"); if (acctName) ca.textContent = acctName; else ca.textContent = t("chip.acct.off"); }
    // 已登录时以服务端资料为准；服务端暂时没给头像时沿用本机缓存，
    // 避免状态轮询的瞬时空值把顶栏头像覆盖成默认图标。
    var cached = readProfileCache();
    if (acctOk && (st.account.avatar_url || acctName)) {
      renderTopAvatar(st.account.avatar_url || (cached && cached.avatar) || "", acctName || "U");
    } else if (cached && (cached.avatar || cached.name || cached.email)) {
      renderTopAvatar(cached.avatar || "", cached.name || cached.email || "U");
    }
    var av = $("#avatar");
    if (av) av.title = acctOk ? (acctName + (st.account.email && st.account.nickname ? " · " + st.account.email : "")) : "未登录账号";
    var un = $("#userName");
    if (un && acctOk && acctName) un.textContent = acctName;
  }

  function loadConnectStatus(cb) {
    api("/jiying/api/connect/status").then(function (r) {
      if (r.code === 200) refreshConnUI(r.data);
      if (cb) cb(r);
    }).catch(function () { if (cb) cb({ code: 0 }); });
  }

  /* ---------- 工作台 ---------- */
  function renderHome() {
    var cont = $("#view-home");
    cont.innerHTML = '<div class="page-head"><div><span class="eyebrow">OVERVIEW / 01</span><h2>工作台</h2><p>即应接入与任务调度实时总览</p></div><div class="spacer"></div><button class="btn btn-ghost btn-sm" onclick="Relay.refreshHome()">刷新</button></div>'
      + '<div class="stat-strip">'
      + statCard("今日收", "-", "g-1", "inbox")
      + statCard("今日发", "-", "g-2", "send")
      + statCard("会话", "-", "g-3", "users")
      + statCard("运行时长", "-", "g-4", "clock")
      + statCard("功能", "-", "g-1", "zap")
      + statCard("定时", "-", "g-2", "repeat")
      + statCard("任务", "-", "g-3", "list")
      + statCard("插件", "-", "g-5", "layers")
      + "</div>"
      + '<div class="grid grid-2 mt-16">'
      + '<div class="panel"><div class="panel-head"><h3>接入状态</h3><span class="hint"><a href="javascript:Relay.go(\'connect\')">connect →</a></span></div><div class="panel-body" id="hcState"></div></div>'
      + '<div class="panel"><div class="panel-head"><h3>快捷入口</h3><span class="hint">4 个常用动作</span></div><div class="panel-body">'
      + quick("chat", "消息收发", "会话列表与发送", "message", "01")
      + quick("scripts", "插件", "安装与调试插件", "puzzle", "02")
      + quick("cron", "定时任务", "创建定时调度", "clock", "03")
      + quick("tasks", "任务队列", "执行记录与重试", "list", "04")
        + "</div></div>"
        + "</div>"
      + '<div class="panel mt-16"><div class="panel-head"><h3>接入检查</h3><span class="hint">首次使用按此顺序逐项自检</span></div><div class="panel-body" id="hcReady">' + loadingState() + '</div></div>';
    Relay.refreshHome();
  }

  function statCard(label, val, cls, ik) {
    return '<div class="stat-cell"><small>' + label + '<span class="ico ' + cls + '" data-ico="' + ik + '">' + ico(ik) + '</span></small><b id="' + label + '">' + val + "</b></div>";
  }
  function quick(id, name, desc, ik, no) {
    return '<div class="list-item cmd-row" onclick="Relay.go(\'' + id + '\')"><span class="cmd-no">' + (no || "·") + '</span><div class="cmd-ico" data-ico="' + ik + '">' + ico(ik) + '</div><div class="grow"><div class="name">' + name + '</div><div class="desc">' + desc + '</div></div><div class="actions">→</div></div>';
  }

  // 接入检查清单：把就绪检查前置到首页，并给出可点击的修复入口。
  var READY_LABEL = { ready: "正常", connected: "已连接", configured: "已配置未连接", invalid: "配置有误", missing: "未配置", blocked: "不可用" };
  function readyOK(check) {
    var s = (check && check.status) || "missing";
    return s === "ready" || s === "connected";
  }
  function readinessPanel(r, opts) {
    opts = opts || {};
    r = r || {};
    var rows = [
      { name: "1. 本地存储", check: r.storage || {}, fix: "" },
      { name: "2. 应用通道", check: r.app || {}, fix: "pool" },
      { name: "3. 账号通道", check: r.account || {}, fix: "connect" },
      { name: "4. 模型 Agent", check: r.agent || {}, fix: "agent" }
    ];
    var bad = 0;
    var html = rows.map(function (row) {
      var ok = readyOK(row.check);
      if (!ok) bad += 1;
      var issues = (row.check.issues || []).filter(function (x) { return x && (x.message || x.hint); });
      var text = issues.length
        ? issues.map(function (x) { return (x.message || "") + (x.hint ? " · " + x.hint : ""); }).join("；")
        : (READY_LABEL[row.check.status] || "未知状态");
      var fixLabel = ok ? "查看" : "去修复";
      var fix = (row.fix && !opts.noFix) ? '<button class="btn btn-ghost btn-sm" onclick="Relay.go(\'' + row.fix + '\')">' + fixLabel + "</button>" : "";
      return '<div class="conn-state mt-8 ' + (ok ? "on" : "off") + '"><span class="dot ' + (ok ? "on" : "off") + '"></span><div class="grow"><b>' + row.name + '</b><div class="muted" style="font-size:12px">' + esc(text) + '</div></div>' + fix + "</div>";
    }).join("");
    if (opts.onlyIssues && bad === 0) return "";
    return '<div class="hint mb-8">' + (bad === 0 ? "全部就绪，可正常接收与回复消息" : bad + " 项待处理") + "</div>" + html;
  }
  function setReady(r) {
    var box = $("#hcReady");
    if (box) box.innerHTML = readinessPanel(r);
  }

  function refreshHome(silent) {
    api("/jiying/api/overview").then(function (r) {
      if (r.code !== 200) { if (!silent) toast(r.msg || "加载失败", "err"); return; }
      var d = r.data;
      setText("#今日收", d.msg_in); setText("#今日发", d.msg_out); setText("#会话", d.conv_cnt);
      setText("#运行时长", d.uptime); setText("#功能", d.features); setText("#定时", d.crons);
      setText("#任务", d.tasks); setText("#插件", d.plugins);
      syncUptime(d.uptime_seconds);
      setReady(d.readiness);
      var ws = String(d.ws).indexOf("已连接") >= 0;
      var acct = String(d.account).indexOf("已登录") >= 0;
      $("#hcState").innerHTML = '<div class="rl-row"><div class="grow">'
        + '<div class="kv"><span class="k">应用通道</span><span class="v">' + statusChip(ws, "已连接", d.ws) + "</span></div>"
        + '<div class="kv"><span class="k">账号通道</span><span class="v">' + statusChip(acct, d.account, d.account) + "</span></div>"
        + '<div class="kv"><span class="k">应用 ID</span><span class="v"><code>' + esc(d.app_id || "-") + "</code></span></div>"
        + '<div class="kv"><span class="k">服务器时间</span><span class="v">' + esc(d.time) + "</span></div>"
        + '<div class="kv"><span class="k">刷新周期</span><span class="v">10 秒自动刷新</span></div>'
        + "</div></div>";
    }).catch(function () { if (!silent) toast("接口请求失败", "err"); });
  }
  function setText(sel, v) { var el = $(sel); if (el) el.textContent = v; }

  // 运行时长本地 1s 递增：以服务端返回的秒数为基准，避免每秒打接口。
  function syncUptime(sec) {
    if (sec == null || isNaN(sec)) return;
    state.baseUptime = Number(sec);
    state.uptimeAt = Date.now();
    startUptimeTick();
  }
  function uptimeSeconds() {
    if (state.baseUptime == null) return null;
    return state.baseUptime + Math.floor((Date.now() - state.uptimeAt) / 1000);
  }
  function startUptimeTick() {
    if (state.uptimeTimer) return;
    state.uptimeTimer = setInterval(function () {
      var s = uptimeSeconds();
      if (s == null) return;
      setText("#运行时长", fmtDur(s));
      var tiles = $$("#observeTiles .tile");
      if (tiles[1]) {
        var v = tiles[1].querySelector(".t-val");
        if (v) v.textContent = fmtDur(s);
      }
    }, 1000);
  }

  /* ---------- 接入系统 ---------- */
  var CONN_TAB = "app";
  function renderConnect() {
    var cont = $("#view-connect");
    cont.innerHTML = '<div class="page-head"><div><span class="eyebrow">ACCESS / 02</span><h2>接入系统</h2><p>推荐顺序：先配「应用密钥」建立应用通道，再用「账号密码 / 邮箱验证码」授权账号；完成后在 <a href="javascript:Relay.go(\'home\')">工作台 · 接入检查</a> 逐项确认。</p></div></div>'
      + '<div class="grid-aside">'
      + '<div class="panel"><div class="panel-head"><h3>接入状态</h3><span class="hint">1 应用密钥 → 2 账号授权 → 3 模型 Agent</span></div><div class="panel-body" id="connStatus"></div></div>'
      + '<div class="panel"><div class="panel-body">'
      + '<div class="card-tabs" id="connTabs">'
      + '<button data-tab="app">应用密钥</button><button data-tab="pw">账号密码</button><button data-tab="code">邮箱验证码</button>'
      + "</div><div id=" + '"connForm"></div>'
      + "</div></div>"
      + "</div>";
    $$("#connTabs button").forEach(function (b) {
      b.addEventListener("click", function () { setConnTab(b.getAttribute("data-tab")); });
    });
    setConnTab(CONN_TAB);
    loadConnectStatus();
  }

  function setConnTab(tab) {
    CONN_TAB = tab;
    $$("#connTabs button").forEach(function (b) { b.classList.toggle("active", b.getAttribute("data-tab") === tab); });
    var form = $("#connForm");
    if (!form) return;
    loadConnectStatus(function (st) {
      renderConnStatus(st);
      var app = st && st.data ? st.data.app : null;
      var acct = st && st.data ? st.data.account : null;
      var html = "";
      if (tab === "app") {
        html = connAppForm(app);
      } else if (tab === "pw") {
        html = connPwForm(acct);
      } else {
        html = connCodeForm(acct);
      }
      form.innerHTML = html;
      bindConnForms();
    });
  }

  function connAppForm(app) {
    var ws = app ? app.ws_url || "" : "";
    var id = app ? app.app_id || "" : "";
    var secret = app && app.configured ? "****" : "";
    var hint = app && app.configured ? '<div class="muted mb-8" style="font-size:12px">已保存的地址以掩码显示，保持掩码或留空表示不修改。</div>' : "";
    return '<div class="field"><label>WebSocket 接入地址</label><input class="input" id="cf_ws" placeholder="wss://<host>/api/app/ws" value="' + esc(ws) + '"></div>'
      + '<div class="field"><label>应用 ID</label><input class="input" id="cf_id" placeholder="8-4-4-4-12 的 UUID，从应用详情复制" value="' + esc(id) + '"></div>'
      + '<div class="muted mb-8" style="font-size:12px">应用 ID 必须是 UUID，与账号通道指向同一个即应服务；不一致会导致应用握手 400。</div>'
      + '<div class="field"><label>应用密钥</label><input class="input" id="cf_secret" type="password" placeholder="连接密钥" value="' + esc(secret) + '"></div>'
      + hint
       + '<div class="input-row"><button class="btn btn-primary btn-lg" id="cf_save">保存并连接</button><button class="btn btn-ghost btn-lg" id="cf_test">测试连接</button></div>';
  }
  function connPwForm(acct) {
    var base = acct ? acct.base_url || "" : "";
    var email = acct ? acct.email || "" : "";
    var secret = acct && acct.configured ? "****" : "";
    var hint = acct && acct.configured ? '<div class="muted mb-8" style="font-size:12px">已保存的地址以掩码显示，保持掩码或留空表示不修改。</div>' : "";
    return '<div class="field"><label>即应服务地址</label><input class="input" id="cf_base" placeholder="https://<host> 或 http://127.0.0.1:20080" value="' + esc(base) + '"></div>'
      + '<div class="field"><label>邮箱账号</label><input class="input" id="cf_email" placeholder="user@example.com" value="' + esc(email) + '"></div>'
      + '<div class="field"><label>密码</label><input class="input" id="cf_pw" type="password" value="' + esc(secret) + '"></div>'
      + hint
      + '<button class="btn btn-primary btn-lg" id="cf_save">登录并接入</button>';
  }
  function connCodeForm(acct) {
    var base = acct ? acct.base_url || "" : "";
    var email = acct ? acct.email || "" : "";
    var hint = acct && acct.configured ? '<div class="muted mb-8" style="font-size:12px">已保存的地址以掩码显示，保持掩码或留空表示不修改。</div>' : "";
    return '<div class="field"><label>即应服务地址</label><input class="input" id="cf_base" placeholder="https://<host> 或 http://127.0.0.1:20080" value="' + esc(base) + '"></div>'
      + '<div class="field"><label>邮箱账号</label><input class="input" id="cf_email" placeholder="user@example.com" value="' + esc(email) + '"></div>'
      + '<div class="input-row"><input class="input" id="cf_code" placeholder="8 位邮箱验证码"><button class="btn btn-ghost" id="cf_sendcode">发送验证码</button></div>'
      + '<div class="muted mb-8" style="font-size:12px" id="cf_codetip">验证码 15 分钟内有效</div>'
      + hint
      + '<button class="btn btn-primary btn-lg" id="cf_save">验证并接入</button>';
  }

  function bindConnForms() {
    var save = $("#cf_save");
    if (!save) return;
    save.addEventListener("click", function () {
      if (CONN_TAB === "app") {
        api("/jiying/api/connect/app", { method: "POST", form: { ws_url: $("#cf_ws").value.trim(), app_id: $("#cf_id").value.trim(), app_secret: $("#cf_secret").value } }).then(function (r) {
          if (r.code === 200) { toast("应用通道已更新", "ok"); CONN_TAB = "app"; setConnTab("app"); }
          else toast(r.msg || "配置失败", "err");
        });
      } else if (CONN_TAB === "pw") {
        api("/jiying/api/connect/account", { method: "POST", form: { base_url: $("#cf_base").value.trim(), email: $("#cf_email").value.trim(), password: $("#cf_pw").value } }).then(function (r) {
          if (r.code === 200) { toast("账号登录成功", "ok"); CONN_TAB = "pw"; setConnTab("pw"); }
          else toast(r.msg || "登录失败", "err");
        });
      } else {
        api("/jiying/api/connect/login-code", { method: "POST", form: { base_url: $("#cf_base").value.trim(), email: $("#cf_email").value.trim(), code: $("#cf_code").value.trim() } }).then(function (r) {
          if (r.code === 200) { toast("账号验证并登录成功", "ok"); CONN_TAB = "code"; setConnTab("code"); }
          else toast(r.msg || "登录失败", "err");
        });
      }
    });
    var test = $("#cf_test");
    if (test) test.addEventListener("click", function () {
      api("/jiying/api/connect/test-app", { method: "POST", form: { ws_url: $("#cf_ws").value.trim(), app_id: $("#cf_id").value.trim(), app_secret: $("#cf_secret").value } }).then(function (r) {
        if (r.code === 200) toast("应用握手成功", "ok");
        else toast(r.msg || "连接测试失败", "err");
      });
    });
    var send = $("#cf_sendcode");
    if (send) send.addEventListener("click", function () {
      var base = $("#cf_base").value.trim(), email = $("#cf_email").value.trim();
      if (!base || !email) { toast("请先填写服务地址与邮箱", "err"); return; }
      api("/jiying/api/connect/request-code", { method: "POST", form: { base_url: base, email: email } }).then(function (r) {
        if (r.code === 200) { toast("验证码已发送", "ok"); var t = $("#cf_codetip"); if (t) t.textContent = "已验证码发送，有效期 " + r.data.expires_in_seconds + " 秒"; }
        else toast(r.msg || "发送失败", "err");
      });
    });
  }

  var ACCOUNT_HEALTH = {
    valid: { label: "会话有效", cls: "badge-ok" },
    expired: { label: "会话已失效", cls: "badge-err" },
    unknown: { label: "状态未知", cls: "badge-warn" },
    unconfigured: { label: "未配置账号", cls: "badge-off" }
  };
  function accountHealthView(h, acct) {
    var key = h.status || (acct.logged_in ? "valid" : "unconfigured");
    if (!acct.configured) key = "unconfigured";
    else if (!acct.logged_in && key === "valid") key = "expired";
    var meta = ACCOUNT_HEALTH[key] || ACCOUNT_HEALTH.unknown;
    var bits = [];
    if (h.last_ok) bits.push("最近成功 " + h.last_ok);
    if (h.checked_at) bits.push("检查于 " + h.checked_at);
    if (!bits.length) {
      if (key === "unconfigured") bits.push("配置账号后可检查登录会话");
      else bits.push(acct.code_login ? "该账号使用邮箱验证码登录" : "点击「立即检查」确认会话状态");
    }
    return { key: key, badge: '<span class="badge ' + meta.cls + '">' + meta.label + "</span>", detail: bits.join(" · ") };
  }
  function checkAccountHealth() {
    api("/jiying/api/connect/account/health/check", { method: "POST", form: {} }).then(function (r) {
      if (r.code === 200 && r.data) {
        var key = r.data.status || "unknown";
        var meta = ACCOUNT_HEALTH[key] || ACCOUNT_HEALTH.unknown;
        toast("账号会话：" + meta.label + (r.data.message ? " · " + r.data.message : ""), key === "valid" ? "ok" : "err");
        loadConnectStatus(function (st) { renderConnStatus(st); });
      } else toast(r.msg || "检查失败", "err");
    });
  }
  function reloginAccount() {
    var codeLogin = !!(state.account && state.account.code_login);
    setConnTab(codeLogin ? "code" : "pw");
    toast(codeLogin ? "请重新获取验证码并登录" : "请重新输入密码并登录", "ok");
  }
  function logoutAccount() {
    confirmDialog("退出账号", "退出后账号通道与自动发送将停止，确认退出？", "退出", function () {
      api("/jiying/api/connect/account/logout", { method: "POST", form: {} }).then(function (r) {
        if (r.code === 200) { toast("已退出账号", "ok"); loadConnectStatus(function (st) { renderConnStatus(st); }); }
        else toast(r.msg || "退出失败", "err");
      });
    });
  }

  // 应用通道头像堆叠：默认应用优先，最多展示 4 个，其余用 +N 表示。
  function connAppAvatars(apps) {
    apps = apps || [];
    if (!apps.length) return "";
    var sorted = apps.slice().sort(function (a, b) { return (b.is_default ? 1 : 0) - (a.is_default ? 1 : 0); });
    var head = sorted.slice(0, 4).map(function (a) {
      return '<span class="conn-av" title="' + esc((a.name || a.app_id) + (a.online ? " · 已连接" : " · 未连接")) + '">' + avatarHTML(a.avatar_url, a.name || a.app_id, "conn-avatar") + "</span>";
    }).join("");
    if (sorted.length > 4) head += '<span class="conn-av-more">+' + (sorted.length - 4) + "</span>";
    return '<span class="conn-av-stack">' + head + "</span>";
  }
  function connAppList(apps) {
    apps = apps || [];
    if (!apps.length) return "";
    return '<div class="conn-app-list">' + apps.map(function (a) {
      var nm = a.name || shortID(a.app_id);
      return '<span class="app-chip" title="' + esc(a.app_id) + '">'
        + '<span class="app-chip-av">' + avatarInner(a.avatar_url, nm) + "</span>"
        + esc(nm)
        + '<span class="dot ' + (a.online ? "on" : "off") + '" style="width:5px;height:5px"></span>'
        + (a.is_default ? '<span class="muted" style="font-size:10px">默认</span>' : "")
        + "</span>";
    }).join("") + "</div>";
  }
  function connAccountList(accounts, activeEmail) {
    accounts = accounts || [];
    if (!accounts.length) {
      return '<div class="muted" style="font-size:12px">还没有账号，先在右侧用密码或邮箱验证码登录。</div>';
    }
    return '<div class="conn-acct-list" id="connAccounts">' + accounts.map(function (a) {
      var nm = a.nickname || a.email;
      var active = a.active || a.email === activeEmail;
      return '<div class="conn-acct-row' + (active ? " active" : "") + '" data-email="' + esc(a.email) + '">'
        + avatarHTML(a.avatar_url, nm, "conn-avatar")
        + '<div class="conn-acct-main"><b title="' + esc(nm) + '">' + esc(nm) + '</b><small title="' + esc(a.email) + '">' + esc(a.email) + "</small></div>"
        + '<span class="badge ' + (a.logged_in ? "badge-ok" : "badge-off") + '">' + (a.logged_in ? "已登录" : "未登录") + "</span>"
        + (active ? '<span class="badge badge-info">当前</span>'
          : '<button type="button" class="btn btn-ghost btn-sm" data-acct-switch="' + esc(a.email) + '">切换</button>'
          + '<button type="button" class="btn btn-outline-danger btn-sm" data-acct-del="' + esc(a.email) + '">移除</button>')
        + "</div>";
    }).join("") + "</div>";
  }
  function switchAccount(email) {
    if (!email) return;
    toast("正在切换到 " + email, "info");
    api("/jiying/api/connections/account/select", { method: "POST", form: { email: email } }).then(function (r) {
      if (r.code === 200) {
        toast("已切换账号", "ok");
        loadConnectStatus(function (st) { renderConnStatus(st); });
      } else toast(r.msg || "账号切换失败", "err");
    });
  }
  function removeAccount(email) {
    confirmDialog("移除账号", "从连接池移除 " + email + "？已保存的登录令牌会一并清除。", "移除", function () {
      api("/jiying/api/connections/account/delete", { method: "POST", form: { email: email } }).then(function (r) {
        if (r.code === 200) {
          toast("已移除账号", "ok");
          loadConnectStatus(function (st) { renderConnStatus(st); });
        } else toast(r.msg || "移除失败", "err");
      });
    });
  }

  function renderConnStatus(st) {
    var box = $("#connStatus");
    if (!box) return;
    var d = st && st.data ? st.data : null;
    if (!d) { errorState(box, "link", "接入状态加载失败", "网络异常，请检查服务状态。", function () { loadConnectStatus(); }); return; }
    refreshConnUI(d);
    var apps = d.apps || [];
    var accounts = d.accounts || [];
    var appOn = d.app.connected, acctOn = d.account.logged_in;
    var readiness = d.readiness || {};
    // 接入页两个通道都可直接编辑：仅在存在待处理项时提示，且不重复给出跳转按钮。
    var readyBlock = readinessPanel(readiness, { noFix: true, onlyIssues: true });
    var readinessHTML = readyBlock ? '<div class="mt-16 mb-8"><b>待处理项</b></div>' + readyBlock : "";
    state.account = d.account;
    var h = d.account.health || {};
    var health = accountHealthView(h, d.account);
    box.innerHTML = '<div class="conn-state ' + (appOn ? "on" : "off") + '"><span class="dot ' + (appOn ? "on" : "off") + '"></span><div class="conn-main">'
      + '<div class="conn-account-head">' + connAppAvatars(apps) + "<b>应用通道</b>"
      + (apps.length ? '<span class="badge badge-off">' + apps.length + " 个应用</span>" : "")
      + '<a href="javascript:Relay.go(\'pool\')" class="hint" style="margin-left:auto">管理应用 →</a>'
      + "</div>"
      + '<div class="muted" style="font-size:12px">' + esc((appOn ? "已连接 · " : "未连接") + (d.app.name ? d.app.name + " · " : "") + (d.app.ws_url || "")) + "</div>"
      + connAppList(apps)
      + "</div></div>"
      + '<div class="conn-state mt-8 ' + (acctOn ? "on" : "off") + '"><span class="dot ' + (acctOn ? "on" : "off") + '"></span><div class="conn-main"><div class="conn-account-head"><b>账号通道</b>'
      + (accounts.length ? '<span class="badge badge-off">' + accounts.length + " 个账号</span>" : "")
      + "</div>"
      + '<div class="muted" style="font-size:12px">' + esc(acctOn ? (d.account.nickname ? d.account.nickname + " · " + d.account.email : d.account.email) : d.account.status || "未登录") + "</div>"
      + connAccountList(accounts, d.account.email)
      + "</div></div>"
      + '<div class="health-card mt-12"><div class="health-head"><b>账号会话</b>' + health.badge + '</div>'
      + '<div class="muted" style="font-size:12px">' + esc(health.detail) + "</div>"
      + (h.message && health.key !== "unconfigured" ? '<div class="muted clamp2" style="font-size:12px;color:var(--err);margin-top:4px" title="' + esc(h.message) + '">' + esc(h.message) + "</div>" : "")
      + '<div class="input-row mt-8">'
      + '<button class="btn btn-ghost btn-sm" id="acctHealthCheck"' + (d.account.configured ? "" : " disabled") + '>立即检查</button>'
      + (d.account.configured ? '<button class="btn btn-ghost btn-sm" id="acctRelogin">重新登录</button>' : "")
      + (acctOn ? '<button class="btn btn-outline-danger btn-sm" id="acctLogout">退出账号</button>' : "")
      + "</div></div>"
      + readinessHTML
      + '<div class="kv mt-16"><span class="k">应用 ID</span><span class="v"><code>' + esc(d.app.app_id || "-") + "</code></span></div>"
      + '<div class="kv"><span class="k">接入状态</span><span class="v">' + esc(d.account.status || "未登录") + "</span></div>"
      + '<div class="kv"><span class="k">运行时长</span><span class="v">' + esc(d.runtime.uptime) + "</span></div>"
      + '<div class="kv"><span class="k">服务器时间</span><span class="v">' + esc(d.runtime.time) + "</span></div>";
    var hc = $("#acctHealthCheck");
    if (hc) hc.addEventListener("click", checkAccountHealth);
    var rl = $("#acctRelogin");
    if (rl) rl.addEventListener("click", reloginAccount);
    var lo = $("#acctLogout");
    if (lo) lo.addEventListener("click", logoutAccount);
    $$("#connAccounts [data-acct-switch]").forEach(function (b) {
      b.addEventListener("click", function () { switchAccount(b.getAttribute("data-acct-switch")); });
    });
    $$("#connAccounts [data-acct-del]").forEach(function (b) {
      b.addEventListener("click", function () { removeAccount(b.getAttribute("data-acct-del")); });
    });
  }

  /* ---------- 消息收发 ---------- */
  function startChatTimer() {
    clearInterval(state.interval);
    state.interval = setInterval(function () { if (state.conv) loadChat(); }, 8000);
  }
  function renderChat() {
    // 进入消息页时保留上次选中的会话，状态轮询和切换页面不再把对话框清空。
    state.chatEpoch += 1;
    state.chatRequest += 1;
    var keepID = state.chatConvID || (state.conv && state.conv.id) || "";
    state.convs = []; state.conv = null; state.msgs = [];
    var cont = $("#view-chat");
    cont.innerHTML = '<div class="page-head"><div><span class="eyebrow">MESSAGES / 03</span><h2>消息收发</h2><p>选择会话查看消息并发送内容</p></div></div>'
      + '<div class="chat-grid">'
      + '<div class="panel"><div class="panel-head"><h3>会话</h3><select class="select" id="convAppFilter" aria-label="' + esc(t("chat.filter.title")) + '" style="max-width:158px;margin-left:auto"></select></div><div class="panel-body" id="convList">' + loadingState() + '</div></div>'
      + '<div class="panel" style="display:flex;flex-direction:column"><div class="panel-head"><h3>消息</h3><span class="hint" id="chatTitle"></span><select class="select" id="chatAccountSelect" style="max-width:220px;margin-left:auto"><option>加载账号中…</option></select></div>'
      + '<div class="panel-body" style="flex:1;display:flex;flex-direction:column"><div class="chat-stream" id="chatStream">' + emptyState("message", "选择一个会话", "从会话列表中选择，即可查看历史消息并发送内容。") + "</div>"
      + '<div class="muted" id="chatCap" style="font-size:12px;margin-top:8px">选择会话后显示可用发送身份</div>'
      + '<div class="input-row mt-8"><select class="select" id="chatIdentity" style="width:126px" title="发送身份"><option value="auto">身份 · 自动</option><option value="app">身份 · 应用</option><option value="user">身份 · 用户</option></select><input class="input" id="chatInput" placeholder="输入消息，回车或点击发送"><button class="btn btn-ghost" id="chatPreview">预演</button><button class="btn btn-primary" id="chatSend">发送</button></div>'
      + "</div></div></div>";
    loadConvs(keepID);
    loadChatAccounts();
    $("#chatSend").addEventListener("click", function () { sendChat(false); });
    $("#chatPreview").addEventListener("click", function () { sendChat(true); });
    $("#chatInput").addEventListener("keydown", function (e) { if (e.key === "Enter") sendChat(false); });
  }

  function loadChatAccounts() {
    var epoch = state.chatEpoch;
    api("/jiying/api/connections").then(function (r) {
      if (epoch !== state.chatEpoch) return;
      var select = $("#chatAccountSelect");
      if (!select) return;
      if (r.code !== 200) { select.innerHTML = '<option value="">账号状态加载失败，点「刷新」重试</option>'; return; }
      var accounts = r.data && r.data.accounts ? r.data.accounts : [];
      state.apps = r.data && r.data.apps ? r.data.apps : [];
      paintConvFilter();
      if (!accounts.length) {
        select.innerHTML = '<option value="">请先在接入系统登录账号</option>';
        return;
      }
      select.innerHTML = accounts.map(function (a) {
        return '<option value="' + esc(a.email) + '"' + (a.active ? " selected" : "") + '>' + esc(a.email) + (a.logged_in ? " · 已登录" : "") + "</option>";
      }).join("");
      select.addEventListener("change", function () {
        if (!select.value) return;
        select.disabled = true;
        api("/jiying/api/connections/account/select", { method: "POST", form: { email: select.value } }).then(function (result) {
           if (result.code === 200) { toast("已切换账号", "ok"); state.chatEpoch += 1; state.chatConvID = ""; state.conv = null; state.msgs = []; loadConvs(); }
          else toast(result.msg || "账号切换失败", "err");
        }).finally(function () { select.disabled = false; });
      });
    });
  }

  // convAppMeta 把后端补充的 relay_* 归属字段解析成前端可直接展示的应用信息。
  function convAppMeta(c) {
    if (!c) return null;
    var id = String(c.relay_app_id || "").trim();
    if (!id) return null;
    var known = state.apps.filter(function (a) { return String(a.app_id || "") === id; })[0] || {};
    var cached = state.appNames[id] || {};
    var name = c.relay_app_name || known.name || cached.name || shortID(id);
    var avatar = known.avatar_url || cached.avatar || "";
    return { id: id, name: name, avatar: avatar, online: c.relay_app_online !== false && known.online !== false, configured: c.relay_app_configured !== false };
  }
  function convAccount(c) {
    return !!c && !convAppMeta(c) && (c.relay_channel === "account" || c.relay_account === true);
  }
  // convKindLabel 用会话类型与成员数区分同名会话，例如「群 · 12 人」「私聊」。
  function convKindLabel(c) {
    var kind = String(c && c.type || "").toLowerCase();
    var n = Number(c && c.member_count || 0);
    if (kind === "group" || kind === "topic") {
      var base = kind === "topic" ? (LANG === "en" ? "Topic" : "话题") : t("chat.group");
      return n > 0 ? base + " · " + n + (LANG === "en" ? "" : " 人") : base;
    }
    if (kind === "app" || kind === "direct" || kind === "dm") return t("chat.dm");
    return "";
  }
  function convOwnLabel(c) {
    var meta = convAppMeta(c);
    if (meta) {
      if (meta.configured === false) return meta.name + " · " + t("chat.appMissing");
      return meta.name + (meta.online ? "" : " · " + t("chat.appOffline"));
    }
    return convAccount(c) ? t("chat.account") : t("chat.unassigned");
  }
  function convVisible(c) {
    var f = state.convFilter;
    if (!f) return true;
    var meta = convAppMeta(c);
    if (f === "__none__") return !meta && !convAccount(c);
    if (f === "__account__") return !meta && convAccount(c);
    return !!meta && meta.id === f;
  }
  function paintConvFilter() {
    var sel = $("#convAppFilter");
    if (!sel) return;
    var counts = {};
    var missing = {};
    var unowned = 0;
    var account = 0;
    (state.convs || []).forEach(function (c) {
      var meta = convAppMeta(c);
      if (!meta) {
        if (convAccount(c)) account += 1;
        else unowned += 1;
      } else {
        counts[meta.id] = (counts[meta.id] || 0) + 1;
        if (meta.configured === false) missing[meta.id] = meta.name;
      }
    });
    var seen = {};
    var html = '<option value="">' + esc(t("chat.allApps")) + "</option>";
    (state.apps || []).forEach(function (a) {
      var id = String(a.app_id || "");
      if (!id || seen[id]) return;
      // 离线且没有会话的应用不进入筛选，避免「全部应用」出现一串 (0)。
      if (a.online === false && !counts[id]) return;
      seen[id] = true;
      var name = a.name || state.appNames[id] && state.appNames[id].name || shortID(id);
      html += '<option value="' + esc(id) + '">' + esc(name) + " (" + (counts[id] || 0) + ")</option>";
    });
    Object.keys(counts).forEach(function (id) {
      if (seen[id]) return;
      var label = missing[id] ? missing[id] + " · " + t("chat.appMissing") : shortID(id);
      html += '<option value="' + esc(id) + '">' + esc(label) + " (" + counts[id] + ")</option>";
    });
    if (account) html += '<option value="__account__">' + esc(t("chat.account")) + " (" + account + ")</option>";
    html += '<option value="__none__">' + esc(t("chat.unassigned")) + " (" + unowned + ")</option>";
    sel.innerHTML = html;
    var want = state.convFilter;
    if (want && want !== "__none__" && want !== "__account__" && !seen[want]) {
      var extra = counts[want] ? want : "";
      if (!extra) { state.convFilter = ""; want = ""; }
    }
    sel.value = want || "";
    sel.onchange = function () {
      state.convFilter = sel.value;
      renderConvs();
    };
  }
  function renderConvs() {
    var box = $("#convList");
    if (!box) return;
    paintConvFilter();
    var list = state.convs || [];
    if (!list.length) { box.innerHTML = emptyState("message", "暂无会话", "收到或发起会话后，会在这里列出。"); return; }
    var nameCounts = {};
    list.forEach(function (c) {
      if (!convVisible(c)) return;
      var peer = c.peer || c.user || c.target || c.contact || {};
      var base = c.name || c.title || pickDisplayName(peer, "") || c.id;
      nameCounts[base] = (nameCounts[base] || 0) + 1;
    });
    var rows = [];
    list.forEach(function (c, i) {
      if (!convVisible(c)) return;
      var peer = c.peer || c.user || c.target || c.contact || {};
      var name = c.name || c.title || pickDisplayName(peer, "") || c.id;
      var baseName = name;
      var meta = convAppMeta(c);
      // 同名会话优先追加所属应用名，便于区分同一群在不同应用下的入口；
      // 应用名相同或缺失时再追加短 ID。
      if (nameCounts[name] > 1) {
        var parts = [];
        if (meta && meta.name && meta.name !== name) parts.push(meta.name);
        var kind = convKindLabel(c);
        if (kind) parts.push(kind);
        parts.push(shortID(c.id));
        name = name + " · " + parts.join(" · ");
      }
      var active = state.conv && state.conv.id === c.id;
      // 会话头像不能回退到当前选中应用的图标：多应用混排时会把所有无头像会话
      // 显示成同一个应用头像，看起来像随机错配。应用归属由下方 appLine 单独展示。
      var av = pickConversationAvatar(c);
      var own = convOwnLabel(c);
      var sameName = !!meta && meta.configured !== false && String(meta.name || "").trim() === String(baseName || "").trim();
      var appLine = "";
      if (meta && meta.configured === false) {
        appLine = '<span class="conv-app-line conv-app-none" title="' + esc(t("chat.appMissing")) + '">' + esc(own) + "</span>";
      } else if (meta && !sameName) {
        appLine = '<span class="conv-app-line" title="' + esc(t("chat.app") + "：" + own) + '">' + avatarHTML(meta.avatar, meta.name, "conv-app-av") + '<span class="conv-app-name">' + esc(own) + "</span></span>";
      } else if (meta && !meta.online) {
        appLine = '<span class="conv-app-line conv-app-none" title="' + esc(t("chat.appOffline")) + '">' + esc(t("chat.appOffline")) + "</span>";
      } else if (!meta && convAccount(c)) {
        appLine = '<span class="conv-app-line conv-app-account" title="' + esc(t("chat.account")) + '">' + esc(t("chat.account")) + "</span>";
      } else if (!meta) {
        appLine = '<span class="conv-app-line conv-app-none" title="' + esc(t("chat.noAssignment")) + '">' + esc(t("chat.unassigned")) + "</span>";
      }
      var tip = name + (own && own !== name && !sameName ? " · " + own : "");
      rows.push('<button type="button" class="conv-item' + (active ? " active" : "") + '" data-conv-index="' + i + '" title="' + esc(tip) + '">' + avatarHTML(av, name, "conv-avatar") + '<span class="conv-main"><span class="cname">' + esc(name) + "</span>" + appLine + "</span></button>");
    });
    if (!rows.length) { box.innerHTML = emptyState("message", t("chat.filter.empty"), ""); return; }
    box.innerHTML = rows.join("");
    $$("[data-conv-index]", box).forEach(function (item) {
      item.addEventListener("click", function () { pickConv(Number(item.getAttribute("data-conv-index"))); });
    });
    hydrateIcons();
  }
  function loadConvs(restoreID) {
    var epoch = state.chatEpoch;
    api("/jiying/api/conversations").then(function (r) {
      if (epoch !== state.chatEpoch) return;
      if (r.code !== 200) { errorState($("#convList"), "message", r.code === 403 ? "权限不足" : "会话加载失败", r.msg || r.message, loadConvs); return; }
      var loaded = r.data && r.data.conversations ? r.data.conversations : [];
      var seenIDs = {};
      state.convs = loaded.filter(function (c) {
        var id = String(c && c.id || "");
        if (!id || seenIDs[id]) return false;
        seenIDs[id] = true;
        return true;
      });
      var want = restoreID || state.chatConvID;
      if (want) {
        var hit = (state.convs || []).filter(function (c) { return String(c.id) === String(want); })[0];
        if (hit && !convVisible(hit)) state.convFilter = "";
      }
      renderConvs();
      if (want) {
        for (var k = 0; k < state.convs.length; k++) {
          if (String(state.convs[k].id) === String(want)) { pickConv(k); break; }
        }
      }
    }).catch(function () { errorState($("#convList"), "message", "会话列表加载失败", "网络异常，请检查服务状态。", loadConvs); });
  }

  function pickConv(i) {
    var c = state.convs[i];
    if (!c) return;
    state.conv = c;
    state.chatConvID = c.id;
    state.msgs = [];
    $$(".conv-item").forEach(function (item) {
      item.classList.toggle("active", Number(item.getAttribute("data-conv-index")) === i);
    });
    var titleName = c.name || "会话";
    var own = convOwnLabel(c);
    $("#chatTitle").textContent = own && own !== titleName ? titleName + " · " + own : titleName;
    loadChat();
    loadChatCapabilities();
    startChatTimer();
  }
  // capsReasonText 在英文模式下翻译后端返回的中文原因片段，应用名等用户数据保持原样。
  function capsReasonText(reason) {
    var raw = String(reason || "");
    if (!raw || LANG !== "en") return raw;
    return raw.split("；").map(function (part) {
      var key = part.trim();
      return i18nTranslate(key) || key;
    }).join("; ");
  }
  function loadChatCapabilities() {
    if (!state.conv) return;
    var convID = state.conv.id;
    state.chatCaps = null;
    var el = $("#chatCap");
    if (el) el.textContent = "正在检查可用发送身份…";
    api("/jiying/api/conversation/capabilities?conversation_id=" + encodeURIComponent(convID)).then(function (r) {
      if (!state.conv || state.conv.id !== convID) return;
      if (r.code !== 200 || !r.data) {
        if (el) el.textContent = r.msg || "可用身份检查失败，发送时会再次校验";
        setChatIdentityAvailability(null);
        return;
      }
      state.chatCaps = r.data;
      var caps = r.data;
      setChatIdentityAvailability(caps);
      if (!el) return;
      var can = [];
      if (caps.can_send_as_app) can.push(t("chat.app"));
      if (caps.can_send_as_user) can.push(t("chat.user"));
      var route = caps.app_known
        ? t("chat.app") + (LANG === "en" ? ": " : "：") + (caps.app_name || shortID(caps.app_id))
        : t("chat.noAssignment");
      if (caps.app_known && !caps.app_online) route += " · " + t("chat.appOffline");
      var reason = capsReasonText(caps.reason);
      el.textContent = can.length
        ? route + " · " + t("chat.canSend") + can.join(" / ") + (reason ? " · " + reason : "")
        : route + " · " + t("chat.noIdentity") + (reason ? " · " + reason : "");
      if (!can.length) el.style.color = "var(--err)";
      else el.style.color = "";
      var send = $("#chatSend"), prev = $("#chatPreview");
      if (send) send.disabled = !can.length;
      if (prev) prev.disabled = !can.length;
    }).catch(function () {
      if (el) el.textContent = "可用身份检查失败，发送时会再次校验";
      setChatIdentityAvailability(null);
    });
  }
  function setChatIdentityAvailability(caps) {
    var sel = $("#chatIdentity");
    if (!sel) return;
    Array.prototype.forEach.call(sel.options, function (o) {
      if (o.value === "app") o.disabled = caps ? !caps.can_send_as_app : false;
      if (o.value === "user") o.disabled = caps ? !caps.can_send_as_user : false;
    });
  }

  function loadChat() {
    if (!state.conv) return;
    var epoch = state.chatEpoch;
    var convID = state.conv.id;
    var request = ++state.chatRequest;
    api("/jiying/api/messages?conversation_id=" + encodeURIComponent(convID)).then(function (r) {
      if (epoch !== state.chatEpoch || request !== state.chatRequest || !state.conv || state.conv.id !== convID) return;
      if (r.code !== 200) { toast(r.msg || "消息加载失败", "err"); return; }
      var msgs = r.data && r.data.messages ? r.data.messages : [];
      state.msgs = msgs;
      renderMsgs();
    });
  }

  function renderMsgs() {
    var box = $("#chatStream");
    if (!box) return;
    // 用户正在向上翻阅历史时不要每次轮询都强行拉回底部。
    var pinned = box.scrollHeight - box.scrollTop - box.clientHeight < 80;
    var prevTop = box.scrollTop;
    if (!state.msgs.length) { box.innerHTML = emptyState("message", "暂无消息", "在下方输入框发送内容，中间件会按功能或 Agent 自动回复。"); return; }
    var app = state.app || {};
    var acct = state.account || {};
    var members = memberIndex(state.conv);
    var convAv = pickConversationAvatar(state.conv);
    box.innerHTML = state.msgs.map(function (m) {
      var body = m.body || {};
      var isMarkdown = body.type === "markdown";
      var content = "";
      if (typeof body === "string") content = body;
      else if (body.type === "text") content = body.content || body.text || "";
      else if (body.type === "image") content = "[图片]";
      else if (body.type === "file") content = "[文件] " + (body.name || body.filename || "");
      else content = m.summary || body.content || body.text || (body.type ? "[" + body.type + "]" : "");
      var sender = (m.sender && typeof m.sender === "object") ? m.sender : {};
      var member = members[String(sender.id)] || null;
      var appSender = sender.type === "app" ? appLookup(sender.id) : null;
      // 账号本人以服务端返回的用户 id 判定；sender.type=app 表示中间件/应用发出。
      var mine = !!(acct.id && sender.id && String(sender.id) === String(acct.id));
      var out = mine || sender.type === "app" || sender.type === "assistant" || m.direction === "out" || m.is_out === true;
      var t = fmtTime(m.created_at || m.createdAt || m.time);
      var who = pickDisplayName(member, "") || pickDisplayName(appSender, "") || pickDisplayName(sender, "") || m.sender_name;
      if (!who && sender.type === "assistant") who = LANG === "en" ? "Jiying Assistant" : "即应助手";
      if (!who && out) who = app.name || acct.nickname || acct.email || "Relay";
      if (!who) who = SENDER_LABEL[sender.type || ""] || sender.type || "对方";
      // 入站优先用成员/发送者头像，其次用会话头像；出站回退到应用/当前账号头像。
      var av = pickAvatarURL(member) || pickAvatarURL(appSender) || pickAvatarURL(sender) || (out ? (app.avatar_url || acct.avatar_url || "") : convAv);
      if (!av && sender.type === "assistant") av = "/assets/apps/assistant.webp";
      var rendered = isMarkdown ? renderMarkdownSafe(content) : esc(content);
      return '<div class="msg-item ' + (out ? "msg-item-out" : "msg-item-in") + '">' + avatarHTML(av, who, "msg-avatar")
        + '<div class="msg-main"><span class="msg-meta" title="' + esc(who) + '">' + esc(who) + " · " + esc(t) + '</span><div class="msg-bubble ' + (out ? "msg-out" : "msg-in") + '">' + rendered + "</div></div></div>";
    }).join("");
    if (pinned) box.scrollTop = box.scrollHeight;
    else box.scrollTop = prevTop;
  }

  function sendChat(dryRun) {
    var input = $("#chatInput");
    var text = input.value.trim();
    if (!state.conv || !text) { toast("请选择会话并输入内容", "err"); return; }
    var identityEl = $("#chatIdentity");
    var identity = identityEl ? identityEl.value : "auto";
    if (!dryRun) input.value = "";
    api("/jiying/api/send/unified", { method: "POST", form: { conversation_id: state.conv.id, text: text, identity: identity, dry_run: dryRun ? "true" : "false" } }).then(function (r) {
      if (r.code === 200) {
        var res = r.data || {};
        var who = res.identity === "user" ? "用户" : res.identity === "app" ? "应用" : "自动";
        if (dryRun) {
          showModal("发送预演", '<div class="kv"><span class="k">会话</span><span class="v mono">' + esc(state.conv.id) + '</span></div>'
            + '<div class="kv"><span class="k">解析身份</span><span class="v">' + esc(who) + '</span></div>'
            + '<div class="kv"><span class="k">内容</span><span class="v">' + esc(text) + '</span></div>'
            + '<div class="muted mt-8" style="font-size:12px">' + esc(res.reason || "未产生实际发送，也不会写入消息记录。") + '</div>',
            '<button class="btn btn-primary" data-bs-dismiss="modal">知道了</button>');
          return;
        }
        toast("已通过" + who + "身份发送", "ok");
        setTimeout(loadChat, 800);
      } else {
        // 发送失败时把内容还回输入框，避免用户手打的消息丢失（已在输入新内容则不覆盖）。
        if (!dryRun) {
          var box = $("#chatInput");
          if (box && !box.value.trim()) box.value = text;
        }
        toast(r.msg || "发送失败", "err");
      }
    });
  }

  /* ---------- 插件（脚本资产） ---------- */
  var SCRIPT_LANG_LABEL = { js: "JavaScript", python: "Python" };
  var SENDER_LABEL = { user: "对方", system: "系统", bot: "机器人", app: "应用" };
  var SCRIPT_EXT = { js: ".js", python: ".py" };
  var SCRIPT_TPL = {
    js: '// [title: 示例插件]\n// [author: 我]\n// [rule: ^示例$]\n// [description: 示例插件]\nresult = "hello " + args;\n',
    python: '# [title: 示例插件]\n# [author: 我]\n# [rule: ^示例$]\n# [description: 示例插件]\nprint("hello " + __import__("os").environ.get("JIYING_ARGS", ""))\n'
  };
  // 模板库降低上手成本：空白模板之外，给出可直接运行的发送与提醒示例。
  var SCRIPT_TEMPLATES = [
    { id: "js-basic", lang: "js", name: "demo.js", title: "JavaScript 空白模板" },
    { id: "py-basic", lang: "python", name: "demo.py", title: "Python 空白模板" },
    {
      id: "js-send", lang: "js", name: "echo.js", title: "回发消息（JavaScript）",
      code: "// [title: 回发消息]\n// [description: 把参数原样发回当前会话]\n// [rule: ^回发\\s+(.+)$]\nvar text = String(args || \"\").trim();\nif (!text) {\n  result = \"用法：回发 内容\";\n} else {\n  sendText(text);\n  result = \"已发送\";\n}\n"
    },
    {
      id: "py-send", lang: "python", name: "remind.py", title: "回发消息（Python）",
      code: "# [title: 定时提醒]\n# [description: 把参数作为提醒发回当前会话]\n# [rule: ^提醒\\s+(.+)$]\nimport os\nmsg = os.environ.get(\"JIYING_ARGS\", \"\").strip() or \"该处理这件事了\"\nprint(\"SEND:\" + msg)\n"
    }
  ];
  function renderScripts() {
    var cont = $("#view-scripts");
    cont.innerHTML = '<div class="page-head"><div><h2>插件</h2><p>按分类归档的插件资产，支持一个插件服务多个应用；安装后会自动登记为可调用功能</p></div><div class="spacer"></div>'
      + '<button class="btn btn-ghost btn-sm" onclick="Relay.go(\'features\')">功能管理 →</button>'
      + '<button class="btn btn-primary" id="newScript">新建插件</button></div>'
      + '<div class="panel"><div class="plugin-toolbar">'
      + '<div class="plugin-tree-map" id="scriptTreeMap"></div>'
      + '<div class="spacer"></div>'
      + '<input class="input plugin-search" id="scriptSearch" type="search" placeholder="搜索插件 / 分类 / 服务 / 应用" aria-label="搜索插件">'
      + '<div class="seg" id="scriptViewToggle" role="group" aria-label="插件视图"><button type="button" data-sview="tree">目录</button><button type="button" data-sview="list">列表</button></div>'
      + '</div><div class="panel-body" id="scriptList">' + loadingState() + '</div></div>'
      + '<div class="panel mt-16"><div class="panel-head"><h3>技能</h3><span class="hint">可被插件 require 复用的 JS 模块，或 Markdown 说明；安装前会隔离试跑</span><div class="spacer"></div><button class="btn btn-primary btn-sm" id="newSkill">安装技能</button></div><div class="panel-body" id="skillList">' + loadingState() + '</div></div>';
    var n = $("#newScript");
    if (n) n.addEventListener("click", function () { openScriptModal(null); });
    var ns = $("#newSkill");
    if (ns) ns.addEventListener("click", openSkillModal);
    var search = $("#scriptSearch");
    if (search) search.addEventListener("input", function () { paintScripts(); });
    $$("#scriptViewToggle [data-sview]").forEach(function (b) {
      b.addEventListener("click", function () {
        state.scriptView = b.getAttribute("data-sview");
        paintScripts();
      });
    });
    bindScriptFolders();
    loadScripts();
    loadSkills();
  }
  var SCRIPT_CACHE = [];
  function loadScripts() {
    api("/jiying/api/scripts").then(function (r) {
      var box = $("#scriptList");
      if (!box) return;
      if (r.code !== 200) { errorState(box, "puzzle", "插件列表加载失败", r.msg || r.message, loadScripts); return; }
      SCRIPT_CACHE = r.data || [];
      if (!SCRIPT_CACHE.length) {
        box.innerHTML = emptyState("puzzle", "暂无插件", "插件放在 scripts 目录下，安装后会自动登记为可调用功能。", '<button class="btn btn-primary btn-sm" id="scriptEmptyAdd">新建插件</button>');
        var b = $("#scriptEmptyAdd");
        if (b) b.addEventListener("click", function () { openScriptModal(null); });
        paintScriptTreeMap();
        return;
      }
      loadScriptAppMap(function () { paintScripts(); });
      paintScripts();
    }).catch(function () { errorState($("#scriptList"), "puzzle", "插件列表加载失败", "网络异常，请检查服务状态。", loadScripts); });
  }
  // 应用名与头像来自连接池；只拉一次，插件列表即可把应用 ID 显示成人名与头像。
  function loadScriptAppMap(cb) {
    if (state.appNamesLoaded) { if (cb) cb(); return; }
    api("/jiying/api/connections", { silent: true }).then(function (r) {
      var apps = (r && r.data && r.data.apps) || [];
      state.apps = apps;
      apps.forEach(function (a) {
        var id = String(a.app_id || "");
        if (!id) return;
        state.appNames[id] = { name: a.name || state.appProfiles[id] && state.appProfiles[id].name || shortID(id), avatar: a.avatar_url || state.appProfiles[id] && state.appProfiles[id].avatar_url || "" };
      });
      state.appNamesLoaded = true;
      if (cb) cb();
    }).catch(function () { state.appNamesLoaded = true; if (cb) cb(); });
  }
  function scriptClass(s) {
    var c = String((s && s.class) || "").trim();
    return c || "未分类";
  }
  // 服务用于把同一业务/项目的多个插件收进一个子文件夹，与分类形成「分类 → 服务 → 插件」三层。
  function scriptService(s) {
    return String((s && s.service) || "").trim();
  }
  function scriptFilterText(s) {
    var apps = (s.apps || []).map(function (id) { var a = state.appNames[id]; return (a && a.name) || id; });
    return [s.title, s.name, s.class, s.service, s.description, (s.rules || []).join(" "), (s.platform || []).join(" "), apps.join(" ")].join(" ").toLowerCase();
  }
  function scriptAppChips(s) {
    var apps = s.apps || [];
    if (!apps.length || apps.indexOf("*") >= 0) return '<span class="app-chip app-chip-all"><span class="ico">' + ico("layers") + '</span>全部应用</span>';
    return apps.map(function (id) {
      var a = state.appNames[id] || {};
      var name = a.name || shortID(id);
      var av = a.avatar ? '<span class="app-chip-av">' + avatarInner(a.avatar, name) + "</span>" : '<span class="app-chip-av app-chip-av-fb">' + esc(name.charAt(0).toUpperCase()) + "</span>";
      return '<span class="app-chip" title="' + esc(id) + '">' + av + esc(name) + "</span>";
    }).join("");
  }
  function scriptRow(s) {
    var lang = s.lang === "python" ? "python" : "js";
    var meta = '<span class="badge badge-off">' + (SCRIPT_LANG_LABEL[lang] || lang) + '</span>';
    if (s.version) meta += ' <span class="badge badge-info">v' + esc(s.version) + '</span>';
    if (s.admin) meta += ' <span class="badge badge-warn">主控</span>';
    var detail = [];
    if (s.description) detail.push(esc(s.description));
    detail.push((s.size || 0) + " B");
    if (s.rules && s.rules.length) detail.push("规则 " + esc(s.rules.join(" / ")));
    return '<div class="list-item plugin-row" data-name="' + esc(s.name) + '"><div class="ico li-ico">' + pluginIcon(s.icon) + '</div>'
      + '<div class="grow rl-clickable" onclick="Relay.editScript(' + jsArg(encodeURIComponent(s.name)) + ')"><div class="name">' + esc(s.title || s.name) + ' <span class="badge badge-off">' + esc(s.name) + '</span> ' + meta + '</div>'
      + '<div class="desc">' + detail.join(" · ") + '</div><div class="app-chips">' + scriptAppChips(s) + "</div></div>"
      + '<div class="actions">'
      + '<button class="btn btn-ghost btn-sm" onclick="Relay.editScriptApps(' + jsArg(encodeURIComponent(s.name)) + ')">应用范围</button>'
      + (s.config_schema && s.config_schema.length ? '<button class="btn btn-ghost btn-sm" onclick="Relay.scriptConfig(' + jsArg(encodeURIComponent(s.name)) + ')">配置</button>' : "")
      + '<button class="btn btn-ghost btn-sm" onclick="Relay.editScript(' + jsArg(encodeURIComponent(s.name)) + ')">编辑</button>'
      + '<button class="btn btn-ghost btn-sm" onclick="Relay.runScript(' + jsArg(encodeURIComponent(s.name)) + ')">运行</button>'
      + '<button class="btn btn-outline-danger btn-sm" onclick="Relay.delScript(' + jsArg(encodeURIComponent(s.name)) + ')">删除</button></div></div>';
  }
  function scriptGroups(list) {
    var groups = {};
    list.forEach(function (s) { var c = scriptClass(s); (groups[c] = groups[c] || []).push(s); });
    return groups;
  }
  function paintScriptTreeMap() {
    var el = $("#scriptTreeMap");
    if (!el) return;
    var groups = scriptGroups(SCRIPT_CACHE || []);
    var classes = Object.keys(groups).sort(function (a, b) { return a.localeCompare(b, "zh"); });
    var appSet = {}, serviceSet = {};
    (SCRIPT_CACHE || []).forEach(function (s) {
      (s.apps || []).forEach(function (id) { if (id !== "*") appSet[id] = 1; });
      var svc = scriptService(s);
      if (svc) serviceSet[scriptClass(s) + "//" + svc] = 1;
    });
    el.innerHTML = '<span class="ptm-node"><span class="ico">' + ico("puzzle") + "</span><b>" + (SCRIPT_CACHE || []).length + "</b> 插件</span>"
      + '<span class="ptm-link" aria-hidden="true"></span>'
      + '<span class="ptm-node"><span class="ico">' + ico("folder") + "</span><b>" + classes.length + "</b> 分类</span>"
      + '<span class="ptm-link" aria-hidden="true"></span>'
      + '<span class="ptm-node"><span class="ico">' + ico("layers") + "</span><b>" + Object.keys(serviceSet).length + "</b> 服务</span>"
      + '<span class="ptm-link" aria-hidden="true"></span>'
      + '<span class="ptm-node"><span class="ico">' + ico("layers") + "</span><b>" + Object.keys(appSet).length + "</b> 应用</span>";
  }
  function paintScripts() {
    var box = $("#scriptList");
    if (!box) return;
    var kw = (($("#scriptSearch") || {}).value || "").trim().toLowerCase();
    var list = (SCRIPT_CACHE || []).filter(function (s) { return !kw || scriptFilterText(s).indexOf(kw) >= 0; });
    $$("#scriptViewToggle [data-sview]").forEach(function (b) { b.classList.toggle("active", b.getAttribute("data-sview") === state.scriptView); });
    paintScriptTreeMap();
    if (!list.length) {
      box.innerHTML = emptyState("search", kw ? "没有匹配的插件" : "暂无插件", kw ? "换个关键词，或清空搜索查看全部插件。" : "插件放在 scripts 目录下，安装后会自动登记为可调用功能。");
      return;
    }
    if (state.scriptView === "list") {
      box.innerHTML = '<div class="list">' + list.slice().sort(function (a, b) {
        return scriptClass(a).localeCompare(scriptClass(b), "zh") || scriptService(a).localeCompare(scriptService(b), "zh") || String(a.name).localeCompare(String(b.name));
      }).map(function (s) {
        var svc = scriptService(s);
        return '<div class="plugin-flat"><span class="plugin-class-tag">' + esc(scriptClass(s)) + "</span>"
          + (svc ? '<span class="plugin-class-tag plugin-service-tag">' + esc(svc) + "</span>" : "")
          + scriptRow(s) + "</div>";
      }).join("") + "</div>";
      hydrateIcons();
      return;
    }
    var groups = scriptGroups(list);
    var classes = Object.keys(groups).sort(function (a, b) { return a.localeCompare(b, "zh"); });
    box.innerHTML = classes.map(function (cls) {
      var folded = state.scriptFolders[cls] === false;
      var items = groups[cls].slice().sort(function (a, b) { return String(a.name).localeCompare(String(b.name)); });
      var direct = [], services = {};
      items.forEach(function (s) {
        var svc = scriptService(s);
        if (svc) (services[svc] = services[svc] || []).push(s);
        else direct.push(s);
      });
      var svcNames = Object.keys(services).sort(function (a, b) { return a.localeCompare(b, "zh"); });
      var body = svcNames.map(function (svc) {
        var key = cls + "//" + svc;
        var subFolded = state.scriptFolders[key] === false;
        var svcItems = services[svc].slice().sort(function (a, b) { return String(a.name).localeCompare(String(b.name)); });
        return '<section class="plugin-folder plugin-folder-sub' + (subFolded ? " folded" : "") + '" data-folder="' + esc(key) + '">'
          + '<button type="button" class="plugin-folder-head" aria-expanded="' + (subFolded ? "false" : "true") + '"><span class="pf-chevron">' + ico("chevron-down") + '</span><span class="pf-ico">' + ico("layers") + '</span><b>' + esc(svc) + '</b><span class="pf-count">' + svcItems.length + '</span><span class="pf-line" aria-hidden="true"></span></button>'
          + '<div class="plugin-folder-body"><div class="list">' + svcItems.map(scriptRow).join("") + "</div></div></section>";
      }).join("");
      if (direct.length) body += '<div class="list">' + direct.map(scriptRow).join("") + "</div>";
      return '<section class="plugin-folder' + (folded ? " folded" : "") + '" data-folder="' + esc(cls) + '">'
        + '<button type="button" class="plugin-folder-head" aria-expanded="' + (folded ? "false" : "true") + '"><span class="pf-chevron">' + ico("chevron-down") + '</span><span class="pf-ico">' + ico("folder") + '</span><b>' + esc(cls) + '</b><span class="pf-count">' + items.length + '</span>'
        + (svcNames.length ? '<span class="pf-svc-count">' + svcNames.length + " 服务</span>" : "")
        + '<span class="pf-line" aria-hidden="true"></span></button>'
        + '<div class="plugin-folder-body">' + body + "</div></section>";
    }).join("");
    bindScriptFolders();
    hydrateIcons();
  }
  function bindScriptFolders() {
    $$("#scriptList .plugin-folder-head").forEach(function (b) {
      b.addEventListener("click", function () {
        var sec = b.parentNode;
        var cls = sec.getAttribute("data-folder");
        var folded = !sec.classList.contains("folded");
        sec.classList.toggle("folded", folded);
        b.setAttribute("aria-expanded", folded ? "false" : "true");
        state.scriptFolders[cls] = !folded;
      });
    });
  }
  function bindCodeEditor(el) {
    el.addEventListener("keydown", function (e) {
      if (e.key === "Tab") {
        e.preventDefault();
        var s = this.selectionStart, en = this.selectionEnd;
        this.value = this.value.slice(0, s) + "  " + this.value.slice(en);
        this.selectionStart = this.selectionEnd = s + 2;
      }
    });
  }
  function openScriptModal(prefill) {
    var isEdit = !!(prefill && prefill.name);
    var lang = (prefill && prefill.lang === "python") ? "python" : "js";
    var name = isEdit ? prefill.name : "demo.js";
    var code = isEdit ? (prefill.content || "") : SCRIPT_TPL[lang];
    showModal(isEdit ? "编辑插件" : "新建插件",
      (isEdit ? "" : '<div class="field"><label>模板</label><select class="select" id="sm_tpl">'
        + SCRIPT_TEMPLATES.map(function (x) { return '<option value="' + esc(x.id) + '">' + esc(x.title) + "</option>"; }).join("")
        + '</select><div class="hint">选一个接近需求的模板，再改文件名和代码即可安装。</div></div>')
      + '<div class="grid grid-2"><div class="field"><label>插件文件名</label><input class="input" id="sm_name" value="' + esc(name) + '" placeholder="demo.js"></div>'
      + '<div class="field"><label>语言</label><select class="select" id="sm_lang"><option value="js">JavaScript</option><option value="python">Python</option></select></div></div>'
      + '<div class="field"><label>插件代码</label><textarea class="input code-editor" id="sm_code" rows="16" spellcheck="false"></textarea></div>'
      + '<p style="color:var(--muted);font-size:12px;margin:6px 0 0">JavaScript 用 <code>result</code> 返回结果；Python 用标准输出返回结果，以 <code>SEND:</code> 开头的行会作为会话消息回发。头部注释可用 <code>[title: ...]</code>、<code>[class: 分类]</code>、<code>[service: 服务名]</code>、<code>[apps: 应用ID]</code>、<code>[rule: ...]</code>、<code>[cron: ...]</code>、<code>[admin: true]</code> 声明元数据（兼容 AutMan）；同一 <code>service</code> 的插件会收进同一服务文件夹。</p>',
      '<button class="btn btn-ghost" onclick="Relay.closeModal()">取消</button><button class="btn btn-primary" id="sm_save">' + (isEdit ? "保存" : "安装") + "</button>");
    var codeEl = $("#sm_code"), langEl = $("#sm_lang"), nameEl = $("#sm_name");
    codeEl.value = code;
    langEl.value = lang;
    bindCodeEditor(codeEl);
    var tplEl = $("#sm_tpl");
    if (tplEl) tplEl.addEventListener("change", function () {
      var tpl = SCRIPT_TEMPLATES.filter(function (x) { return x.id === tplEl.value; })[0];
      if (!tpl) return;
      lang = tpl.lang;
      langEl.value = tpl.lang;
      nameEl.value = tpl.name;
      codeEl.value = tpl.code || SCRIPT_TPL[tpl.lang];
    });
    langEl.addEventListener("change", function () {
      var prevTpl = SCRIPT_TPL[lang];
      lang = this.value;
      if (codeEl.value === "" || codeEl.value === prevTpl) codeEl.value = SCRIPT_TPL[lang];
      if (/\.(js|py)$/i.test(nameEl.value.trim())) nameEl.value = nameEl.value.trim().replace(/\.(js|py)$/i, SCRIPT_EXT[lang]);
    });
    $("#sm_save").addEventListener("click", function () {
      var nm = nameEl.value.trim();
      var body = codeEl.value;
      if (!nm || !body) { toast("请填写文件名与代码", "err"); return; }
      if (!/\.(js|py)$/i.test(nm)) nm += SCRIPT_EXT[langEl.value];
      var risk = scriptRiskReasons(nm, body);
      var form = { name: nm, code: body };
      if (risk.length && !window.confirm("脚本包含：" + risk.join("、") + "。确认安装并允许其以服务权限运行？")) return;
      if (risk.length) form.confirm = "true";
      api("/jiying/api/script/install", { method: "POST", form: form }).then(function (r) {
        if (r.code === 200) { closeModal(); toast(r.data || "已安装", "ok"); loadScripts(); }
        else toast(r.msg || "安装失败", "err");
      });
    });
  }
  function editScript(name) {
    var nm = decodeURIComponent(name);
    api("/jiying/api/script/get", { method: "POST", form: { name: nm } }).then(function (r) {
      if (r.code !== 200 || !r.data) { toast(r.msg || "读取失败", "err"); return; }
      openScriptModal({ name: r.data.name, lang: r.data.lang, content: r.data.content });
    });
  }
  function editPluginOf(target) {
    var nm = decodeURIComponent(target || "");
    if (!nm) { toast("该功能未关联插件文件", "err"); return; }
    switchView("scripts");
    editScript(encodeURIComponent(nm));
  }
  function runScript(name) {
    var nm = decodeURIComponent(name);
    showModal("运行插件 · " + esc(nm),
      '<div class="field"><label>参数（可选）</label><input class="input" id="sm_args" placeholder="参数"></div>'
      + '<div class="field"><label>输出</label><pre class="script-out" id="sm_out">点击运行查看结果</pre></div>',
      '<button class="btn btn-ghost" onclick="Relay.closeModal()">关闭</button><button class="btn btn-primary" id="sm_run">运行</button>');
    $("#sm_run").addEventListener("click", function () {
      var out = $("#sm_out");
      out.textContent = "运行中…";
      api("/jiying/api/script/run", { method: "POST", form: { name: nm, args: $("#sm_args").value } }).then(function (r) {
        if (r.code === 200) out.textContent = (r.data === "" || r.data == null) ? "(无输出)" : String(r.data);
        else out.textContent = r.msg || "运行失败";
      }).catch(function () { out.textContent = "运行失败"; });
    });
  }
  function delScript(name) {
    var nm = decodeURIComponent(name);
    confirmDialog("删除插件", "确认删除插件 " + nm + " ？其登记的可调用功能会一并移除。", "删除", function () {
      api("/jiying/api/script/delete", { method: "POST", form: { name: nm } }).then(function (r) {
        if (r.code === 200) { toast("已删除", "ok"); loadScripts(); }
        else toast(r.msg || "删除失败", "err");
      });
    });
  }

  /* ---------- 插件配置（头部 [config: ...] 动态渲染表单） ---------- */
  function configFieldHTML(f, values, set) {
    var id = "cfg_" + f.key;
    var val = values && values[f.key] != null ? String(values[f.key]) : "";
    var label = '<label for="' + id + '">' + esc(f.key) + (f.desc ? ' <span class="hint">' + esc(f.desc) + "</span>" : "") + "</label>";
    var control;
    if (f.type === "select") {
      var opts = (f.options || []).map(function (o) {
        return '<option value="' + esc(o) + '"' + (o === val ? " selected" : "") + ">" + esc(o) + "</option>";
      }).join("");
      control = '<select class="select" id="' + id + '" data-key="' + esc(f.key) + '" data-type="select">' + opts + "</select>";
    } else if (f.type === "text") {
      control = '<textarea class="input code-editor" rows="5" id="' + id + '" data-key="' + esc(f.key) + '" data-type="text">' + esc(val) + "</textarea>";
    } else if (f.type === "bool") {
      control = '<select class="select" id="' + id + '" data-key="' + esc(f.key) + '" data-type="bool">'
        + '<option value="true"' + (val === "true" ? " selected" : "") + ">true</option>"
        + '<option value="false"' + (val !== "true" ? " selected" : "") + ">false</option></select>";
    } else if (f.type === "secret") {
      var configured = set && set.indexOf(f.key) >= 0;
      control = '<input class="input" type="password" id="' + id + '" data-key="' + esc(f.key) + '" data-type="secret" autocomplete="new-password" placeholder="' + (configured ? "已设置，留空则不修改" : "未设置，请输入") + '">';
    } else if (f.type === "number") {
      control = '<input class="input" type="number" step="any" id="' + id + '" data-key="' + esc(f.key) + '" data-type="number" value="' + esc(val) + '">';
    } else {
      control = '<input class="input" type="text" id="' + id + '" data-key="' + esc(f.key) + '" data-type="string" value="' + esc(val) + '">';
    }
    return '<div class="field config-field">' + label + control + "</div>";
  }
  function configControls() { return $$("#modalBody [data-key]"); }
  function collectConfigValues() {
    var out = {}, bad = "";
    configControls().forEach(function (el) {
      var key = el.getAttribute("data-key"), type = el.getAttribute("data-type"), v = el.value;
      if (type === "number" && v !== "" && isNaN(Number(v))) { bad = key; return; }
      out[key] = v;
    });
    if (bad) { toast("配置 " + bad + " 需要数字", "err"); return null; }
    return out;
  }
  function fillConfigDefaults(schema) {
    var byKey = {};
    (schema || []).forEach(function (f) { byKey[f.key] = f; });
    configControls().forEach(function (el) {
      var f = byKey[el.getAttribute("data-key")];
      if (!f) return;
      if (f.type === "secret") { el.value = ""; return; }
      el.value = f.default != null ? String(f.default) : "";
    });
  }
  function openScriptConfig(name) {
    var nm = decodeURIComponent(name);
    api("/jiying/api/script/config", { method: "POST", form: { name: nm } }).then(function (r) {
      if (r.code !== 200 || !r.data) { toast(r.msg || "读取配置失败", "err"); return; }
      var schema = r.data.schema || [], values = r.data.values || {}, set = r.data.set || [];
      var fields = schema.map(function (f) { return configFieldHTML(f, values, set); }).join("");
      var body = fields
        ? '<p class="hint" style="margin:0 0 10px">配置保存在服务端；运行时以 <code>config</code> 对象（JS）或 <code>JIYING_CONFIG</code> 环境变量（Python）注入。密钥值不会回显，留空表示保持不变。</p><div class="config-form">' + fields + "</div>"
        : emptyState("sliders", "无配置项", "该插件未声明 [config: ...]。");
      showModal("插件配置 · " + esc(nm), body,
        '<button class="btn btn-ghost" id="cfg_reset">恢复默认</button><button class="btn btn-ghost" onclick="Relay.closeModal()">取消</button><button class="btn btn-primary" id="cfg_save">保存</button>');
      var rs = $("#cfg_reset");
      if (rs) rs.addEventListener("click", function () { fillConfigDefaults(schema); });
      var sv = $("#cfg_save");
      if (sv) sv.addEventListener("click", function () {
        var out = collectConfigValues();
        if (out === null) return;
        api("/jiying/api/script/config/save", { method: "POST", form: { name: nm, values: JSON.stringify(out) } }).then(function (rr) {
          if (rr.code === 200) { closeModal(); toast(rr.data || "配置已保存", "ok"); }
          else toast(rr.msg || "保存失败", "err");
        });
      });
    }).catch(function () { toast("读取配置失败", "err"); });
  }

  /* ---------- 技能（可被插件 require 复用的模块） ---------- */
  var SKILL_TPL = {
    js: '// [title: 示例技能]\nmodule.exports = function (args) {\n  return "hello " + args;\n};\n',
    md: "# 示例技能\n\n在此记录可复用的说明或提示词。\n"
  };
  function loadSkills() {
    api("/jiying/api/skills").then(function (r) {
      var box = $("#skillList");
      if (!box) return;
      if (r.code !== 200) { errorState(box, "code", r.code === 403 ? "权限不足" : "技能列表加载失败", r.msg || r.message, loadSkills); return; }
      var list = r.data || [];
      if (!list.length) {
        box.innerHTML = emptyState("code", "暂无技能", "技能可被插件通过 require 复用；安装 JS 技能时会先在沙箱内试跑。", '<button class="btn btn-primary btn-sm" id="skillEmptyAdd">安装技能</button>');
        var b = $("#skillEmptyAdd");
        if (b) b.addEventListener("click", openSkillModal);
        return;
      }
      box.innerHTML = '<div class="list">' + list.map(function (s) {
        var isJS = s.kind === "script";
        return '<div class="list-item"><div class="ico li-ico">' + ico("code") + "</div>"
          + '<div class="grow"><div class="name">' + esc(s.name) + " " + metaBadge(isJS ? "JS 模块" : "Markdown", isJS ? "badge-info" : "badge-off") + "</div>"
          + '<div class="desc">' + (s.size || 0) + " B</div></div>"
          + '<div class="actions">' + (isJS ? '<button class="btn btn-ghost btn-sm" onclick="Relay.runSkill(' + jsArg(encodeURIComponent(s.name)) + ')">运行</button>' : "")
          + '<button class="btn btn-outline-danger btn-sm" onclick="Relay.delSkill(' + jsArg(encodeURIComponent(s.name)) + ')">删除</button></div></div>';
      }).join("") + "</div>";
      hydrateIcons();
    }).catch(function () { errorState($("#skillList"), "code", "技能列表加载失败", "网络异常，请检查服务状态。", loadSkills); });
  }
  function openSkillModal() {
    var lang = "js";
    showModal("安装技能",
      '<div class="grid grid-2"><div class="field"><label>技能文件名</label><input class="input" id="sk_name" value="helper.js" placeholder="helper.js 或 note.md"></div>'
      + '<div class="field"><label>类型</label><select class="select" id="sk_lang"><option value="js">JavaScript 模块</option><option value="md">Markdown</option></select></div></div>'
      + '<div class="field"><label>技能内容</label><textarea class="input code-editor" id="sk_code" rows="14" spellcheck="false"></textarea></div>'
      + '<p style="color:var(--muted);font-size:12px;margin:6px 0 0">JS 技能用 <code>module.exports</code> 导出函数，可被插件通过 <code>require("helper.js")</code> 复用；安装前会在沙箱内试跑，失败不落盘。Markdown 技能用于沉淀说明与提示词。</p>',
      '<button class="btn btn-ghost" onclick="Relay.closeModal()">取消</button><button class="btn btn-primary" id="sk_save">安装</button>');
    var codeEl = $("#sk_code"), nameEl = $("#sk_name"), langEl = $("#sk_lang");
    codeEl.value = SKILL_TPL.js;
    bindCodeEditor(codeEl);
    langEl.addEventListener("change", function () {
      var prev = SKILL_TPL[lang];
      lang = this.value;
      if (codeEl.value === "" || codeEl.value === prev) codeEl.value = SKILL_TPL[lang];
      nameEl.value = nameEl.value.trim().replace(/\.(js|md)$/i, lang === "js" ? ".js" : ".md");
    });
    $("#sk_save").addEventListener("click", function () {
      var nm = nameEl.value.trim(), body = codeEl.value;
      if (!nm || !body) { toast("请填写文件名与内容", "err"); return; }
      if (!/\.(js|md)$/i.test(nm)) nm += lang === "js" ? ".js" : ".md";
      api("/jiying/api/skill/install", { method: "POST", form: { name: nm, code: body } }).then(function (r) {
        if (r.code === 200) { closeModal(); toast(r.data || "已安装", "ok"); loadSkills(); }
        else toast(r.msg || "安装失败", "err");
      });
    });
  }
  function runSkill(name) {
    var nm = decodeURIComponent(name);
    showModal("运行技能 · " + esc(nm),
      '<div class="field"><label>参数（可选）</label><input class="input" id="sk_args" placeholder="参数"></div>'
      + '<div class="field"><label>输出</label><pre class="script-out" id="sk_out">点击运行查看结果</pre></div>',
      '<button class="btn btn-ghost" onclick="Relay.closeModal()">关闭</button><button class="btn btn-primary" id="sk_run">运行</button>');
    $("#sk_run").addEventListener("click", function () {
      var out = $("#sk_out");
      out.textContent = "运行中…";
      api("/jiying/api/skill/run", { method: "POST", form: { name: nm, args: $("#sk_args").value } }).then(function (r) {
        if (r.code === 200) out.textContent = (r.data === "" || r.data == null) ? "(无输出)" : String(r.data);
        else out.textContent = r.msg || "运行失败";
      }).catch(function () { out.textContent = "运行失败"; });
    });
  }
  function delSkill(name) {
    var nm = decodeURIComponent(name);
    confirmDialog("删除技能", "确认删除技能 " + nm + " ？引用它的插件可能因此报错。", "删除", function () {
      api("/jiying/api/skill/delete", { method: "POST", form: { name: nm } }).then(function (r) {
        if (r.code === 200) { toast("已删除", "ok"); loadSkills(); }
        else toast(r.msg || "删除失败", "err");
      });
    });
  }

  /* ---------- 功能管理 ---------- */
  var FEATURE_KIND = { builtin: "内置", http: "HTTP", script: "脚本" };
  var FEATURE_SCOPE = { public: "公开", master: "主控" };
  var FEATURE_IDENTITY = { app: "应用", user: "账号" };
  function metaBadge(text, cls) {
    return '<span class="badge ' + (cls || "badge-off") + '">' + esc(text) + "</span>";
  }
  function renderFeatures() {
    var cont = $("#view-features");
    cont.innerHTML = '<div class="page-head"><div><h2>功能管理</h2><p>面向即应会话的可调用功能：内置、HTTP 与插件功能的启停、试跑与配置</p></div><div class="spacer"></div>'
      + '<button class="btn btn-ghost btn-sm" onclick="Relay.loadFeatures()">刷新</button>'
      + '<button class="btn btn-primary btn-sm" id="newFeature">安装功能</button></div>'
      + '<div class="panel"><div class="panel-body" id="featureList">' + loadingState() + '</div></div>';
    var n = $("#newFeature");
    if (n) n.addEventListener("click", openFeatureModal);
    loadFeatures();
  }
  function featureSwitch(f) {
    var enc = encodeURIComponent(f.name);
    return '<div class="form-check form-switch rl-switch" title="' + (f.enabled ? "点击停用" : "点击启用") + '">'
      + '<input class="form-check-input" type="checkbox" role="switch" aria-label="启用 ' + esc(f.name) + '" ' + (f.enabled ? "checked" : "")
      + ' onchange="Relay.setFeature(decodeURIComponent(' + jsArg(enc) + '), this.checked)"></div>';
  }
  function featureRow(f) {
    var on = Boolean(f.enabled);
    var enc = encodeURIComponent(f.name);
    var kind = FEATURE_KIND[f.kind] || f.kind || "-";
    var ik = f.kind === "script" ? "code" : "zap";
    return '<div class="list-item rl-clickable' + (on ? "" : " rl-dim") + '" data-feature="' + esc(f.name) + '">'
      + '<div class="ico li-ico">' + ico(ik) + "</div>"
      + '<div class="grow"><div class="name">' + esc(f.name) + " " + metaBadge(kind) + " "
      + metaBadge(FEATURE_SCOPE[f.scope] || f.scope, f.scope === "public" ? "badge-info" : "badge-off") + " "
      + metaBadge(FEATURE_IDENTITY[f.identity] || f.identity, "badge-off")
      + (f.class ? " " + metaBadge(f.class, "badge-info") : "")
      + (f.service ? " " + metaBadge(f.service, "badge-off") : "")
      + (f.apps_locked ? " " + metaBadge("手动范围", "badge-warn") : "")
      + (f.confirm ? " " + metaBadge("需确认", "badge-warn") : "") + "</div>"
      + (f.kind === "script" ? '<div class="app-chips">' + scriptAppChips(f) + "</div>" : "")
      + '<div class="desc mono clip" title="' + esc(f.target) + '">' + esc(f.target || "-") + "</div></div>"
      + '<div class="actions">' + featureSwitch(f)
      + (f.kind === "script" ? '<button class="btn btn-ghost btn-sm" onclick="Relay.editPluginOf(' + jsArg(encodeURIComponent(f.target || "")) + ')">编辑插件</button>' : "")
      + '<button class="btn btn-ghost btn-sm" onclick="Relay.featureDetail(' + jsArg(f.name) + ')">详情</button>'
      + '<button class="btn btn-ghost btn-sm" onclick="Relay.runFeature(decodeURIComponent(' + jsArg(enc) + '))">运行</button>'
      + '<button class="btn btn-outline-danger btn-sm" onclick="Relay.delFeature(decodeURIComponent(' + jsArg(enc) + '), ' + jsArg(f.kind || "") + ')">删除</button></div></div>';
  }
  function loadFeatures() {
    loadScriptAppMap(function () { if (state.view === "features") paintFeatureList(); });
    api("/jiying/api/features").then(function (r) {
      var box = $("#featureList");
      if (!box) return;
      if (r.code !== 200) { errorState(box, "toggles", "功能列表加载失败", r.msg || r.message, loadFeatures); return; }
      var list = r.code === 200 && r.data ? r.data : [];
      if (!list.length) {
        box.innerHTML = emptyState("toggles", "暂无功能", "安装 HTTP 功能或添加插件后，会在这里统一管理。", '<button class="btn btn-primary btn-sm" id="featureEmptyAdd">安装功能</button>');
        var b = $("#featureEmptyAdd");
        if (b) b.addEventListener("click", openFeatureModal);
        return;
      }
      state.featureRows = list;
      paintFeatureList();
    }).catch(function () { errorState($("#featureList"), "toggles", "功能列表加载失败", "网络异常，请检查服务状态。", loadFeatures); });
  }
  function paintFeatureList() {
    var box = $("#featureList");
    if (!box || !state.featureRows) return;
    var list = state.featureRows.slice().sort(function (a, b) {
      return String(a.class || "").localeCompare(String(b.class || ""), "zh") || String(a.kind).localeCompare(String(b.kind)) || String(a.name).localeCompare(String(b.name), "zh");
    });
    box.innerHTML = '<div class="list">' + list.map(featureRow).join("") + "</div>";
    box.onclick = function (e) {
      if (e.target.closest("button, input, select, a, label")) return;
      var row = e.target.closest("[data-feature]");
      if (row) featureDetail(row.getAttribute("data-feature"));
    };
    hydrateIcons();
  }
  function setFeature(name, on) {
    api("/jiying/api/feature/toggle", { method: "POST", form: { name: name, enabled: on ? "true" : "false" } }).then(function (r) {
      if (r.code === 200) toast(on ? "已启用 " + name : "已停用 " + name, "ok");
      else toast(r.msg || "操作失败", "err");
      loadFeatures();
    });
  }
  function runFeature(name) {
    showModal("运行功能 · " + name, '<div class="field"><label>参数（可选）</label><input class="input" id="fm_args" placeholder="参数"></div>',
      '<button class="btn btn-ghost" onclick="Relay.closeModal()">取消</button><button class="btn btn-primary" id="fm_run">运行</button>');
    $("#fm_run").addEventListener("click", function () {
      api("/jiying/api/feature/run", { method: "POST", form: { name: name, args: $("#fm_args").value } }).then(function (r) {
        if (r.code === 200) { closeModal(); toast(String(r.data).slice(0, 80), "ok"); }
        else toast(r.msg || "运行失败", "err");
      });
    });
  }
  function featureByName(name) {
    var list = state.featureRows || [];
    for (var i = 0; i < list.length; i++) {
      if (String(list[i].name) === String(name)) return list[i];
    }
    return null;
  }
  function featureAppLabel(id) {
    var known = state.appNames && state.appNames[id];
    if (known && known.name) return known.name;
    var app = appLookup(id);
    return (app && app.name) || shortID(id);
  }
  function featureAppPicker(f) {
    var selected = (f.apps || []).slice();
    var all = !selected.length || selected.indexOf("*") >= 0;
    var ids = [], seen = {};
    function addApp(id) {
      id = String(id || "");
      if (id && id !== "*" && !seen[id]) { seen[id] = 1; ids.push(id); }
    }
    (state.apps || []).forEach(function (a) { addApp(a.app_id); });
    // 未接入连接池时，仍保留功能与插件清单中出现过的应用 ID，避免只能选「全部应用」。
    (state.featureRows || []).forEach(function (row) { (row.apps || []).forEach(addApp); });
    (SCRIPT_CACHE || []).forEach(function (row) { (row.apps || []).forEach(addApp); });
    selected.forEach(addApp);
    var boxes = ids.map(function (id) {
      return '<label class="check app-pick"><input type="checkbox" data-app="' + esc(id) + '"' + (!all && selected.indexOf(id) >= 0 ? " checked" : "") + "> "
        + esc(featureAppLabel(id)) + ' <span class="mono muted">' + esc(shortID(id)) + "</span></label>";
    }).join("");
    if (!boxes) boxes = '<div class="muted" style="font-size:12px">尚未接入应用；接入后可为该功能指定应用范围。</div>';
    return '<div class="field"><label>应用范围</label>'
      + '<label class="check"><input type="checkbox" id="fa_all"' + (all ? " checked" : "") + "> 全部应用</label>"
      + '<div class="app-picker" id="fa_apps">' + boxes + "</div>"
      + '<div class="hint">取消「全部应用」后可勾选部分应用；保存后以手动设置为准，插件重启不会被清单覆盖。</div></div>';
  }
  function featureDetail(name) {
    var nm = String(name || "");
    var f = featureByName(nm);
    if (!f) { toast("功能不存在或列表未加载", "err"); return; }
    var isHttp = f.kind === "http";
    var title = "功能详情 · " + nm;
    var body = '<div class="feature-detail">'
      + '<div class="kv"><span class="k">名称</span><span class="v"><b>' + esc(f.name) + '</b> ' + metaBadge(FEATURE_KIND[f.kind] || f.kind || "-", "badge-off") + "</span></div>"
      + '<div class="field"><label>' + (isHttp ? "HTTP 地址" : "目标") + '</label><input class="input" id="fd_target" value="' + esc(f.target || "") + '"' + (isHttp ? "" : " readonly") + ">"
      + (isHttp ? "" : '<div class="hint">插件目标由插件文件决定，请通过「编辑插件」修改。</div>') + "</div>"
      + '<div class="grid grid-2"><div class="field"><label>权限范围</label><select class="select" id="fd_scope">'
      + '<option value="public"' + (f.scope === "public" ? " selected" : "") + '>public · 公开</option>'
      + '<option value="master"' + (f.scope === "master" ? " selected" : "") + '>master · 主控</option></select></div>'
      + '<div class="field"><label>调用身份</label><select class="select" id="fd_identity">'
      + '<option value="app"' + (f.identity === "app" ? " selected" : "") + '>应用</option>'
      + '<option value="user"' + (f.identity === "user" ? " selected" : "") + '>账号</option></select></div></div>'
      + '<label class="check"><input type="checkbox" id="fd_confirm"' + (f.confirm ? " checked" : "") + "> 调用前需要用户确认</label>"
      + featureAppPicker(f)
      + '<div class="rl-row" style="margin-top:10px">'
      + (f.kind === "script" ? '<button type="button" class="btn btn-ghost btn-sm" id="fd_edit_plugin">编辑插件</button><button type="button" class="btn btn-ghost btn-sm" id="fd_config">插件配置</button>' : "")
      + '<button type="button" class="btn btn-ghost btn-sm" id="fd_reset_apps">' + (f.kind === "script" ? "恢复插件声明范围" : "恢复全部应用") + "</button>"
      + '<div class="spacer"></div><button type="button" class="btn btn-ghost btn-sm" id="fd_run">运行</button></div>'
      + "</div>";
    showModal(title, body,
      '<button class="btn btn-ghost" onclick="Relay.closeModal()">取消</button><button class="btn btn-primary" id="fd_save">保存</button>');
    var allEl = $("#fa_all"), boxes = $$("#fa_apps input[type=checkbox]");
    function syncApps() {
      boxes.forEach(function (b) {
        b.disabled = allEl.checked;
        if (allEl.checked) b.checked = false;
      });
    }
    allEl.addEventListener("change", syncApps);
    syncApps();
    function refresh() {
      closeModal();
      loadFeatures();
      if (state.view === "scripts") loadScripts();
    }
    $("#fd_save").addEventListener("click", function () {
      var apps;
      if (allEl.checked) {
        apps = ["*"];
      } else {
        apps = boxes.filter(function (b) { return b.checked; }).map(function (b) { return b.getAttribute("data-app"); });
        if (!apps.length) { toast("请选择至少一个应用，或勾选全部应用", "err"); return; }
      }
      var form = {
        name: nm,
        scope: $("#fd_scope").value,
        identity: $("#fd_identity").value,
        confirm: $("#fd_confirm").checked ? "true" : "false",
        apps: JSON.stringify(apps)
      };
      if (isHttp) form.target = $("#fd_target").value.trim();
      api("/jiying/api/feature/update", { method: "POST", form: form }).then(function (r) {
        if (r.code === 200) { toast("功能设置已保存", "ok"); refresh(); }
        else toast(r.msg || r.message || "保存失败", "err");
      });
    });
    $("#fd_reset_apps").addEventListener("click", function () {
      api("/jiying/api/feature/update", { method: "POST", form: { name: nm, reset_apps: "true" } }).then(function (r) {
        if (r.code === 200) { toast("应用范围已恢复", "ok"); refresh(); }
        else toast(r.msg || "恢复失败", "err");
      });
    });
    var runBtn = $("#fd_run");
    if (runBtn) runBtn.addEventListener("click", function () { closeModal(); runFeature(nm); });
    var editBtn = $("#fd_edit_plugin");
    if (editBtn) editBtn.addEventListener("click", function () { closeModal(); editPluginOf(encodeURIComponent(f.target || "")); });
    var cfgBtn = $("#fd_config");
    if (cfgBtn) cfgBtn.addEventListener("click", function () { closeModal(); openScriptConfig(encodeURIComponent(f.target || "")); });
  }
  // 插件页直接调整应用范围：按脚本目标找到对应功能后复用功能详情弹窗。
  function editScriptApps(target) {
    var nm = decodeURIComponent(target || "");
    if (!nm) { toast("该插件未关联功能记录", "err"); return; }
    api("/jiying/api/features").then(function (r) {
      var list = (r && r.data) || [];
      var hit = list.filter(function (f) { return String(f.target || "") === nm; })[0];
      if (!hit) { toast("该插件尚未登记为功能", "err"); return; }
      state.featureRows = list;
      featureDetail(hit.name);
    });
  }
  function delFeature(name, kind) {
    var extra = kind === "script" ? "对应插件文件也会一并删除。"
      : kind === "builtin" ? "内置功能会在服务重启后自动恢复。"
      : "";
    confirmDialog("删除功能", "确认删除功能 " + name + " ？" + extra, "删除", function () {
      api("/jiying/api/feature/delete", { method: "POST", form: { name: name } }).then(function (r) {
        if (r.code === 200) { toast("已删除", "ok"); loadFeatures(); }
        else toast(r.msg || "删除失败", "err");
      });
    });
  }
  function openFeatureModal() {
    showModal("安装 HTTP 功能", '<div class="field"><label>功能名</label><input class="input" id="fm_name" placeholder="例如：天气"></div>'
      + '<div class="field"><label>HTTP 地址</label><input class="input" id="fm_url" placeholder="https://example.com/api"></div>',
      '<button class="btn btn-ghost" onclick="Relay.closeModal()">取消</button><button class="btn btn-primary" id="fm_save">安装</button>');
    $("#fm_save").addEventListener("click", function () {
      var name = $("#fm_name").value.trim(), url = $("#fm_url").value.trim();
      if (!name || !url) { toast("请填写功能名与地址", "err"); return; }
      api("/jiying/api/feature/install", { method: "POST", form: { name: name, url: url } }).then(function (r) {
        if (r.code === 200) { closeModal(); toast(r.data || "已安装", "ok"); loadFeatures(); }
        else toast(r.msg || "安装失败", "err");
      });
    });
  }

  /* ---------- Agent 工作台 ---------- */
  function renderAgent() {
    var cont = $("#view-agent");
    cont.innerHTML = '<div class="page-head"><div><h2>Agent 工作台</h2><p>薄 Agent 配置与在线试跑，模型只负责选功能与填参数</p></div></div>'
      + '<div class="grid grid-2">'
      + '<div class="panel"><div class="panel-head"><h3>模型配置</h3><span class="hint">仅主控可改</span></div><div class="panel-body" id="agentCfg">' + loadingState() + '</div></div>'
      + '<div class="panel"><div class="panel-head"><h3>试跑</h3><span class="hint">不经过即应会话</span></div><div class="panel-body">'
      + '<div class="field"><label>会话 ID（可选）</label><input class="input" id="ag_conv" placeholder="留空则使用临时会话"></div>'
      + '<div class="field"><label>提问</label><textarea class="input" id="ag_prompt" rows="4" placeholder="例如：现在系统状态怎么样"></textarea></div>'
      + '<div class="rl-row"><label class="check"><input type="checkbox" id="ag_reset"> 先清空上下文</label><div class="spacer"></div>'
      + '<button class="btn btn-primary btn-sm" id="ag_run">运行</button></div>'
      + '<pre class="mono rl-out" id="ag_out">等待运行…</pre>'
      + "</div></div>"
      + '<div class="panel" style="margin-top:16px"><div class="panel-head"><h3>工具契约</h3><span class="hint">由功能与插件清单生成的 JSON Schema</span></div><div class="panel-body" id="agentTools">' + loadingState() + '</div></div>'
      + "</div>";
    $("#ag_run").addEventListener("click", runAgent);
    loadAgentConfig();
    loadAgentTools();
  }
  function loadAgentConfig() {
    api("/jiying/api/agent/config").then(function (r) {
      var box = $("#agentCfg");
      if (!box) return;
      if (r.code !== 200) { box.innerHTML = emptyState("robot", "无权限或未就绪", r.msg || r.message || "需要主控账号才能查看与修改 Agent 配置。"); return; }
      var d = r.data || {};
      var baseHint = /:\/\/\*+$/.test(String(d.base_url || "")) ? '<div class="hint">已保存地址以掩码显示，保持掩码或留空表示不修改。</div>' : "";
      // 候选项只取本部署已配置过的模型（当前模型 + 后备模型），避免回填与本部署供应商无关的模型名。
      var knownModels = [];
      [d.model].concat(d.fallback_models || []).forEach(function (m) {
        m = String(m || "").trim();
        if (m && knownModels.indexOf(m) < 0) knownModels.push(m);
      });
      var modelList = knownModels.length ? '<datalist id="agentModelOptions">' + knownModels.map(function (m) { return '<option value="' + esc(m) + '"></option>'; }).join("") + "</datalist>" : "";
      box.innerHTML = '<div class="field"><label>启用</label><div class="form-check form-switch rl-switch"><input class="form-check-input" type="checkbox" role="switch" id="ag_enabled" ' + (d.enabled ? "checked" : "") + "></div></div>"
        + '<div class="field"><label>Base URL</label><input class="input" id="ag_base" value="' + esc(d.base_url || "") + '" placeholder="https://api.example.com/v1">' + baseHint + "</div>"
       + '<div class="field"><label>模型</label><input class="input" id="ag_model" list="agentModelOptions" value="' + esc(d.model || "") + '" placeholder="填写服务商支持的模型名">' + modelList + "</div>"
         + '<div class="field"><label>后备模型</label><input class="input" id="ag_fallback" value="' + esc((d.fallback_models || []).join(", ")) + '" placeholder="模型A, 模型B"><div class="hint">主模型不可用时按顺序降级，逗号分隔；留空表示不降级。</div></div>'
        + '<div class="field"><label>API Key</label><input class="input" id="ag_key" type="password" placeholder="' + (d.has_key ? "已配置，留空保持不变" : "尚未配置") + '"></div>'
        + '<div class="field"><label>全局系统提示</label><textarea class="input" id="ag_system" rows="3">' + esc(d.system || "") + "</textarea></div>"
        + '<details class="agent-advanced" open><summary>按应用覆盖系统提示</summary>'
        + '<div class="grid grid-2"><div class="field"><label>应用</label><select class="select" id="ag_app"></select></div>'
        + '<div class="field"><label>该应用的系统提示</label><textarea class="input" id="ag_app_system" rows="3" placeholder="留空则继承全局系统提示"></textarea><div class="hint" id="ag_app_hint">在系统提示后追加该应用的约束与上下文。</div></div></div>'
        + '<div class="rl-row"><div class="spacer"></div><button class="btn btn-ghost btn-sm" id="ag_app_save">保存应用提示</button></div></details>'
         + '<div class="field"><label>上下文消息数</label><input class="input" id="ag_hist" type="number" min="2" max="64" value="' + esc(String(d.max_history || 12)) + '"><div class="hint">每个会话保留的最近消息数量，数值越大越能延续上下文。</div></div>'
         + '<details class="agent-advanced"><summary>高级运行保护</summary>'
         + '<div class="grid grid-2"><div class="field"><label>最大模型轮次</label><input class="input" id="ag_turns" type="number" min="1" max="12" value="' + esc(String(d.max_turns || 6)) + '"><div class="hint">模型连续决策的最大轮次，用于防止异常循环。</div></div>'
         + '<div class="field"><label>工具调用上限</label><input class="input" id="ag_tools" type="number" min="1" max="32" value="' + esc(String(d.max_tools || 8)) + '"><div class="hint">单次运行实际执行的功能调用总数，超出后要求模型直接作答。</div></div></div>'
         + '<div class="field"><label>上下文预算（KB）</label><input class="input" id="ag_ctx" type="number" min="32" max="1024" value="' + esc(String(Math.round((d.max_context_bytes || 131072) / 1024))) + '"><div class="hint">系统提示、历史消息与当前输入的总预算；超出时优先丢弃最旧历史。</div></div>'
         + '<div class="field"><label>模型超时（秒）</label><input class="input" id="ag_timeout" type="number" min="30" max="600" value="' + esc(String(d.timeout_seconds || 150)) + '"><div class="hint">单次模型请求等待上限，默认 150 秒；长回复可适当调大。</div></div></details>'
        + '<div class="rl-row"><div class="spacer"></div><button class="btn btn-primary btn-sm" id="ag_save">保存配置</button></div>';
      $("#ag_save").addEventListener("click", saveAgentConfig);
      $("#ag_app_save").addEventListener("click", saveAgentAppSystem);
      loadAgentAppOptions(d);
    }).catch(function () { errorState($("#agentCfg"), "robot", "Agent 配置加载失败", "网络异常，请检查服务状态。", loadAgentConfig); });
  }
  function loadAgentTools() {
    api("/jiying/api/agent/tools").then(function (r) {
      var box = $("#agentTools");
      if (!box) return;
      if (r.code !== 200) { errorState(box, "toggles", r.code === 403 ? "权限不足" : "工具列表加载失败", r.msg || r.message, loadAgentTools); return; }
      var list = r.data ? r.data : [];
      if (!list.length) { box.innerHTML = emptyState("toggles", "暂无可调用功能", "启用功能或安装插件后，Agent 会自动获得对应工具。", '<button class="btn btn-primary btn-sm" id="agentToolsEmptyGo">去功能管理</button>'); var b = $("#agentToolsEmptyGo"); if (b) b.addEventListener("click", function () { switchView("features"); }); return; }
      box.innerHTML = '<div class="list">' + list.map(function (t) {
        var params = (t.params && t.params.length) ? t.params.join(" / ") : "args";
        var badges = t.scope === "master" ? ' <span class="badge badge-warn">主控</span>' : "";
        if (t.identity === "user") badges += ' <span class="badge badge-info">用户身份</span>';
        return '<div class="list-item"><div class="grow"><div class="name">' + esc(t.feature) + ' <span class="badge badge-off">' + esc(t.tool) + '</span>' + badges + '</div>'
          + '<div class="desc">' + esc(t.description || "") + ' · 参数 ' + esc(params) + '</div></div></div>';
      }).join("") + "</div>";
    }).catch(function () { errorState($("#agentTools"), "toggles", "工具列表加载失败", "网络异常，请检查服务状态。", loadAgentTools); });
  }
  function saveAgentConfig() {
    var form = {
      enabled: $("#ag_enabled").checked ? "true" : "false",
      base_url: $("#ag_base").value.trim(),
      model: $("#ag_model").value.trim(),
      fallback_models: $("#ag_fallback").value.trim(),
      api_key: $("#ag_key").value,
      system: $("#ag_system").value,
      max_turns: $("#ag_turns").value.trim(),
      max_tools: $("#ag_tools").value.trim(),
      max_context_kb: $("#ag_ctx").value.trim(),
      max_history: $("#ag_hist").value.trim(),
      timeout_seconds: $("#ag_timeout").value.trim()
    };
    api("/jiying/api/agent/config", { method: "POST", form: form }).then(function (r) {
      if (r.code === 200) { toast("配置已保存", "ok"); loadAgentConfig(); }
      else toast(r.msg || "保存失败", "err");
    });
  }
  function loadAgentAppOptions(d) {
    var sel = $("#ag_app");
    if (!sel) return;
    function fill(apps) {
      state.apps = apps || [];
      var overrides = (d && d.app_systems) || {};
      var ids = [], seen = {};
      state.apps.forEach(function (a) {
        var id = String(a.app_id || "");
        if (id && !seen[id]) { seen[id] = 1; ids.push(id); }
      });
      Object.keys(overrides).forEach(function (id) {
        if (id && !seen[id]) { seen[id] = 1; ids.push(id); }
      });
      if (!ids.length) {
        sel.innerHTML = '<option value="">暂无应用</option>';
        sel.disabled = true;
        $("#ag_app_system").disabled = true;
        $("#ag_app_save").disabled = true;
        $("#ag_app_hint").textContent = "接入应用后可为其设置独立系统提示。";
        return;
      }
      sel.innerHTML = ids.map(function (id) {
        return '<option value="' + esc(id) + '">' + esc(featureAppLabel(id)) + " · " + esc(shortID(id)) + "</option>";
      }).join("");
      function paint() {
        var id = sel.value;
        $("#ag_app_system").value = overrides[id] || "";
        $("#ag_app_hint").textContent = overrides[id] ? "该应用当前使用独立系统提示；留空保存即恢复继承全局。" : "留空保存即继承全局系统提示。";
      }
      sel.addEventListener("change", paint);
      paint();
    }
    if (state.apps && state.apps.length) { fill(state.apps); return; }
    api("/jiying/api/connections", { silent: true }).then(function (r) {
      fill((r && r.data && r.data.apps) || []);
    }).catch(function () { fill([]); });
  }
  function saveAgentAppSystem() {
    var sel = $("#ag_app");
    if (!sel || !sel.value) { toast("请选择应用", "err"); return; }
    api("/jiying/api/agent/app-system", { method: "POST", form: { app_id: sel.value, system: $("#ag_app_system").value } }).then(function (r) {
      if (r.code === 200) { toast($("#ag_app_system").value.trim() ? "应用系统提示已保存" : "已清除覆盖，继承全局", "ok"); loadAgentConfig(); }
      else toast(r.msg || "保存失败", "err");
    });
  }
  function runAgent() {
    var prompt = $("#ag_prompt").value.trim();
    if (!prompt) { toast("请输入提问内容", "err"); return; }
    var out = $("#ag_out");
    out.textContent = "运行中…";
    api("/jiying/api/agent/run", { method: "POST", form: { conversation_id: $("#ag_conv").value.trim(), prompt: prompt, reset: $("#ag_reset").checked ? "true" : "false" } }).then(function (r) {
      out.textContent = r.code === 200 ? String(r.data) : (r.msg || "运行失败");
    }).catch(function () { out.textContent = "运行失败"; });
  }

  /* ---------- 执行记录 ---------- */
  function renderExecutions() {
    var cont = $("#view-executions");
    var f = state.execFilter || (state.execFilter = { kind: "", status: "", name: "" });
    cont.innerHTML = '<div class="page-head"><div><h2>执行记录</h2><p>定时任务与手动触发的执行结果、耗时和追踪号</p></div><div class="spacer"></div>'
      + '<select class="select" id="execKind" style="width:120px" title="按类型筛选"><option value="">全部类型</option><option value="cron">定时</option><option value="manual">手动</option><option value="flow">流程</option></select>'
      + '<select class="select" id="execStatus" style="width:120px" title="按状态筛选"><option value="">全部状态</option><option value="success">成功</option><option value="failed">失败</option><option value="running">执行中</option><option value="skipped">跳过</option></select>'
      + '<input class="input" id="execName" placeholder="筛选名称 / 命令" style="width:190px">'
      + '<button class="btn btn-ghost btn-sm" onclick="Relay.loadExecutions(false)">刷新</button></div>'
      + '<div class="panel"><div class="table-wrap"><table class="table align-middle"><thead><tr><th class="nowrap" style="width:150px">开始时间</th><th class="nowrap" style="width:80px">类型</th><th style="width:170px">名称</th><th class="nowrap" style="width:88px">状态</th><th class="nowrap" style="width:88px">耗时</th><th>摘要</th></tr></thead><tbody id="execList"><tr><td colspan="6">' + loadingState() + '</td></tr></tbody></table></div></div>'
      + '<div class="muted" id="execMeta" style="margin-top:8px;font-size:12px"></div>';
    var k = $("#execKind"), st = $("#execStatus"), nm = $("#execName");
    if (k) k.value = f.kind;
    if (st) st.value = f.status;
    if (nm) nm.value = f.name;
    function sync() {
      state.execFilter = { kind: k.value, status: st.value, name: nm.value.trim() };
      loadExecutions(false);
    }
    if (k) k.addEventListener("change", sync);
    if (st) st.addEventListener("change", sync);
    if (nm) nm.addEventListener("keydown", function (e) { if (e.key === "Enter") sync(); });
    loadExecutions(false);
  }
  function loadExecutions(silent) {
    var f = state.execFilter || {};
    var qs = "?limit=200&kind=" + encodeURIComponent(f.kind || "") + "&status=" + encodeURIComponent(f.status || "") + "&name=" + encodeURIComponent(f.name || "");
    api("/jiying/api/executions" + qs).then(function (r) {
      var box = $("#execList");
      if (!box) return;
      if (r.code !== 200) {
        state.executions = [];
        state.executionsErr = r.msg || r.message || "请稍后重试。";
        tableError(box, 6, "activity", r.code === 403 ? "权限不足" : "执行记录加载失败", state.executionsErr, function () { state.executionsErr = ""; loadExecutions(false); });
        setExecMeta(0);
        return;
      }
      state.executionsErr = "";
      state.executions = r.data || [];
      paintExecutions();
    }).catch(function () {
      var box = $("#execList");
      if (!box) return;
      state.executions = [];
      state.executionsErr = "网络异常，请稍后重试。";
      tableError(box, 6, "activity", "执行记录加载失败", state.executionsErr, function () { state.executionsErr = ""; loadExecutions(false); });
      setExecMeta(0);
    });
  }
  var EXEC_KIND = { cron: "定时", manual: "手动", flow: "流程" };
  var EXEC_STATUS = {
    success: '<span class="badge badge-ok">成功</span>',
    failed: '<span class="badge badge-err">失败</span>',
    timeout: '<span class="badge badge-err">超时</span>',
    running: '<span class="badge badge-run">执行中</span>',
    skipped: '<span class="badge badge-off">跳过</span>'
  };
  function paintExecutions() {
    var box = $("#execList");
    if (!box || state.executionsErr) return;
    var list = state.executions || [];
    if (!list.length) {
      box.innerHTML = '<tr><td colspan="6">' + emptyState("activity", "暂无执行记录", "定时任务执行或手动触发后会在这里留下结果。") + "</td></tr>";
      setExecMeta(0);
      return;
    }
    box.innerHTML = list.map(function (e) {
      var id = String(e.id || "");
      var detail = String(e.detail || e.error || "").replace(/\s+/g, " ");
      var dur = Number(e.duration_ms || 0);
      var durText = dur >= 1000 ? (dur / 1000).toFixed(1) + " s" : (dur ? dur + " ms" : "-");
      return '<tr class="exec-row" data-exec-id="' + esc(id) + '"><td class="mono nowrap">' + esc(e.started_at || "-") + "</td>"
        + '<td class="nowrap">' + esc(EXEC_KIND[e.kind] || e.kind || "-") + "</td>"
        + '<td class="clip" title="' + esc(e.name || "") + '">' + esc(e.name || "-") + "</td>"
        + '<td class="nowrap">' + (EXEC_STATUS[e.status] || '<span class="badge badge-off">' + esc(e.status || "-") + "</span>") + "</td>"
        + '<td class="mono nowrap">' + esc(durText) + "</td>"
        + '<td class="muted"><div class="clamp2" title="' + esc(detail) + '">' + esc(detail || "-") + "</div></td></tr>";
    }).join("");
    $$(".exec-row", box).forEach(function (tr) {
      tr.addEventListener("click", function () { executionDetail(tr.getAttribute("data-exec-id")); });
    });
    setExecMeta(list.length);
  }
  function setExecMeta(n) {
    var el = $("#execMeta");
    if (el) el.textContent = n ? "共 " + n + " 条（最多保留 500 条）" : "";
  }
  function executionDetail(id) {
    var e = (state.executions || []).filter(function (x) { return String(x.id) === String(id); })[0];
    if (!e) return;
    var rows = [
      ["记录 ID", e.id],
      ["类型", EXEC_KIND[e.kind] || e.kind || "-"],
      ["名称", e.name || "-"],
      ["命令", e.command || "-"],
      ["状态", e.status || "-"],
      ["开始", e.started_at || "-"],
      ["结束", e.finished_at || "-"],
      ["耗时", e.duration_ms ? e.duration_ms + " ms" : "-"],
      ["执行者", e.owner || "-"],
      ["追踪号", e.trace_id || "-"],
      ["摘要", e.detail || "-"],
      ["错误", e.error || "-"]
    ];
    var html = '<table class="table table-sm align-middle" style="margin:0"><tbody>'
      + rows.map(function (kv) { return '<tr><th class="nowrap" style="width:96px">' + esc(kv[0]) + '</th><td class="mono" style="word-break:break-all">' + esc(String(kv[1] == null ? "" : kv[1])) + "</td></tr>"; }).join("")
      + "</tbody></table>";
    var actions = '<button class="btn btn-ghost" data-bs-dismiss="modal">关闭</button>';
    if (e.trace_id) actions += '<button class="btn btn-primary" id="exec_logs">在日志中查找</button>';
    showModal("执行详情", html, actions);
    var btn = $("#exec_logs");
    if (btn) btn.addEventListener("click", function () { closeModal(); searchLogs(e.trace_id); });
  }
  function gotoExecutions(name) {
    state.execFilter = { kind: "", status: "", name: name || "" };
    switchView("executions");
  }

  /* ---------- 定时任务 ---------- */
  function cronCommandLabel(cmd) {
    var raw = String(cmd || "").trim();
    if (raw.charAt(0) === "{") {
      try {
        var o = JSON.parse(raw);
        if (o.type === "feature" || o.feature) {
          var label = "功能 · " + (o.feature || "");
          if (o.args) label += " · " + o.args;
          return label;
        }
        if (o.type === "command" && o.command) return o.command;
      } catch (e) { /* 保持原文 */ }
    }
    return raw;
  }
  // 把常见 cron 规则翻译成一句话，列表里先看人话再看表达式。
  function cronHuman(spec) {
    spec = String(spec || "").trim();
    var known = {
      "*/1 * * * * *": "每秒", "* * * * * *": "每秒",
      "0 * * * * *": "每分钟", "*/5 * * * * *": "每 5 秒", "*/10 * * * * *": "每 10 秒",
      "*/15 * * * * *": "每 15 秒", "*/30 * * * * *": "每 30 秒",
      "0 */5 * * * *": "每 5 分钟", "0 */10 * * * *": "每 10 分钟",
      "0 */15 * * * *": "每 15 分钟", "0 */30 * * * *": "每 30 分钟",
      "0 0 * * * *": "每小时整点", "0 0 9 * * *": "每天 09:00", "0 0 18 * * *": "每天 18:00",
      "0 0 9 * * 1-5": "工作日 09:00", "0 0 9 * * 1": "每周一 09:00",
      "@hourly": "每小时整点", "@daily": "每天 00:00", "@weekly": "每周一 00:00"
    };
    if (known[spec]) return known[spec];
    var p = spec.split(/\s+/);
    var sec, min, hour, dom, mon, dow;
    if (p.length === 5) {
      sec = "0"; min = p[0]; hour = p[1]; dom = p[2]; mon = p[3]; dow = p[4];
    } else if (p.length === 6) {
      sec = p[0]; min = p[1]; hour = p[2]; dom = p[3]; mon = p[4]; dow = p[5];
    } else {
      return spec;
    }
    if (dom !== "*" || mon !== "*") return spec;
    if (sec === "0" && /^\*\/\d+$/.test(min) && hour === "*" && dow === "*") return "每 " + min.slice(2) + " 分钟";
    if (sec === "0" && min === "0" && /^\*\/\d+$/.test(hour) && dow === "*") return "每 " + hour.slice(2) + " 小时";
    if (sec === "0" && /^\d+$/.test(min) && hour === "*" && dow === "*") return "每小时第 " + Number(min) + " 分";
    if (sec === "0" && /^\d+$/.test(min) && /^\d+$/.test(hour)) {
      var hm = ("0" + Number(hour)).slice(-2) + ":" + ("0" + Number(min)).slice(-2);
      if (dow === "*") return "每天 " + hm;
      if (dow === "1-5") return "工作日 " + hm;
      if (/^[0-6]$/.test(dow)) return "每周" + ["日", "一", "二", "三", "四", "五", "六"][Number(dow)] + " " + hm;
    }
    return spec;
  }
  var CRON_PRESETS = [
    ["0 */5 * * * *", "每 5 分钟"],
    ["0 */30 * * * *", "每 30 分钟"],
    ["0 0 * * * *", "每小时整点"],
    ["0 0 9 * * *", "每天 09:00"],
    ["0 0 18 * * *", "每天 18:00"],
    ["0 0 9 * * 1-5", "工作日 09:00"],
    ["0 0 9 * * 1", "每周一 09:00"]
  ];
  function renderCron() {
    var cont = $("#view-cron");
    cont.innerHTML = '<div class="page-head"><div><h2>定时任务</h2><p>按 cron 表达式执行系统命令或已启用功能，结果写入执行记录</p></div><div class="spacer"></div>'
      + '<button class="btn btn-ghost btn-sm" onclick="Relay.gotoExecutions(\'\')">执行记录</button>'
      + '<button class="btn btn-primary" id="newCron">新建定时</button></div>'
      + '<div class="panel"><div class="table-wrap"><table class="table align-middle"><thead><tr><th style="width:150px">名称</th><th class="nowrap" style="width:112px">调度</th><th class="nowrap" style="width:150px">下次运行</th><th style="width:136px">上次结果</th><th class="nowrap" style="width:88px">运行 / 失败</th><th class="nowrap" style="width:76px">状态</th><th class="nowrap text-right" style="width:214px">操作</th></tr></thead><tbody id="cronList"><tr><td colspan="7">' + loadingState() + "</td></tr></tbody></table></div></div>"
      + '<div class="muted" style="margin-top:8px;font-size:12px">连续失败 3 次会自动停用该定时，修正后可手动启用。</div>';
    $("#newCron").addEventListener("click", openCronModal);
    loadCrons();
  }
  function loadCrons() {
    api("/jiying/api/crons").then(function (r) {
      var box = $("#cronList");
      if (!box) return;
      if (r.code !== 200) { tableError(box, 7, "clock", r.code === 403 ? "权限不足" : "定时任务加载失败", r.msg || r.message, loadCrons); return; }
      var list = r.data ? r.data : [];
      state.crons = list;
      if (!list.length) { box.innerHTML = '<tr><td colspan="7">' + emptyState("clock", "暂无定时任务", "新建定时后，中间件会按 cron 表达式自动执行，并记录每次结果。", '<button class="btn btn-primary btn-sm" id="cronEmptyAdd">新建定时</button>') + "</td></tr>"; var b = $("#cronEmptyAdd"); if (b) b.addEventListener("click", openCronModal); return; }
      box.innerHTML = list.map(function (c) {
        var on = Boolean(c.status);
        var last = String(c.last_status || "");
        var badge = last === "success" ? '<span class="badge badge-ok">成功</span>'
          : last === "failed" ? '<span class="badge badge-err">失败</span>'
          : last === "timeout" ? '<span class="badge badge-err">超时</span>'
          : last === "running" ? '<span class="badge badge-run">执行中</span>'
          : c.last_success ? '<span class="badge badge-ok">成功</span>' : '<span class="badge badge-off">未运行</span>';
        var when = c.last_run ? '<div class="muted mono" style="font-size:11px">' + esc(c.last_run) + "</div>" : "";
        var err = c.last_error ? '<div class="muted clamp1" style="font-size:11px;color:var(--err)" title="' + esc(c.last_error) + '">' + esc(c.last_error) + "</div>" : "";
        var fail = Number(c.fail_count || 0);
        var count = Number(c.run_count || 0);
        return '<tr><td class="nowrap"><div>' + esc(c.name) + '</div><div class="muted mono" style="font-size:11px" title="' + esc(cronCommandLabel(c.command)) + '">' + esc(cronCommandLabel(c.command)) + "</div></td>"
          + '<td class="nowrap"><code>' + esc(c.cron) + '</code><div class="muted" style="font-size:11px">' + esc(cronHuman(c.cron)) + "</div></td>"
          + '<td class="nowrap mono">' + esc(c.next_run_at || "-") + "</td>"
          + "<td>" + badge + when + err + "</td>"
          + '<td class="mono nowrap">' + count + " / " + (fail ? '<span style="color:var(--err)">' + fail + "</span>" : "0") + "</td>"
          + '<td class="nowrap">' + (on ? '<span class="badge badge-ok">启用</span>' : '<span class="badge badge-off">停用</span>') + "</td>"
          + '<td class="text-right nowrap"><button class="btn btn-ghost btn-sm" title="立即执行一次" onclick="Relay.runCron(' + jsArg(c.id) + ')">运行一次</button>'
          + '<button class="btn btn-ghost btn-sm" title="查看该任务执行记录" onclick="Relay.gotoExecutions(' + jsArg(c.name) + ')">记录</button>'
          + '<button class="btn btn-ghost btn-sm" onclick="Relay.toggleCron(' + jsArg(c.id) + "," + (on ? "false" : "true") + ')">' + (on ? "停用" : "启用") + "</button>"
          + '<button class="btn btn-ghost btn-sm" onclick="Relay.editCron(' + jsArg(c.id) + ')">编辑</button>'
          + '<button class="btn btn-outline-danger btn-sm" onclick="Relay.delCron(' + jsArg(c.id) + ')">删除</button></td></tr>';
      }).join("");
    }).catch(function () { tableError($("#cronList"), 7, "clock", "定时任务加载失败", "网络异常，请检查服务状态。", loadCrons); });
  }
  function parseCronCommand(cmd) {
    var raw = String(cmd || "").trim();
    var out = { type: "command", command: raw };
    if (raw.charAt(0) === "{") {
      try {
        var o = JSON.parse(raw);
        if (o.type === "feature" || o.feature) {
          out = { type: "feature", feature: o.feature || "", args: o.args || "", conv: o.conversation_id || "" };
        } else if (o.type === "command" && o.command) {
          out = { type: "command", command: o.command };
        }
      } catch (e) { /* 保持纯命令 */ }
    }
    return out;
  }
  function openCronModal(c) {
    var editing = !!c;
    var parsed = parseCronCommand(editing ? c.command : "");
    var initType = parsed.type === "command" ? "command" : "feature";
    var preset = editing ? String(c.cron || "") : CRON_PRESETS[0][0];
    var knownPreset = CRON_PRESETS.some(function (p) { return p[0] === preset; });
    var selectPreset = editing && !knownPreset ? "custom" : preset;
    showModal(editing ? "编辑定时任务" : "新建定时任务", '<div class="field"><label>名称</label><input class="input" id="cm_name" placeholder="每日额度重置" value="' + esc(editing ? c.name : "") + '"></div>'
      + '<div class="field"><label>执行频率</label><select class="select" id="cm_preset">'
      + CRON_PRESETS.map(function (p) { return '<option value="' + esc(p[0]) + '"' + (p[0] === selectPreset ? " selected" : "") + '>' + esc(p[1]) + "</option>"; }).join("")
      + '<option value="custom"' + (selectPreset === "custom" ? " selected" : "") + '>自定义</option></select><div class="hint" id="cm_preset_hint"></div></div>'
      + '<div class="field" id="cm_cron_box" hidden><label>Cron 表达式</label><input class="input" id="cm_cron" placeholder="0 5 * * * *" value="' + esc(selectedCronValue(selectPreset, preset)) + '"><div class="hint">秒 分 时 日 月 周，例如 <code>0 5 * * * *</code> 表示每小时第 5 分钟。</div></div>'
      + '<div class="field"><label>执行类型</label><select class="select" id="cm_type"><option value="feature"' + (initType === "feature" ? " selected" : "") + '>已启用功能</option><option value="command"' + (initType === "command" ? " selected" : "") + '>系统命令</option></select></div>'
      + '<div id="cm_feature_box"><div class="field"><label>功能名</label><input class="input" id="cm_feature" list="cm_features" placeholder="选择或输入已启用功能" value="' + esc(parsed.feature || "") + '"><datalist id="cm_features"></datalist></div>'
      + '<div class="field"><label>功能参数（可选）</label><input class="input" id="cm_args" placeholder="例如：重置" value="' + esc(parsed.args || "") + '"></div>'
      + '<div class="field"><label>会话 ID（可选）</label><input class="input" id="cm_conv" placeholder="功能需要会话上下文时填写" value="' + esc(parsed.conv || "") + '"></div></div>'
      + '<div id="cm_command_box" hidden><div class="field"><label>系统命令</label><input class="input" id="cm_command" placeholder="status" value="' + esc(parsed.command || "") + '"></div><div class="hint">管理员命令仅主人角色可创建与运行。</div></div>'
      + '<pre class="rl-codeblock" id="cm_preview" style="margin:0;max-height:120px"></pre>',
      '<button class="btn btn-ghost" onclick="Relay.closeModal()">取消</button><button class="btn btn-primary" id="cm_save">' + (editing ? "保存" : "创建") + "</button>");
    var typeEl = $("#cm_type"), fbox = $("#cm_feature_box"), cbox = $("#cm_command_box"), prev = $("#cm_preview");
    var cronEl = $("#cm_cron"), presetEl = $("#cm_preset"), cronBox = $("#cm_cron_box"), presetHint = $("#cm_preset_hint");
    function syncCron() {
      var custom = presetEl.value === "custom";
      cronBox.hidden = !custom;
      if (!custom) cronEl.value = presetEl.value;
      presetHint.textContent = (custom ? "按秒 分 时 日 月 周填写" : presetEl.options[presetEl.selectedIndex].text + "执行一次");
      syncPreview();
    }
    presetEl.addEventListener("change", syncCron);
    function syncType() {
      var isFeature = typeEl.value === "feature";
      fbox.hidden = !isFeature;
      cbox.hidden = isFeature;
      syncPreview();
    }
    function syncPreview() {
      if (!prev) return;
      prev.textContent = buildCronCommand();
    }
    function buildCronCommand() {
      if (typeEl.value === "command") return ($("#cm_command").value || "").trim();
      var feature = ($("#cm_feature").value || "").trim();
      if (!feature) return "";
      var spec = { type: "feature", feature: feature };
      var args = ($("#cm_args").value || "").trim();
      var conv = ($("#cm_conv").value || "").trim();
      if (args) spec.args = args;
      if (conv) spec.conversation_id = conv;
      return JSON.stringify(spec);
    }
    typeEl.addEventListener("change", syncType);
    ["cm_feature", "cm_args", "cm_conv", "cm_command"].forEach(function (id) {
      var el = $("#" + id);
      if (el) el.addEventListener("input", syncPreview);
    });
    syncCron();
    syncType();
    api("/jiying/api/features").then(function (r) {
      var dl = $("#cm_features");
      if (!dl || r.code !== 200 || !r.data) return;
      dl.innerHTML = r.data.filter(function (f) { return f.enabled; }).map(function (f) { return '<option value="' + esc(f.name) + '">' + esc(f.kind || "") + "</option>"; }).join("");
    }).catch(function () {});
    $("#cm_save").addEventListener("click", function () {
      var name = $("#cm_name").value.trim(), cron = cronEl.value.trim(), cmd = buildCronCommand();
      if (!name || !cron || !cmd) { toast("请填写名称、执行频率与执行内容", "err"); return; }
      var form = { name: name, cron: cron, command: cmd };
      var url = "/jiying/api/cron/add";
      if (editing) { url = "/jiying/api/cron/update"; form.id = c.id; }
      api(url, { method: "POST", form: form }).then(function (r) {
        if (r.code === 200) { closeModal(); toast(editing ? "已保存" : "已创建", "ok"); loadCrons(); }
        else toast(r.msg || (editing ? "保存失败" : "创建失败"), "err");
      });
    });
  }
  function selectedCronValue(preset, custom) {
    return preset === "custom" ? custom : preset;
  }
  function runCron(id) {
    api("/jiying/api/cron/run", { method: "POST", form: { id: id } }).then(function (r) {
      if (r.code === 200) {
        toast("已触发执行，稍后可在执行记录查看结果", "ok");
        setTimeout(loadCrons, 1500);
      } else toast(r.msg || "触发失败", "err");
    });
  }
  function toggleCron(id, on) {
    api("/jiying/api/cron/toggle", { method: "POST", form: { id: id, enabled: on ? "true" : "false" } }).then(function (r) {
      if (r.code === 200) { toast(on ? "已启用" : "已停用", "ok"); loadCrons(); }
      else toast(r.msg || "操作失败", "err");
    });
  }
  function editCron(id) {
    var c = (state.crons || []).filter(function (x) { return String(x.id) === String(id); })[0];
    if (!c) { toast("定时任务已不在列表中，请刷新", "err"); return; }
    openCronModal(c);
  }
  function delCron(id) {
    confirmDialog("删除定时任务", "确认删除该定时任务？", "删除", function () {
      api("/jiying/api/cron/delete", { method: "POST", form: { id: id } }).then(function (r) { toast(r.code === 200 ? "已删除" : r.msg, r.code === 200 ? "ok" : "err"); loadCrons(); });
    });
  }

  /* ---------- 任务队列 ---------- */
  function renderTasks() {
    var cont = $("#view-tasks");
    cont.innerHTML = '<div class="page-head"><div><h2>任务队列</h2><p>功能执行记录与状态</p></div><div class="spacer"></div>'
      + '<select class="select" id="taskStatus" style="width:130px" title="按状态筛选"><option value="">全部状态</option><option value="failed">失败 / 超时</option><option value="running">执行中</option><option value="pending">排队中</option><option value="waiting">待确认</option><option value="done">完成</option><option value="canceled">已取消</option></select>'
      + '<label class="muted nowrap" style="font-size:12px;display:inline-flex;align-items:center;gap:6px"><input type="checkbox" id="taskSilent"> 含 Agent 调用</label>'
      + '<input class="input" id="taskFilter" placeholder="筛选功能 / 会话 / 结果" style="width:200px">'
      + '<button class="btn btn-ghost btn-sm" onclick="Relay.loadTasks()">刷新</button></div>'
      + '<div class="panel"><div class="table-wrap"><table class="table align-middle"><thead><tr><th style="width:128px">ID</th><th class="nowrap" style="width:88px">功能</th><th style="width:120px">会话</th><th class="nowrap" style="width:88px">状态</th><th>结果</th><th class="nowrap text-right" style="width:120px">操作</th></tr></thead><tbody id="taskList"><tr><td colspan="6">' + loadingState() + '</td></tr></tbody></table></div></div>'
      + '<div class="muted" id="taskMeta" style="margin-top:8px;font-size:12px"></div>';
    $("#taskStatus").addEventListener("change", paintTasks);
    $("#taskFilter").addEventListener("input", paintTasks);
    $("#taskSilent").addEventListener("change", loadTasks);
    Relay.loadTasks();
  }
  // 状态筛选把 timeout 归入「失败」，与徽标语义一致。
  var TASK_STATUS_GROUP = { failed: ["failed", "timeout"], waiting: ["waiting", "waiting_confirmation"] };
  function taskMatchStatus(t, want) {
    if (!want) return true;
    return (TASK_STATUS_GROUP[want] || [want]).indexOf(String(t.status)) >= 0;
  }
  function loadTasks() {
    var withSilent = !!($("#taskSilent") && $("#taskSilent").checked);
    var silentReq = withSilent ? api("/jiying/api/tasks?silent=1&limit=50") : Promise.resolve(null);
    Promise.all([api("/jiying/api/tasks"), silentReq]).then(function (res) {
      var r = res[0], sr = res[1];
      var box = $("#taskList");
      if (!box) return;
      if (r.code !== 200) {
        state.tasks = [];
        state.silentTasks = [];
        state.tasksErr = r.msg || r.message || "请稍后重试。";
        tableError(box, 6, "list", r.code === 403 ? "权限不足" : "任务列表加载失败", state.tasksErr, function () { state.tasksErr = ""; loadTasks(); });
        setTaskMeta(0, 0);
        return;
      }
      state.tasksErr = "";
      state.tasks = r.data ? r.data : [];
      state.silentTasks = (sr && sr.code === 200 && sr.data) ? sr.data : [];
      paintTasks();
    }).catch(function () {
      var box = $("#taskList");
      if (!box) return;
      state.tasks = [];
      state.silentTasks = [];
      state.tasksErr = "网络异常，请稍后重试。";
      tableError(box, 6, "list", "任务列表加载失败", state.tasksErr, function () { state.tasksErr = ""; loadTasks(); });
      setTaskMeta(0, 0);
    });
  }
  function paintTasks() {
    var box = $("#taskList");
    if (!box) return;
    if (state.tasksErr) return;
    var all = (state.tasks || []).concat(state.silentTasks || []);
    all.sort(function (a, b) { return String(b.created_at || "").localeCompare(String(a.created_at || "")); });
    if (!all.length) { box.innerHTML = '<tr><td colspan="6">' + emptyState("list", "暂无任务", "功能执行后会在这里留下记录。") + "</td></tr>"; setTaskMeta(0, 0); return; }
    var want = (($("#taskStatus") || {}).value || "");
    var kw = (($("#taskFilter") || {}).value || "").trim().toLowerCase();
    var shown = all.filter(function (t) {
      if (!taskMatchStatus(t, want)) return false;
      if (!kw) return true;
      return [t.feature, t.conversation_id, t.result, t.error_code, t.id].some(function (v) { return String(v || "").toLowerCase().indexOf(kw) >= 0; });
    });
    box.innerHTML = shown.map(function (t) {
        var stb = { done: '<span class="badge badge-ok">完成</span>', running: '<span class="badge badge-run">执行中</span>', pending: '<span class="badge badge-warn">排队中</span>', waiting: '<span class="badge badge-warn">待确认</span>', waiting_confirmation: '<span class="badge badge-warn">待确认</span>', failed: '<span class="badge badge-err">失败</span>', timeout: '<span class="badge badge-err">超时</span>', canceled: '<span class="badge badge-off">已取消</span>' }[t.status] || '<span class="badge badge-off">' + esc(t.status) + "</span>";
        var tid = String(t.id || "-");
        var cid = String(t.conversation_id || "");
        var res = String(t.result || "");
        var isAgent = !!t.silent;
        var featureCell = esc(t.feature) + (isAgent ? ' <span class="badge badge-run" title="Agent 工具调用">Agent</span>' : "");
        var actions = isAgent
          ? '<button class="btn btn-ghost btn-sm" onclick="Relay.taskDetail(' + jsArg(tid) + ')">详情</button>'
          : (t.status === "failed" ? '<button class="btn btn-ghost btn-sm" onclick="Relay.retryTask(' + jsArg(tid) + ')">重试</button> ' : "") + '<button class="btn btn-outline-danger btn-sm" onclick="Relay.delTask(' + jsArg(tid) + ')">删除</button>';
        return '<tr><td class="mono clip" title="' + esc(tid) + '"><a href="javascript:void(0)" onclick="Relay.taskDetail(' + jsArg(tid) + ')">' + esc(shortID(tid)) + "</a></td>"
          + '<td class="nowrap">' + featureCell + "</td>"
          + '<td class="mono clip" title="' + esc(cid) + '">' + esc(cid ? shortID(cid) : "-") + "</td>"
          + '<td class="nowrap">' + stb + "</td>"
          + '<td class="muted"><div class="clamp2" title="' + esc(res) + '">' + esc(res) + "</div><small class=\"muted\">" + esc(t.error_code || "") + (t.duration_ms ? " · " + esc(String(t.duration_ms)) + " ms" : "") + "</small></td>"
          + '<td class="text-right nowrap">' + actions + "</td></tr>";
      }).join("") || '<tr><td colspan="6"><div class="empty">没有符合筛选条件的任务</div></td></tr>';
    setTaskMeta(shown.length, all.length);
  }
  function setTaskMeta(shown, total) {
    var el = $("#taskMeta");
    if (!el) return;
    if (!total) { el.textContent = ""; return; }
    // 接口默认只取最近 100 条，筛选范围受此限制，需在计数里说明。
    var scope = total >= 100 ? "（最近 100 条）" : "";
    el.textContent = shown === total ? "共 " + total + " 条" + scope : "匹配 " + shown + " / " + total + " 条" + scope;
  }
  function retryTask(id) { api("/jiying/api/task/retry", { method: "POST", form: { id: id } }).then(function (r) { toast(r.code === 200 ? "已重试" : r.msg, r.code === 200 ? "ok" : "err"); loadTasks(); }); }
  function taskDetail(id) {
    var t = (state.tasks || []).concat(state.silentTasks || []).filter(function (x) { return String(x.id) === String(id); })[0];
    if (!t) { toast("任务已不在列表中，请刷新", "err"); return; }
    var rows = [
      ["任务 ID", t.id],
      ["功能", t.feature],
      ["状态", t.status],
      ["会话", t.conversation_id || "-"],
      ["用户", t.user_id || "-"],
      ["创建时间", fmtTime(t.created_at)],
      ["开始时间", fmtTime(t.started_at)],
      ["结束时间", fmtTime(t.finished_at)],
      ["耗时", t.duration_ms ? t.duration_ms + " ms" : "-"],
      ["重试次数", t.retry_count == null ? "-" : String(t.retry_count)],
      ["错误码", t.error_code || "-"],
      ["结果", t.result || "-"]
    ];
    var html = '<table class="table table-sm align-middle" style="margin:0"><tbody>'
      + rows.map(function (kv) { return '<tr><th class="nowrap" style="width:110px">' + esc(kv[0]) + '</th><td class="mono" style="word-break:break-all">' + esc(String(kv[1])) + "</td></tr>"; }).join("")
      + "</tbody></table>";
    var kw = String(t.id || t.feature || "").slice(0, 12);
    showModal("任务详情",
      html,
      '<button class="btn btn-ghost" data-bs-dismiss="modal">关闭</button>'
      + '<button class="btn btn-primary" id="rl_task_logs">在日志中查找</button>');
    var btn = $("#rl_task_logs");
    if (btn) btn.addEventListener("click", function () { closeModal(); searchLogs(kw); });
  }
  function delTask(id) {
    confirmDialog("删除任务", "确认删除任务？", "删除", function () {
      api("/jiying/api/task/delete", { method: "POST", form: { id: id } }).then(function (r) { toast(r.code === 200 ? "已删除" : r.msg, r.code === 200 ? "ok" : "err"); loadTasks(); });
    });
  }

  /* ---------- 运行日志 ---------- */
  var LOG_LEVEL_LABEL = { error: "错误", warn: "警告", info: "信息", debug: "调试", log: "常规" };
  function renderLogs() {
    stopLogAuto();
    var cont = $("#view-logs");
    state.logLevel = state.logLevel || "all";
    state.logPinned = true;
    state.logFromBottom = 0;
    var day = state.logDay || today();
    cont.innerHTML = '<div class="page-head"><div><h2>运行日志</h2><p>按级别、追踪号与关键字检索；点击 trace 可反向筛选整条链路</p></div><div class="spacer"></div>'
      + '<label class="log-auto muted"><input type="checkbox" id="logAuto"> 自动刷新</label>'
      + '<button class="btn btn-ghost btn-sm" id="logExport">导出当前视图</button>'
      + '<button class="btn btn-primary btn-sm" id="logReload">刷新</button></div>'
      + '<div class="panel log-panel">'
      + '<div class="log-toolbar"><div class="log-filters">'
      + '<label class="log-field"><span>日期</span><input class="input" id="logDay" type="date"></label>'
      + '<label class="log-field"><span>行数</span><select class="select" id="logTail"><option>100</option><option>200</option><option selected>500</option><option>1000</option></select></label>'
      + '<label class="log-field log-search"><span>搜索</span><input class="input" id="logFilter" placeholder="关键字 / trace / 路径" spellcheck="false"></label>'
      + '<label class="log-check"><input type="checkbox" id="logWrap"> 自动换行</label>'
      + '</div><div class="log-stats" id="logStats"></div></div>'
      + '<div class="log-vp"><div class="log-viewport" id="logBox" role="log" aria-live="polite" aria-label="运行日志"></div>'
      + '<button type="button" class="log-latest" id="logLatest" hidden>回到底部</button></div>'
      + '<div class="log-foot"><span id="logMeta">加载中…</span><span class="hint">双击日志行可复制 · 点击 trace 可按追踪号筛选</span></div>'
      + '</div>';
    $("#logDay").value = day;
    $("#logReload").addEventListener("click", loadLogs);
    $("#logDay").addEventListener("change", function () { state.logDay = this.value; loadLogs(); });
    $("#logTail").addEventListener("change", loadLogs);
    $("#logExport").addEventListener("click", exportLogs);
    $("#logAuto").addEventListener("change", function () { setLogAuto(this.checked); });
    $("#logWrap").addEventListener("change", function () { setLogWrap(this.checked); });
    try { if (localStorage.getItem("relay-log-wrap") === "1") { $("#logWrap").checked = true; setLogWrap(true); } } catch (e) { /* 忽略存储失败 */ }
    var filter = $("#logFilter"), ft = null;
    filter.addEventListener("input", function () {
      clearTimeout(ft);
      ft = setTimeout(function () { paintLogs(true); }, 120);
    });
    var box = $("#logBox");
    box.addEventListener("scroll", updateLogPinned, { passive: true });
    box.addEventListener("click", function (e) {
      var trace = e.target.closest(".log-trace");
      if (!trace) return;
      e.preventDefault();
      var value = trace.getAttribute("data-trace") || "";
      if (!value) return;
      state.logPending = value;
      filter.value = value;
      paintLogs(true);
    });
    box.addEventListener("dblclick", function (e) {
      if (e.target.closest(".log-trace")) return;
      var row = e.target.closest(".log-row");
      if (!row) return;
      var raw = (state.logShown || [])[Number(row.getAttribute("data-i")) || 0];
      if (raw == null) return;
      copyText(String(raw));
    });
    $("#logLatest").addEventListener("click", function () {
      var b = $("#logBox");
      if (b) b.scrollTop = b.scrollHeight;
      state.logPinned = true;
      updateLogPinned();
    });
    loadLogs();
  }
  function setLogWrap(on) {
    var box = $("#logBox");
    if (box) box.classList.toggle("rl-logbox-wrap", !!on);
    try { localStorage.setItem("relay-log-wrap", on ? "1" : "0"); } catch (e) { /* 忽略存储失败 */ }
  }
  function setLogAuto(on) {
    stopLogAuto();
    if (on) { state.logTimer = setInterval(loadLogs, 5000); loadLogs(); }
  }
  function stopLogAuto() {
    if (state.logTimer) { clearInterval(state.logTimer); state.logTimer = null; }
  }
  function today() {
    var d = new Date();
    return d.getFullYear() + "-" + String(d.getMonth() + 1).padStart(2, "0") + "-" + String(d.getDate()).padStart(2, "0");
  }
  function fmtClock(d) {
    function p(n) { return String(n).padStart(2, "0"); }
    return p(d.getHours()) + ":" + p(d.getMinutes()) + ":" + p(d.getSeconds());
  }
  // 后端时间多为 RFC3339（UTC），统一转成本地「YYYY-MM-DD HH:mm:ss」以对齐服务器时间展示；无法解析则原样返回。
  function fmtTime(s) {
    s = String(s == null ? "" : s).trim();
    if (!s) return "-";
    var d = new Date(s);
    if (isNaN(d.getTime())) return s;
    function p(n) { return String(n).padStart(2, "0"); }
    return d.getFullYear() + "-" + p(d.getMonth() + 1) + "-" + p(d.getDate()) + " " + p(d.getHours()) + ":" + p(d.getMinutes()) + ":" + p(d.getSeconds());
  }
  function loadLogs() {
    var dayEl = $("#logDay");
    if (!dayEl) return;
    var box = $("#logBox");
    if (box) {
      state.logFromBottom = Math.max(0, box.scrollHeight - box.scrollTop - box.clientHeight);
      state.logPinned = state.logFromBottom <= 32;
      box.setAttribute("aria-busy", "true");
    }
    var day = (dayEl.value || today()).replace(/-/g, "");
    state.logDay = dayEl.value || today();
    var tail = $("#logTail").value;
    api("/jiying/api/logs?day=" + day + "&tail=" + tail).then(function (r) {
      var b = $("#logBox");
      if (!b) return;
      b.removeAttribute("aria-busy");
      if (r.code !== 200) {
        state.logLines = [];
        state.logShown = [];
        b.innerHTML = '<div class="log-empty"><b>日志加载失败</b><p>' + esc(r.msg || "请稍后重试。") + '</p><button type="button" class="btn btn-ghost btn-sm" id="logRetry">重试</button></div>';
        var retry = $("#logRetry");
        if (retry) retry.addEventListener("click", loadLogs);
        var meta = $("#logMeta");
        if (meta) meta.textContent = "加载失败";
        renderLogStats({ all: 0, error: 0, warn: 0, info: 0, debug: 0 }, 0);
        return;
      }
      state.logLines = r.data.lines || [];
      if (state.logPending) { var f = $("#logFilter"); if (f) f.value = state.logPending; state.logPending = ""; }
      state.logLoadedAt = fmtClock(new Date());
      paintLogs(false);
    }).catch(function () {
      var b = $("#logBox");
      if (!b) return;
      b.removeAttribute("aria-busy");
      state.logLines = [];
      state.logShown = [];
      b.innerHTML = '<div class="log-empty"><b>日志加载失败</b><p>网络异常，请检查服务状态。</p><button type="button" class="btn btn-ghost btn-sm" id="logRetry">重试</button></div>';
      var retry = $("#logRetry");
      if (retry) retry.addEventListener("click", loadLogs);
      var meta = $("#logMeta");
      if (meta) meta.textContent = "加载失败";
      renderLogStats({ all: 0, error: 0, warn: 0, info: 0, debug: 0 }, 0);
    });
  }
  function searchLogs(kw) {
    state.logPending = String(kw || "");
    if (state.view === "logs") {
      var f = $("#logFilter");
      if (f) f.value = state.logPending;
      state.logPending = "";
      paintLogs(true);
    } else switchView("logs");
  }
  function parseLogLine(raw) {
    var m = String(raw || "").match(/^\[\[\s*([^\]]+?)\s*\]\s*\[\s*([A-Za-z]+)\s*\]\s*\[\s*([^\]]*?)\s*\]\s*([\s\S]*?)\]\s*$/);
    if (!m) {
      var level = /(\bFATAL\b|\bPANIC\b)/.test(raw) ? "fatal" : (/\bERROR\b/.test(raw) ? "error" : (/\bWARN(ING)?\b/.test(raw) ? "warn" : "log"));
      return { raw: true, time: "", clock: "", level: level, src: "", srcFull: "", msg: String(raw || "") };
    }
    var time = m[1].trim();
    var srcFull = m[3].trim();
    return {
      raw: false,
      time: time,
      clock: time.split(/\s+/).pop() || time,
      level: m[2].toLowerCase(),
      src: srcFull.replace(/^.*[\\/]/, "").replace(/\s*:\s*/g, ":"),
      srcFull: srcFull,
      msg: m[4].trim()
    };
  }
  function logLevelCounts(lines) {
    var counts = { error: 0, warn: 0, info: 0, debug: 0, other: 0 };
    lines.forEach(function (l) {
      var lv = parseLogLine(l).level;
      if (lv === "fatal") counts.error++;
      else if (lv === "error") counts.error++;
      else if (lv === "warn" || lv === "warning") counts.warn++;
      else if (lv === "info") counts.info++;
      else if (lv === "debug" || lv === "trace") counts.debug++;
      else counts.other++;
    });
    return counts;
  }
  function renderLogStats(counts, total) {
    var box = $("#logStats");
    if (!box) return;
    var active = state.logLevel || "all";
    var chips = [
      ["all", "全部", total],
      ["issue", "异常", counts.error + counts.warn],
      ["error", "错误", counts.error],
      ["warn", "警告", counts.warn],
      ["info", "信息", counts.info],
      ["debug", "调试", counts.debug]
    ];
    box.innerHTML = chips.map(function (c) {
      return '<button type="button" class="log-chip' + (active === c[0] ? " active" : "") + '" data-level="' + c[0] + '">' + c[1] + ' <b>' + c[2] + "</b></button>";
    }).join("");
    $$(".log-chip", box).forEach(function (b) {
      b.addEventListener("click", function () {
        state.logLevel = b.getAttribute("data-level");
        paintLogs(true);
      });
    });
  }
  function logLevelPass(level, want) {
    if (!want || want === "all") return true;
    var lv = level === "fatal" ? "error" : (level === "warning" ? "warn" : level);
    if (want === "issue") return lv === "error" || lv === "warn";
    return lv === want;
  }
  function logHighlight(text, kw) {
    text = String(text == null ? "" : text);
    if (!kw) return esc(text);
    var low = text.toLowerCase(), needle = kw.toLowerCase(), out = "", pos = 0, idx;
    if (!needle) return esc(text);
    while ((idx = low.indexOf(needle, pos)) >= 0) {
      out += esc(text.slice(pos, idx)) + "<mark>" + esc(text.slice(idx, idx + needle.length)) + "</mark>";
      pos = idx + needle.length;
    }
    return out + esc(text.slice(pos));
  }
  function logTraceHTML(trace, kw) {
    return '<button type="button" class="log-trace" data-trace="' + esc(trace) + '" title="按此追踪号筛选">trace=' + logHighlight(trace, kw) + "</button>";
  }
  function logMessageHTML(msg, kw) {
    var m = msg.match(/^(?:gin access\s+)?(GET|POST|PUT|PATCH|DELETE|HEAD|OPTIONS)\s+(\S+)\s+(\d{3})\s+(\S+)\s+trace=(\S+)\s*$/i);
    if (!m) return logHighlight(msg, kw);
    var code = Number(m[3]);
    var cls = code >= 500 ? "s-err" : code >= 400 ? "s-warn" : code >= 300 ? "s-info" : "s-ok";
    return '<span class="log-method">' + esc(m[1].toUpperCase()) + '</span> <span class="log-path">' + logHighlight(m[2], kw) + '</span> <span class="log-status ' + cls + '">' + esc(m[3]) + '</span> <span class="log-dur">' + esc(m[4]) + "</span> " + logTraceHTML(m[5], kw);
  }
  function logRowHTML(raw, i, kw) {
    var p = parseLogLine(raw);
    if (p.raw) {
      return '<div class="log-row lv-' + esc(p.level) + '" data-i="' + i + '" title="双击复制该行"><span class="log-no">' + (i + 1) + '</span><span class="log-msg">' + logHighlight(p.msg, kw) + "</span></div>";
    }
    var lv = p.level === "warning" ? "warn" : p.level;
    return '<div class="log-row lv-' + esc(lv) + '" data-i="' + i + '" title="双击复制该行">'
      + '<span class="log-no">' + (i + 1) + '</span>'
      + '<span class="log-time" title="' + esc(p.time) + '">' + esc(p.clock) + '</span>'
      + '<span class="log-level">' + esc(p.level.toUpperCase()) + '</span>'
      + '<span class="log-src" title="' + esc(p.srcFull) + '">' + esc(p.src) + '</span>'
      + '<span class="log-msg">' + logMessageHTML(p.msg, kw) + "</span></div>";
  }
  function paintLogs(resetToTop) {
    var box = $("#logBox");
    if (!box) return;
    var kw = (($("#logFilter") || {}).value || "").trim();
    var lower = kw.toLowerCase();
    var base = lower ? state.logLines.filter(function (l) { return String(l).toLowerCase().indexOf(lower) >= 0; }) : state.logLines;
    var counts = logLevelCounts(base);
    renderLogStats({ all: base.length, error: counts.error, warn: counts.warn, info: counts.info, debug: counts.debug }, state.logLines.length);
    var want = state.logLevel || "all";
    var shown = base.filter(function (l) { return logLevelPass(parseLogLine(l).level, want); });
    state.logShown = shown;
    if (!shown.length) {
      box.innerHTML = '<div class="log-empty"><b>' + (state.logLines.length ? "无匹配日志" : "当日暂无日志") + "</b><p>" + esc(state.logLines.length ? "调整级别或关键字后重试。" : "切换日期或刷新后查看。") + "</p></div>";
    } else {
      box.innerHTML = shown.map(function (l, i) { return logRowHTML(l, i, lower); }).join("");
    }
    if (resetToTop) {
      box.scrollTop = 0;
      state.logPinned = false;
    } else if (state.logPinned) {
      box.scrollTop = box.scrollHeight;
    } else {
      box.scrollTop = Math.max(0, box.scrollHeight - box.clientHeight - (state.logFromBottom || 0));
    }
    var meta = $("#logMeta");
    if (meta) {
      var text = kw || want !== "all"
        ? (kw && want !== "all" ? "匹配 " + shown.length + " / 共 " + state.logLines.length + " 行" : (kw ? "匹配 " + shown.length + " / 共 " + state.logLines.length + " 行" : "筛选 " + shown.length + " / 共 " + state.logLines.length + " 行"))
        : "共 " + state.logLines.length + " 行";
      if (state.logLoadedAt) text += " · 最后刷新 " + state.logLoadedAt;
      meta.textContent = text;
    }
    updateLogPinned();
  }
  function updateLogPinned() {
    var box = $("#logBox"), btn = $("#logLatest");
    if (!box) return;
    var fromBottom = Math.max(0, box.scrollHeight - box.scrollTop - box.clientHeight);
    state.logFromBottom = fromBottom;
    state.logPinned = fromBottom <= 32;
    if (btn) btn.hidden = state.logPinned;
  }
  function exportLogs() {
    var lines = state.logShown || [];
    if (!lines.length) { toast("暂无日志可导出", "err"); return; }
    var day = ($("#logDay").value || today());
    downloadBlob(new Blob([lines.join("\n") + "\n"], { type: "text/plain;charset=utf-8" }), "relay-logs-" + day + ".log");
    toast("已导出当前视图 " + lines.length + " 行", "ok");
  }
  function copyText(text) {
    text = String(text == null ? "" : text);
    var done = function () { toast("已复制该行日志", "ok"); };
    if (navigator.clipboard && navigator.clipboard.writeText) {
      navigator.clipboard.writeText(text).then(done).catch(function () { copyTextFallback(text, done); });
      return;
    }
    copyTextFallback(text, done);
  }
  function copyTextFallback(text, done) {
    var ta = document.createElement("textarea");
    ta.value = text;
    ta.setAttribute("readonly", "");
    ta.style.position = "fixed";
    ta.style.top = "-1000px";
    ta.style.opacity = "0";
    document.body.appendChild(ta);
    ta.select();
    try { document.execCommand("copy"); done(); }
    catch (e) { toast("复制失败，请手动选择文本", "err"); }
    document.body.removeChild(ta);
  }
  function downloadBlob(blob, name) {
    var url = URL.createObjectURL(blob);
    var a = document.createElement("a");
    a.href = url;
    a.download = name;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    setTimeout(function () { URL.revokeObjectURL(url); }, 1000);
  }

  /* ---------- 连接池 ---------- */
  function renderPool() {
    var cont = $("#view-pool");
    cont.innerHTML = '<div class="page-head"><div><h2>连接池</h2><p>多应用通道与个人账号状态，按应用隔离运行</p></div><div class="spacer"></div><button class="btn btn-primary" id="newApp">接入应用</button></div>'
      + '<div class="grid grid-2">'
      + '<div class="panel"><div class="panel-head"><h3>应用通道</h3></div><div class="panel-body" id="poolApps">' + loadingState() + '</div></div>'
       + '<div class="panel"><div class="panel-head"><h3>个人账号</h3></div><div class="panel-body" id="poolAccountProfile"></div><div class="panel-head pool-account-list-head"><h3>已授权账号</h3></div><div class="panel-body" id="poolAccountList"></div></div>'
      + "</div>"
      + (roleAllowed("ops") ? '<div class="panel mt-16"><div class="panel-head"><h3>应用矩阵</h3><div class="spacer"></div><button class="btn btn-ghost btn-sm" id="poolMatrixRefresh">刷新</button></div><div class="panel-body" id="poolMatrix">' + loadingState() + '</div></div>' : "")
      + '<div class="panel mt-16"><div class="panel-head"><h3>接入指引</h3></div><div class="panel-body"><div class="rl-cols">'
      + '<div><b>1. 准备应用</b><p class="muted" style="margin:6px 0 0;font-size:13px">在 MagicChat 创建应用，取得应用 ID 与应用密钥。</p></div>'
      + '<div><b>2. 接入通道</b><p class="muted" style="margin:6px 0 0;font-size:13px">填写 WebSocket 地址、应用 ID 与密钥，保存后即建立通道。</p></div>'
      + '<div><b>3. 按应用隔离</b><p class="muted" style="margin:6px 0 0;font-size:13px">多个应用各自运行与记账，默认应用用于未指定来源的任务。</p></div>'
      + "</div></div></div>";
    var n = $("#newApp");
    if (n) n.addEventListener("click", openAppModal);
    var mr = $("#poolMatrixRefresh");
    if (mr) mr.addEventListener("click", loadPoolMatrix);
    loadPool();
  }
  function loadPool() {
    api("/jiying/api/connections").then(function (r) {
      if (r.code !== 200) { toast(r.msg || "加载失败", "err"); return; }
      var apps = r.data.apps || [];
      state.apps = apps;
      renderPoolApps(apps);
      ensureAppProfiles(apps);
      renderPoolAccount(r.data.account || {});
      renderPoolAccounts(r.data.accounts || []);
      if ($("#poolMatrix")) loadPoolMatrix();
    }).catch(function () { errorState($("#poolApps"), "link", "连接池状态加载失败", "网络异常，请检查服务状态。", loadPool); });
  }
  // 应用资料由服务端异步缓存；刚连接时按需补拉一次，成功后重绘。
  function ensureAppProfiles(apps) {
    var pending = false;
    (apps || []).forEach(function (a) {
      var id = String(a.app_id || "");
      if (!a.online || !id || a.name || state.appProfiles[id]) return;
      pending = true;
      state.appProfiles[id] = { pending: true };
      api("/jiying/api/apps/profile?app_id=" + encodeURIComponent(id), { silent: true }).then(function (r) {
        if (r.code === 200 && r.data) state.appProfiles[id] = r.data;
        else delete state.appProfiles[id];
      }).catch(function () { delete state.appProfiles[id]; }).finally(function () {
        if (state.view === "pool" && !$("#poolApps .rl-loading")) loadPool();
      });
    });
    return pending;
  }
  function renderPoolApps(apps) {
    var box = $("#poolApps");
    if (!box) return;
    if (!apps.length) { box.innerHTML = emptyState("link", "尚未接入应用", "接入后可在此查看通道在线状态，并按应用隔离运行。", '<button class="btn btn-primary btn-sm" id="poolEmptyAdd">接入应用</button>'); bindPoolEmptyAdd(); return; }
    box.innerHTML = apps.map(function (a) {
      var id = String(a.app_id || "");
      var extra = state.appProfiles[id] && !state.appProfiles[id].pending ? state.appProfiles[id] : {};
      var name = a.name || extra.name || "";
      var avatar = a.avatar_url || extra.avatar_url || "";
      var desc = a.description || extra.description || a.ws_url || "";
      var short = shortID(id);
      var offline = !a.online && (a.last_error || a.retry_at);
      return '<div class="list-item"><span class="dot ' + (a.online ? "on" : "off") + '"></span>' + avatarHTML(avatar, name || short, "pool-avatar")
        + '<div class="grow"><div class="name" title="' + esc(id) + '">' + esc(name || short || "(未命名)") + (a.is_default ? ' <span class="badge badge-ok">默认</span>' : "") + (a.conversation_count ? ' <span class="badge">' + Number(a.conversation_count) + " 会话</span>" : "") + "</div>"
        + '<div class="desc mono" title="' + esc(a.ws_url || "") + '">' + esc(desc || "-") + "</div>"
        + (offline ? '<div class="desc" style="color:var(--danger,#d9534f)">' + esc(a.last_error || "通道离线") + (a.retry_at ? " · 下次重连 " + esc(fmtTime(a.retry_at)) : "") + "</div>" : "")
        + '</div><div class="actions"><button class="btn btn-ghost btn-sm" onclick="Relay.showAppMatrix(' + jsArg(id) + ')">矩阵</button><button class="btn btn-ghost btn-sm" onclick="Relay.editApp(' + jsArg(id) + ')">编辑</button>' + (a.is_default ? "" : '<button class="btn btn-ghost btn-sm" onclick="Relay.defaultApp(' + jsArg(id) + ')">设为默认</button>') + '<button class="btn btn-outline-danger btn-sm" onclick="Relay.delApp(' + jsArg(id) + ')">移除</button></div></div>';
    }).join("");
  }
  function bindPoolEmptyAdd() {
    var b = $("#poolEmptyAdd");
    if (b) b.addEventListener("click", openAppModal);
  }
  function renderPoolAccount(a) {
    var box = $("#poolAccountProfile");
    if (!box) return;
    var name = a.nickname || a.email || "U";
    var avatar = avatarHTML(a.avatar_url, name, "pool-avatar");
    var st = String(a.status || "未登录");
    var badge = st.indexOf("已登录") >= 0 ? '<span class="badge badge-ok">已登录</span>' : '<span class="badge badge-off">未登录</span>';
    box.innerHTML = '<div class="rl-row">' + avatar
      + '<div class="grow"><div class="kv"><span class="k">账号</span><span class="v">' + esc(a.email || "未配置") + "</span></div>"
      + '<div class="kv"><span class="k">昵称</span><span class="v">' + (a.nickname ? esc(a.nickname) : '<span class="muted">未设置</span>') + "</span></div>"
      + '<div class="kv"><span class="k">状态</span><span class="v">' + badge + "</span></div></div></div>";
  }
  function renderPoolAccounts(accounts) {
    var box = $("#poolAccountList");
    if (!box) return;
    if (!accounts || !accounts.length) { box.innerHTML = emptyState("users", "暂无已授权账号", "授权账号后可按账号身份调用功能，并与应用通道隔离运行。", '<button class="btn btn-primary btn-sm" id="poolAcctEmptyAdd">去授权账号</button>'); var b = $("#poolAcctEmptyAdd"); if (b) b.addEventListener("click", function () { switchView("connect"); }); return; }
    box.innerHTML = accounts.map(function (a) {
      var status = a.logged_in ? '<span class="badge badge-ok">已登录</span>' : (a.active ? '<span class="badge badge-warn">待登录</span>' : '<span class="badge badge-off">未选中</span>');
      var action = a.active ? "" : '<button class="btn btn-ghost btn-sm" onclick="Relay.selectAccount(' + jsArg(a.email) + ')">切换</button>';
      return '<div class="list-item"><div class="grow"><div class="name">' + esc(a.email) + (a.active ? ' <span class="badge badge-ok">当前</span>' : "") + '</div><div class="desc">' + esc(a.base_url || "") + " · " + status + '</div></div><div class="actions">' + action + '</div></div>';
    }).join("");
  }
  function selectAccount(email) {
    api("/jiying/api/connections/account/select", { method: "POST", form: { email: email } }).then(function (r) {
      if (r.code === 200) { toast("账号已切换", "ok"); loadPool(); }
      else toast(r.msg || "账号切换失败", "err");
    });
  }
  function openAppModal() {
    showModal("接入应用", '<div class="field"><label>WebSocket 接入地址</label><input class="input" id="pa_ws" placeholder="wss://<host>/api/app/ws"></div>'
      + '<div class="field"><label>应用 ID</label><input class="input" id="pa_id" placeholder="应用标识"></div>'
      + '<div class="field"><label>应用密钥</label><input class="input" id="pa_secret" type="password" placeholder="连接密钥"></div>',
      '<button class="btn btn-ghost" onclick="Relay.closeModal()">取消</button><button class="btn btn-primary" id="pa_save">保存</button>');
    $("#pa_save").addEventListener("click", function () {
      var ws = $("#pa_ws").value.trim(), id = $("#pa_id").value.trim(), secret = $("#pa_secret").value;
      if (!ws || !id || !secret) { toast("请填写完整信息", "err"); return; }
      api("/jiying/api/connections/app", { method: "POST", form: { ws_url: ws, app_id: id, app_secret: secret } }).then(function (r) {
        if (r.code === 200) { closeModal(); toast("已接入", "ok"); loadPool(); }
        else toast(r.msg || "接入失败", "err");
      });
    });
  }
  function editApp(appID) {
    var id = String(appID || "");
    var apps = state.apps || [];
    var a = apps.filter(function (x) { return String(x.app_id || "") === id; })[0];
    if (!a) { toast("应用不在当前连接池中", "err"); return; }
    showModal("编辑应用 · " + (a.name || shortID(id)),
      '<div class="field"><label>应用 ID</label><input class="input mono" value="' + esc(id) + '" readonly><div class="hint">应用 ID 由 MagicChat 分配，不可修改。</div></div>'
      + '<div class="field"><label>WebSocket 接入地址</label><input class="input" id="pa_ws" value="' + esc(a.ws_url || "") + '" placeholder="wss://<host>/api/app/ws"></div>'
      + '<div class="field"><label>应用密钥</label><input class="input" id="pa_secret" type="password" value="****"><div class="hint">保持 **** 表示沿用已保存密钥；更换接入地址时必须重新填写密钥。</div></div>',
      '<button class="btn btn-ghost" onclick="Relay.closeModal()">取消</button><button class="btn btn-primary" id="pa_save">保存</button>');
    $("#pa_save").addEventListener("click", function () {
      var ws = $("#pa_ws").value.trim(), secret = $("#pa_secret").value.trim();
      if (!ws || !secret) { toast("请填写完整信息", "err"); return; }
      api("/jiying/api/connections/app", { method: "POST", form: { ws_url: ws, app_id: id, app_secret: secret } }).then(function (r) {
        if (r.code === 200) { closeModal(); toast("应用已更新", "ok"); loadPool(); }
        else toast(r.msg || "更新失败", "err");
      });
    });
  }
  function delApp(appID) {
    confirmDialog("移除应用", "确认移除应用 " + appID + " ？", "移除", function () {
      api("/jiying/api/connections/app/delete", { method: "POST", form: { app_id: appID } }).then(function (r) {
        if (r.code === 200) { toast("已移除", "ok"); loadPool(); }
        else toast(r.msg || "移除失败", "err");
      });
    });
  }
  function defaultApp(appID) {
    api("/jiying/api/connections/app/default", { method: "POST", form: { app_id: appID } }).then(function (r) {
      if (r.code === 200) { toast("默认应用已切换", "ok"); loadPool(); }
      else toast(r.msg || "切换失败", "err");
    });
  }

  function loadPoolMatrix() {
    var box = $("#poolMatrix");
    if (!box) return;
    api("/jiying/api/connections/app/matrix", { silent: true }).then(function (r) {
      if (r.code !== 200) { box.innerHTML = '<div class="muted">应用矩阵加载失败：' + esc(r.msg || r.message || "权限不足") + "</div>"; return; }
      state.appMatrix = r.data.apps || [];
      renderPoolMatrix(state.appMatrix);
    }).catch(function () { box.innerHTML = '<div class="muted">应用矩阵加载失败，请检查网络。</div>'; });
  }
  function renderPoolMatrix(rows) {
    var box = $("#poolMatrix");
    if (!box) return;
    if (!rows || !rows.length) { box.innerHTML = emptyState("grid", "暂无应用矩阵", "接入应用后可查看功能、角色、会话与定时的隔离情况。"); return; }
    box.innerHTML = '<div class="table-wrap"><table class="table align-middle"><thead><tr><th>应用</th><th class="nowrap">状态</th><th class="nowrap">模型</th><th class="nowrap">功能</th><th class="nowrap">会话</th><th class="nowrap">定时</th><th class="nowrap">角色</th></tr></thead><tbody>'
      + rows.map(function (a) {
        var roles = a.roles ? Object.keys(a.roles).length : 0;
        return '<tr><td><button class="btn btn-ghost btn-sm" onclick="Relay.showAppMatrix(' + jsArg(a.app_id) + ')">' + esc(a.name || shortID(a.app_id)) + '</button>' + (a.is_default ? ' <span class="badge badge-ok">默认</span>' : "") + '</td>'
          + '<td>' + (a.online ? '<span class="badge badge-ok">在线</span>' : '<span class="badge badge-err">离线</span>') + '</td>'
          + '<td class="mono clip" title="' + esc(a.model || "") + '">' + esc(a.model || "-") + '</td>'
          + '<td>' + Number((a.features || []).length) + '</td><td>' + Number(a.conversation_count || 0) + '</td><td>' + Number(a.schedules || 0) + '</td><td>' + (a.roles ? Number(roles) + " 条覆盖" : "继承全局") + '</td></tr>';
      }).join("") + "</tbody></table></div>";
  }
  function showAppMatrix(appID) {
    var rows = state.appMatrix || [];
    var a = rows.filter(function (x) { return String(x.app_id || "") === String(appID); })[0];
    if (!a) { toast("应用矩阵尚未加载", "err"); return; }
    var features = (a.features || []).map(function (n) { return '<span class="badge">' + esc(n) + "</span>"; }).join(" ") || '<span class="muted">无可见功能</span>';
    var roles = "";
    if (a.roles) {
      roles = Object.keys(a.roles).sort().map(function (u) { return '<div class="kv"><span class="k">' + esc(u) + '</span><span class="v"><span class="badge">' + esc(a.roles[u]) + "</span></span></div>"; }).join("") || '<span class="muted">无应用级角色覆盖</span>';
    } else {
      roles = '<span class="muted">仅主控可见</span>';
    }
    showModal("应用矩阵 · " + (a.name || shortID(a.app_id)),
      '<div class="kv"><span class="k">应用 ID</span><span class="v mono">' + esc(a.app_id) + "</span></div>"
      + '<div class="kv"><span class="k">通道</span><span class="v">' + (a.online ? "在线" : "离线" + (a.last_error ? " · " + esc(a.last_error) : "")) + "</span></div>"
      + '<div class="kv"><span class="k">模型</span><span class="v mono">' + esc(a.model || "-") + "</span></div>"
      + '<div class="kv"><span class="k">会话 / 定时</span><span class="v">' + Number(a.conversation_count || 0) + " / " + Number(a.schedules || 0) + "</span></div>"
      + '<div class="field" style="margin-top:12px"><label>可见功能</label><div>' + features + "</div></div>"
      + '<div class="field"><label>应用级角色</label><div>' + roles + "</div></div>",
      '<button class="btn btn-ghost" onclick="Relay.closeModal()">关闭</button>');
  }

  /* ---------- 钉钉连接器 ---------- */
  function renderDingTalk() {
    var cont = $("#view-dingtalk");
    var master = roleAllowed("master");
    var actions = master ? '<button class="btn btn-ghost" id="dtProbe">检查登录</button><button class="btn btn-ghost" id="dtConfig">配置</button><button class="btn btn-primary" id="dtToggle">启动</button>' : "";
    cont.innerHTML = '<div class="page-head"><div style="display:flex;align-items:center;gap:12px"><img src="/assets/dingtalk.svg" alt="钉钉" width="40" height="40" style="border-radius:10px"><div><h2>钉钉连接器</h2><p>把钉钉私聊、@我与群提醒接入 Relay，并让 Agent 通过 dws 操作钉钉</p></div></div><div class="spacer"></div>' + actions + '</div>'
      + '<div class="grid grid-2"><div class="panel"><div class="panel-head"><h3>连接状态</h3></div><div class="panel-body" id="dtStatus">' + loadingState() + '</div></div>'
      + '<div class="panel"><div class="panel-head"><h3>运行说明</h3></div><div class="panel-body"><div class="rl-cols">'
      + '<div><b>1. 安装 dws</b><p class="muted" style="margin:6px 0 0;font-size:13px">在宿主机安装 dingtalk-workspace-cli，并执行 dws auth login 授权接收通知的账号。</p></div>'
      + '<div><b>2. 订阅事件</b><p class="muted" style="margin:6px 0 0;font-size:13px">连接器会常驻执行 event consume，解析私聊、@我 与 @所有人 事件。</p></div>'
      + '<div><b>3. 受控推送</b><p class="muted" style="margin:6px 0 0;font-size:13px">可把事件自动推送到指定应用会话；Agent 仅在连接器启用时获得钉钉工具。</p></div>'
      + "</div></div></div></div>"
      + '<div class="panel mt-16"><div class="panel-head"><h3>最近事件</h3><div class="spacer"></div><button class="btn btn-ghost btn-sm" id="dtRefresh">刷新</button></div><div class="panel-body" id="dtEvents">' + loadingState() + '</div></div>';
    var cfgBtn = $("#dtConfig");
    var toggleBtn = $("#dtToggle");
    var probeBtn = $("#dtProbe");
    if (cfgBtn) cfgBtn.addEventListener("click", openDingTalkConfig);
    if (toggleBtn) toggleBtn.addEventListener("click", toggleDingTalk);
    if (probeBtn) probeBtn.addEventListener("click", probeDingTalk);
    $("#dtRefresh").addEventListener("click", loadDingTalk);
    loadDingTalk();
  }
  function loadDingTalk() {
    api("/jiying/api/dingtalk/status").then(function (r) {
      if (r.code !== 200) { errorState($("#dtStatus"), "chat-dots", "状态加载失败", r.msg || "请稍后重试。", loadDingTalk); return; }
      renderDingTalkStatus(r.data || {});
    }).catch(function () { errorState($("#dtStatus"), "chat-dots", "状态加载失败", "网络异常，请检查服务状态。", loadDingTalk); });
    api("/jiying/api/dingtalk/events?limit=50", { silent: true }).then(function (r) {
      if (r.code === 200) renderDingTalkEvents((r.data && r.data.events) || []);
    }).catch(function () { });
  }
  function renderDingTalkStatus(st) {
    state.dingtalk = st;
    var cfg = st.config || {};
    var badge = st.running ? '<span class="badge badge-ok">运行中</span>' : '<span class="badge badge-err">已停止</span>';
    var toggle = $("#dtToggle");
    if (toggle) { toggle.textContent = cfg.enabled ? "停止" : "启动"; toggle.className = "btn " + (cfg.enabled ? "btn-outline-danger" : "btn-primary"); }
    var rows = '<div class="kv"><span class="k">状态</span><span class="v">' + badge + (st.pid ? ' <span class="muted mono">PID ' + Number(st.pid) + "</span>" : "") + "</span></div>"
      + '<div class="kv"><span class="k">dws</span><span class="v">' + (st.bin_found ? '<span class="badge badge-ok">已找到</span>' : '<span class="badge badge-err">未找到</span>') + ' <span class="mono">' + esc(cfg.bin || "dws") + "</span></span></div>"
      + '<div class="kv"><span class="k">自动推送</span><span class="v">' + (cfg.auto_push ? (cfg.topic_mode ? "已开启 · 按会话分话题" : "已开启 · 推送到指定会话") : "未开启") + (cfg.push_conversation_id ? ' · <span class="mono">' + esc(cfg.push_conversation_id) + "</span>" : "") + "</span></div>"
      + (st.started_at ? '<div class="kv"><span class="k">启动时间</span><span class="v mono">' + esc(fmtTime(st.started_at)) + "</span></div>" : "")
      + (st.retry_at ? '<div class="kv"><span class="k">下次重连</span><span class="v mono">' + esc(fmtTime(st.retry_at)) + "</span></div>" : "")
      + (st.last_error ? '<div class="kv"><span class="k">最近错误</span><span class="v" style="color:var(--danger,#d9534f)">' + esc(st.last_error) + "</span></div>" : "")
      + (st.stderr ? '<div class="kv"><span class="k">dws 输出</span><span class="v mono clip" title="' + esc(st.stderr) + '">' + esc(st.stderr) + "</span></div>" : "");
    var box = $("#dtStatus");
    if (box) box.innerHTML = rows;
  }
  function renderDingTalkEvents(events) {
    var box = $("#dtEvents");
    if (!box) return;
    if (!events || !events.length) { box.innerHTML = emptyState("inbox", "暂无钉钉事件", "连接器运行后，私聊、@我 与 @所有人 事件会显示在这里。"); return; }
    var reasonMap = { "dm": "私聊", "mention": "@我", "at-all": "@所有人" };
    box.innerHTML = events.slice().reverse().map(function (e) {
      var badge = e.reason === "at-all" ? "badge-err" : (e.reason === "mention" ? "badge-warn" : "badge-ok");
      return '<div class="list-item"><div class="grow"><div class="name">' + esc(e.sender || "未知发送人") + ' <span class="badge ' + badge + '">' + esc(reasonMap[e.reason] || e.reason || "事件") + '</span> <span class="muted mono">' + esc(e.conversation_id || "") + "</span></div>"
        + '<div class="desc">' + esc((e.content || "").slice(0, 300)) + "</div>"
        + '<div class="desc muted mono">' + esc(fmtTime(e.received_at)) + (e.time ? " · " + esc(e.time) : "") + "</div></div></div>";
    }).join("");
  }
  function openDingTalkConfig() {
    var st = state.dingtalk || {};
    var cfg = st.config || {};
    var apps = (state.apps || []).map(function (a) { return '<option value="' + esc(a.app_id) + '">' + esc(a.name || a.app_id) + "</option>"; }).join("");
    showModal("钉钉连接器配置",
      '<div class="field"><label>dws 可执行文件</label><input class="input mono" id="dt_bin" value="' + esc(cfg.bin || "dws") + '"><div class="hint">可填绝对路径，例如 /usr/local/bin/dws。</div></div>'
      + '<div class="field"><label><input type="checkbox" id="dt_auto"' + (cfg.auto_push ? " checked" : "") + '"> 自动推送事件</label><div class="hint">开启后把钉钉事件推送到下方应用会话。</div></div>'
      + '<div class="field"><label><input type="checkbox" id="dt_topic"' + (cfg.topic_mode ? " checked" : "") + '"> 按钉钉会话分话题</label><div class="hint">每个钉钉群/私聊复用一个即应话题；在话题里回复会直接转发回钉钉。关闭时全部推送到下方会话。</div></div>'
      + '<div class="field"><label>推送应用 ID</label><input class="input mono" id="dt_app" list="dt_apps" value="' + esc(cfg.push_app_id || "") + '"><datalist id="dt_apps">' + apps + "</datalist></div>"
      + '<div class="field"><label>推送会话 ID</label><input class="input mono" id="dt_conv" value="' + esc(cfg.push_conversation_id || "") + '"><div class="hint">即应中的目标会话 ID；留空则只在面板展示事件。</div></div>',
      '<button class="btn btn-ghost" onclick="Relay.closeModal()">取消</button><button class="btn btn-primary" id="dt_save">保存</button>');
    $("#dt_save").addEventListener("click", function () {
      var form = {
        enabled: cfg.enabled ? "true" : "false",
        bin: $("#dt_bin").value.trim(),
        auto_push: $("#dt_auto").checked ? "true" : "false",
        topic_mode: $("#dt_topic").checked ? "true" : "false",
        push_app_id: $("#dt_app").value.trim(),
        push_conversation_id: $("#dt_conv").value.trim()
      };
      api("/jiying/api/dingtalk/config", { method: "POST", form: form }).then(function (r) {
        if (r.code === 200) { closeModal(); toast("钉钉配置已保存", "ok"); loadDingTalk(); }
        else toast(r.msg || r.message || "保存失败", "err");
      });
    });
  }
  function toggleDingTalk() {
    var cfg = (state.dingtalk || {}).config || {};
    var url = cfg.enabled ? "/jiying/api/dingtalk/stop" : "/jiying/api/dingtalk/start";
    api(url, { method: "POST" }).then(function (r) {
      if (r.code === 200) { toast(cfg.enabled ? "连接器已停止" : "连接器已启动", "ok"); loadDingTalk(); }
      else toast(r.msg || r.message || "操作失败", "err");
    });
  }
  function probeDingTalk() {
    api("/jiying/api/dingtalk/probe", { method: "POST" }).then(function (r) {
      if (r.code === 200) {
        var text = (r.data && r.data.auth) || "{}";
        showModal("dws 登录状态", '<pre class="mono" style="white-space:pre-wrap;max-height:50vh;overflow:auto">' + esc(text) + "</pre>", '<button class="btn btn-ghost" onclick="Relay.closeModal()">关闭</button>');
      } else {
        toast(r.msg || r.message || "dws 检查失败", "err");
      }
    });
  }

  /* ---------- 插件市场 ---------- */
  function renderMarket() {
    var cont = $("#view-market");
    cont.innerHTML = '<div class="page-head"><div><h2>插件市场</h2><p>发现、上架与分发包，已安装的插件在「插件」页统一管理</p></div><div class="spacer"></div><button class="btn btn-ghost" id="marketPublishers">信任发布者</button><button class="btn btn-primary" id="newPlugin">上架插件</button></div>'
      + '<div class="panel"><div class="panel-body" id="marketList">' + loadingState() + '</div></div>'
      + '<div class="panel mt-16"><div class="panel-head"><h3>插件规范</h3></div><div class="panel-body"><div class="rl-cols">'
      + '<div><b>作用范围</b><p class="muted" style="margin:6px 0 0;font-size:13px">插件归属 <span class="mono">public</span> 或 <span class="mono">master</span>，安装时按应用或个人账号隔离。</p></div>'
      + '<div><b>类型</b><p class="muted" style="margin:6px 0 0;font-size:13px">支持脚本型与 http 型，脚本型随插件一同分发源码。</p></div>'
      + '<div><b>版本</b><p class="muted" style="margin:6px 0 0;font-size:13px">同名插件按版本区分，已安装的会在列表中标注。</p></div>'
      + "</div></div></div>";
    var n = $("#newPlugin");
    if (n) n.addEventListener("click", openPluginModal);
    var p = $("#marketPublishers");
    if (p) p.addEventListener("click", openPublisherModal);
    loadMarket();
  }
  function loadMarket() {
    api("/jiying/api/market").then(function (r) {
      var box = $("#marketList");
      if (!box) return;
      if (r.code !== 200) { errorState(box, "inbox", "插件市场加载失败", r.msg || r.message, loadMarket); return; }
      var list = r.code === 200 && r.data ? r.data : [];
      if (!list.length) { box.innerHTML = emptyState("inbox", "市场暂无插件", "上架插件后会在这里展示，供按应用或个人账号范围安装分发。", '<button class="btn btn-primary btn-sm" id="marketEmptyAdd">上架插件</button>'); bindMarketEmptyAdd(); return; }
      box.innerHTML = '<div class="list">' + list.map(function (p) {
        return '<div class="list-item"><div class="ico li-ico" data-ico="inbox">' + ico("inbox") + '</div>'
          + '<div class="grow"><div class="name">' + esc(p.name) + ' <span class="muted">@' + esc(p.version) + '</span> '
          + metaBadge(p.installed ? "已安装" : "未安装", p.installed ? "badge-ok" : "badge-off") + " "
          + metaBadge(FEATURE_KIND[p.kind] || p.kind || "-") + " "
          + metaBadge(FEATURE_SCOPE[p.scope] || p.scope, p.scope === "public" ? "badge-info" : "badge-off") + " "
          + metaBadge(FEATURE_IDENTITY[p.identity] || p.identity, "badge-off") + "</div>"
          + '<div class="desc">' + esc(p.description || "-") + "</div></div>"
           + '<div class="actions"><button class="btn btn-ghost btn-sm" onclick="Relay.installPlugin(' + jsArg(p.name) + ')">' + (p.installed ? "重装" : "安装") + '</button>'
           + '<button class="btn btn-outline-danger btn-sm" onclick="Relay.delPlugin(' + jsArg(p.name) + ')">下架</button></div></div>';
      }).join("") + "</div>";
      hydrateIcons();
    });
  }
  function openPublisherModal() {
    api("/jiying/api/market/publishers").then(function (r) {
      var list = r.code === 200 && r.data ? r.data : [];
      var rows = list.length ? list.map(function (p) {
        return '<div class="list-item"><div class="grow"><div class="name">' + esc(p.name) + '</div>'
          + '<div class="desc mono" style="word-break:break-all">' + esc(p.public_key) + '</div></div>'
          + '<div class="actions"><button class="btn btn-outline-danger btn-sm" data-pub="' + esc(p.name) + '">删除</button></div></div>';
      }).join("") : '<p class="muted">尚未登记外部发布者。本机发布会自动登记 local 公钥。</p>';
      showModal("信任发布者", '<div class="field"><label>名称</label><input class="input" id="pub_name" placeholder="vendor"></div>'
        + '<div class="field"><label>Ed25519 公钥（Base64）</label><input class="input mono" id="pub_key" placeholder="32 字节公钥"></div>'
        + '<div class="panel mt-16"><div class="panel-body"><div class="list">' + rows + "</div></div></div>"
        + '<p class="muted" style="font-size:12px">公钥登记后不可更换；删除后该发布者已上架插件将无法通过签名校验。</p>',
        '<button class="btn btn-ghost" onclick="Relay.closeModal()">关闭</button><button class="btn btn-primary" id="pub_save">登记</button>');
      $("#pub_save").addEventListener("click", function () {
        var name = $("#pub_name").value.trim(), key = $("#pub_key").value.trim();
        if (!name || !key) { toast("请填写名称与公钥", "err"); return; }
        api("/jiying/api/market/publishers", { method: "POST", form: { name: name, public_key: key } }).then(function (res) {
          if (res.code === 200) { toast("已登记", "ok"); openPublisherModal(); }
          else toast(res.msg || "登记失败", "err");
        });
      });
      Array.prototype.forEach.call(document.querySelectorAll("[data-pub]"), function (btn) {
        btn.addEventListener("click", function () {
          var name = btn.getAttribute("data-pub");
          confirmDialog("删除发布者", "删除 " + name + " 后，其已上架插件将无法安装，确认？", "删除", function () {
            api("/jiying/api/market/publishers/delete", { method: "POST", form: { name: name } }).then(function (res) {
              if (res.code === 200) { toast("已删除", "ok"); openPublisherModal(); }
              else toast(res.msg || "删除失败", "err");
            });
          });
        });
      });
    });
  }
  function bindMarketEmptyAdd() {
    var b = $("#marketEmptyAdd");
    if (b) b.addEventListener("click", openPluginModal);
  }
  function openPluginModal() {
    showModal("上架插件", '<div class="grid grid-2"><div class="field"><label>名称</label><input class="input" id="pm_name" placeholder="greeter"></div>'
      + '<div class="field"><label>版本</label><input class="input" id="pm_ver" placeholder="1.0.0"></div></div>'
      + '<div class="field"><label>说明</label><input class="input" id="pm_desc" placeholder="插件用途"></div>'
      + '<div class="grid grid-2"><div class="field"><label>类型</label><select class="select" id="pm_kind"><option value="script">script</option><option value="http">http</option></select></div>'
      + '<div class="field"><label>身份</label><select class="select" id="pm_identity"><option value="app">应用插件</option><option value="user">个人账号插件</option></select></div></div>'
      + '<div class="grid grid-2"><div class="field"><label>范围</label><select class="select" id="pm_scope"><option value="public">public</option><option value="master">master</option></select></div>'
      + '<div class="field"><label>目标（http 地址 / 脚本文件名）</label><input class="input" id="pm_target" placeholder="demo.js"></div></div>'
      + '<div class="field"><label>脚本代码（http 插件可留空）</label><textarea class="input" id="pm_content" rows="8" placeholder="// 脚本代码"></textarea></div>',
      '<button class="btn btn-ghost" onclick="Relay.closeModal()">取消</button><button class="btn btn-primary" id="pm_save">上架</button>');
    $("#pm_save").addEventListener("click", function () {
      var form = {
        name: $("#pm_name").value.trim(), version: $("#pm_ver").value.trim(),
        description: $("#pm_desc").value.trim(), kind: $("#pm_kind").value,
        identity: $("#pm_identity").value, scope: $("#pm_scope").value,
        target: $("#pm_target").value.trim(), content: $("#pm_content").value
      };
      if (!form.name || !form.target) { toast("请填写名称与目标", "err"); return; }
      if (form.kind === "script" && !form.content) { toast("script 插件需要代码", "err"); return; }
      api("/jiying/api/market/publish", { method: "POST", form: form }).then(function (r) {
        if (r.code === 200) { closeModal(); toast("已上架", "ok"); loadMarket(); }
        else toast(r.msg || "上架失败", "err");
      });
    });
  }
  function scriptRiskReasons(name, code) {
    var lower = String(code || "").toLowerCase();
    var py = /\.py$/i.test(name || "");
    var checks = py ? [
      ["os.system", "执行系统命令"], ["os.popen", "执行系统命令"], ["subprocess", "执行子进程"],
      ["shutil.rmtree", "删除目录"], ["socket.", "网络连接"], ["requests.", "网络请求"],
      ["urllib.", "网络请求"], ["httpx.", "网络请求"], ["ftplib", "网络连接"], ["paramiko", "网络连接"],
      ["eval(", "动态执行代码"], ["exec(", "动态执行代码"]
    ] : [
      ["child_process", "执行子进程"], ["require('fs')", "读写文件"], ['require("fs")', "读写文件"],
      ["readfile", "读写文件"], ["writefile", "读写文件"], ["fetch(", "网络请求"],
      ["xmlhttprequest", "网络请求"], ["eval(", "动态执行代码"], ["new function", "动态执行代码"]
    ];
    var seen = {}, out = [];
    checks.forEach(function (c) {
      if (!seen[c[1]] && lower.indexOf(c[0]) >= 0) { seen[c[1]] = 1; out.push(c[1]); }
    });
    return out;
  }
  function installPlugin(name) {
    confirmDialog("安装插件", "插件将以服务进程权限运行，请确认来源可信。是否安装 " + name + " ？", "安装", function () {
      api("/jiying/api/market/install", { method: "POST", form: { name: name, confirm: "true" } }).then(function (r) {
        if (r.code === 200) { toast(r.data || "已安装", "ok"); loadMarket(); }
        else toast(r.msg || "安装失败", "err");
      });
    });
  }
  function delPlugin(name) {
    confirmDialog("下架插件", "确认从市场上架列表移除 " + name + " ？已安装的插件与功能不受影响。", "下架", function () {
      api("/jiying/api/market/delete", { method: "POST", form: { name: name } }).then(function (r) {
        if (r.code === 200) { toast("已删除", "ok"); loadMarket(); }
        else toast(r.msg || "删除失败", "err");
      });
    });
  }

  /* ---------- 审计日志 ---------- */
  function renderAudit() {
    var cont = $("#view-audit");
    cont.innerHTML = '<div class="page-head"><div><h2>审计日志</h2><p>敏感操作留痕：操作者、动作、目标与结果</p></div><div class="spacer"></div>'
      + '<input class="input" id="auditFilter" placeholder="筛选操作者 / 动作 / 目标 / 详情" style="width:240px">'
      + '<button class="btn btn-sm" id="auditExport">导出</button>'
      + '<button class="btn btn-ghost btn-sm" onclick="Relay.loadAudit()">刷新</button></div>'
      + '<div class="panel"><div class="table-wrap"><table class="table align-middle"><thead><tr><th class="nowrap" style="width:150px">时间</th><th class="nowrap" style="width:110px">操作者</th><th class="nowrap" style="width:120px">应用</th><th class="nowrap" style="width:130px">动作</th><th style="width:150px">目标</th><th>详情</th><th class="nowrap" style="width:130px">结果</th></tr></thead><tbody id="auditList"><tr><td colspan="7">' + loadingState() + '</td></tr></tbody></table></div></div>'
      + '<div class="muted" id="auditMeta" style="margin-top:8px;font-size:12px"></div>';
    $("#auditFilter").addEventListener("input", paintAudit);
    $("#auditExport").addEventListener("click", exportAudit);
    Relay.loadAudit();
  }
  function loadAudit() {
    api("/jiying/api/audit?limit=100").then(function (r) {
      var box = $("#auditList");
      if (!box) return;
      if (r.code !== 200) {
        state.auditRows = [];
        state.auditErr = r.msg || r.message || "请稍后重试。";
        tableError(box, 7, "shield", r.code === 403 ? "权限不足" : "审计日志加载失败", state.auditErr, function () { state.auditErr = ""; loadAudit(); });
        setAuditMeta(0, 0);
        return;
      }
      state.auditErr = "";
      var list = r.data ? r.data : [];
      state.auditRows = list.slice().reverse();
      paintAudit();
    }).catch(function () {
      var box = $("#auditList");
      if (!box) return;
      state.auditRows = [];
      state.auditErr = "网络异常，请稍后重试。";
      tableError(box, 7, "shield", "审计日志加载失败", state.auditErr, function () { state.auditErr = ""; loadAudit(); });
      setAuditMeta(0, 0);
    });
  }
  // 审计行与筛选逻辑同时服务于渲染和导出，集中一处避免两处过滤条件漂移。
  function filteredAudit(rows) {
    var kw = (($("#auditFilter") || {}).value || "").trim().toLowerCase();
    if (!kw) return rows;
    return rows.filter(function (a) {
      return [a.app_id, a.actor, a.action, a.target, a.detail, a.result].some(function (v) { return String(v || "").toLowerCase().indexOf(kw) >= 0; });
    });
  }
  function paintAudit() {
    var box = $("#auditList");
    if (!box) return;
    if (state.auditErr) return;
    var rows = state.auditRows || [];
    if (!rows.length) { box.innerHTML = '<tr><td colspan="7">' + emptyState("shield", "暂无审计记录", "敏感操作（接入、插件、角色等）会在这里留痕。") + "</td></tr>"; setAuditMeta(0, 0); return; }
    var kw = (($("#auditFilter") || {}).value || "").trim().toLowerCase();
    var shown = filteredAudit(rows);
    box.innerHTML = shown.map(function (a) {
      var ok = auditOK(a.result);
      return '<tr><td class="mono nowrap">' + esc(fmtTime(a.time)) + '</td><td class="nowrap">' + esc(a.actor) + '</td><td class="mono nowrap">' + esc(a.app_id || "-") + '</td><td class="mono nowrap">' + esc(a.action) + '</td><td class="mono clip" title="' + esc(a.target) + '">' + esc(a.target) + '</td><td>' + esc(a.detail) + '</td><td class="nowrap">' + (ok ? '<span class="badge badge-ok">成功</span>' : '<span class="badge badge-err">失败</span>') + ' <button class="btn btn-ghost btn-sm" title="在运行日志中查找" onclick="Relay.searchLogs(' + jsArg(a.action) + ')"><i class="bi bi-search"></i></button></td></tr>';
    }).join("") || '<tr><td colspan="7"><div class="empty">没有匹配「' + esc(kw) + '」的记录</div></td></tr>';
    setAuditMeta(shown.length, rows.length);
  }
  // 导出当前筛选结果为 CSV，带 BOM 以兼容 Excel 的中文显示。
  function exportAudit() {
    var rows = filteredAudit(state.auditRows || []);
    if (!rows.length) { toast("暂无审计记录可导出", "err"); return; }
    var head = ["时间", "操作者", "应用", "动作", "目标", "详情", "结果"];
    var lines = [head].concat(rows.map(function (a) {
      return [fmtTime(a.time), a.actor, a.app_id || "", a.action, a.target, a.detail, a.result];
    })).map(function (cells) {
      return cells.map(function (v) {
        var s = String(v == null ? "" : v);
        return /[",\r\n]/.test(s) ? '"' + s.replace(/"/g, '""') + '"' : s;
      }).join(",");
    });
    downloadBlob(new Blob(["\ufeff" + lines.join("\r\n") + "\r\n"], { type: "text/csv;charset=utf-8" }), "relay-audit-" + today() + ".csv");
  }
  function setAuditMeta(shown, total) {
    var el = $("#auditMeta");
    if (!el) return;
    el.textContent = total ? (shown === total ? "共 " + total + " 条" : "匹配 " + shown + " / " + total + " 条（最近 100 条）") : "";
  }
  // 审计结果除「失败:」外还有「拒绝:」「执行超时」等表述，需一并判定为非成功。
  var AUDIT_FAIL_MARKERS = ["失败", "拒绝", "超时", "错误", "异常", "无效", "未授权"];
  function auditOK(result) {
    var s = String(result || "");
    return !AUDIT_FAIL_MARKERS.some(function (m) { return s.indexOf(m) >= 0; });
  }

  /* ---------- 运行观测 ---------- */
  function renderObserve() {
    var cont = $("#view-observe");
    cont.innerHTML = '<div class="page-head"><div><h2>运行观测</h2><p>匿名探针与指标，不暴露任何凭据</p></div><div class="spacer"></div><button class="btn btn-ghost btn-sm" onclick="Relay.loadObserve()">刷新</button></div>'
      + '<div class="stat-strip cols-3" id="observeTiles">'
      + tile("健康探针", "…", "GET /healthz") + tile("运转时长", "…", "进程已运行")
      + tile("已收消息", "…", "累计接收") + tile("任务队列", "…", "执行中")
      + tile("通道状态", "…", "应用 · 账号") + tile("限流拒绝", "…", "被拒绝任务")
      + "</div>"
      + '<div class="panel mt-16"><div class="panel-head"><h3>RPC 调用</h3><span class="hint">按方法聚合 · 来自 /metrics</span></div><div class="panel-body" id="observeRpc">' + loadingState() + '</div></div>'
      + '<div class="panel mt-16"><div class="panel-head"><h3>原始指标</h3><span class="hint">Prometheus 文本格式</span></div><div class="panel-body"><pre class="mono rl-codeblock" id="observeRaw">加载中…</pre></div></div>';
    Relay.loadObserve();
  }

  function renderRoles() {
    var cont = $("#view-roles");
      cont.innerHTML = '<div class="page-head"><div><h2>权限管理</h2><p>主人权限面向即应用户 ID（在即应会话里发送「我的ID」获取）；运营与只读面向控制台登录账号。</p></div></div>'
        + '<div class="panel"><div class="panel-head"><h3>授权成员</h3><span class="hint" id="myAccount"></span></div><div class="panel-body"><div class="input-row"><input class="input" id="roleUser" placeholder="输入用户 ID"><select class="select" id="roleValue" style="max-width:150px"><option value="ops">运营</option><option value="readonly">只读</option><option value="master">主人</option></select><button class="btn btn-primary" id="grantRole">授予权限</button></div></div></div>'
       + '<div class="panel mt-16"><div class="panel-head"><h3>成员角色</h3></div><div class="panel-body" id="roleList">' + loadingState() + '</div></div>'
       + '<div class="panel mt-16"><div class="panel-head"><h3>主人列表</h3></div><div class="panel-body" id="masterList">' + loadingState() + '</div></div>';
     $("#grantRole").addEventListener("click", grantRole);
    loadRoles();
  }
  function loadRoles() {
    api("/jiying/api/roles").then(function (r) {
      var box = $("#masterList");
      if (!box) return;
      if (r.code !== 200) {
        var reason = r.msg || r.message || "请稍后重试。";
        errorState(box, "shield", r.code === 403 ? "权限不足" : "主人列表加载失败", reason, loadRoles);
        // 同一接口同时驱动成员角色面板，出错时也要收尾，避免它一直停在「加载中…」。
        var roleBox = $("#roleList");
        if (roleBox) roleBox.innerHTML = emptyState("users", r.code === 403 ? "权限不足" : "成员角色加载失败", reason);
        return;
      }
      var masters = r.data && r.data.masters ? r.data.masters : [];
      var roles = r.data && r.data.roles ? r.data.roles : {};
      var myBox = $("#myAccount");
      if (myBox && r.data) {
        var meId = String(r.data.me_id || "").trim();
        var meRole = ROLE_LABEL[r.data.me] || r.data.me || "";
        myBox.textContent = meId ? "当前控制台账号 " + meId + " · " + meRole : "";
      }
      var roleBox = $("#roleList");
      var roleNames = { ops: "运营", readonly: "只读" };
      var roleEntries = Object.keys(roles);
      if (roleBox) roleBox.innerHTML = roleEntries.length ? roleEntries.map(function (id) {
        var role = roles[id] || "";
        return '<div class="list-item"><div class="grow"><div class="name mono">' + esc(id) + '</div><div class="desc">' + esc(roleNames[role] || role) + '</div></div><button class="btn btn-outline-danger btn-sm" onclick="Relay.deleteRole(' + jsArg(id) + ')">撤销</button></div>';
      }).join("") : emptyState("users", "暂无显式角色", "运营和只读成员会显示在这里。");
      if (!masters.length) { box.innerHTML = emptyState("users", "尚未设置主人", "主人可以使用主控范围的功能和管理指令。"); return; }
      box.innerHTML = masters.map(function (id) {
         return '<div class="list-item"><div class="grow"><div class="name mono">' + esc(id) + '</div><div class="desc">主控权限</div></div><button class="btn btn-outline-danger btn-sm" onclick="Relay.removeMaster(' + jsArg(id) + ')">移除</button></div>';
      }).join("");
    });
  }
  function grantRole() {
    var user = $("#roleUser").value.trim();
    var role = $("#roleValue").value;
    if (!user) { toast("请输入用户 ID", "err"); return; }
    if (role === "master") {
      api("/jiying/api/masters", { method: "POST", form: { user: user } }).then(function (r) {
        if (r.code === 200) { toast("主人权限已授予", "ok"); $("#roleUser").value = ""; loadRoles(); }
        else toast(r.msg || "授权失败", "err");
      });
      return;
    }
    api("/jiying/api/roles", { method: "POST", form: { user: user, role: role } }).then(function (r) {
      if (r.code === 200) { toast("成员角色已更新", "ok"); $("#roleUser").value = ""; loadRoles(); }
      else toast(r.msg || "授权失败", "err");
    });
  }
  function deleteRole(user) {
    confirmDialog("撤销角色", "确认撤销该成员的显式角色？", "撤销", function () {
      api("/jiying/api/roles/delete", { method: "POST", form: { user: user } }).then(function (r) {
        if (r.code === 200) { toast("角色已撤销", "ok"); loadRoles(); }
        else toast(r.msg || "撤销失败", "err");
      });
    });
  }
  function removeMaster(user) {
    confirmDialog("移除主人", "确认移除用户 " + user + " 的主控权限？", "移除", function () {
      api("/jiying/api/masters/delete", { method: "POST", form: { user: user } }).then(function (r) {
        if (r.code === 200) { toast("主人已移除", "ok"); loadRoles(); }
        else toast(r.msg || "移除失败", "err");
      });
    });
  }
  /* ---------- AI 接入 ---------- */
  function aiStep(no, title, desc) {
    return '<div class="panel ai-step"><div class="panel-body"><span class="eyebrow">' + esc(no) + '</span><h3>' + esc(title) + "</h3><p>" + esc(desc) + "</p></div></div>";
  }
  function aiPromptText(token) {
    var base = location.origin;
    var key = token || "在「AI 接入」页创建后填入";
    if (LANG === "en") {
      key = token || "create one on the AI access page";
      return "You are the AI operator for the Relay Jiying console. Use the REST API to complete my requests.\n\n"
        + "Base URL: " + base + "\n"
        + "Auth: Authorization: Bearer " + key + "\n"
        + "Read first: " + base + "/jiying/api/ai/manifest (features, schedules, scopes, endpoints)\n"
        + "Guide: " + base + "/jiying/ai.txt\n"
        + "OpenAPI: " + base + "/jiying/openapi.json\n\n"
        + "Rules:\n"
        + "1. Read before writing; describe the intended change before making it.\n"
        + "2. Ask for confirmation before deleting, disabling, signing out, changing roles or bulk sending.\n"
        + "3. Use application/x-www-form-urlencoded. A response of {\"code\":200,\"data\":...} means success.\n"
        + "4. Dry-run sends with dry_run=true; on failure read /jiying/api/logs.\n"
        + "5. Upload local files with POST /jiying/api/upload (multipart field `file`, max 20 MB), then pass the returned file_id to /jiying/api/send/file.\n"
        + "6. Never read or reveal keys, passwords or cookies.\n";
    }
    return "你是 Relay 即应控制台的 AI 操作助手。请通过 REST API 完成我的请求。\n\n"
      + "Base URL: " + base + "\n"
      + "认证: Authorization: Bearer " + key + "\n"
      + "先读: " + base + "/jiying/api/ai/manifest（当前功能、定时任务、权限与端点）\n"
      + "接入说明: " + base + "/jiying/ai.txt\n"
      + "OpenAPI: " + base + "/jiying/openapi.json\n\n"
      + "规则:\n"
      + "1. 先读后写，写操作前说明将要做什么。\n"
      + "2. 删除、停用、退出、改角色、批量发送必须先征得我确认。\n"
      + "3. 参数用 application/x-www-form-urlencoded 传递；响应为 {\"code\":200,\"data\":...} 表示成功。\n"
      + "4. 发送消息前先用 dry_run=true 预演；失败时读取 /jiying/api/logs 定位原因。\n"
      + "5. 本地文件用 POST /jiying/api/upload 上传（multipart 字段 file，最大 20 MB），再把 file_id 交给 /jiying/api/send/file。\n"
      + "6. 不要读取或输出任何密钥、密码、Cookie。\n";
  }
  function copyPlain(text, okMsg) {
    var done = function () { toast(okMsg || "已复制", "ok"); };
    if (navigator.clipboard && navigator.clipboard.writeText) {
      navigator.clipboard.writeText(text).then(done).catch(function () { copyTextFallback(text, done); });
      return;
    }
    copyTextFallback(text, done);
  }
  function renderAI() {
    var cont = $("#view-ai");
    var master = state.role === "master";
    cont.innerHTML = '<div class="page-head"><div><span class="eyebrow">ACCESS / AI</span><h2>AI 接入</h2><p>创建 API Key，把控制台能力交给外部 AI 或脚本；接口文档无需登录即可阅读。</p></div><div class="spacer"></div>'
      + '<a class="btn btn-ghost btn-sm" href="/jiying/ai.txt" target="_blank" rel="noopener">接入说明</a>'
      + '<a class="btn btn-ghost btn-sm" href="/jiying/openapi.json" target="_blank" rel="noopener">OpenAPI</a></div>'
      + '<div class="grid grid-3">'
      + aiStep("01", "创建密钥", "为不同用途分别建钥匙，权限最小化。")
      + aiStep("02", "复制说明", "说明包含地址、认证与常用流程，直接粘贴给 AI。")
      + aiStep("03", "直接调用", "密钥放请求头，先读清单再执行操作。")
      + "</div>"
      + '<div class="panel mt-16"><div class="panel-head"><h3>API Key</h3><span class="hint key-warn">密钥只在创建时显示一次，妥善保存。</span></div><div class="panel-body">'
      + '<div class="grid grid-3 key-create-grid" id="keyCreateRow">'
      + '<div class="field"><label for="keyLabel">备注</label><input class="input" id="keyLabel" placeholder="例如 巡检机器人" maxlength="40"></div>'
      + '<div class="field"><label for="keyScope">权限范围</label><select class="select" id="keyScope"><option value="ops">运营</option><option value="readonly">只读</option><option value="master">主人</option></select></div>'
      + '<div class="field"><label for="keyDays">有效期</label><div class="input-row"><select class="select" id="keyDays" style="flex:1"><option value="30">30 天</option><option value="7">7 天</option><option value="90">90 天</option><option value="0">永久</option></select><button class="btn btn-primary" id="newKey">创建密钥</button></div></div>'
      + '</div>'
      + '<div id="keyList" class="mt-16">' + loadingState() + "</div></div></div>"
      + (roleAllowed("ops") ? uploadPanelHTML() : "")
      + '<div class="panel mt-16"><div class="panel-head"><h3>给 AI 的说明书</h3><div class="spacer"></div><button class="btn btn-ghost btn-sm" id="copyPrompt">复制说明</button><button class="btn btn-ghost btn-sm" id="downloadGuide">下载说明</button></div><div class="panel-body"><pre class="ai-prompt mono" id="aiPrompt"></pre></div></div>';
    $("#aiPrompt").textContent = aiPromptText("");
    if (state.role && !master) $("#keyCreateRow").hidden = true;
    $("#newKey").addEventListener("click", createAPIKey);
    if (roleAllowed("ops")) bindUploadPanel();
    $("#copyPrompt").addEventListener("click", function () { copyPlain(aiPromptText(""), "已复制说明"); });
    $("#downloadGuide").addEventListener("click", function () {
      downloadBlob(new Blob([aiPromptText("")], { type: "text/plain;charset=utf-8" }), "relay-ai-guide.txt");
    });
    loadAPIKeys();
  }
  function uploadPanelHTML() {
    return '<div class="panel mt-16" id="uploadPanel"><div class="panel-head"><h3>文件上传</h3><span class="hint">最大 20 MB。上传后把 file_id 交给发送接口。</span></div><div class="panel-body">'
      + '<div class="grid grid-2"><div class="field"><label for="uploadFile">选择文件</label><input class="input" type="file" id="uploadFile"></div>'
      + '<div class="field"><label for="uploadName">文件名（可选）</label><input class="input" id="uploadName" placeholder="留空则使用原文件名"></div></div>'
      + '<div class="input-row"><button class="btn btn-primary" id="uploadBtn">上传并获取 file_id</button></div>'
      + '<div id="uploadResult" class="mt-16" hidden></div></div></div>';
  }
  function bindUploadPanel() {
    var btn = $("#uploadBtn");
    if (btn) btn.addEventListener("click", uploadTempFile);
  }
  function uploadTempFile() {
    var input = $("#uploadFile");
    var file = input && input.files && input.files[0];
    if (!file) { toast("请选择文件", "err"); return; }
    if (file.size > 20 * 1024 * 1024) { toast("文件不能超过 20 MB", "err"); return; }
    var fd = new FormData();
    fd.append("file", file);
    var name = ($("#uploadName").value || "").trim();
    if (name) fd.append("name", name);
    var btn = $("#uploadBtn");
    btn.disabled = true;
    api("/jiying/api/upload", { method: "POST", body: fd, timeout: 120000 }).then(function (r) {
      btn.disabled = false;
      if (r.code !== 200) { toast(r.msg || r.message || "上传失败", "err"); return; }
      var d = r.data || {};
      var box = $("#uploadResult");
      box.hidden = false;
      box.innerHTML = '<div class="field"><label>file_id</label><div class="input-row"><input class="input mono" id="uploadFileId" readonly value="' + esc(d.file_id || "") + '">'
        + '<button class="btn btn-ghost btn-sm" id="copyFileId">复制</button></div></div>'
        + '<p class="hint">发送时调用 POST /jiying/api/send/file，传入 conversation_id、file_id、name、size。</p>';
      $("#copyFileId").addEventListener("click", function () { copyPlain(d.file_id || "", "已复制 file_id"); });
      toast("上传成功", "ok");
    });
  }
  function apiKeyStatus(k) {
    if (k.expires_at && Date.parse(k.expires_at) < Date.now()) return { cls: "badge-off", label: "已过期" };
    return { cls: "badge-ok", label: "有效" };
  }
  function loadAPIKeys() {
    api("/jiying/api/keys", { silent: true }).then(function (r) {
      var box = $("#keyList");
      if (!box) return;
      if (r.code === 403) {
        var row = $("#keyCreateRow");
        if (row) row.hidden = true;
        box.innerHTML = emptyState("shield", "仅主人可管理密钥。", "当前账号为只读或运营，可阅读文档但不能签发密钥。");
        return;
      }
      var createRow = $("#keyCreateRow");
      if (createRow) createRow.hidden = false;
      if (r.code !== 200) {
        errorState(box, "key", "密钥加载失败", r.msg || r.message || "请稍后重试。", loadAPIKeys);
        return;
      }
      var keys = (r.data && r.data.keys) || [];
      if (!keys.length) {
        box.innerHTML = emptyState("key", "暂无 API Key", "新建一把密钥后即可让 AI 接入。", '<button class="btn btn-primary btn-sm" id="keyEmptyAdd">创建密钥</button>');
        var add = $("#keyEmptyAdd");
        if (add) add.addEventListener("click", function () {
          var label = $("#keyLabel");
          if (label) { label.focus(); label.scrollIntoView({ block: "center", behavior: "smooth" }); }
        });
        return;
      }
      box.innerHTML = '<div class="table-wrap"><table class="table align-middle"><thead><tr><th>备注</th><th class="nowrap">前缀</th><th class="nowrap">权限</th><th class="nowrap">创建于</th><th class="nowrap">最后使用</th><th class="nowrap">状态</th><th class="text-right">操作</th></tr></thead><tbody>'
        + keys.map(function (k) {
          var st = apiKeyStatus(k);
          return "<tr><td>" + esc(k.label || "-") + '</td><td class="mono">' + esc(k.prefix || "") + '…</td><td>' + esc(EN_TEXT[k.scope === "master" ? "主人" : k.scope === "ops" ? "运营" : "只读"] || k.scope)
            + "</td><td class=\"mono\">" + esc((k.created_at || "").slice(0, 10)) + "</td><td class=\"mono\">" + esc(k.last_used_at ? k.last_used_at.slice(0, 16).replace("T", " ") : "-")
            + '</td><td><span class="badge ' + st.cls + '">' + esc(st.label) + '</span></td><td class="text-right nowrap"><button class="btn btn-ghost btn-sm" data-rotate="' + esc(k.id) + '" data-label="' + esc(k.label || "") + '">轮换</button> <button class="btn btn-outline-danger btn-sm" data-revoke="' + esc(k.id) + '" data-label="' + esc(k.label || "") + '">撤销</button></td></tr>';
        }).join("") + "</tbody></table></div>";
      $$("[data-revoke]", box).forEach(function (btn) {
        btn.addEventListener("click", function () { revokeAPIKey(btn.getAttribute("data-revoke"), btn.getAttribute("data-label")); });
      });
      $$("[data-rotate]", box).forEach(function (btn) {
        btn.addEventListener("click", function () { rotateAPIKey(btn.getAttribute("data-rotate"), btn.getAttribute("data-label")); });
      });
    });
  }
  function createAPIKey() {
    var label = ($("#keyLabel").value || "").trim();
    if (!label) { toast("请输入备注", "err"); return; }
    var btn = $("#newKey");
    btn.disabled = true;
    api("/jiying/api/keys", { method: "POST", form: { label: label, scope: $("#keyScope").value, expires_days: $("#keyDays").value } }).then(function (r) {
      btn.disabled = false;
      if (r.code !== 200) { toast(r.msg || r.message || "生成失败", "err"); return; }
      $("#keyLabel").value = "";
      loadAPIKeys();
      showNewKey(r.data && r.data.token);
    });
  }
  function revokeAPIKey(id, label) {
    confirmDialog("撤销 API Key", "撤销后，使用「" + label + "」的调用会立即失效。", "撤销", function () {
      api("/jiying/api/keys/delete", { method: "POST", form: { id: id } }).then(function (r) {
        if (r.code === 200) { toast("已撤销", "ok"); loadAPIKeys(); }
        else toast(r.msg || r.message || "撤销失败", "err");
      });
    });
  }
  function rotateAPIKey(id, label) {
    confirmDialog("轮换 API Key", "将立即作废「" + label + "」的旧密钥并生成新密钥，正在使用旧密钥的调用会中断。", "轮换", function () {
      api("/jiying/api/keys/rotate", { method: "POST", form: { id: id } }).then(function (r) {
        if (r.code === 200) { toast("已轮换", "ok"); loadAPIKeys(); showNewKey(r.data && r.data.token); }
        else toast(r.msg || r.message || "轮换失败", "err");
      });
    });
  }
  var keyRevealOpen = false;
  function showNewKey(token) {
    if (!token) return;
    keyRevealOpen = true;
    showModal("API Key 已创建",
      '<p class="rl-confirm-msg">这是唯一一次显示明文，关闭后无法找回。</p>'
      + '<div class="key-token mono" id="newKeyToken">' + esc(token) + "</div>"
      + '<div class="input-row mt-16"><button class="btn btn-ghost btn-sm" id="copyToken">复制密钥</button>'
      + '<button class="btn btn-ghost btn-sm" id="copyKeyPrompt">复制接入说明</button></div>',
      '<button class="btn btn-primary" id="keySaved">我已保存</button>');
    $("#copyToken").addEventListener("click", function () { copyPlain(token, "已复制"); });
    $("#copyKeyPrompt").addEventListener("click", function () { copyPlain(aiPromptText(token), "已复制说明"); });
    $("#keySaved").addEventListener("click", function () {
      keyRevealOpen = false;
      closeModal();
    });
  }

  function tile(label, val, sub) {
    return '<div class="stat-cell tile"><span class="t-lab">' + esc(label) + '</span><span class="t-val">' + esc(val) + "</span>" + (sub ? '<span class="t-sub">' + esc(sub) + "</span>" : "") + "</div>";
  }
  function setTile(i, val, sub, tone) {
    var t = $$("#observeTiles .tile")[i];
    if (!t) return;
    var v = t.querySelector(".t-val");
    if (v) {
      var changed = v.textContent !== val;
      v.textContent = val;
      v.className = "t-val" + (tone ? " t-" + tone : "");
      if (changed) { v.classList.remove("rl-flash"); void v.offsetWidth; v.classList.add("rl-flash"); }
    }
    if (sub) { var s = t.querySelector(".t-sub"); if (s) s.textContent = sub; }
  }
  function fmtDur(sec) {
    sec = Math.max(0, Math.floor(sec));
    var d = Math.floor(sec / 86400), h = Math.floor((sec % 86400) / 3600), m = Math.floor((sec % 3600) / 60);
    if (d) return d + "d " + h + "h";
    if (h) return h + "h " + m + "m";
    if (m) return m + "m " + (sec % 60) + "s";
    return sec + "s";
  }
  // 解析 /metrics 里的 RPC 聚合行，返回按方法名排序的表格行。
  function rpcRows(txt) {
    var rows = {};
    function row(method) {
      if (!rows[method]) rows[method] = { method: method, ok: 0, failed: 0, avg: null, codes: [] };
      return rows[method];
    }
    txt.split("\n").forEach(function (line) {
      var m = line.match(/^jiying_app_calls_total\{method="([^"]+)",outcome="(ok|error)"\}\s+([0-9.eE+-]+)$/);
      if (m) {
        var r = row(m[1]);
        if (m[2] === "ok") r.ok += parseFloat(m[3]);
        else r.failed += parseFloat(m[3]);
        return;
      }
      m = line.match(/^jiying_app_call_errors_total\{method="([^"]+)",code="([^"]+)"\}\s+([0-9.eE+-]+)$/);
      if (m) { row(m[1]).codes.push(m[2] + " × " + m[3]); return; }
      m = line.match(/^jiying_app_call_duration_milliseconds_avg\{method="([^"]+)"\}\s+([0-9.eE+-]+)$/);
      if (m) { row(m[1]).avg = parseFloat(m[2]); }
    });
    return Object.keys(rows).sort().map(function (k) { return rows[k]; });
  }
  function renderRpc(txt) {
    var box = $("#observeRpc");
    if (!box) return;
    var rows = rpcRows(txt);
    if (!rows.length) {
      box.innerHTML = emptyState("activity", "暂无 RPC 调用", "应用通道产生调用后，这里会按方法显示成功率与耗时。");
      return;
    }
    box.innerHTML = '<div class="table-wrap"><table class="table table-sm"><thead><tr>'
      + '<th>方法</th><th class="text-end">成功</th><th class="text-end">失败</th><th class="text-end">平均耗时</th><th>错误码</th>'
      + "</tr></thead><tbody>"
      + rows.map(function (r) {
        var avg = r.avg == null ? "—" : (r.avg < 10 ? r.avg.toFixed(1) : Math.round(r.avg)) + " ms";
        return "<tr><td class=\"mono\">" + esc(r.method) + "</td>"
          + '<td class="text-end mono">' + r.ok + "</td>"
          + '<td class="text-end mono' + (r.failed ? " text-danger" : "") + '">' + r.failed + "</td>"
          + '<td class="text-end mono">' + avg + "</td>"
          + '<td class="mono muted">' + (r.codes.length ? esc(r.codes.join(" · ")) : "—") + "</td></tr>";
      }).join("")
      + "</tbody></table></div>";
  }
  // 从 Prometheus 文本中取指标值：metric(name,{labels}) 精确匹配样本名。
  function metricVal(txt, name, labelMatch) {
    var re = new RegExp("^" + name + "(?:\\{[^}]*\\})?\\s+([0-9.eE+-]+)\\s*$", "m");
    var lines = txt.split("\n");
    for (var i = 0; i < lines.length; i++) {
      if (lines[i].charAt(0) === "#") continue;
      if (labelMatch && lines[i].indexOf(labelMatch) < 0) continue;
      var m = lines[i].match(re);
      if (m) return parseFloat(m[1]);
    }
    return null;
  }
  function loadObserve() {
    fetch("/healthz").then(function (r) { return r.status; }).then(function (s) {
      setTile(0, s === 200 ? "正常" : "异常", s === 200 ? "GET /healthz · 200" : "GET /healthz · " + s, s === 200 ? "ok" : "err");
    }).catch(function () { });
    fetch("/metrics").then(function (r) { return r.text(); }).then(function (txt) {
      var box = $("#observeRaw");
      if (box) box.textContent = txt || "（无指标）";
      renderRpc(txt || "");
      var up = metricVal(txt, "jiying_uptime_seconds");
      if (up != null) { setTile(1, fmtDur(up), "进程已运行"); syncUptime(up); }
      var msg = metricVal(txt, "jiying_messages_total");
      if (msg != null) setTile(2, String(msg), "累计接收");
      var run = metricVal(txt, "jiying_tasks", 'status="running"');
      var pend = metricVal(txt, "jiying_tasks", 'status="pending"');
      var done = metricVal(txt, "jiying_tasks", 'status="done"');
      var fail = metricVal(txt, "jiying_tasks", 'status="failed"');
      if (run != null) setTile(3, String(run), "待 " + (pend || 0) + " · 完成 " + (done || 0));
      var app = metricVal(txt, "jiying_channel_up", 'channel="app"');
      var acc = metricVal(txt, "jiying_channel_up", 'channel="account"');
      if (app != null) setTile(4, (app ? "应用在线" : "应用离线"), "账号" + (acc ? "已登录" : "未登录"), app ? "ok" : "err");
      var rl = metricVal(txt, "jiying_ratelimit_rejected_total");
      if (rl != null) setTile(5, String(rl), "累计拒绝 · 任务失败 " + (fail || 0));
    }).catch(function () { });
  }

  /* ---------- 渲染表 ---------- */
  var renderers = {
    home: renderHome,
    connect: renderConnect,
    pool: renderPool,
    dingtalk: renderDingTalk,
    chat: renderChat,
    scripts: renderScripts,
    features: renderFeatures,
    agent: renderAgent,
    cron: renderCron,
    executions: renderExecutions,
    tasks: renderTasks,
    market: renderMarket,
    audit: renderAudit,
    observe: renderObserve,
    logs: renderLogs,
    roles: renderRoles,
    ai: renderAI
  };

  // 需要定时轮询的视图：切换视图时自动启动，离开时由 switchView 清理。
  var POLL = { home: function () { refreshHome(true); }, features: loadFeatures, observe: loadObserve, executions: function () { loadExecutions(true); } };
  var POLL_MS = { home: 10000, features: 10000, observe: 5000, executions: 8000 };

  /* ---------- 启动 ---------- */
  /* ---------- 吉祥物挂件 ---------- */
  // 素材为 18 帧交付动画 + 8 方向像素切片。
  // 默认开启，关闭后只留一个可恢复的小标签；偏好写入本机，不经过服务端。
  var MASCOT_KEY = "relay-mascot";
  var MASCOT_BASE = "/assets/mascot/";
  var MASCOT_SLICES = ["front", "front_right", "right", "back_right", "back", "back_left", "left", "front_left"];
  var mascotState = { ready: false, acting: false, timer: null, ws: null, acct: null };
  function mascotEnabled() {
    try { return localStorage.getItem(MASCOT_KEY) !== "0"; } catch (e) { return true; }
  }
  function setMascotEnabled(on) {
    try { localStorage.setItem(MASCOT_KEY, on ? "1" : "0"); } catch (e) { /* 忽略存储失败 */ }
  }
  function mascotT(zh, en) { return LANG === "en" ? en : zh; }
  // 点击挂件时轮换的短句，避免每次都同一句。
  var MASCOT_LINES = [
    ["手机给你，验收一下", "Here you go. Take a look"],
    ["活儿干完了，随时喊我", "All done. Call me anytime"],
    ["消息我盯着，放心", "I'm watching the messages"],
    ["插件装好了，试试看", "Plugins are ready. Try them"],
    ["出错别慌，我在这儿", "Don't panic. I'm here"],
    ["通道畅通，消息不丢", "Channel is clear. No lost messages"],
    ["今天也要顺利交付", "Smooth delivery today"]
  ];
  var mascotLineIndex = 0;
  function mascotNextLine() {
    var line = MASCOT_LINES[mascotLineIndex % MASCOT_LINES.length];
    mascotLineIndex++;
    return mascotT(line[0], line[1]);
  }
  function mascotSlice(dir) {
    var img = $("#duduImg");
    if (!img || !dir || mascotState.acting) return;
    img.src = MASCOT_BASE + "slices/" + dir + ".webp";
  }
  function mascotShow(on) {
    var box = $("#dudu"), restore = $("#duduRestore");
    if (box) box.hidden = !on;
    if (restore) restore.hidden = on;
    setMascotEnabled(on);
    if (on) setTimeout(function () { mascotPlay(mascotT("我在这儿，有事叫我", "Right here if you need me"), 3800); }, 220);
  }
  // 播放交付动画：GIF 重新挂载以从头播放，结束后回到正面静态切片。
  function mascotPlay(text, ms) {
    if (!mascotState.ready || !mascotEnabled()) return;
    var box = $("#dudu"), img = $("#duduImg"), bubble = $("#duduBubble");
    if (!box || !img) return;
    mascotState.acting = true;
    box.classList.add("acting");
    if (bubble) {
      bubble.textContent = text || mascotT("交给我", "On it");
      bubble.hidden = false;
      bubble.classList.remove("pop");
      void bubble.offsetWidth;
      bubble.classList.add("pop");
    }
    img.src = MASCOT_BASE + "dudu-take-phone.gif?t=" + Date.now();
    clearTimeout(mascotState.timer);
    mascotState.timer = setTimeout(function () {
      mascotState.acting = false;
      box.classList.remove("acting");
      img.src = MASCOT_BASE + "slices/front.webp";
      if (bubble) bubble.hidden = true;
    }, ms || 7200);
  }
  function mascotReact(kind, msg) {
    if (!mascotState.ready || kind === "err" || !mascotEnabled()) return;
    var line = mascotT("搞定，交给我", "Done. Leave it to me");
    if (/连接|接入|通道/.test(msg || "")) line = mascotT("通道已接通", "Channel connected");
    else if (/保存|创建|安装|更新|成功/.test(msg || "")) line = mascotT("已经办好了", "All set");
    else if (/删除|停用|撤销/.test(msg || "")) line = mascotT("清理完毕", "Cleaned up");
    mascotPlay(line, 6800);
  }
  function installMascot() {
    if (document.getElementById("dudu")) return;
    var wrap = document.createElement("div");
    wrap.id = "dudu";
    wrap.className = "dudu";
    wrap.setAttribute("data-no-i18n", "");
    wrap.innerHTML = '<div class="dudu-bubble" id="duduBubble" role="status" aria-live="polite" hidden></div>'
      + '<div class="dudu-card">'
      + '<button type="button" class="dudu-stage" id="duduStage" title="' + esc(mascotT("点一下看我表演", "Click for a trick")) + '" aria-label="' + esc(mascotT("点击播放交付动画", "Play the handoff animation")) + '"><img id="duduImg" src="' + MASCOT_BASE + 'slices/front.webp" alt="' + esc(mascotT("吉祥物", "Mascot")) + '" draggable="false"></button>'
      + '<div class="dudu-bar"><span class="dudu-dot" aria-hidden="true"></span>'
      + '<button type="button" class="dudu-x" id="duduClose" title="' + esc(mascotT("关闭挂件", "Hide mascot")) + '" aria-label="' + esc(mascotT("关闭挂件", "Hide mascot")) + '">×</button></div>'
      + '</div>';
    document.body.appendChild(wrap);
    var restore = document.createElement("button");
    restore.type = "button";
    restore.id = "duduRestore";
    restore.className = "dudu-restore";
    restore.hidden = true;
    restore.title = mascotT("叫回挂件", "Bring mascot back");
    restore.setAttribute("aria-label", mascotT("显示挂件", "Show mascot"));
    restore.innerHTML = '<img src="' + MASCOT_BASE + 'slices/front.webp" alt="">';
    document.body.appendChild(restore);
    mascotState.ready = true;
    var stage = $("#duduStage");
    if (stage) {
      stage.addEventListener("click", function () { mascotPlay(mascotNextLine(), 7200); });
      stage.addEventListener("pointermove", function (e) {
        if (mascotState.acting || window.innerWidth < 760) return;
        var r = stage.getBoundingClientRect();
        var dx = e.clientX - (r.left + r.width / 2);
        var dy = e.clientY - (r.top + r.height / 2);
        var ang = (Math.atan2(dy, dx) * 180 / Math.PI + 360 + 22.5) % 360;
        mascotSlice(MASCOT_SLICES[Math.floor(ang / 45) % 8]);
      });
      stage.addEventListener("pointerleave", function () { mascotSlice("front"); });
    }
    var close = $("#duduClose");
    if (close) close.addEventListener("click", function () { mascotShow(false); });
    restore.addEventListener("click", function () { mascotShow(true); });
    // 预载 8 个方向，鼠标环绕时切换不闪白。
    MASCOT_SLICES.forEach(function (d) { var im = new Image(); im.src = MASCOT_BASE + "slices/" + d + ".webp"; });
    if (!mascotEnabled()) mascotShow(false);
    else {
      wrap.hidden = false;
      restore.hidden = true;
      setTimeout(function () {
        if (location.pathname.indexOf("login.html") >= 0) mascotPlay(mascotT("登录后，活都交给我", "Sign in and leave the work to me"), 7200);
      }, 900);
    }
  }

  function init() {
    installErrorSink();
    installTheme();
    // 语言：读取本机偏好并翻译静态外壳，登录页同样生效。
    LANG = readLang();
    document.documentElement.lang = (LANG === "en") ? "en" : "zh-CN";
    applyStaticI18n();
    refreshLangButtons();
    if (LANG === "en") { i18nStartObserver(); i18nLocalize(document.body); }
    hydrateIcons();
    var lb = $("#langBtn");
    if (lb) lb.addEventListener("click", toggleLang);
    installMascot();
    if (location.pathname.endsWith("/login.html") || location.pathname === "/login.html") return;
    hydrateIcons();
    renderNav();
    var nt = $("#navToggle");
    if (nt) nt.addEventListener("click", toggleNav);
    var nc = $("#navCustomBtn");
    if (nc) nc.addEventListener("click", openNavCustomModal);
    var nb = $("#navBackdrop");
    if (nb) nb.addEventListener("click", closeNav);
    var ncl = $("#navClose");
    if (ncl) ncl.addEventListener("click", closeNav);
    document.addEventListener("keydown", function (e) { if (e.key === "Escape") closeNav(); });
    window.addEventListener("resize", function () { if (!isMobile()) closeNav(); hintScroll(); });
    // 快捷键：“/” 或 Ctrl/Cmd+K 聚焦菜单筛选框（输入框内不拦截）。
    document.addEventListener("keydown", function (e) {
      var slash = e.key === "/" && !e.ctrlKey && !e.metaKey && !e.altKey;
      var cmdK = e.key.toLowerCase() === "k" && (e.ctrlKey || e.metaKey);
      if (!slash && !cmdK) return;
      if (cmdK) e.preventDefault();
      var t = e.target;
      if (t && (t.tagName === "INPUT" || t.tagName === "TEXTAREA" || t.tagName === "SELECT" || t.isContentEditable)) return;
      var f = $("#navFilter");
      if (f && !isMobile()) { e.preventDefault(); f.focus(); f.select(); }
    });
    // 表格内容异步载入后再计算一次横向溢出提示
    var content = document.querySelector(".content");
    if (content && window.MutationObserver) {
      new MutationObserver(hintScroll).observe(content, { childList: true, subtree: true });
    }
    var lo = $("#logoutBtn");
    if (lo) lo.addEventListener("click", function () {
      api("/logout", { method: "POST" }).then(function () { location.href = "login.html"; });
    });
    api("/login").then(function (r) {
      if (r.code === 200) {
        $("#userName").textContent = r.data && r.data.user ? r.data.user : "管理员";
      } else {
        if (location.pathname.indexOf("login.html") < 0) location.href = "login.html";
        return;
      }
      applyCachedProfile();
      loadConnectStatus();
      // 顶栏显示当前角色等级，便于用户判断自己可执行的操作范围。
      api("/jiying/api/roles", { silent: true }).then(function (rr) {
        if (rr.code === 200 && rr.data) {
          state.role = rr.data.me || "";
          var el = $("#userRole");
          if (el && ROLE_LABEL[state.role]) el.textContent = "角色 · " + ROLE_LABEL[state.role];
          renderNav();
          var cur = NAV.filter(function (x) { return x.id === state.view; })[0];
          if (cur && !roleAllowed(cur.minRole)) switchView("home");
          else if (state.view === "ai" && !roleAllowed("ops")) renderAI();
        }
      });
      var wanted = savedViewId();
      var target = wanted && NAV.filter(function (x) { return x.id === wanted; })[0];
      switchView(target && roleAllowed(target.minRole) ? wanted : defaultViewId());
      setInterval(function () { loadConnectStatus(); }, 15000);
    }).catch(function () { location.href = "login.html"; });
  }
  var modal = $("#modal");
  if (modal) {
    modal.addEventListener("click", function (e) { if (e.target === this) closeModal(); });
    modal.addEventListener("hide.bs.modal", function (e) {
      if (keyRevealOpen) {
        e.preventDefault();
        toast("请先复制密钥，再点击「我已保存」", "info");
      }
    });
  }

  window.Relay = {
    go: switchView,
    closeModal: closeModal,
    customizeNav: openNavCustomModal,
    refreshHome: refreshHome,
    pickConv: pickConv,
    editScript: editScript,
    scriptConfig: openScriptConfig,
    editPluginOf: editPluginOf,
    runScript: runScript,
    delScript: delScript,
    runSkill: runSkill,
    delSkill: delSkill,
    loadFeatures: loadFeatures,
    setFeature: setFeature,
    runFeature: runFeature,
    delFeature: delFeature,
    featureDetail: featureDetail,
    editScriptApps: editScriptApps,
    toggleCron: toggleCron,
    editCron: editCron,
    delCron: delCron,
    loadExecutions: loadExecutions,
    gotoExecutions: gotoExecutions,
    runCron: runCron,
    checkAccountHealth: checkAccountHealth,
    loadTasks: loadTasks,
    retryTask: retryTask,
    delTask: delTask,
    renderConnStatus: renderConnStatus,
    selectAccount: selectAccount,
    showAppMatrix: showAppMatrix,
    renderDingTalk: renderDingTalk,
    removeMaster: removeMaster,
    deleteRole: deleteRole,
    loadPool: loadPool,
    editApp: editApp,
    delApp: delApp,
    defaultApp: defaultApp,
    loadMarket: loadMarket,
    installPlugin: installPlugin,
    delPlugin: delPlugin,
    loadAudit: loadAudit,
    loadObserve: loadObserve,
    searchLogs: searchLogs,
    taskDetail: taskDetail,
    auditOK: auditOK
  };
  // 供登录页内联脚本与自测脚本复用。
  window.RelayI18N = {
    t: t,
    lang: function () { return LANG; },
    toggle: toggleLang,
    apply: applyStaticI18n,
    refresh: refreshLangButtons
  };

  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", init);
  else init();
})();
