# 即应中间件开发方案

## 目标

即应中间件作为即应应用的中间件运行。即应只做操作面：收消息、回消息、发选择卡片。中间件负责认人、排队、执行、把结果发回原来的会话。

接上即应应用，或再接一个普通账号之后，主控可以在聊天里管理中间件自己：看状态、改定时、安装功能、写脚本并测试。普通用户只能触发已经授权的功能。

不接 QQ、微信、京东、售卡。不把即应自带的 assistant 进程嵌进来。

## 两个身份

应用身份走 `wss://<host>/api/app/ws`，请求头带 `X-MagicChat-App-ID` 和 `Authorization: Bearer <连接密钥>`。它能做第三方文档里的应用 RPC：发消息、列会话、读历史、话题、群管理、文件签名 URL、确认事件。

账号身份用邮箱密码登录，拿会话后再走用户通道。联系人、项目、任务、未点名的群消息，只在功能声明需要这个身份时使用。

改插件、改定时、看日志、写脚本，不看即应权限，只认中间件里保存的主控用户 ID。

密钥放环境变量：`JIYING_WS_URL`、`JIYING_APP_ID`、`JIYING_APP_SECRET`。主控 ID 存配置，逗号分隔。密码和连接密钥不进仓库，不回显到页面。

## 消息怎么走

```
即应事件
  → 应用通道
  → 现有消息入口
  → 关键词回复 / 系统命令 / 功能
  → 任务队列
  → 结果发回该会话
```

`message.created` 变成一条内部消息。用户 ID、会话 ID、消息 ID 保留，回复时带上原消息 ID。群里的普通消息由即应决定是否推送，中间件不再滤一遍。

`choice.response_created` 用来放行挂起的任务。`topic.closed` 之后不再往该话题发消息。`conversation.status` 是瞬时状态，不确认、不补发。

可靠事件按 cursor 串行处理。业务和后续调用都成功后才 `events.ack`。失败不确认，重连后由服务端重放。同一应用可能多连接收到同一事件，用应用 ID 加 cursor 去重，结果落到本地存储。断线后指数退避重连，并在退避窗口内加入抖动，避免多实例同时重连形成尖峰。

多实例边界：去重状态由 `ackWatermark`、已确认集合与最近 512 条 cursor 台账组成，落在本地 `BucketApp`（键 `jiying_ack_<appID>` 与 `jiying_evt_<appID>`）。`events.ack` 单调推进水位，确认失败但业务已成功时也会先记台账，重放时只确认不重复执行。因此同一 `appID` 应只运行一个写入实例；多个进程若各自使用独立存储，无法共享水位与台账，会让同一条事件在每个实例各执行一次。需要横向扩容时，让共享同一存储的实例串行消费，或改用单写者加外部去重表，并保证业务处理本身幂等。

## 任务

聊天、定时、外部 webhook 进同一个队列。一条任务记录功能名、参数、会话、发起人、状态和结果。状态为待执行、待确认、执行中、完成、失败、已取消、超时，单任务执行超时默认 2 分钟；记录耗时、重试次数与错误码。重启时，执行中的任务改回待执行。

定时沿用现有 cron。到点不直接跑，而是入队。跑完把摘要发回绑定的会话。

需要确认的功能先发选择卡片，用户点确认才入队执行。取消则标记失败，不调用外部服务。

## 功能目录

功能是登记项，不是写死的分支。字段：名字、类型、目标、范围、是否确认、使用哪个身份、是否启用。

类型三种：

- 内置：状态、日志、帮助、插件列表、定时列表、任务列表。
- HTTP：主控登记一个地址。调用时 POST JSON，带功能名、参数和用户 ID。
- 脚本：指向 `scripts/` 下的文件。

范围是 `public` 或 `master`。停用后命令和后续 Agent 都看不到它。

没有模型时，`跑 <功能名> <参数>` 直接入队。有模型以后，模型只负责选择已启用的功能，不能自己拿密钥，也不能调用未登记的地址。

## 脚本和 skill

脚本写入 `scripts/`，skill 写入 `skills/`。路径只允许这两个目录下的文件名，拒绝 `..`。

安装顺序：写入临时内容，在独立运行时里跑一遍，通过后再热加载并启用。失败则删除，不留在线上。只有主控能写、能装、能停用。

脚本与技能按 CommonJS 语义执行，运行时提供 `module` / `exports` / `require`：安装时先在沙箱试跑，热加载到线上引擎时同样具备这些全局，避免「试跑通过、上线报 `module is not defined`」。技能若以 `module.exports` 导出函数，控制台运行与插件 `require('xxx.js')` 都会以参数调用该函数；未导出函数时回退读取脚本的 `result`，空值统一为 `ok`。

插件可在头部用 `[config: key:type:default:desc]` 声明配置项，类型为 `string` / `number` / `bool` / `select` / `text` / `secret`（`select` 用 `a|b|c`，首项默认）。简写 `[config: key:说明]` 等价于 string。控制台按 schema 动态生成表单，`secret` 不回显（界面标注「已设置，留空则不修改」），保存时留空保持旧值。运行时把配置注入脚本：JS 通过全局 `config` 对象读取（number/bool 已类型化），Python 通过 `JIYING_CONFIG`（JSON）与逐项 `JIYING_CFG_<KEY>` 读取。配置按插件文件名存于桶 `plugincfg:<basename>`，删除插件时一并清理；审计只记录变更的配置键名，不记值。

插件还可用 `[class: 分类名]` 归档，用 `[service: 服务名]` 把同一业务/项目的多个插件收进同一文件夹，用 `[apps: app-a, app-b]` 声明可服务的应用白名单（`*` 或省略表示全部应用）。`/jiying/api/scripts` 会返回 `class` / `service` / `apps`，功能记录同步保存；执行功能与 Agent 工具裁剪都会校验当前应用是否命中白名单，未授权返回「该插件未授权当前应用」。控制台按「分类 → 服务 → 插件」三层展示：分类是顶层文件夹，服务是分类内的子文件夹，未声明服务的插件直接挂在分类下；搜索同时匹配分类、服务、应用与插件说明。

## 聊天命令

所有人：`帮助`、`我的ID`、`状态`、`功能列表`，以及已授权功能。

主控：`任务列表`、`定时列表`、`插件列表`、`安装功能 <名字> <地址>`、`安装脚本 <文件名.js> <代码>`、启停插件、增删定时。

命令结果必须回当前即应会话，不能只写本地日志。

## 控制台

保留一个说明页，告诉操作者环境变量和命令，不展示密钥。登录后的状态接口只返回功能列表、最近任务和账号是否已登录，不返回 Cookie。

