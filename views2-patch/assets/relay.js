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
    "nav.connect": { zh: "接入系统", en: "Connect" },
    "nav.pool": { zh: "连接池", en: "Connection pool" },
    "nav.roles": { zh: "权限管理", en: "Access control" },
    "nav.scripts": { zh: "插件", en: "Plugins" },
    "nav.market": { zh: "插件市场", en: "Plugin market" },
    "nav.features": { zh: "功能管理", en: "Features" },
    "nav.agent": { zh: "Agent 工作台", en: "Agent" },
    "nav.cron": { zh: "定时任务", en: "Schedules" },
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
    setLang(LANG === "en" ? "zh" : "en");
    applyStaticI18n();
    refreshLangButtons();
    renderNav();
    if (state.view) {
      var n = NAV.filter(function (x) { return x.id === state.view; })[0];
      var c = $("#crumbs");
      if (c) c.textContent = n ? "Relay · " + t(n.name) : "Relay";
    }
    loadConnectStatus();
    if (state.view && renderers[state.view]) renderers[state.view]();
  }

  var NAV = [
    { id: "home", name: "nav.home", ik: "dashboard", group: "nav.group.overview" },
    { id: "chat", name: "nav.chat", ik: "message", group: "nav.group.access" },
    { id: "connect", name: "nav.connect", ik: "link", group: "nav.group.access" },
    { id: "pool", name: "nav.pool", ik: "layers", group: "nav.group.access" },
    { id: "roles", name: "nav.roles", ik: "users", group: "nav.group.access" },
    { id: "scripts", name: "nav.scripts", ik: "puzzle", group: "nav.group.run" },
    { id: "market", name: "nav.market", ik: "inbox", group: "nav.group.run" },
    { id: "features", name: "nav.features", ik: "toggles", group: "nav.group.run" },
    { id: "agent", name: "nav.agent", ik: "robot", group: "nav.group.run" },
    { id: "cron", name: "nav.cron", ik: "clock", group: "nav.group.run" },
    { id: "tasks", name: "nav.tasks", ik: "list", group: "nav.group.run" },
    { id: "audit", name: "nav.audit", ik: "shield", group: "nav.group.run" },
    { id: "observe", name: "nav.observe", ik: "activity", group: "nav.group.run" },
    { id: "logs", name: "nav.logs", ik: "file", group: "nav.group.run" }
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
    refresh: "bi-arrow-clockwise",
    user: "bi-person",
    logout: "bi-box-arrow-right",
    theme: "bi-moon-stars",
    themeLight: "bi-sun",
    settings: "bi-sliders",
    language: "bi-translate"
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
  // 仅补齐「声明了 data-ico 但没有内联图标」的静态节点（顶栏头像、退出按钮）。
  function hydrateIcons() {
    $$("[data-ico]").forEach(function (el) {
      if (el.querySelector("i, svg")) return;
      el.innerHTML = ico(el.getAttribute("data-ico"));
    });
  }
  // 深浅色主题：优先用户选择（localStorage），否则跟随系统；内联在 <head> 的脚本已先行写入属性避免闪白。
  var THEME_KEY = "relay-theme";
  function preferredTheme() {
    try { var t = localStorage.getItem(THEME_KEY); if (t === "dark" || t === "light") return t; } catch (e) {}
    return window.matchMedia && window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
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
    return '<div class="rl-empty"><div class="rl-empty-ico">' + ico(kw) + "</div>"
      + "<h4>" + esc(title) + "</h4>"
      + (hint ? "<p>" + esc(hint) + "</p>" : "")
      + (actionHTML ? '<div class="rl-empty-act">' + actionHTML + "</div>" : "")
      + "</div>";
  }
  // 加载失败态：与空态区分开，并给一个重试入口，避免用户只能刷新整页。
  function errorState(box, kw, title, hint, retry) {
    if (!box) return;
    box.innerHTML = emptyState(kw, title, hint || "请稍后重试。", '<button class="btn btn-ghost btn-sm">重试</button>');
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

  var state = { view: "home", cerr: null, convs: [], conv: null, msgs: [], interval: null, baseUptime: null, uptimeAt: 0, uptimeTimer: null, chatEpoch: 0, chatRequest: 0, logLines: [], logPending: "", logTimer: null, tasks: [], tasksErr: "", role: "", auditRows: [], auditErr: "" };
  var ROLE_LABEL = { master: "主人", ops: "运营", readonly: "只读" };

  function $(sel, root) { return (root || document).querySelector(sel); }
  function $$(sel, root) { return Array.prototype.slice.call((root || document).querySelectorAll(sel)); }
  function esc(s) {
    return String(s == null ? "" : s).replace(/[&<>"']/g, function (c) {
      return { "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c];
    });
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
    }
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
    }).catch(function () {
      if (!opts.silent) toast("网络请求失败，请检查服务状态后重试", "err");
      return { code: 0, msg: "网络请求失败" };
    });
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
  function orderedNav(p) {
    var byId = {};
    NAV.forEach(function (x) { byId[x.id] = x; });
    var visible = p.order.map(function (id) { return byId[id]; }).filter(function (x) { return x && p.hidden.indexOf(x.id) < 0; });
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
      if (bd) bd.classList.toggle("show", sb.classList.contains("open"));
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
    var toggle = $("#navToggle");
    if (toggle && isMobile()) toggle.setAttribute("aria-expanded", "false");
  }

  function switchView(id) {
    if (renderers[id]) renderers[id]();
     $$("#nav [data-view]").forEach(function (a) { a.classList.toggle("active", a.getAttribute("data-view") === id); });
    $$(".page").forEach(function (p) { p.hidden = p.id !== "view-" + id; });
    state.view = id;
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

  function refreshConnUI(st) {
    if (!st) return;
    var wsOk = st.app && st.app.connected;
    var acctOk = st.account && st.account.logged_in;
    var dWs = $("#dotWs"), dAcct = $("#dotAcct");
    if (dWs) { dWs.className = "dot " + (wsOk ? "on" : "off"); $("#sideWs").textContent = t("side.ws") + " " + (wsOk ? t("side.connected") : t("side.disconnected")); }
    if (dAcct) { dAcct.className = "dot " + (acctOk ? "on" : "off"); $("#sideAcct").textContent = t("side.acct") + " " + (acctOk ? st.account.email : t("side.notlogged")); }
    var cw = $("#chipWs"), ca = $("#chipAcct");
    if (cw) { cw.className = "chip " + (wsOk ? "chip-success" : "chip-danger"); cw.textContent = wsOk ? t("chip.app.on") : t("chip.app.off"); }
    if (ca) { ca.className = "chip " + (acctOk ? "chip-success" : "chip-danger"); if (st.account.email) ca.textContent = st.account.email; else ca.textContent = t("chip.acct.off"); }
    var av = $("#avatar");
    if (av && !av.querySelector("svg, i")) av.textContent = (st.account.email || "U").charAt(0).toUpperCase();
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
    cont.innerHTML = '<div class="page-head"><div><h2>工作台</h2><p>即应接入与任务调度实时总览</p></div><div class="spacer"></div><button class="btn btn-ghost btn-sm" onclick="Relay.refreshHome()">刷新</button></div>'
      + '<div class="grid grid-4">'
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
      + '<div class="panel"><div class="panel-head"><h3>接入状态</h3><span class="hint"><a href="javascript:Relay.go(\'connect\')">进入接入系统 →</a></span></div><div class="panel-body" id="hcState"></div></div>'
      + '<div class="panel"><div class="panel-head"><h3>快捷入口</h3></div><div class="panel-body"><div class="grid grid-2">'
      + quick("chat", "消息收发", "会话列表与发送", "message")
      + quick("scripts", "插件", "安装与调试插件", "puzzle")
      + quick("cron", "定时任务", "创建定时调度", "clock")
      + quick("tasks", "任务队列", "执行记录与重试", "list")
        + "</div></div></div>"
        + "</div>"
      + '<div class="panel mt-16"><div class="panel-head"><h3>接入检查</h3><span class="hint">首次使用按此顺序逐项自检</span></div><div class="panel-body" id="hcReady"><div class="empty">加载中…</div></div></div>';
    Relay.refreshHome();
  }

  function statCard(label, val, cls, ik) {
    return '<div class="panel stat"><div class="ico ' + cls + '" data-ico="' + ik + '">' + ico(ik) + '</div><div><b id="' + label + '">' + val + "</b><small>" + label + "</small></div></div>";
  }
  function quick(id, name, desc, ik) {
    return '<div class="list-item" style="cursor:pointer" onclick="Relay.go(\'' + id + '\')"><div class="ico ' + id + '" data-ico="' + ik + '" style="width:38px;height:38px;border-radius:10px;background:var(--sidebar-accent);display:flex;align-items:center;justify-content:center;font-size:18px;flex:0 0 38px">' + ico(ik) + '</div><div><div class="name">' + name + '</div><div class="desc">' + desc + "</div></div><div class=\"actions\">→</div></div>";
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
    cont.innerHTML = '<div class="page-head"><div><h2>接入系统</h2><p>推荐顺序：先配「应用密钥」建立应用通道，再用「账号密码 / 邮箱验证码」授权账号；完成后在 <a href="javascript:Relay.go(\'home\')">工作台 · 接入检查</a> 逐项确认。</p></div></div>'
      + '<div class="grid grid-2">'
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

  function renderConnStatus(st) {
    var box = $("#connStatus");
    if (!box) return;
    var d = st && st.data ? st.data : null;
    if (!d) { errorState(box, "link", "接入状态加载失败", "网络异常，请检查服务状态。", function () { loadConnectStatus(); }); return; }
    refreshConnUI(d);
    var appOn = d.app.connected, acctOn = d.account.logged_in;
    var readiness = d.readiness || {};
    // 接入页两个通道都可直接编辑：仅在存在待处理项时提示，且不重复给出跳转按钮。
    var readyBlock = readinessPanel(readiness, { noFix: true, onlyIssues: true });
    var readinessHTML = readyBlock ? '<div class="mt-16 mb-8"><b>待处理项</b></div>' + readyBlock : "";
    box.innerHTML = '<div class="conn-state ' + (appOn ? "on" : "off") + '"><span class="dot ' + (appOn ? "on" : "off") + '"></span><div><b>应用通道</b><div class="muted" style="font-size:12px">' + esc((appOn ? "已连接 · " : "未连接") + (d.app.ws_url || "")) + "</div></div></div>"
      + '<div class="conn-state mt-8 ' + (acctOn ? "on" : "off") + '"><span class="dot ' + (acctOn ? "on" : "off") + '"></span><div><b>账号通道</b><div class="muted" style="font-size:12px">' + esc(acctOn ? d.account.email : d.account.status || "未登录") + "</div></div></div>"
      + readinessHTML
      + '<div class="kv mt-16"><span class="k">应用 ID</span><span class="v"><code>' + esc(d.app.app_id || "-") + "</code></span></div>"
      + '<div class="kv"><span class="k">接入状态</span><span class="v">' + esc(d.account.status || "未登录") + "</span></div>"
      + '<div class="kv"><span class="k">运行时长</span><span class="v">' + esc(d.runtime.uptime) + "</span></div>"
      + '<div class="kv"><span class="k">服务器时间</span><span class="v">' + esc(d.runtime.time) + "</span></div>";
  }

  /* ---------- 消息收发 ---------- */
  function startChatTimer() {
    clearInterval(state.interval);
    state.interval = setInterval(function () { if (state.conv) loadChat(); }, 8000);
  }
  function renderChat() {
    state.chatEpoch += 1;
    state.chatRequest += 1;
    state.convs = []; state.conv = null; state.msgs = [];
    var cont = $("#view-chat");
    cont.innerHTML = '<div class="page-head"><div><h2>消息收发</h2><p>选择会话查看消息并发送内容</p></div></div>'
      + '<div class="chat-grid">'
      + '<div class="panel"><div class="panel-head"><h3>会话</h3></div><div class="panel-body" id="convList"><div class="empty">加载中…</div></div></div>'
      + '<div class="panel" style="display:flex;flex-direction:column"><div class="panel-head"><h3>消息</h3><span class="hint" id="chatTitle"></span><select class="select" id="chatAccountSelect" style="max-width:220px;margin-left:auto"><option>加载账号中…</option></select></div>'
      + '<div class="panel-body" style="flex:1;display:flex;flex-direction:column"><div class="chat-stream" id="chatStream">' + emptyState("message", "选择一个会话", "从会话列表中选择，即可查看历史消息并发送内容。") + "</div>"
      + '<div class="input-row mt-8"><input class="input" id="chatInput" placeholder="输入消息，回车或点击发送"><button class="btn btn-primary" id="chatSend">发送</button></div>'
      + "</div></div></div>";
    loadConvs();
    loadChatAccounts();
    $("#chatSend").addEventListener("click", sendChat);
    $("#chatInput").addEventListener("keydown", function (e) { if (e.key === "Enter") sendChat(); });
  }

  function loadChatAccounts() {
    var epoch = state.chatEpoch;
    api("/jiying/api/connections").then(function (r) {
      if (epoch !== state.chatEpoch) return;
      var select = $("#chatAccountSelect");
      if (!select) return;
      if (r.code !== 200) { select.innerHTML = '<option value="">账号状态加载失败，点「刷新」重试</option>'; return; }
      var accounts = r.data && r.data.accounts ? r.data.accounts : [];
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
           if (result.code === 200) { toast("已切换账号", "ok"); state.chatEpoch += 1; state.conv = null; state.msgs = []; loadConvs(); }
          else toast(result.msg || "账号切换失败", "err");
        }).finally(function () { select.disabled = false; });
      });
    });
  }

  function loadConvs() {
    var epoch = state.chatEpoch;
    api("/jiying/api/conversations").then(function (r) {
      if (epoch !== state.chatEpoch) return;
      if (r.code !== 200) { errorState($("#convList"), "message", r.code === 403 ? "权限不足" : "会话加载失败", r.msg || r.message, loadConvs); return; }
      var list = r.data && r.data.conversations ? r.data.conversations : [];
      state.convs = list;
      var box = $("#convList");
      if (!box) return;
      if (!list.length) { box.innerHTML = emptyState("message", "暂无会话", "收到或发起会话后，会在这里列出。"); return; }
      box.innerHTML = list.map(function (c, i) {
        var name = c.name || c.title || c.id;
        var active = state.conv && state.conv.id === c.id;
        return '<button type="button" class="conv-item' + (active ? " active" : "") + '" data-conv-index="' + i + '"><span class="ico" data-ico="message" style="width:30px;height:30px;border-radius:8px;background:var(--primary-soft);display:flex;align-items:center;justify-content:center;flex:0 0 30px">' + ico("message") + '</span><span class="cname">' + esc(name) + "</span></button>";
      }).join("");
      $$("[data-conv-index]", box).forEach(function (item) {
        item.addEventListener("click", function () { pickConv(Number(item.getAttribute("data-conv-index"))); });
      });
    hydrateIcons();
    }).catch(function () { errorState($("#convList"), "message", "会话列表加载失败", "网络异常，请检查服务状态。", loadConvs); });
  }

  function pickConv(i) {
    var c = state.convs[i];
    state.conv = c;
    state.msgs = [];
    loadConvs();
    $("#chatTitle").textContent = c.name || "会话";
    loadChat();
    startChatTimer();
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
    if (!state.msgs.length) { box.innerHTML = emptyState("message", "暂无消息", "在下方输入框发送内容，中间件会按功能或 Agent 自动回复。"); return; }
    box.innerHTML = state.msgs.map(function (m) {
      var content = "";
      var body = m.body || {};
      if (body.type === "text") content = body.content || "";
      else if (body.type === "image") content = "[图片]";
      else content = m.summary || JSON.stringify(body || "");
      var out = (m.sender && m.sender.type) === "app" || (m.sender && m.sender.type) === "assistant";
      var t = fmtTime(m.created_at);
      // 入站消息把原始 sender.type 映射为中文，避免直接暴露 user / system 等协议字段。
      var who = out ? "Relay" : (SENDER_LABEL[(m.sender && m.sender.type) || ""] || (m.sender && m.sender.type) || "对方");
      return '<div style="display:flex;flex-direction:column;align-items:' + (out ? "flex-end" : "flex-start") + '"><span class="msg-meta">' + esc(who) + " · " + esc(t) + '</span><div class="msg-bubble ' + (out ? "msg-out" : "msg-in") + '">' + esc(content) + "</div></div>";
    }).join("");
    box.scrollTop = box.scrollHeight;
  }

  function sendChat() {
    var input = $("#chatInput");
    var text = input.value.trim();
    if (!state.conv || !text) { toast("请选择会话并输入内容", "err"); return; }
    input.value = "";
    api("/jiying/api/send", { method: "POST", form: { conversation_id: state.conv.id, text: text } }).then(function (r) {
      if (r.code === 200) { toast("已发送", "ok"); setTimeout(loadChat, 800); }
      else {
        // 发送失败时把内容还回输入框，避免用户手打的消息丢失（已在输入新内容则不覆盖）。
        var box = $("#chatInput");
        if (box && !box.value.trim()) box.value = text;
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
  function renderScripts() {
    var cont = $("#view-scripts");
    cont.innerHTML = '<div class="page-head"><div><h2>插件</h2><p>已安装的插件资产，支持 JavaScript 与 Python；安装后会自动登记为可调用功能</p></div><div class="spacer"></div>'
      + '<button class="btn btn-ghost btn-sm" onclick="Relay.go(\'features\')">功能管理 →</button>'
      + '<button class="btn btn-primary" id="newScript">新建插件</button></div>'
      + '<div class="panel"><div class="panel-body" id="scriptList"><div class="empty">加载中…</div></div></div>'
      + '<div class="panel mt-16"><div class="panel-head"><h3>技能</h3><span class="hint">可被插件 require 复用的 JS 模块，或 Markdown 说明；安装前会隔离试跑</span><div class="spacer"></div><button class="btn btn-primary btn-sm" id="newSkill">安装技能</button></div><div class="panel-body" id="skillList"><div class="empty">加载中…</div></div></div>';
    var n = $("#newScript");
    if (n) n.addEventListener("click", function () { openScriptModal(null); });
    var ns = $("#newSkill");
    if (ns) ns.addEventListener("click", openSkillModal);
    loadScripts();
    loadSkills();
  }
  function loadScripts() {
    api("/jiying/api/scripts").then(function (r) {
      var box = $("#scriptList");
      if (!box) return;
      if (r.code !== 200) { errorState(box, "puzzle", "插件列表加载失败", r.msg || r.message, loadScripts); return; }
      if (!r.data || !r.data.length) { box.innerHTML = emptyState("puzzle", "暂无插件", "插件放在 scripts 目录下，安装后会自动登记为可调用功能。", '<button class="btn btn-primary btn-sm" id="scriptEmptyAdd">新建插件</button>'); var b = $("#scriptEmptyAdd"); if (b) b.addEventListener("click", function () { openScriptModal(null); }); return; }
      box.innerHTML = '<div class="list">' + r.data.map(function (s) {
        var lang = s.lang === "python" ? "python" : "js";
        var meta = '<span class="badge badge-off">' + (SCRIPT_LANG_LABEL[lang] || lang) + '</span>';
        if (s.version) meta += ' <span class="badge badge-info">v' + esc(s.version) + '</span>';
        if (s.admin) meta += ' <span class="badge badge-warn">主控</span>';
        var detail = [];
        if (s.description) detail.push(esc(s.description));
        detail.push((s.size || 0) + " B");
        if (s.rules && s.rules.length) detail.push("规则 " + esc(s.rules.join(" / ")));
        if (s.platform && s.platform.length) detail.push("平台 " + esc(s.platform.join(" ")));
        return '<div class="list-item"><div class="ico" style="width:34px;height:34px;border-radius:9px;background:var(--primary-soft);display:flex;align-items:center;justify-content:center;flex:0 0 34px">' + pluginIcon(s.icon) + '</div>'
          + '<div class="grow"><div class="name">' + esc(s.title || s.name) + ' <span class="badge badge-off">' + esc(s.name) + '</span> ' + meta + '</div><div class="desc">' + detail.join(" · ") + '</div></div>'
          + '<div class="actions">'
          + (s.config_schema && s.config_schema.length ? '<button class="btn btn-ghost btn-sm" onclick="Relay.scriptConfig(' + jsArg(encodeURIComponent(s.name)) + ')">配置</button>' : "")
          + '<button class="btn btn-ghost btn-sm" onclick="Relay.editScript(' + jsArg(encodeURIComponent(s.name)) + ')">编辑</button>'
          + '<button class="btn btn-ghost btn-sm" onclick="Relay.runScript(' + jsArg(encodeURIComponent(s.name)) + ')">运行</button>'
          + '<button class="btn btn-outline-danger btn-sm" onclick="Relay.delScript(' + jsArg(encodeURIComponent(s.name)) + ')">删除</button></div></div>';
      }).join("") + "</div>";
      hydrateIcons();
    }).catch(function () { errorState($("#scriptList"), "puzzle", "插件列表加载失败", "网络异常，请检查服务状态。", loadScripts); });
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
      '<div class="grid grid-2"><div class="field"><label>插件文件名</label><input class="input" id="sm_name" value="' + esc(name) + '" placeholder="demo.js"></div>'
      + '<div class="field"><label>语言</label><select class="select" id="sm_lang"><option value="js">JavaScript</option><option value="python">Python</option></select></div></div>'
      + '<div class="field"><label>插件代码</label><textarea class="input code-editor" id="sm_code" rows="16" spellcheck="false"></textarea></div>'
      + '<p style="color:var(--muted);font-size:12px;margin:6px 0 0">JavaScript 用 <code>result</code> 返回结果；Python 用标准输出返回结果，以 <code>SEND:</code> 开头的行会作为会话消息回发。头部注释可用 <code>[title: ...]</code>、<code>[rule: ...]</code>、<code>[cron: ...]</code>、<code>[admin: true]</code> 声明元数据（兼容 AutMan），安装后自动登记到插件列表。</p>',
      '<button class="btn btn-ghost" onclick="Relay.closeModal()">取消</button><button class="btn btn-primary" id="sm_save">' + (isEdit ? "保存" : "安装") + "</button>");
    var codeEl = $("#sm_code"), langEl = $("#sm_lang"), nameEl = $("#sm_name");
    codeEl.value = code;
    langEl.value = lang;
    bindCodeEditor(codeEl);
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
      api("/jiying/api/script/install", { method: "POST", form: { name: nm, code: body } }).then(function (r) {
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
        return '<div class="list-item"><div class="ico" style="width:34px;height:34px;border-radius:9px;background:var(--primary-soft);display:flex;align-items:center;justify-content:center;flex:0 0 34px">' + ico("code") + "</div>"
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
      + '<div class="panel"><div class="panel-body" id="featureList"><div class="empty">加载中…</div></div></div>';
    var n = $("#newFeature");
    if (n) n.addEventListener("click", openFeatureModal);
    loadFeatures();
  }
  function featureSwitch(f) {
    var enc = encodeURIComponent(f.name);
    return '<div class="form-check form-switch rl-switch" title="' + (f.enabled ? "点击停用" : "点击启用") + '">'
      + '<input class="form-check-input" type="checkbox" role="switch" aria-label="启用 ' + esc(f.name) + '" ' + (f.enabled ? "checked" : "")
      + ' onchange="Relay.setFeature(decodeURIComponent(\'' + enc + '\'), this.checked)"></div>';
  }
  function featureRow(f) {
    var on = Boolean(f.enabled);
    var enc = encodeURIComponent(f.name);
    var kind = FEATURE_KIND[f.kind] || f.kind || "-";
    var ik = f.kind === "script" ? "code" : "zap";
    return '<div class="list-item' + (on ? "" : " rl-dim") + '">'
      + '<div class="ico" style="width:34px;height:34px;border-radius:9px;background:var(--primary-soft);display:flex;align-items:center;justify-content:center;flex:0 0 34px">' + ico(ik) + "</div>"
      + '<div class="grow"><div class="name">' + esc(f.name) + " " + metaBadge(kind) + " "
      + metaBadge(FEATURE_SCOPE[f.scope] || f.scope, f.scope === "public" ? "badge-info" : "badge-off") + " "
      + metaBadge(FEATURE_IDENTITY[f.identity] || f.identity, "badge-off")
      + (f.confirm ? " " + metaBadge("需确认", "badge-warn") : "") + "</div>"
      + '<div class="desc mono clip" title="' + esc(f.target) + '">' + esc(f.target || "-") + "</div></div>"
      + '<div class="actions">' + featureSwitch(f)
      + (f.kind === "script" ? '<button class="btn btn-ghost btn-sm" onclick="Relay.editPluginOf(\'' + encodeURIComponent(f.target || "") + '\')">编辑插件</button>' : "")
      + '<button class="btn btn-ghost btn-sm" onclick="Relay.runFeature(decodeURIComponent(\'' + enc + '\'))">运行</button>'
      + '<button class="btn btn-outline-danger btn-sm" onclick="Relay.delFeature(decodeURIComponent(\'' + enc + '\'), \'' + (f.kind || "") + '\')">删除</button></div></div>';
  }
  function loadFeatures() {
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
      list.sort(function (a, b) { return String(a.kind).localeCompare(String(b.kind)) || String(a.name).localeCompare(String(b.name), "zh"); });
      box.innerHTML = '<div class="list">' + list.map(featureRow).join("") + "</div>";
      hydrateIcons();
    }).catch(function () { errorState($("#featureList"), "toggles", "功能列表加载失败", "网络异常，请检查服务状态。", loadFeatures); });
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
      + '<div class="panel"><div class="panel-head"><h3>模型配置</h3><span class="hint">仅主控可改</span></div><div class="panel-body" id="agentCfg"><div class="empty">加载中…</div></div></div>'
      + '<div class="panel"><div class="panel-head"><h3>试跑</h3><span class="hint">不经过即应会话</span></div><div class="panel-body">'
      + '<div class="field"><label>会话 ID（可选）</label><input class="input" id="ag_conv" placeholder="留空则使用临时会话"></div>'
      + '<div class="field"><label>提问</label><textarea class="input" id="ag_prompt" rows="4" placeholder="例如：现在系统状态怎么样"></textarea></div>'
      + '<div class="rl-row"><label class="check"><input type="checkbox" id="ag_reset"> 先清空上下文</label><div class="spacer"></div>'
      + '<button class="btn btn-primary btn-sm" id="ag_run">运行</button></div>'
      + '<pre class="mono rl-out" id="ag_out">等待运行…</pre>'
      + "</div></div>"
      + '<div class="panel" style="margin-top:16px"><div class="panel-head"><h3>工具契约</h3><span class="hint">由功能与插件清单生成的 JSON Schema</span></div><div class="panel-body" id="agentTools"><div class="empty">加载中…</div></div></div>'
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
        + '<div class="field"><label>系统提示</label><textarea class="input" id="ag_system" rows="3">' + esc(d.system || "") + "</textarea></div>"
         + '<div class="field"><label>上下文消息数</label><input class="input" id="ag_hist" type="number" min="2" max="64" value="' + esc(String(d.max_history || 12)) + '"><div class="hint">每个会话保留的最近消息数量，数值越大越能延续上下文。</div></div>'
         + '<details class="agent-advanced"><summary>高级运行保护</summary><div class="field"><label>单次工具调用上限</label><input class="input" id="ag_turns" type="number" min="1" max="20" value="' + esc(String(d.max_turns || 6)) + '"><div class="hint">模型连续调用功能的最大次数，用于防止异常循环。</div></div></details>'
        + '<div class="rl-row"><div class="spacer"></div><button class="btn btn-primary btn-sm" id="ag_save">保存配置</button></div>';
      $("#ag_save").addEventListener("click", saveAgentConfig);
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
      max_history: $("#ag_hist").value.trim()
    };
    api("/jiying/api/agent/config", { method: "POST", form: form }).then(function (r) {
      if (r.code === 200) { toast("配置已保存", "ok"); loadAgentConfig(); }
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

  /* ---------- 定时任务 ---------- */
  function renderCron() {
    var cont = $("#view-cron");
    cont.innerHTML = '<div class="page-head"><div><h2>定时任务</h2><p>按 cron 表达式定时入队执行功能</p></div><div class="spacer"></div><button class="btn btn-primary" id="newCron">新建定时</button></div>'
      + '<div class="panel"><div class="table-wrap"><table class="table align-middle"><thead><tr><th style="width:180px">名称</th><th class="nowrap" style="width:150px">表达式</th><th>命令</th><th class="nowrap" style="width:80px">状态</th><th class="nowrap text-right" style="width:150px">操作</th></tr></thead><tbody id="cronList"><tr><td colspan="5"><div class="empty">加载中…</div></td></tr></tbody></table></div></div>';
    $("#newCron").addEventListener("click", openCronModal);
    loadCrons();
  }
  function loadCrons() {
    api("/jiying/api/crons").then(function (r) {
      var box = $("#cronList");
      if (!box) return;
      if (r.code !== 200) { tableError(box, 5, "clock", r.code === 403 ? "权限不足" : "定时任务加载失败", r.msg || r.message, loadCrons); return; }
      var list = r.data ? r.data : [];
      if (!list.length) { box.innerHTML = '<tr><td colspan="5">' + emptyState("clock", "暂无定时任务", "新建定时后，中间件会按 cron 表达式自动入队执行。") + "</td></tr>"; return; }
      box.innerHTML = list.map(function (c) {
        var on = Boolean(c.status);
        return '<tr><td class="nowrap">' + esc(c.name) + '</td><td class="nowrap"><code>' + esc(c.cron) + '</code></td><td class="clip" title="' + esc(c.command) + '">' + esc(c.command) + '</td><td class="nowrap">' + (on ? '<span class="badge badge-ok">启用</span>' : '<span class="badge badge-off">停用</span>') + '</td><td class="text-right nowrap"><button class="btn btn-ghost btn-sm" onclick="Relay.toggleCron(' + jsArg(c.id) + ',' + (on ? "false" : "true") + ')">' + (on ? "停用" : "启用") + "</button> " + '<button class="btn btn-outline-danger btn-sm" onclick="Relay.delCron(' + jsArg(c.id) + ')">删除</button></td></tr>';
      }).join("");
    }).catch(function () { tableError($("#cronList"), 5, "clock", "定时任务加载失败", "网络异常，请检查服务状态。", loadCrons); });
  }
  function openCronModal() {
    showModal("新建定时任务", '<div class="field"><label>名称</label><input class="input" id="cm_name" placeholder="示例任务"></div>'
      + '<div class="field"><label>Cron 表达式</label><input class="input" id="cm_cron" placeholder="*/30 * * * * *"></div>'
      + '<div class="field"><label>命令 / 功能名</label><input class="input" id="cm_cmd" placeholder="状态"></div>',
      '<button class="btn btn-ghost" onclick="Relay.closeModal()">取消</button><button class="btn btn-primary" id="cm_save">创建</button>');
    $("#cm_save").addEventListener("click", function () {
      var name = $("#cm_name").value.trim(), cron = $("#cm_cron").value.trim(), cmd = $("#cm_cmd").value.trim();
      if (!name || !cron || !cmd) { toast("请填写名称、表达式与命令", "err"); return; }
      api("/jiying/api/cron/add", { method: "POST", form: { name: name, cron: cron, command: cmd } }).then(function (r) {
        if (r.code === 200) { closeModal(); toast("已创建", "ok"); loadCrons(); }
        else toast(r.msg || "创建失败", "err");
      });
    });
  }
  function toggleCron(id, on) {
    api("/jiying/api/cron/toggle", { method: "POST", form: { id: id, enabled: on ? "true" : "false" } }).then(function (r) {
      if (r.code === 200) { toast(on ? "已启用" : "已停用", "ok"); loadCrons(); }
      else toast(r.msg || "操作失败", "err");
    });
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
      + '<input class="input" id="taskFilter" placeholder="筛选功能 / 会话 / 结果" style="width:200px">'
      + '<button class="btn btn-ghost btn-sm" onclick="Relay.loadTasks()">刷新</button></div>'
      + '<div class="panel"><div class="table-wrap"><table class="table align-middle"><thead><tr><th style="width:128px">ID</th><th class="nowrap" style="width:88px">功能</th><th style="width:120px">会话</th><th class="nowrap" style="width:88px">状态</th><th>结果</th><th class="nowrap text-right" style="width:120px">操作</th></tr></thead><tbody id="taskList"><tr><td colspan="6"><div class="empty">加载中…</div></td></tr></tbody></table></div></div>'
      + '<div class="muted" id="taskMeta" style="margin-top:8px;font-size:12px"></div>';
    $("#taskStatus").addEventListener("change", paintTasks);
    $("#taskFilter").addEventListener("input", paintTasks);
    Relay.loadTasks();
  }
  // 状态筛选把 timeout 归入「失败」，与徽标语义一致。
  var TASK_STATUS_GROUP = { failed: ["failed", "timeout"], waiting: ["waiting", "waiting_confirmation"] };
  function taskMatchStatus(t, want) {
    if (!want) return true;
    return (TASK_STATUS_GROUP[want] || [want]).indexOf(String(t.status)) >= 0;
  }
  function loadTasks() {
    api("/jiying/api/tasks").then(function (r) {
      var box = $("#taskList");
      if (!box) return;
      if (r.code !== 200) {
        state.tasks = [];
        state.tasksErr = r.msg || r.message || "请稍后重试。";
        tableError(box, 6, "list", r.code === 403 ? "权限不足" : "任务列表加载失败", state.tasksErr, function () { state.tasksErr = ""; loadTasks(); });
        setTaskMeta(0, 0);
        return;
      }
      state.tasksErr = "";
      state.tasks = r.data ? r.data : [];
      paintTasks();
    }).catch(function () {
      var box = $("#taskList");
      if (!box) return;
      state.tasks = [];
      state.tasksErr = "网络异常，请稍后重试。";
      tableError(box, 6, "list", "任务列表加载失败", state.tasksErr, function () { state.tasksErr = ""; loadTasks(); });
      setTaskMeta(0, 0);
    });
  }
  function paintTasks() {
    var box = $("#taskList");
    if (!box) return;
    if (state.tasksErr) return;
    var all = state.tasks || [];
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
        return '<tr><td class="mono clip" title="' + esc(tid) + '"><a href="javascript:void(0)" onclick="Relay.taskDetail(' + jsArg(tid) + ')">' + esc(shortID(tid)) + "</a></td>"
          + '<td class="nowrap">' + esc(t.feature) + "</td>"
          + '<td class="mono clip" title="' + esc(cid) + '">' + esc(cid ? shortID(cid) : "-") + "</td>"
          + '<td class="nowrap">' + stb + "</td>"
          + '<td class="muted"><div class="clamp2" title="' + esc(res) + '">' + esc(res) + "</div><small class=\"muted\">" + esc(t.error_code || "") + (t.duration_ms ? " · " + esc(String(t.duration_ms)) + " ms" : "") + "</small></td>"
          + '<td class="text-right nowrap">' + (t.status === "failed" ? '<button class="btn btn-ghost btn-sm" onclick="Relay.retryTask(' + jsArg(tid) + ')">重试</button> ' : "") + '<button class="btn btn-outline-danger btn-sm" onclick="Relay.delTask(' + jsArg(tid) + ')">删除</button></td></tr>';
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
    var t = (state.tasks || []).filter(function (x) { return String(x.id) === String(id); })[0];
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
  function renderLogs() {
    stopLogAuto();
    var cont = $("#view-logs");
    cont.innerHTML = '<div class="page-head"><div><h2>运行日志</h2><p>中间件运行日志（需登录控制台；访问日志仅记方法与路径）</p></div><div class="spacer"></div>'
      + '<input class="input" id="logDay" style="width:150px" type="date">'
      + '<select class="select" id="logTail" style="width:110px"><option>100</option><option>200</option><option selected>500</option><option>1000</option></select>'
      + '<input class="input" id="logFilter" placeholder="筛选关键字" style="width:180px">'
      + '<label class="log-auto muted"><input type="checkbox" id="logAuto"> 自动刷新</label>'
      + '<label class="log-auto muted"><input type="checkbox" id="logWrap"> 自动换行</label>'
      + '<button class="btn btn-sm" id="logExport">导出</button>'
      + '<button class="btn btn-primary btn-sm" id="logReload">刷新</button></div>'
      + '<div class="panel"><div class="panel-body"><pre class="mono rl-logbox" id="logBox" style="margin:0;max-height:calc(100vh - 220px);overflow:auto;background:oklch(0.153 0.006 107.1);color:oklch(0.9 0.02 106.5);padding:14px;border-radius:10px;font-size:12px;line-height:1.6">加载中…</pre></div></div>'
      + '<div class="muted" id="logMeta" style="margin-top:8px;font-size:12px"></div>';
    $("#logReload").addEventListener("click", loadLogs);
    $("#logDay").value = today();
    $("#logFilter").addEventListener("input", paintLogs);
    $("#logExport").addEventListener("click", exportLogs);
    $("#logAuto").addEventListener("change", function () { setLogAuto(this.checked); });
    $("#logWrap").addEventListener("change", function () { setLogWrap(this.checked); });
    loadLogs();
  }
  // 默认按行展示（长行横向滚动），避免 trace= 等 token 被从中间断开；需要时再开自动换行。
  function setLogWrap(on) {
    var box = $("#logBox");
    if (box) box.classList.toggle("rl-logbox-wrap", !!on);
  }
  // 自动刷新为「tail -f」式流式查看，5 秒拉取一次当日日志并滚到底部。
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
    var day = ($("#logDay").value || today()).replace(/-/g, "");
    var tail = $("#logTail").value;
    api("/jiying/api/logs?day=" + day + "&tail=" + tail).then(function (r) {
      var box = $("#logBox");
      if (!box) return;
      if (r.code !== 200) { box.textContent = r.msg || "加载失败"; return; }
      state.logLines = r.data.lines || [];
      if (state.logPending) { $("#logFilter").value = state.logPending; state.logPending = ""; }
      paintLogs();
    });
  }
  function searchLogs(kw) {
    state.logPending = String(kw || "");
    if (state.view === "logs") { $("#logFilter").value = state.logPending; state.logPending = ""; paintLogs(); }
    else switchView("logs");
  }
  function paintLogs() {
    var box = $("#logBox");
    if (!box) return;
    var kw = (($("#logFilter") || {}).value || "").trim().toLowerCase();
    var lines = state.logLines;
    var shown = kw ? lines.filter(function (l) { return String(l).toLowerCase().indexOf(kw) >= 0; }) : lines;
    box.textContent = shown.join("\n") || (lines.length ? "（无匹配行）" : "（当日暂无日志）");
    box.scrollTop = box.scrollHeight;
    var meta = $("#logMeta");
    if (meta) meta.textContent = kw ? "匹配 " + shown.length + " / " + lines.length + " 行" : "共 " + lines.length + " 行";
  }
  function exportLogs() {
    var lines = state.logLines || [];
    if (!lines.length) { toast("暂无日志可导出", "error"); return; }
    var kw = (($("#logFilter") || {}).value || "").trim().toLowerCase();
    var out = kw ? lines.filter(function (l) { return String(l).toLowerCase().indexOf(kw) >= 0; }) : lines;
    var day = ($("#logDay").value || today());
    downloadBlob(new Blob([out.join("\n") + "\n"], { type: "text/plain;charset=utf-8" }), "relay-logs-" + day + ".log");
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
      + '<div class="panel"><div class="panel-head"><h3>应用通道</h3></div><div class="panel-body" id="poolApps"><div class="empty">加载中…</div></div></div>'
       + '<div class="panel"><div class="panel-head"><h3>个人账号</h3></div><div class="panel-body" id="poolAccountProfile"></div><div class="panel-head pool-account-list-head"><h3>已授权账号</h3></div><div class="panel-body" id="poolAccountList"></div></div>'
      + "</div>"
      + '<div class="panel mt-16"><div class="panel-head"><h3>接入指引</h3></div><div class="panel-body"><div class="rl-cols">'
      + '<div><b>1. 准备应用</b><p class="muted" style="margin:6px 0 0;font-size:13px">在 MagicChat 创建应用，取得应用 ID 与应用密钥。</p></div>'
      + '<div><b>2. 接入通道</b><p class="muted" style="margin:6px 0 0;font-size:13px">填写 WebSocket 地址、应用 ID 与密钥，保存后即建立通道。</p></div>'
      + '<div><b>3. 按应用隔离</b><p class="muted" style="margin:6px 0 0;font-size:13px">多个应用各自运行与记账，默认应用用于未指定来源的任务。</p></div>'
      + "</div></div></div>";
    var n = $("#newApp");
    if (n) n.addEventListener("click", openAppModal);
    loadPool();
  }
  function loadPool() {
    api("/jiying/api/connections").then(function (r) {
      if (r.code !== 200) { toast(r.msg || "加载失败", "err"); return; }
      renderPoolApps(r.data.apps || []);
        renderPoolAccount(r.data.account || {});
        renderPoolAccounts(r.data.accounts || []);
    }).catch(function () { errorState($("#poolApps"), "link", "连接池状态加载失败", "网络异常，请检查服务状态。", loadPool); });
  }
  function renderPoolApps(apps) {
    var box = $("#poolApps");
    if (!box) return;
    if (!apps.length) { box.innerHTML = emptyState("link", "尚未接入应用", "接入后可在此查看通道在线状态，并按应用隔离运行。", '<button class="btn btn-primary btn-sm" id="poolEmptyAdd">接入应用</button>'); bindPoolEmptyAdd(); return; }
    box.innerHTML = apps.map(function (a) {
      var id = String(a.app_id || "");
      var short = shortID(id);
      return '<div class="list-item"><span class="dot ' + (a.online ? "on" : "off") + '"></span>'
        + '<div class="grow"><div class="name" title="' + esc(id) + '">' + esc(short || "(未命名)") + (a.is_default ? ' <span class="badge badge-ok">默认</span>' : "") + "</div>"
        + '<div class="desc mono">' + esc(a.ws_url || "-") + "</div></div>"
          + '<div class="actions">' + (a.is_default ? "" : '<button class="btn btn-ghost btn-sm" onclick="Relay.defaultApp(' + jsArg(id) + ')">设为默认</button>') + '<button class="btn btn-outline-danger btn-sm" onclick="Relay.delApp(' + jsArg(id) + ')">移除</button></div></div>';
    }).join("");
  }
  function bindPoolEmptyAdd() {
    var b = $("#poolEmptyAdd");
    if (b) b.addEventListener("click", openAppModal);
  }
  function renderPoolAccount(a) {
    var box = $("#poolAccountProfile");
    if (!box) return;
    var avatar = a.avatar_url ? '<img class="avatar pool-avatar" src="' + esc(a.avatar_url) + '" alt="">' : '<div class="avatar pool-avatar">' + esc((a.email || "U").charAt(0).toUpperCase()) + "</div>";
    var st = String(a.status || "未登录");
    var badge = st.indexOf("已登录") >= 0 ? '<span class="badge badge-ok">已登录</span>' : '<span class="badge badge-off">未登录</span>';
    box.innerHTML = '<div class="rl-row"><div class="grow"><div class="kv"><span class="k">账号</span><span class="v">' + esc(a.email || "未配置") + "</span></div>"
      + '<div class="kv"><span class="k">昵称</span><span class="v">' + (a.nickname ? esc(a.nickname) : '<span class="muted">未设置</span>') + "</span></div>"
      + '<div class="kv"><span class="k">状态</span><span class="v">' + badge + "</span></div></div>" + avatar + "</div>";
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

  /* ---------- 插件市场 ---------- */
  function renderMarket() {
    var cont = $("#view-market");
    cont.innerHTML = '<div class="page-head"><div><h2>插件市场</h2><p>发现、上架与分发包，已安装的插件在「插件」页统一管理</p></div><div class="spacer"></div><button class="btn btn-primary" id="newPlugin">上架插件</button></div>'
      + '<div class="panel"><div class="panel-body" id="marketList"><div class="empty">加载中…</div></div></div>'
      + '<div class="panel mt-16"><div class="panel-head"><h3>插件规范</h3></div><div class="panel-body"><div class="rl-cols">'
      + '<div><b>作用范围</b><p class="muted" style="margin:6px 0 0;font-size:13px">插件归属 <span class="mono">public</span> 或 <span class="mono">master</span>，安装时按应用或个人账号隔离。</p></div>'
      + '<div><b>类型</b><p class="muted" style="margin:6px 0 0;font-size:13px">支持脚本型与 http 型，脚本型随插件一同分发源码。</p></div>'
      + '<div><b>版本</b><p class="muted" style="margin:6px 0 0;font-size:13px">同名插件按版本区分，已安装的会在列表中标注。</p></div>'
      + "</div></div></div>";
    var n = $("#newPlugin");
    if (n) n.addEventListener("click", openPluginModal);
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
        return '<div class="list-item"><div class="ico" data-ico="inbox" style="width:34px;height:34px;border-radius:9px;background:var(--primary-soft);display:flex;align-items:center;justify-content:center;flex:0 0 34px">' + ico("inbox") + '</div>'
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
  function installPlugin(name) {
    api("/jiying/api/market/install", { method: "POST", form: { name: name } }).then(function (r) {
      if (r.code === 200) { toast(r.data || "已安装", "ok"); loadMarket(); }
      else toast(r.msg || "安装失败", "err");
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
      + '<div class="panel"><div class="table-wrap"><table class="table align-middle"><thead><tr><th class="nowrap" style="width:150px">时间</th><th class="nowrap" style="width:110px">操作者</th><th class="nowrap" style="width:130px">动作</th><th style="width:150px">目标</th><th>详情</th><th class="nowrap" style="width:130px">结果</th></tr></thead><tbody id="auditList"><tr><td colspan="6"><div class="empty">加载中…</div></td></tr></tbody></table></div></div>'
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
        tableError(box, 6, "shield", r.code === 403 ? "权限不足" : "审计日志加载失败", state.auditErr, function () { state.auditErr = ""; loadAudit(); });
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
      tableError(box, 6, "shield", "审计日志加载失败", state.auditErr, function () { state.auditErr = ""; loadAudit(); });
      setAuditMeta(0, 0);
    });
  }
  // 审计行与筛选逻辑同时服务于渲染和导出，集中一处避免两处过滤条件漂移。
  function filteredAudit(rows) {
    var kw = (($("#auditFilter") || {}).value || "").trim().toLowerCase();
    if (!kw) return rows;
    return rows.filter(function (a) {
      return [a.actor, a.action, a.target, a.detail, a.result].some(function (v) { return String(v || "").toLowerCase().indexOf(kw) >= 0; });
    });
  }
  function paintAudit() {
    var box = $("#auditList");
    if (!box) return;
    if (state.auditErr) return;
    var rows = state.auditRows || [];
    if (!rows.length) { box.innerHTML = '<tr><td colspan="6">' + emptyState("shield", "暂无审计记录", "敏感操作（接入、插件、角色等）会在这里留痕。") + "</td></tr>"; setAuditMeta(0, 0); return; }
    var kw = (($("#auditFilter") || {}).value || "").trim().toLowerCase();
    var shown = filteredAudit(rows);
    box.innerHTML = shown.map(function (a) {
      var ok = auditOK(a.result);
      return '<tr><td class="mono nowrap">' + esc(fmtTime(a.time)) + '</td><td class="nowrap">' + esc(a.actor) + '</td><td class="mono nowrap">' + esc(a.action) + '</td><td class="mono clip" title="' + esc(a.target) + '">' + esc(a.target) + '</td><td>' + esc(a.detail) + '</td><td class="nowrap">' + (ok ? '<span class="badge badge-ok">成功</span>' : '<span class="badge badge-err">失败</span>') + ' <button class="btn btn-ghost btn-sm" title="在运行日志中查找" onclick="Relay.searchLogs(' + jsArg(a.action) + ')"><i class="bi bi-search"></i></button></td></tr>';
    }).join("") || '<tr><td colspan="6"><div class="empty">没有匹配「' + esc(kw) + '」的记录</div></td></tr>';
    setAuditMeta(shown.length, rows.length);
  }
  // 导出当前筛选结果为 CSV，带 BOM 以兼容 Excel 的中文显示。
  function exportAudit() {
    var rows = filteredAudit(state.auditRows || []);
    if (!rows.length) { toast("暂无审计记录可导出", "error"); return; }
    var head = ["时间", "操作者", "动作", "目标", "详情", "结果"];
    var lines = [head].concat(rows.map(function (a) {
      return [fmtTime(a.time), a.actor, a.action, a.target, a.detail, a.result];
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
      + '<div class="grid grid-3" id="observeTiles">'
      + tile("健康探针", "…", "GET /healthz") + tile("运转时长", "…", "进程已运行")
      + tile("已收消息", "…", "累计接收") + tile("任务队列", "…", "执行中")
      + tile("通道状态", "…", "应用 · 账号") + tile("限流拒绝", "…", "被拒绝任务")
      + "</div>"
      + '<div class="panel mt-16"><div class="panel-head"><h3>原始指标</h3><span class="hint">Prometheus 文本格式</span></div><div class="panel-body"><pre class="mono" id="observeRaw" style="margin:0;white-space:pre-wrap;background:var(--sidebar-accent);padding:14px;border-radius:10px;font-size:12px;max-height:420px;overflow:auto">加载中…</pre></div></div>';
    Relay.loadObserve();
  }

  function renderRoles() {
    var cont = $("#view-roles");
      cont.innerHTML = '<div class="page-head"><div><h2>权限管理</h2><p>主人权限面向即应用户 ID（在即应会话里发送「我的ID」获取）；运营与只读面向控制台登录账号。</p></div></div>'
        + '<div class="panel"><div class="panel-head"><h3>授权成员</h3><span class="hint" id="myAccount"></span></div><div class="panel-body"><div class="input-row"><input class="input" id="roleUser" placeholder="输入用户 ID"><select class="select" id="roleValue" style="max-width:150px"><option value="ops">运营</option><option value="readonly">只读</option><option value="master">主人</option></select><button class="btn btn-primary" id="grantRole">授予权限</button></div></div></div>'
       + '<div class="panel mt-16"><div class="panel-head"><h3>成员角色</h3></div><div class="panel-body" id="roleList"><div class="empty">加载中…</div></div></div>'
       + '<div class="panel mt-16"><div class="panel-head"><h3>主人列表</h3></div><div class="panel-body" id="masterList"><div class="empty">加载中…</div></div></div>';
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
  function tile(label, val, sub) {
    return '<div class="panel tile"><span class="t-lab">' + esc(label) + '</span><span class="t-val">' + esc(val) + "</span>" + (sub ? '<span class="t-sub">' + esc(sub) + "</span>" : "") + "</div>";
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
    chat: renderChat,
    scripts: renderScripts,
    features: renderFeatures,
    agent: renderAgent,
    cron: renderCron,
    tasks: renderTasks,
    market: renderMarket,
    audit: renderAudit,
    observe: renderObserve,
    logs: renderLogs,
    roles: renderRoles
  };

  // 需要定时轮询的视图：切换视图时自动启动，离开时由 switchView 清理。
  var POLL = { home: function () { refreshHome(true); }, features: loadFeatures, observe: loadObserve };
  var POLL_MS = { home: 10000, features: 10000, observe: 5000 };

  /* ---------- 启动 ---------- */
  function init() {
    installErrorSink();
    installTheme();
    // 语言：读取本机偏好并翻译静态外壳，登录页同样生效。
    LANG = readLang();
    document.documentElement.lang = (LANG === "en") ? "en" : "zh-CN";
    applyStaticI18n();
    refreshLangButtons();
    hydrateIcons();
    var lb = $("#langBtn");
    if (lb) lb.addEventListener("click", toggleLang);
    if (location.pathname.endsWith("/login.html") || location.pathname === "/login.html") return;
    hydrateIcons();
    renderNav();
    var nt = $("#navToggle");
    if (nt) nt.addEventListener("click", toggleNav);
    var nc = $("#navCustomBtn");
    if (nc) nc.addEventListener("click", openNavCustomModal);
    var nb = $("#navBackdrop");
    if (nb) nb.addEventListener("click", closeNav);
    window.addEventListener("resize", function () { if (!isMobile()) closeNav(); hintScroll(); });
    // 快捷键：按 “/” 聚焦菜单筛选框（输入框内不拦截）。
    document.addEventListener("keydown", function (e) {
      if (e.key !== "/" || e.ctrlKey || e.metaKey || e.altKey) return;
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
      loadConnectStatus();
      // 顶栏显示当前角色等级，便于用户判断自己可执行的操作范围。
      api("/jiying/api/roles", { silent: true }).then(function (rr) {
        if (rr.code === 200 && rr.data) {
          state.role = rr.data.me || "";
          var el = $("#userRole");
          if (el && ROLE_LABEL[state.role]) el.textContent = "角色 · " + ROLE_LABEL[state.role];
        }
      });
      switchView(defaultViewId());
      setInterval(function () { loadConnectStatus(); }, 15000);
    }).catch(function () { location.href = "login.html"; });
  }
  var modal = $("#modal");
  if (modal) modal.addEventListener("click", function (e) { if (e.target === this) closeModal(); });

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
    toggleCron: toggleCron,
    delCron: delCron,
    loadTasks: loadTasks,
    retryTask: retryTask,
    delTask: delTask,
    renderConnStatus: renderConnStatus,
    selectAccount: selectAccount,
    removeMaster: removeMaster,
    deleteRole: deleteRole,
    loadPool: loadPool,
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
