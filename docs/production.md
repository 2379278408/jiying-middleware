# 生产部署

本文覆盖单实例生产部署、反向代理、备份恢复、告警与多实例边界。默认数据目录为 `/data`，资源目录为 `/app`。

## 1. 二进制部署

```bash
# 编译
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o dist/jymw .

# 准备资源目录
sudo install -d -o jiying -g jiying /opt/jiying /var/lib/jiying
sudo install -m 0755 dist/jymw /opt/jiying/jymw
sudo cp -R views2-patch/assets /opt/jiying/views2
```

`/etc/jiying/jiying.env`：

```bash
JIYING_RES=/opt/jiying
JIYING_DATA=/var/lib/jiying
JIYING_PORT=8080
JIYING_WS_URL=wss://<host>/api/app/ws
JIYING_APP_ID=<应用 ID>
JIYING_APP_SECRET=<应用密钥>
TZ=Asia/Shanghai
```

```bash
# 安装 systemd 单元
sudo install -m 0644 deploy/jiying.service /etc/systemd/system/jiying.service
sudo systemctl daemon-reload
sudo systemctl enable --now jiying
sudo systemctl status jiying
```

`JIYING_RES` 必须包含 `views2/`，否则控制台静态资源 404。日志写入 `<JIYING_DATA>/logs/YYYYMMDD.log`。

## 2. Docker 部署

```bash
# 构建镜像
docker build -t jiying/jymw:latest .

# 启动，数据目录必须持久化
docker run -d --name jymw --restart unless-stopped \
  -p 127.0.0.1:8080:8080 \
  -v /srv/jiying:/data \
  --env-file /etc/jiying/jiying.env \
  jiying/jymw:latest
```

镜像内已安装 `python3`、`util-linux`（`prlimit`、`unshare`）和 CA 证书。Python 插件的 `[network: false]` 依赖内核允许非特权用户命名空间；受限内核上该插件会直接执行失败，属于安全兜底。

## 3. 反向代理

使用 `deploy/nginx.conf`，替换域名与证书路径。关键点：

- 关闭 `proxy_buffering`，控制台状态推送才能实时到达。
- `client_max_body_size` 至少 32m，媒体上传与插件包会超过默认 1m。
- 只暴露反向代理的 443，`jymw` 监听 `127.0.0.1:8080`。

```bash
sudo install -m 0644 deploy/nginx.conf /etc/nginx/conf.d/jiying.conf
sudo nginx -t && sudo systemctl reload nginx
```

## 4. 备份与恢复

需要备份：

- `<JIYING_DATA>/data/buckets/middleware.db`（bbolt，配置、API Key、审计、事件台账）
- `<JIYING_DATA>/.market_signing_key`（本机插件发布签名私钥）
- `/etc/jiying/jiying.env`

在线直接 `cp` bbolt 文件可能得到不一致快照。推荐停服务后复制：

```bash
# 停服务后备份
sudo systemctl stop jiying
sudo tar -C /var/lib/jiying -czf /backup/jiying-$(date +%F).tgz .
sudo systemctl start jiying
```

若已安装 bbolt CLI，可在线备份：

```bash
bbolt backup /var/lib/jiying/data/buckets/middleware.db /backup/middleware-$(date +%F).db
```

恢复：

```bash
# 停服务，保留旧目录用于回退
sudo systemctl stop jiying
sudo mv /var/lib/jiying /var/lib/jiying.bak
sudo install -d -o jiying -g jiying /var/lib/jiying
sudo tar -C /var/lib/jiying -xzf /backup/jiying-2026-10-19.tgz
sudo chown -R jiying:jiying /var/lib/jiying
sudo systemctl start jiying
```

恢复后登录控制台检查连接就绪度、插件列表与审计，并确认 `.market_signing_key` 权限为 `0600`。

## 5. 告警

`jiying` 单元已设置 `Restart=on-failure`。建议再加三条外部检查：

```bash
# 进程存活
systemctl is-active --quiet jiying

# 错误日志（按天文件）
grep -E "ERROR|panic|失败" /var/lib/jiying/logs/$(date +%Y%m%d).log

# 存活与就绪：readyz 在存储未就绪时返回 503
curl -fsS https://jiying.example.com/healthz
curl -fsS https://jiying.example.com/readyz

# 关键计数
curl -fsS https://jiying.example.com/metrics
```

接入配置是否完整看控制台连接页，或登录后请求 `/jiying/api/connect/readiness`，返回 `status=invalid` 时按 `issues[].hint` 修复。磁盘告警重点看 `.db`、`logs/` 和插件目录增长。

## 6. 多实例

bbolt 是单写者嵌入式数据库，`middleware.db` 不能由多个进程同时打开，也不能放在 NFS 上共享。多副本部署时每个实例必须使用独立数据目录，这会导致插件、定时任务和事件台账各自独立，同一即应事件可能被重复处理。

当前版本按单实例部署。需要水平扩展时，先拆分共享状态（配置、事件台账、任务队列）到外部数据库，再让多个无状态实例共享；在此之前用反向代理指向单一实例，并通过重启与备份保障可用性。