日志推送只允许本机访问。

## 开发顺序

1. 应用通道：连上、收到文本、回复文本、确认事件、去重。
2. 主控命令把结果回会话。
3. 任务队列接上现有定时任务。
4. 功能目录，先支持 HTTP 和确认卡片。
5. 账号通道，只给声明了该身份的功能用。
6. 脚本安装、隔离测试、热加载。
7. 可选的薄 Agent：选功能或起草脚本，执行仍走功能目录。

每一步用假连接做测试，不依赖真实即应。联调时再提供部署地址、应用 ID 和连接密钥。

## 不做的事

不代用户发送，不使用即应未对普通应用开放的能力。不在中间件里保存即应连接密钥的副本到日志。不把外部 HTTP 功能开放给非主控登记。

## 当前实现补充（2026-09-25）

以下在原始设计之上落地，描述的是现有行为。

### 连接与身份

- 多应用连接池：可登记多条应用连接，其中一条为默认出口；未指定来源的任务走默认应用。控制台可切换默认应用、增删应用。
- 多账号池：可登记多个个人账号，同一时刻只有一个是当前账号；切换账号会以该账号重新登录。旧版单账号配置在首次读取时自动迁移为池中首个账号。账号密钥同样以 `enc:v1:` 密文落盘。
- 应用与账号通道分别显示在线与登录状态，并提供就绪自检与连通测试：应用侧做真实 WS 握手，账号侧调用 `/api/client/me`。
- 控制台回填的地址以掩码显示（如 `wss://********`）；提交时掩码与空值都按「不修改」处理，回退到已存地址，避免把掩码误存成真实地址导致通道失效。密钥同理，回传 `****` 时沿用已存密钥。
- 就绪清单前置到工作台：`/jiying/api/overview` 一并返回存储、应用、账号、模型四项就绪结果，首页按顺序编号显示状态、原因与跳转入口（就绪项显示「查看」，待处理项显示「去修复」）；接入页在存在待处理项时同步列出。
- 就绪项会一次列出该项的全部问题与修复提示（不只第一条）。应用通道额外校验 App ID 必须是 `8-4-4-4-12` 的 UUID：格式不符时保存前即拒绝（`connect/app` 与连接池写入同一规则），因为 MagicChat 对非 UUID 的应用 ID 会直接返回 400。

### 权限

- 角色为 master（主人）、ops（运营）、readonly（只读）三档，按操作而非按即应权限判定。
- 主人列表与角色分配在控制台管理；成员可用 `我的ID` 命令获取自己的用户 ID 后由主人授权。
- 主控范围的功能对主人与运营管理员均放行。
- 运行日志接口要求运营或主人：`GET /jiying/api/logs` 使用 `requireRole(RoleOps)`，只读角色在侧栏看不到入口，直接访问也会被拒绝。
- Agent 可见工具按调用者权限裁剪：主控专属与清单 Admin 标记的功能不会出现在无权用户的工具列表中，避免模型调用后才被拒绝。
- 控制台登录成功、失败与触发锁定都会写入审计（动作 `login.success` / `login.fail` / `login.blocked`，附带来源 IP），便于事后追溯口令猜解；登录失败按「来源 + 账号」计数锁定 5 分钟，避免共享出口 IP 下不同账号相互牵连。

### 任务与 Agent

- 任务生命周期含待确认、已取消、超时；失败任务可重试，重试累计次数。
- 控制台任务列表支持按状态（失败与超时合并为一组）和关键字（功能、会话、结果、错误码、任务 ID）过滤，并实时显示匹配计数；列表展示错误码与耗时，点击任务 ID 打开详情弹窗，可查看会话、用户、各阶段时间与结果，并一键跳转到运行日志按任务 ID 查找。
- Agent 唤醒策略：应用会话中用户的普通文本直接触发自动回复；群聊需 @ 应用才触发。Agent 只选功能与填参数，执行仍走队列。
- Agent 自动回复记录结果日志（会话、用户、提示与回复字数、耗时），失败单独告警，便于从运行日志排查。
- 功能既可用显示名（如 `状态`）调用，也可用唯一内部标识（如 `status`）调用；标识重复时只接受显示名，避免歧义。
- 队列限流按全局深度、单功能深度、单用户并发与单用户速率四层保护（阈值可用 `JIYING_MAX_QUEUE`、`JIYING_MAX_FEATURE_QUEUE`、`JIYING_MAX_USER_QUEUE`、`JIYING_USER_RATE_PER_MIN` 覆盖），被拒任务计入 `rate_limit_rejected` 指标；单用户令牌桶空闲超过一个满载周期后会被周期回收，避免长期运行时按用户数无界增长。
- 消息触发的直接执行（功能命令与 Agent 自动回复）另有并发上限（`JIYING_MSG_EXEC_CONCURRENCY`，默认 8）：槽位用满时立即回「系统繁忙，请稍后再试」并确认消息，避免消息洪峰把上游模型与本地脚本一起打爆或在通道内堆积重放。

### 接口与观测

