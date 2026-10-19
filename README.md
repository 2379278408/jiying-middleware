# 即应中间件（Relay Jiying Middleware）

即应（Jiying）应用的中间件与控制台。中间件负责认人、排队、执行任务，并把结果发回原会话；控制台提供连接管理、功能与插件、定时任务、消息发送、AI 接入和审计。

![Relay 即应中间件](docs/images/relay-banner.svg)

## 能力

- 应用通道与账号通道双身份接入，事件按 cursor 去重、确认与重放。
- 聊天、定时、Webhook 统一任务队列，支持确认卡片、超时与重试。
- 内置 / HTTP / 脚本三类功能，插件支持 `[config: ...]` 配置 schema。
- 插件市场使用 Ed25519 发布者签名，安装前校验签名与内容摘要，禁止静默降级。
- Python 脚本在独立工作目录执行，带环境变量白名单、CPU/内存/输出限制，`[network: false]` 使用无网络命名空间。
- 公开 AI 文档按 scope 裁剪：匿名只暴露只读 GET，运维与主控接口按 API Key 角色收敛。
- 控制台：会话、消息发送、媒体上传、功能、插件、技能、定时、执行记录、审计。
- API Key（readonly / ops / master）、公开 AI 文档与 OpenAPI、角色分级。
- 任务与定时按归属隔离：主控可见全部，其他身份只能查看和操作自己创建的任务；危险操作写审计。
- `/metrics` 输出任务排队等待、执行耗时与终态计数，便于接 Prometheus 告警。
- Agent 可配置轮次、工具调用次数与上下文预算，超预算时裁剪历史并停止下发工具定义。

## 界面

登录入口。应用通道与账号通道未就绪时，控制台会直接给出修复项。

![登录控制台](docs/images/login.png)

<table>
  <tr>
    <td width="50%"><img src="docs/images/console-overview.png" alt="工作台"></td>
    <td width="50%"><img src="docs/images/plugins.png" alt="插件管理"></td>
  </tr>
  <tr>
    <td><img src="docs/images/ai-access.png" alt="AI 接入"></td>
    <td><img src="docs/images/features.png" alt="功能管理"></td>
  </tr>
</table>

## 运行架构

即应的事件进入接入层，归一后写入统一任务队列，由执行器调用内置、HTTP 或脚本功能，结果回发会话并写入执行账本。控制台覆盖连接、插件、定时、AI Key 与审计。

![Relay 运行架构](docs/images/architecture.svg)

## 环境要求

- Go 1.25 或更新版本
- 可访问的即应服务（应用身份或账号身份）

## 构建与启动

```bash
# 编译
go build -o dist/jymw .

# 控制台静态资源需要与可执行文件放在同一资源目录
mkdir -p dist/views2
cp -R views2-patch/assets dist/views2/

# 启动，默认监听 0.0.0.0:8080
./dist/jymw
```

自定义资源目录与数据目录：

```bash
JIYING_RES=/opt/jiying JIYING_DATA=/var/lib/jiying JIYING_PORT=8080 ./jymw
```

`JIYING_RES` 默认是可执行文件所在目录，`JIYING_DATA` 默认与资源目录相同。日志位于 `<数据目录>/logs/YYYYMMDD.log`，脚本、插件、存储桶也落在数据目录下。

端口优先级为控制台配置 `port` > 环境变量 `JIYING_PORT` > 默认 `8080`；监听 `443` 时启用 TLS，需要在控制台配置 `certFile` 与 `keyFile`。其余端口按 HTTP 启动，生产环境建议由反向代理终止 TLS。

## 配置

连接参数通过环境变量提供，不写入仓库：

```bash
export JIYING_WS_URL='wss://<host>/api/app/ws'
export JIYING_APP_ID='<应用 ID>'
export JIYING_APP_SECRET='<应用连接密钥>'
```

控制台账号、主控用户 ID、账号通道等配置在控制台内维护。生产环境请放在反向代理之后并启用 HTTPS。

调试端点默认关闭；只在受信任的本机环境临时开启：

```bash
JIYING_PPROF=1 ./jymw
```

Gin 默认以发布模式运行，排障时再显式打开调试日志：

```bash
JIYING_DEBUG=1 ./jymw
```

出站连接默认拒绝回环、私网与云元数据地址；确认部署在内网且需要访问内网服务时：

```bash
JIYING_ALLOW_PRIVATE_HOSTS=1 ./jymw
```

rank 编码使用部署侧密钥，未配置时相关编码直接失败关闭：

```bash
export JIYING_RANK3_KEY='<32 字节以上随机串>'
export JIYING_RANK4_KEY='<32 字节以上随机串>'
```

## 文档与接口

- 开发方案：`docs/jiying-middleware.md`
- 生产部署、Docker、反向代理、备份恢复、告警与多实例边界：`docs/production.md`
- AI 接入说明：`GET /jiying/ai.txt`（无需登录）
- OpenAPI：`GET /jiying/openapi.json`（无需登录）
- 能力清单：`GET /jiying/api/ai/manifest`（需登录或 API Key）
- 健康检查：`GET /healthz`、`GET /readyz`、`GET /metrics`

API Key 在控制台「AI 接入」页创建，明文只显示一次。权限从低到高为只读、运营、主人；密钥管理只接受主人账号的浏览器会话。

## 媒体上传

`POST /jiying/api/upload`（运营及以上）支持 multipart 字段 `file`，或表单字段 `name`、`content_type`、`data_base64`，单文件最大 20 MB。返回的 `file_id` 可直接用于 `/jiying/api/send/file`。

## 测试

```bash
gofmt -l .
go build ./...
go vet ./...
go test ./...
```

