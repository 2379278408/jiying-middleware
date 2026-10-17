# 即应中间件（Relay Jiying Middleware）

即应（Jiying）应用的中间件与控制台。中间件负责认人、排队、执行任务，并把结果发回原会话；控制台提供连接管理、功能与插件、定时任务、消息发送、AI 接入和审计。

## 能力

- 应用通道与账号通道双身份接入，事件按 cursor 去重、确认与重放。
- 聊天、定时、Webhook 统一任务队列，支持确认卡片、超时与重试。
- 内置 / HTTP / 脚本三类功能，插件支持 `[config: ...]` 配置 schema。
- 控制台：会话、消息发送、媒体上传、功能、插件、技能、定时、执行记录、审计。
- API Key（readonly / ops / master）、公开 AI 文档与 OpenAPI、角色分级。

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

## 文档与接口

- 开发方案：`docs/jiying-middleware.md`
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