- 控制台 API 统一返回 `{ok, code, message, error_code, data, trace_id}`；每个请求都会分配追踪 ID 并同时通过响应头 `X-Trace-ID` 回传与写入访问日志（`trace=`），客户端传入的 `X-Trace-ID` 会被透传复用，便于把一次调用与运行日志逐条关联。前端在错误提示中附带短追踪号（`（追踪 xxxxxxxx）`）与「去日志」按钮，点击即跳运行日志并把该前缀填入筛选框。
- 匿名探针 `/healthz`、`/readyz`、`/metrics` 供进程存活、就绪与 Prometheus 指标采集；就绪检查覆盖存储、任务队列与应用通道。运行观测页除健康、时长、任务与通道状态外，新增「RPC 调用」表，从 `/metrics` 解析各方法的成功/失败数、平均耗时与错误码分布，通道产生调用后自动出现。
- 访问日志跳过高频心跳 `/jiying/api/connect/status` 与静态资源请求，保留有意义的接口调用记录。
- 运行日志支持按关键字筛选、导出为 `.log` 文件，以及可选的自动刷新（勾选后每 5 秒拉取当日日志并滚到底部，离开视图即停止），便于 tail 式跟踪；审计日志支持按操作者 / 动作 / 目标 / 详情关键字筛选并显示匹配计数，每行可一键跳到运行日志并带入该动作作为筛选条件，配合 `trace_id` 完成从操作到日志的关联。审计日志可把当前筛选结果导出为 CSV（带 BOM 便于 Excel 显示中文，含时间 / 操作者 / 动作 / 目标 / 详情 / 结果六列），便于合规归档；审计加载失败时显示错误态与「重试」，不再与「暂无审计记录」混淆。审计结果的成功与失败按关键字判定（失败、拒绝、超时、错误、异常、无效、未授权），避免只匹配「失败」二字而把「拒绝」「执行超时」误标为成功。审计、任务与消息时间统一按本地 `YYYY-MM-DD HH:mm:ss` 展示，避免直接暴露 RFC3339 的 `T`/`Z`。
- 日志接口只从文件尾部读取（最多 2 MiB）再按 `tail` 参数截取所需行数，并在从中间截断时丢弃首行残段，避免为了最后几百行把整天的日志读进内存。
- 审计留痕写入 URL 前只保留协议、主机与路径，剥离 userinfo、查询串与片段，避免把挂在 URL 上的 token、签名写进审计存储。
- 侧栏菜单提供筛选框，按名称即时过滤并按分组隐藏空组，快捷键 `/` 聚焦筛选框、回车打开首个匹配项。
- 侧栏支持自定义：每项可置顶 / 上移 / 下移 / 隐藏，偏好存于 `localStorage`（`relay-nav-prefs`），置顶项集中显示在「置顶」分组，新增菜单自动补齐，弹窗内可一键恢复默认。
- 插件列表按头部 `icon` 渲染图标，支持 `bi-*` 图标名、`http(s)` / `data:image` / 相对图片路径，其他值按文本转义显示，避免注入；插件若声明了配置 schema，列表项出现「配置」按钮，打开动态表单填写并保存。
- 顶栏提供深浅色主题切换：默认深色，切换后写入 `localStorage`，登录页与主控制台共用同一偏好；`<head>` 内联脚本在样式加载前写入 `data-bs-theme`，避免首帧闪白。两套主题共用同一组设计令牌，基于 Bootstrap 5.3 的 `data-bs-theme` 生效。
- 提交类弹窗（运行功能、安装功能、新建定时、接入应用、上架插件）只在后端返回成功时关闭；失败时保持打开并保留已填内容，便于修正后重试，无需重新输入。
- 列表类面板把「加载失败」与「空数据」分开呈现：接口报错时显示错误态与「重试」按钮（原地重跑该加载函数），不再与空态混用，网络恢复后可一键恢复列表，无需刷新整页。覆盖插件、功能、市场、连接池、权限、审计以及任务队列 / 定时任务 / Agent 工具列表；表格内的错误态包在 `tr`/`td` 中，避免块级元素被浏览器移出 `tbody` 破坏表格布局。
- 顶栏在登录后显示当前角色等级（主人 / 运营 / 只读），取自 `GET /jiying/api/roles` 的 `me` 字段，便于用户判断可执行的操作范围。
- 前端内置全局错误采集（`window.__jsErrors` 保留最近 20 条未捕获异常与 Promise 拒绝），自测脚本据此判定页面无 JS 错误。
- 媒体能力贯穿功能与脚本：脚本可上传临时文件、读取签名地址与内容，并把图片、文件、卡片等结果发回会话。
- 插件页内嵌「技能」面板（不新增导航项）：列出 `skills/` 下的 JS 模块与 Markdown 文档，展示类型徽标与大小；JS 技能可一键运行并回显结果、可删除（删除前提示会波及引用它的插件），安装入口与插件一致，均走独立运行时试跑后再落盘。
- 模型降级：Agent 配置支持一个主模型加多个后备模型，主模型不可用（无渠道、限流、超时）时按顺序自动重试后备模型，成功与最终失败都会写入审计；后备模型可用环境变量 `USER_LLM_FALLBACK_MODELS` 或控制台配置，逗号分隔。
- Agent 配置里的 Base URL 同样以掩码回显，提交掩码或空值即保持原值；API Key 留空或回传 `****` 也保持原值，界面只提示「已配置」。

## 当前实现补充（2026-10-16）

对照《第三方应用开发指南》（协议 v1）补齐协议面能力。

### 应用通道协议

- 接收回执：命中处理（命令匹配或 Agent 唤醒）后立即异步回发 `message.reaction.set`，文本为 `🔥`、`reacted=true`。回执在业务执行前发出，不阻塞队列、不等待模型，失败只记 Debug 日志，不影响 `events.ack`。常量 `receiptReaction` 定义于 `core/jiying_app.go`。
- 限流退避：`message.send` 的滚动 60 秒限额（单应用 300 条 / 30 个不同用户）超限时服务端返回 `rate_limited`。`Call` 解析错误体中的 `retry_after_seconds`，按该值退避（上限 10 秒）后复用原请求 ID 重试一次；限流次数计入指标 `jiying_app_ratelimited_total`。
- 错误分类：`internal_error` 与传输层失败复用原请求 ID 重试；`rate_limited` 按退避重试；`unsupported_version`、`request_id_conflict`、`response_too_large`、`request_too_large`、`invalid_conversation`、`not_found`、`invalid_request`、`invalid_target`、`permission_denied` 属确定性失败，不重试并写一条带排查提示的告警日志（提示表 `jiyingFailureHints`）。
- 调用观测：`Call` 按方法聚合成功/失败次数、平均耗时与错误码，输出到 `/metrics` 的 `jiying_app_calls_total{method,outcome}`、`jiying_app_call_errors_total{method,code}` 与 `jiying_app_call_duration_milliseconds_avg{method}`；快照经 `JiyingCallStats()` 读取，方法名与错误码都做了归一，避免高基数标签。
- 话题上下文：`message.created` 解析 `conversation.parent`、`conversation.source_message` 与 `conversation.created_by_app_id`，落在 `JiyingSender` 的 `ParentConvID` / `ParentConvType` / `ParentConvName` / `SourceMessageID` / `CreatedByAppID`；`IsTopic()` 判断话题会话，父会话同时登记进会话归属映射以保证出站路由。
- 握手诊断：应用通道握手失败按 HTTP 状态给出可读原因（400 应用 ID 格式错误、401 密钥错误、403 应用已停用），最近一次失败原因经 `AppChannelLastError()` 在 `/jiying/api/connect/readiness` 的应用项中作为 `app_connect_failed` 问题展示，便于就地排障。
- 出站保护：单条文本经 `clampOutboundText` 截断到 256 KiB 以内（按 UTF-8 边界），避免 Envelope 触顶 1 MiB；`SendChoice` 在发送前校验正文长度（≤5000）、选项数量（2~20）、选项 ID（1~64，限 `[A-Za-z0-9_-]` 且不重复）与展示文本（1~200、无换行），非法参数直接返回错误而不发请求。
- 单测覆盖：`core/jiying_test.go` 新增接收回执、限流复用请求 ID、话题上下文解析、选择参数校验、握手错误文案、出站文本截断等用例。

