(function () {
  "use strict";
  // 复用 relay.js 暴露的翻译函数；缺失时回退中文常量。
  function T(k, fallback) {
    return (window.RelayI18N && window.RelayI18N.t) ? window.RelayI18N.t(k) : fallback;
  }
  function doLogin() {
    var un = document.getElementById("username").value.trim();
    var pw = document.getElementById("password").value;
    var msg = document.getElementById("loginMsg");
    var btn = document.getElementById("loginBtn");
    if (!un || !pw) { msg.textContent = T("login.empty", "请输入用户名与密码"); return; }
    if (btn.disabled) return;
    btn.disabled = true;
    btn.setAttribute("aria-busy", "true");
    msg.textContent = T("login.busy", "登录中…");
    fetch("/login", {
      method: "POST",
      credentials: "same-origin",
      headers: { "Content-Type": "application/x-www-form-urlencoded" },
      body: new URLSearchParams({ username: un, password: pw })
     }).then(function (r) {
       return r.text().then(function (raw) { try { return raw ? JSON.parse(raw) : {}; } catch (e) { return { msg: T("login.parse", "服务返回了无法解析的响应") }; } });
     }).then(function (r) {
       if (r.code === 200) { location.href = "index.html"; }
       else { msg.textContent = r.msg || T("login.fail", "登录失败"); btn.disabled = false; btn.removeAttribute("aria-busy"); }
     }).catch(function () { msg.textContent = T("login.network", "网络错误，请稍后重试"); btn.disabled = false; btn.removeAttribute("aria-busy"); });
  }
  document.getElementById("loginBtn").addEventListener("click", doLogin);
  ["username", "password"].forEach(function (id) {
    document.getElementById(id).addEventListener("keydown", function (e) { if (e.key === "Enter") doLogin(); });
  });
  fetch("/login", { credentials: "same-origin" }).then(function (r) {
    if (r.ok) location.href = "index.html";
  }).catch(function () {});
})();
