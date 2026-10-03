# 前端第三方资源（vendor）

控制台是无构建（no-build）的静态页面，第三方库以本地文件形式随包分发，
由中间件从 `views2/assets` 目录直接托管，运行时不依赖任何外部 CDN。

## 内容

| 文件 | 版本 | 来源 | 许可 |
|------|------|------|------|
| `bootstrap/bootstrap.min.css` | 5.3.3 | `https://cdn.jsdelivr.net/npm/bootstrap@5.3.3/dist/css/bootstrap.min.css` | MIT |
| `bootstrap/bootstrap.bundle.min.js` | 5.3.3 | `https://cdn.jsdelivr.net/npm/bootstrap@5.3.3/dist/js/bootstrap.bundle.min.js` | MIT |
| `bootstrap/bootstrap-icons.min.css` | 1.11.3 | `https://cdn.jsdelivr.net/npm/bootstrap-icons@1.11.3/font/bootstrap-icons.min.css` | MIT |
| `bootstrap/fonts/bootstrap-icons.woff2` | 1.11.3 | `https://cdn.jsdelivr.net/npm/bootstrap-icons@1.11.3/font/fonts/bootstrap-icons.woff2` | MIT |
| `bootstrap/fonts/bootstrap-icons.woff` | 1.11.3 | `https://cdn.jsdelivr.net/npm/bootstrap-icons@1.11.3/font/fonts/bootstrap-icons.woff` | MIT |

## 约定

- 加载顺序必须是 `bootstrap.min.css` → `bootstrap-icons.min.css` → `relay.css`。
  `relay.css` 后置生效，因此在 `.btn`/`.badge`/`.nav`/`.modal`/`.toast` 这些同名类上
  Relay 自有样式优先，引入 Bootstrap 不会改变既有外观。
- `relay.css` 末尾的「Bootstrap 5.3 主题桥接」把 Bootstrap 的 `--bs-*` 变量
  映射到 Relay 的 OKLCH 令牌，新增组件默认沿用暖色配色。
- Bootstrap 栅格依赖 `.row` / `.col-*`，因此 Relay 自有的横排容器命名为 `.rl-row`。
- 图标统一使用 Bootstrap Icons 的 `<i class="bi bi-xxx">`，不再请求外部图标服务。

## 升级方式

按上表 URL 覆盖同名文件即可，无需构建步骤；升级后确认 `bootstrap-icons.min.css`
中字体引用仍为相对路径 `fonts/bootstrap-icons.woff2`。