### 应用通道 RPC 面（`core/jiying_rpc.go`）

按指南把应用通道对普通应用开放的方法补齐为类型化的 Go API，并接入脚本运行时。

- 目标类型化发送：`rpcTarget{Type,UserID,ConversationID}` 覆盖 `user` / `app` / `group` / `topic` / `conversation`，`SendTo` 统一构造 `message.send` 请求体；`SendMessage` 改为委托 `SendTo`，消除重复实现。`SendToConv` 按会话归属选通道。
- 查询类：`UsersGet`（`users.get`）、`AppsGet`（`apps.get`）、`ListConversations`（`conversations.list`）、`ListMessages`（`conversation.messages.list`，要求 `before_or_equal_seq` 为正整数）。
- 话题类：`TopicCreate` / `TopicGet` / `TopicClose`（`conversation.topic.create|get|close`），`JiyingTopic` 暴露话题会话 ID、父会话、来源消息与 `last_message_seq`。
- 群聊类：`GroupCreate` / `GroupGet` / `GroupUpdate` / `GroupDissolve` 与 `GroupMembers` / `GroupAddMembers` / `GroupRemoveMember` / `GroupSetRole`，并在发送前校验群名称（1~120 字符）、公告长度（≤200）、名称与公告只能改其一、`member_type`（user/app）与 `role`（admin/member）。
- 脚本能力：`registerSendApis` 末尾调用 `registerAppApis`，向 goja 注入完整应用通道能力。查询与回执：`appUser` / `appSelf` / `appConversations` / `appHistory` / `appReaction`；话题与状态：`appTopic`（查询）/ `appTopicCreate` / `appTopicClose` / `appStatus`；群管理：`groupCreate` / `groupGet` / `groupRename` / `groupAnnounce` / `groupDissolve` / `groupMembers` / `groupAddMembers` / `groupRemoveMember` / `groupSetRole`。查询类失败返回 `null`、写操作返回布尔，便于脚本判空。数组参数（成员/应用 ID）统一经 `gojaArgStrings` 清洗，空数组视为未提供。
- 选择与媒体便捷函数：脚本 `sendChoice(content, [{id, label}, ...])` 发送单选集卡片并返回消息 ID（失败返回空串），参数先经 `validateChoice` 校验；`lastMediaUrl()` 直接返回当前会话最近一条入站媒体的临时访问地址，等价于 `readFileUrl(lastMediaFileId())` 且省去脚本侧拼接。
- 覆盖范围：上述脚本接口覆盖指南对普通应用开放的全部 20 个方法，不含 `runas`、代用户发送与 `entity_card`。
- 单测覆盖：`core/jiying_rpc_test.go` 覆盖目标校验、用户/会话发送、查询解析、话题解析、群参数校验、脚本 API 行为（含群/话题/状态注入）与数组参数清洗。

### Agent 服务端历史上下文（`core/agent.go`）

- 序号捕获：`message.created` 解析 `message.seq` 存入 `JiyingSender.Seq`，作为读取服务端历史的锚点。
- 上下文来源：`RunAgent` 新增 `seq` 参数，经 `agentContext` 优先通过 `conversation.messages.list`（`before_or_equal_seq=seq`）拉取服务端可见历史；`seq` 缺失、未连接或读取失败时回退本地短历史（`getAgentHistory`），读取失败仅记 Debug，不阻断回复。
- 角色映射：`messagesToChatHistory` 把发送方 `type=app`（或 ID 等于本应用）映射为 assistant，其余为 user；`seq` 不小于锚点的消息（即当前 prompt）从历史中剔除以免重复；消息体经 `plainText` 取文本，缺失时回退 `summary`。
- 调用点：`handle_msg.go` 传入 `sender.Seq`；即应命令 `问` 经 `senderSeq` 传入；控制台 Agent 无锚点传入 0。
- 选择卡片：内置工具 `ask_choice(question, options)` 恒随功能工具一起下发。模型调用后发送单选集卡片并返回消息 ID，挂起项记录会话、用户与选项标签；`choice.response_created` 到达时由 `resumeAgentChoice` 异步恢复 Agent，把所选标签作为新提示继续对话并回发最终答复。未命中挂起项时仍走原有任务确认逻辑。
- 单测覆盖：`TestMessageCreatedEnrichesSender` 断言 seq 捕获；`TestMessagesToChatHistory` / `TestJiyingMessagePlainTextFallsBackToSummary` 覆盖角色映射、剔除当前消息与文本回退；`TestBuildAgentToolsIncludesAskChoice` 断言内置工具始终可用。

### 媒体消息 URL 化（`core/media.go`）

按指南要求，图片/文件消息体使用服务端可拉取的 URL，不再发送仅应用可见的 `file_id`。

- 解析入口：`resolveMediaContent` 统一处理媒体来源——传入绝对 `http(s)` URL 时原样返回，传入 `file_id` 时经 `appReadTemporaryURL`（`temporary_files.read_urls`）换取签名 URL；空白来源直接报错。`buildImageMessage` / `buildFileMessage` 先解析再构造消息体。
- 消息体字段差异（对齐服务端实现）：图片的远程 URL 放 `content`（服务端 `createRemoteImageMessageBody` 接受 `content` 或 `url`，`url` 为别名）；**文件的远程 URL 必须放 `url`**，`content` 在服务端只作内联纯文本（≤64KiB、`text/plain`，走 `createInlineFileMessageBody`）。两者同时提供会报「文件 URL 和内容只能二选一」。`ImageMessage(content, caption, captionType)` 写 `content` 与可选 `caption` / `caption_type`（默认 `text`，为 `markdown` 时保留）；`FileMessage(url, name, sizeBytes)` 写 `url` / `name` / `size_bytes`；新增 `MarkdownMessage(content)` 构造 `markdown` 富文本。
- 服务端拉取约束：URL 须为 `http(s)`，DNS 解析不得命中 loopback/私有/链路本地/多播/`100.64.0.0/10`（SSRF 防护，`validateLinkFetchAddr`），否则报「URL 不允许访问」；图片下载后转 WebP，服务端上限 `request_too_large`（图片 >5MiB、文件 >20MiB）。
- 调用面：控制台 `/send/image`、`/send/file` 同时接受 `url` 或 `file_id`（无 `url` 时回退 `file_id`），新增 `/send/markdown`；脚本 `sendImage` / `sendFile` 接受 URL 或 `file_id` 并支持 `caption_type`，新增 `sendMarkdown`；便捷函数 `SendImageToConv` / `SendFileToConv` 语义同步为「URL 或 `file_id`」，新增 `SendMarkdownToConv`。
- 单测覆盖：`TestMediaMessageBuilders` 断言图片 `content`/`caption_type` 与文件 `url`（并断言文件体不出现 `content`）；`TestResolveMediaContentPassthroughURL` 覆盖 URL 直通与空白报错。


### 控制台视觉系统（Ops Console，2026-10-17）

- 方向：工控台 / Ops Console。近黑石墨底、单一信号色（琥珀）、细线分隔；圆角统一 3px，阴影只用于浮层（弹窗、下拉、Toast）。
- 令牌：`relay.css` 顶部定义 `--bg` / `--panel` / `--panel-2` / `--line` / `--line-strong` / `--text` / `--muted` / `--primary` / `--ok` / `--warn` / `--err` / `--info` / `--sidebar*`，深浅色各自覆盖；语义色只用于状态，不用于装饰。
- 字体：界面用 IBM Plex Sans + Noto Sans SC，ID、指标、时间、键值、日志用 IBM Plex Mono（`Tabular numbers`），通过 Google Fonts 加载并带系统字体回退。
- 结构：左轨固定 236px（收起 60px），顶栏 54px；页面标题带等宽编号眉标（`OVERVIEW / 01` … `LOGS / 14`），由 `decoratePage` 统一补齐。工作台指标改为单一 `stat-strip`（8 格），观测页复用同一组件（6 格）。
- 列表：面板内列表改成发丝线分隔的行（`.panel-body > .list > .list-item`），不再使用独立卡片；表格表头为等宽大写小字。
- 状态：chip / badge 统一为直角 + 前置状态点，颜色由语义色按 `color-mix` 生成浅底与描边。
- 交互：保留 `/` 聚焦菜单筛选，新增 `Ctrl/Cmd+K`；窄栏可收起，860px 以下切换为抽屉并带遮罩。
- 无障碍：保留 `prefers-reduced-motion` 关闭入场动画，所有可聚焦控件使用 `:focus-visible` 描边。

## 当前实现补充（2026-10-17）

### 执行记录与定时任务

- 执行账本：`core/executions.go` 记录定时、手动与流程执行，内存保留最近 500 条，并异步单槽落盘到 `executions.json`（权限 `0600`，临时文件 + rename）；记录字段包含 `id` / `kind` / `name` / `command` / `status` / `started_at` / `finished_at` / `duration_ms` / `owner` / `trace_id` / `detail` / `error`。状态取 `success` / `failed` / `timeout` / `skipped` / `running`。
- 查询接口：`GET /jiying/api/executions?kind=&status=&name=&limit=`，按类型、状态与名称过滤；执行详情可在控制台打开并跳转运行日志按 `trace_id` 查找。
- 定时执行：`runCronCmd` 按任务 `owner_role` / `owner` 解析执行身份，写入开始与结束记录，并真实回写 `last_run` / `last_success` / `last_status` / `last_error` / `run_count` / `fail_count`；连续失败 3 次自动停用任务（`status=false`），panic 会被捕获并记为失败。
- 下次运行：`GET /jiying/api/crons` 按 cron 表达式计算并返回 `next_run_at`，控制台直接展示；`POST /jiying/api/cron/run` 可立即触发一次，执行异步进行，结果进入执行记录。
- 命令格式：定时任务接受纯系统命令名，或 JSON 功能命令 `{"type":"feature","feature":"功能名","args":"参数","conversation_id":"会话 ID"}`；`type=command` 可显式包装系统命令。功能模式会校验功能存在与 `conversation_id` 合法性，执行超时 2 分钟。系统命令若标记 `Admin`，仅 `system` / 主人 / `admin` 可运行，避免通过定时任务提权。

### 账号会话持久化与健康

- 登录 token 按邮箱加密落盘到 `user_sessions.json`（复用 `sealSecret` / `openSecret`，`0600`，临时文件 + rename）；重启后自动恢复，无需重复验证码。
- 账号健康状态为 `valid` / `expired` / `unknown` / `unconfigured`，记录最近成功、检查时间与原因；启动时恢复并每小时调用 `GET /api/client/me` 检查一次。网络错误只标记 `unknown`，仅 401 才判定失效。
- 接口：`GET /jiying/api/connect/account/health` 读取快照，`POST /jiying/api/connect/account/health/check` 主动检查；`connect/status` 的 `account.health` 同步返回，并带 `code_login` 标识验证码账号。
- 失效处理：`UserCall` 遇到 `CodeLogin` 返回 `errAccountNeedsCode`，不再静默降级；登录成功即保存 token，401 时清除落盘 token。验证码账号不会自动登录，切号优先使用已落盘 token。

### 统一发送与身份

- `POST /jiying/api/send/unified` 统一发送文本，参数 `conversation_id` / `text` / `identity`（`auto` / `user` / `app`）/ `reply_to` / `dry_run`。`auto` 优先应用通道，探测失败直接报错，不默认降级为用户身份。
- `GET /jiying/api/conversation/capabilities?conversation_id=` 返回 `can_send_as_user` / `can_send_as_app` / `reason`，控制台据此禁用不可用身份并展示原因。
- 用户身份发送走 `POST /api/client/conversations/{id}/messages`，请求体为 `{client_message_id, body:{type:"text",content}}`；应用身份走既有应用通道。`dry_run=true` 只解析身份与校验参数，不发送、不计数、不写消息，仅落一条 `message.dry_run` 审计。

### 控制台面板（2026-10-17）

- 新增「执行记录」页：按类型 / 状态 / 名称过滤，展示开始时间、类型、名称、状态、耗时与摘要；点击行打开详情，可复制字段并跳转运行日志。定时任务页的「记录」按钮会带上任务名跳转并自动筛选。
- 定时任务页改版：展示下次运行、上次结果（含错误摘要）、运行 / 失败次数与启停状态；支持立即运行、查看记录、启停与删除。新建弹窗可选「已启用功能」或「系统命令」，功能模式提供功能名、参数与会话 ID，并实时预览最终命令 JSON；创建失败保留已填内容。
- 接入页新增账号会话卡片：展示 `valid` / `expired` / `unknown` / `unconfigured`、最近成功与检查时间，提供「立即检查」「重新登录」「退出账号」；验证码账号重新登录会切到验证码页签。
- 消息收发页新增发送身份选择（自动 / 应用 / 用户）与「预演」：选中会话后加载可用身份，不可用选项置灰并显示原因；预演弹窗展示会话、解析身份与内容，不产生实际消息。
- 英文模式覆盖以上新增页面与弹窗；执行摘要、账号健康与身份原因等动态文案通过正则规则翻译，用户数据（任务名、功能名、错误原文）保持原样。

### 运行日志查看器（2026-10-17）

- 结构化解析：按 `[[ 时间 ] [ 级别 ] [ 来源 ] 消息]` 拆分为行号、时间、级别、来源与正文；gin access 行进一步拆出方法、路径、状态码、耗时与 `trace=`，状态码按 2xx / 3xx / 4xx / 5xx 着色。
- 过滤与检索：级别 chips（全部 / 异常 / 错误 / 警告 / 信息 / 调试）带计数，可与关键字搜索叠加；命中关键字用 `<mark>` 高亮，点击 access 行的 trace 按钮按 `trace_id` 反查并保留当前级别筛选。
- 交互：支持自动换行（本机记忆 `relay-log-wrap`）、手动刷新与自动刷新切换、双击行复制原始文本、导出当前筛选结果；轮询刷新时保持滚动位置，贴底时自动跟随，并提供「回到底部」悬浮按钮。
- 状态与无障碍：加载骨架、空态、错误态均可重试；工具栏 meta 显示匹配行数与最后刷新时间；窄屏下工具栏与行内字段自动换行，日期、行数与筛选在 390px 宽度仍可用。

### 头像与昵称同步（2026-10-17）

- 个人账号资料读取 `/api/client/me`，服务端缓存 5 分钟；登录、切号、退出或会话 401 时立即失效。`connect/status` 与 `connections` 统一返回 `nickname` / `avatar_url`，避免状态轮询反复请求上游。
- 多账号资料按邮箱持久化到桶 `relay_account_profiles`：每个账号登录或被轮询到时记录昵称与头像，重启后仍可展示；`connect/status` 额外返回 `accounts`，包含每个账号的 `nickname` / `avatar_url` / `logged_in` / `active`，接入页据此渲染账号列表并支持切换、移除。
- 应用资料来自应用通道 RPC `apps.get`：连接建立后 3 秒拉取，未连通时短间隔重试，成功后每 10 分钟刷新；连接池快照返回 `name` / `description` / `avatar_url`，并提供 `GET /jiying/api/apps/profile?app_id=` 供控制台按需刷新。
- 接入页的应用通道展示默认应用头像与最多 4 个应用头像堆叠（其余以 `+N` 表示），下方列出每个应用的名称、在线状态与默认标记；账号通道按账号逐行展示头像、昵称、邮箱、登录状态与当前标记。
- 控制台顶栏头像、侧栏账号、接入状态、连接池应用与账号卡片、会话列表与消息发送者均优先展示头像，图片加载失败自动回退为首字母；头像地址兼容 `avatar` / `avatar_url` / `icon` 以及 `peer` / `user` / `target` / `members` 等嵌套结构。
- 顶栏头像不再带 `data-ico`：切换页面时图标水合会跳过头像节点，未登录或状态轮询暂时拿不到资料时保留本地缓存，避免头像被默认用户图标覆盖。
- 消息气泡按发送方向兜底头像：入站依次取发送者头像、会话头像，出站取应用头像、当前账号头像；昵称缺失时按方向回退为「Relay / 对方」。会话列表即使没有头像也显示名称首字母，不再退化为通用消息图标。

### API Key 与 AI 接入（2026-10-17）

- 密钥模型：`core/apikeys.go` 生成 `jy_` + 32 字节随机串，落盘只存 SHA-256 摘要、前缀、备注、scope、创建/到期/最后使用时间；明文仅在创建响应返回一次。默认有效期 30 天，可选 7 / 30 / 90 天或永久；单把密钥限流 180 次/分钟，超限返回 429。
- 鉴权：`requireLogin` 先解析 `Authorization: Bearer jy_...`，失败直接 401；无该头时回退 `jiying_session` Cookie。`requireRole` 对密钥按 `scope`（readonly / ops / master）比较角色等级，`consoleUser` 返回 `key:备注(前缀)` 写入审计，密钥调用与人工操作可区分。
- 管理边界：密钥的创建、列表、轮换、撤销仅允许浏览器会话（`requireConsoleSession`），API Key 不能自我复制或提权；撤销与轮换即时生效。
- 生命周期：单实例最多保留 100 把有效密钥；读取列表与创建时自动清理已过期密钥；轮换会作废旧密钥并签发同备注、同 scope 的新密钥，留空有效期时沿用旧密钥剩余天数（永久密钥保持永久）；限流窗口表超过 256 项时回收两个窗口周期前的条目。
- 公开文档：`GET /jiying/ai.txt` 为面向 AI 的纯文本说明（认证、响应约定、常用流程、定时与插件规范），`GET /jiying/openapi.json` 为 OpenAPI 3.0 文档，均无需登录。
- 能力清单：`GET /jiying/api/ai/manifest` 返回当前功能、定时任务、就绪度、调用者角色、端点表、cron 示例与插件协议，供 AI 在执行前自助读取。端点表覆盖连接池应用/账号管理与资料刷新；删除类接口按路由实际权限标记为 `ops`。OpenAPI 中密钥管理使用 `cookieAuth`，并注明不支持 Bearer。
- 控制台新增「AI 接入」页（`view-ai`）：三步说明、密钥列表（状态 / 最后使用 / 轮换 / 撤销）、带可见标签的创建表单（备注 / 权限 / 有效期）、文件上传区与「给 AI 的说明书」；创建或轮换成功后弹窗提供复制密钥与复制带密钥的接入说明，未点击「我已保存」前阻止关闭，避免明文丢失。只读账号可阅读文档，文件上传、签发与撤销仍要求运营或主人。
- 文件上传：`POST /jiying/api/upload`（运营及以上）接受 multipart 字段 `file`，或表单字段 `name` / `content_type` / `data_base64`，单文件最大 20 MB，同时最多处理 2 个上传，返回 `file_id` 供 `send/file` 引用；审计只记录文件名、大小与类型，不记录内容。
- 测试：`core/apikeys_test.go` 覆盖创建、摘要落盘、校验、过期、清理、轮换、撤销、非法输入与限流窗口；`core/aidocs_test.go` 校验 `aiEndpoints` 与 Gin 实际路由逐条一致，并检查 OpenAPI 安全方案与操作数量。

### 定时任务与插件易用性（2026-10-17）

- 定时新建改为频率优先：预设每 5 分钟、每 30 分钟、每小时整点、每天 09:00 / 18:00、工作日 09:00、每周一 09:00 与自定义 cron；预设自动写入表达式，自定义时展开原始输入，弹窗内实时提示「多久执行一次」。
- 定时列表在表达式下补充人话说明（如「每 30 分钟」「工作日 09:00」），常见 5 段与 6 段写法均可识别，其余保留原始表达式。6 段预设显式把秒位固定为 0（如 `0 */30 * * * *`），避免把「每 30 分钟」误写成每 30 秒。
- 新建插件提供模板库：JavaScript / Python 空白模板、JS 回发消息、Python 回发消息；选择模板自动切换语言、文件名与代码，减少从零编写。已有插件编辑时仍直接进入源码编辑。
- Python 模板演示 `JIYING_ARGS` 入参与 `SEND:` 回发协议，JS 模板演示 `args` / `result` 与 `sendText()`；安装后自动登记为功能，可在功能管理或定时任务中直接引用。

### 插件分类、多应用与控制台体验（2026-10-17）

- 插件元数据：头部清单支持 `[class: ...]` 归档、`[service: ...]` 服务分组与 `[apps: app-a, app-b]` 应用白名单，`*` 或省略表示全部应用；服务重启或安装后扫描同步到功能记录，`/jiying/api/scripts` 返回 `class` / `service` / `apps`。
- 应用隔离：功能执行与 Agent 工具列表两处都做白名单校验，避免模型选中未授权插件后才被拒绝。
- 插件页：顶部「插件 → 分类 → 服务 → 应用」树脉络统计，支持按分类和服务两级折叠、关键词搜索（插件名/标题/分类/服务/应用）与目录、列表两种视图；同一 `service` 的插件收进分类内的子文件夹，未声明服务的插件直接挂在分类下；每个插件显示分类标签与服务应用 chip，未声明白名单时显示「全部应用」。
- 功能页：脚本功能同样显示分类、服务与应用 chip；应用名与头像来自连接池，未知应用回退短 ID。
- 头像与状态保持：头像底色按名称稳定散列，图片失败保留首字母；顶栏资料写入本地缓存，切页面不再被图标水合覆盖；多账号资料按邮箱持久化并在接入页列出；会话页记住上次选中的会话，侧栏记住上次停留页面。
- 吉祥物挂件：登录页与控制台右下角常驻，指针环绕切换 8 个方向切片，点击时轮换短句并播放交付动画，连接成功、保存成功等事件同样触发；可关闭并保留恢复入口，开关状态存于 `relay-mascot`。

### 生产加固（2026-10-17）

- pprof 默认关闭：`main.go` 仅在 `JIYING_PPROF=1` 时挂载 `/debug/pprof/`，启动日志会明确提示已启用；生产环境保持默认关闭，避免堆栈、内存与 CPU 采样数据暴露。
- 端点目录一致性：`core/aidocs_test.go` 把 `aiEndpoints` 与 `Server.Routes()` 的 `/jiying/api/` 路由双向比对，新增或改名接口未同步文档时测试直接失败；同时校验 OpenAPI 的 `bearerAuth` / `cookieAuth` 与操作总数。
- OpenAPI 增强：写接口附带可直接替换的 form 示例，统一成功响应包含 `ok/code/message/trace_id/data` 结构，错误响应补充 401、403、429 语义，并输出 `x-required-role`。
- 工程化：新增顶层 `README.md`，说明构建、资源目录、环境变量、数据目录、文档入口与测试命令；仓库根目录准备了 GitHub Actions 工作流，需使用具备 `workflow` 权限的令牌推送后生效。

### 安全审计修复（第二轮，2026-10-17）

- 权限与身份：定时任务的创建、修改、删除、执行、启停均按会话或 API Key 身份判定；主控任务仅主控可管，其余用户只能管理自己的任务。调度与手动执行共用 `cronIdentity`，只有主控任务以 `system` 运行。`cron:` 前缀在完成鉴权后才允许派发。
- 凭据保护：连接地址统一经 `validateOutboundURL` 校验，默认拒绝回环、私网、链路本地与云元数据地址；内网自建部署可设 `JIYING_ALLOW_PRIVATE_HOSTS=1`。掩码密钥只能复用同一应用的已存端点，杜绝借测试接口外带凭据。
- 认证会话：重置验证码改为服务端票据（Cookie 仅存随机票据 ID，验证码只存 SHA-256、10 分钟有效、一次性、限次）；会话 Cookie 增加代次校验，改密、禁用、删除用户后旧会话立即失效；登录/发码/重置/退出增加同源校验，HTTPS 下自动加 `Secure`。
- 上传与渲染：上传与脚本路径统一走 `safeUploadRel` / `safeJoinUnder`，限制根目录、扩展名与单文件 4 MB；multipart 文件名剥离 CR/LF/引号，Content-Type 经 `mime.ParseMediaType` 校验；控制台功能列表改用 `jsArg` 编码内联参数，修复存储型 XSS。
- 日志与并发：应用日志、审计与访问日志在落盘前剥离控制字符；发送者上下文改为按 goroutine 隔离；确认卡片绑定发起人与会话；插件权限表加互斥锁；定时任务状态更新加锁，任务队列使用原子桶引用消除后台竞态。
- 静态加密：连接密钥、账号密码、应用密钥、模型密钥与会话令牌统一 AES-GCM（随机 nonce）；密钥不可用时中止保存，禁止回退明文。AES-CBC 与 SM4-CBC 改为随机 IV 并带版本头；rank 编码密钥改由 `JIYING_RANK3_KEY` / `JIYING_RANK4_KEY` 提供，未配置时失败关闭。
- 暴露面与响应头：默认 `gin.ReleaseMode`，仅 `JIYING_DEBUG=1` 开启调试；移除 `GET /exit`、`GET /js/debug`，插件启停改为 POST；`/local/msg` 拒绝携带浏览器上下文头的请求；静态资源移除接入测试页；全局下发 `X-Frame-Options: DENY`、CSP `frame-ancestors 'none'`、`X-Content-Type-Options: nosniff`、`Referrer-Policy: no-referrer`，HTTPS 下追加 HSTS。
- 文件权限：数据、日志、配置目录统一 `0700`，日志与配置、BoltDB 文件 `0600`，脚本目录 `0750`；启动时同步收紧已存在目录权限。

## 当前实现补充（2026-10-18）

### 消息计数与就绪判定

- 收发计数落盘：`jiyingMsgStats` 以桶键 `jiying_msg_stats` 保存当日 `inbound` / `outbound` 与日期，首次读取时懒加载；跨日自动清零。计数在内存聚合，最多每 2 秒批量落盘一次，避免每条消息都写桶；落盘失败只记日志，不影响消息处理，进程退出前可调用 `flushMsgStats` 补写。
- 验证码账号就绪：`accountReadiness` 接受「已保存密码」或 `code_login=true` 两种凭据；已登录时直接判定 `connected`，未登录才按配置完整性返回 `configured` / `invalid`。`UserLogin` 与 `connect/status` 同步暴露 `CodeLogin`，避免验证码账号被误报为缺少密钥。

### Agent 回复与工具调用

- 回复格式：Agent 最终答复优先走 `ReplyMarkdown`（即应 `markdown` 富文本），标题、列表、表格与代码块可正常渲染；发送失败自动回退纯文本。`SendMarkdownToConv` 与纯文本共用 `clampOutboundText`，超长内容按上限截断。
- 身份注入：`agentSystemPrompt` 在系统提示末尾追加「当前用户ID（主控权限/普通用户）」与可信度约束（实时数据先调工具、失败如实说明、不编造数据或图片）；用户自定义 system 不再覆盖这两项，身份为空时仍保留可信度约束。
- 实时意图强制工具：`agentToolChoice` 在首轮命中天气、新闻、股价、快递、实时等关键词且存在匹配工具时，向模型传 `tool_choice=required`；未命中或没有对应工具仍为 `auto`，避免为普通对话强行调用工具。
- 失败可感知：工具执行被拒绝、超时或失败时，回灌内容以 `FAILED:` 开头并附带「不要编造结果」提示，模型据此如实回复；成功结果保持原样。等待任务时使用调用方上下文并覆盖 `taskRunTimeout`，只有 `done` 状态才算成功，`failed` / `timeout` / `canceled` 及取消时仍为 `running` 的快照都按失败处理。
- 图片降级：模型最终文本中的 Markdown 图片语法 `![alt](url)` 统一改写为 `[图片：alt](url)`。Agent 需要发图必须调用图片工具，防止未真正发送却让客户端渲染出图片。
- 静默任务可追溯：`taskq` 为静默任务保留最近 50 条内存副本（`RecentSilentTasks`），`GET /jiying/api/tasks?silent=1` 返回；控制台任务队列新增「含 Agent 调用」开关，Agent 行带标记并只提供详情，不参与重试/删除。该接口要求运维及以上角色，主控可看全量，其他角色只返回自己发起的静默任务。静默任务仍不进入持久层，重启即清空。
- 参数边界：`max_turns` 限制在 1~12，`max_history` 限制在 1~50，保存与加载都会归一化；回灌给模型的单次工具结果截断到 12 KiB，避免超长输出挤占上下文。

### 插件运行环境与失败详情

- 控制台环境注入：Python 插件子进程新增 `JIYING_PORT`、`JIYING_BASE_URL`（本机 `127.0.0.1` 控制台地址）、`JIYING_CONSOLE_USER`、`JIYING_CONSOLE_PASS`。控制台账号固定为 `script`，口令是首次使用时随机生成并以 AES-GCM 加密保存的专用令牌，登录后映射到控制台管理员身份；即应账号密码不再注入脚本。需要账号身份时使用 `JIYING_ACCOUNT_BASE_URL` / `JIYING_ACCOUNT_EMAIL`，密码始终留在服务端。仅管理员安装的插件可执行；插件自身配置的同名变量仍可覆盖。
- 输出脱敏：脚本成功/失败输出、任务结果、审计与 Agent 工具回灌统一经过 `redactSecretText`，账号密码与脚本控制台令牌替换为 `***`；任务详情不再展示原始凭据。
- 失败详情：脚本非零退出时合并 `stderr` 与 `stdout`（各带标签，总计截断 2000 字符）返回。插件把错误 JSON 打印到 stdout 时不再只剩 `exit status 1`，Agent 与任务列表都能看到真实原因。
- 账号通道边界：账号通道只有 HTTP 会话与 token，不存在应用通道那样的实时事件流；「账号侧收到入站消息」依赖应用通道，文档与就绪检查均按此语义描述。

### 总览会话数缓存

- `overviewConversationCount` 以「账号 BaseURL + 邮箱」为键缓存会话数 30 秒，切号后键变化自然失效；上游调用失败时回退该账号上次结果，没有历史结果则返回 0，并对失败结果做 8 秒负缓存，避免远端故障时每次刷新都等待超时。

### 媒体发送的部署限制

- 即应服务端拉取媒体时执行 SSRF 防护，`localhost`、回环、私网与链路本地地址一律拒绝。中间件部署在私网时，图片/文件必须先上传到公网可访问的地址（对象存储、带公网域名的静态服务等），不能直接使用本机临时文件路径。
- 发送失败会返回明确错误（`URL 不允许访问` 或下载超限），控制台、脚本与 Agent 工具调用都应把失败原样告知用户，并降级为文字说明或链接，不要假定图片已送达。

### 消息身份与头像（2026-10-18 补充）

- 账号资料补 `id`：`parseAccountProfile` 解析 `/api/client/me` 的 `data.user.id`，`AccountProfile` 与 `GET /jiying/api/connections` 的 `account.id` 一并返回；切换账号后由资料缓存刷新。
- 消息方向：前端优先用 `sender.id === account.id` 判定本人消息，`sender.type=app/assistant`、`direction` 与 `is_out` 作为兼容兜底，避免把 `type=app` 一律当成自己。
- 昵称与头像：根据会话 `members` 建立 id 索引，`sender` 只带 id 时也能显示成员昵称与头像；应用发送者按 `app_id` 匹配应用快照，助手消息回退 `/assets/apps/assistant.webp`（随包提供）。头像加载失败时保留首字母占位并记录控制台警告，不再静默移除。

### 出站下载与脚本进程（2026-10-18 补充）

- 媒体下载：`ReadFileContent` 在请求前校验临时文件 URL，并通过 `safeOutboundHTTPClient` 在建立连接时重新解析、校验 IP（防 DNS 重绑定），重定向逐跳校验，最多 5 次；`JIYING_ALLOW_PRIVATE_HOSTS=1` 仍可放行私网自建部署，云元数据地址始终拒绝。
- 脚本进程组：Python 脚本及其子进程使用独立进程组，超时或取消时整组终止；工作目录切换到脚本所在目录，便于读取同目录数据文件。
- 任务队列：默认 2 个 worker 并发消费，单个慢插件不再阻塞后续任务；执行完成一个任务后若仍有待执行任务，会唤醒其他 worker。
