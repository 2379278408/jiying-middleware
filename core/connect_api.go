package core

import (
	"context"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func registerConnectAPI() {
	g := Server.Group("/jiying/api/connect", requireLogin)
	g.GET("/status", connectStatus)
	g.GET("/readiness", connectReadiness)
	g.POST("/app", requireRole(RoleOps), connectApp)
	g.POST("/test-app", requireRole(RoleOps), connectTestApp)
	g.POST("/account", requireRole(RoleOps), connectAccount)
	g.POST("/test-account", requireRole(RoleOps), connectTestAccount)
	g.POST("/account/logout", requireRole(RoleOps), connectAccountLogout)
	g.GET("/account/health", connectAccountHealth)
	g.POST("/account/health/check", requireRole(RoleOps), connectAccountHealthCheck)
	g.POST("/request-code", requireRole(RoleOps), connectRequestCode)
	g.POST("/login-code", requireRole(RoleOps), connectLoginCode)
}

func maskSecret(s string) string {
	if strings.TrimSpace(s) == "" {
		return ""
	}
	// 只表示“已配置”，不暴露密钥的任何字符。
	return "****"
}

func connectStatus(c *gin.Context) {
	apps := AppPoolSnapshot()
	acctConn := CurrentAccountConn()
	acctCfg := UserLogin{BaseURL: acctConn.BaseURL, Email: acctConn.Email, Password: acctConn.Secret}
	var def AppPoolItem
	for _, a := range apps {
		if a.IsDefault {
			def = a
			break
		}
	}
	jiyingJSON(c, gin.H{
		"app": gin.H{
			"configured": def.AppID != "",
			"ws_url":     def.WSURL,
			"app_id":     def.AppID,
			"connected":  def.Online,
			"count":      len(apps),
		},
		"apps": apps,
		"account": gin.H{
			"configured":    acctConn.configured(),
			"code_login":    acctConn.CodeLogin,
			"base_url":      maskURL(acctCfg.BaseURL),
			"email":         acctCfg.Email,
			"secret_masked": maskSecret(acctCfg.Password),
			"logged_in":     userLoggedIn(),
			"status":        UserSessionStatus(),
			"health":        accountHealthSnapshot(),
		},
		"runtime": gin.H{
			"uptime": JiyingUptime(),
			"time":   time.Now().Format("2006-01-02 15:04:05"),
		},
		"readiness": connectionReadiness(),
	})
}

func connectReadiness(c *gin.Context) {
	jiyingJSON(c, connectionReadiness())
}

func maskURL(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	i := strings.Index(s, "://")
	if i < 0 {
		return s
	}
	return s[:i+3] + strings.Repeat("*", 8)
}

// stripURLCredentials 去掉地址中的 userinfo（user:password@），
// 只读展示账号地址时避免把嵌在 URL 里的凭据一并回显。
func stripURLCredentials(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	u, err := url.Parse(s)
	if err != nil || u.Host == "" || u.User == nil {
		return s
	}
	u.User = nil
	return u.String()
}

// maskedSentinel 判断前端回传的值是否为掩码占位（形如 wss://********）或空值。
// 控制台表单会用掩码值回填地址字段，提交时若原样回传，必须按“未修改”处理，
// 否则会把掩码当成真实地址落库，导致连接失效。
func maskedSentinel(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return true
	}
	i := strings.Index(s, "://")
	if i < 0 {
		return false
	}
	host := s[i+3:]
	if j := strings.IndexAny(host, "/?#"); j >= 0 {
		host = host[:j]
	}
	return host != "" && strings.Trim(host, "*") == ""
}

// resolveAppWSURL 在提交值为空或掩码时回退到已存的同应用地址。
func resolveAppWSURL(submitted, appID string) string {
	if !maskedSentinel(submitted) {
		return strings.TrimSpace(submitted)
	}
	for _, a := range loadAppConns() {
		if a.AppID == appID && strings.TrimSpace(a.WSURL) != "" {
			return a.WSURL
		}
	}
	return ""
}

// resolveAccountBaseURL 在提交值为空或掩码时回退到当前账号的已存地址。
func resolveAccountBaseURL(submitted string) string {
	submitted = strings.TrimRight(strings.TrimSpace(submitted), "/")
	if !maskedSentinel(submitted) {
		return submitted
	}
	return strings.TrimRight(loadRelayConn().effectiveAccount().BaseURL, "/")
}

func connectApp(c *gin.Context) {
	appID := strings.TrimSpace(c.PostForm("app_id"))
	wsURL := resolveAppWSURL(c.PostForm("ws_url"), appID)
	secret := c.PostForm("app_secret")
	if wsURL == "" || appID == "" || secret == "" {
		RecordAudit(consoleUser(c), "connect.app", appID, "", "失败: 缺少参数")
		jiyingErr(c, "缺少应用地址 / ID / 密钥")
		return
	}
	if !isUUID(appID) {
		RecordAudit(consoleUser(c), "connect.app", appID, "", "失败: 应用 ID 不是 UUID")
		jiyingErr(c, "应用 ID 必须是 UUID（8-4-4-4-12），请从 MagicChat 应用详情复制")
		return
	}
	if secret == "****" || secret == "" {
		for _, a := range loadAppConns() {
			if a.AppID == appID {
				secret = a.AppSecret
				break
			}
		}
	}
	if err := SyncAppConn(AppConn{WSURL: wsURL, AppID: appID, AppSecret: secret}); err != nil {
		RecordAudit(consoleUser(c), "connect.app", appID, "", "失败: "+err.Error())
		jiyingErr(c, err.Error())
		return
	}
	connected := "即应未连接"
	for i := 0; i < 30; i++ {
		if JiyingState() == "即应已连接" {
			connected = "即应已连接"
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	RecordAudit(consoleUser(c), "connect.app", appID, connected, "成功")
	jiyingJSON(c, gin.H{"connected": connected})
}

func connectTestApp(c *gin.Context) {
	cfg := AppConn{AppID: strings.TrimSpace(c.PostForm("app_id")), AppSecret: c.PostForm("app_secret")}
	cfg.WSURL = resolveAppWSURL(c.PostForm("ws_url"), cfg.AppID)
	if cfg.AppSecret == "****" || cfg.AppSecret == "" {
		for _, a := range loadAppConns() {
			if a.AppID == cfg.AppID {
				cfg.AppSecret = a.AppSecret
				break
			}
		}
	}
	if err := testAppConnection(c.Request.Context(), cfg); err != nil {
		jiyingErr(c, "应用连接测试失败: "+err.Error())
		return
	}
	jiyingJSON(c, gin.H{"tested": true, "status": "connected"})
}

func connectAccount(c *gin.Context) {
	email := strings.TrimSpace(c.PostForm("email"))
	secret := c.PostForm("password")
	base := resolveAccountBaseURL(c.PostForm("base_url"))
	if base == "" || email == "" || secret == "" {
		RecordAudit(consoleUser(c), "connect.account", email, "", "失败: 缺少参数")
		jiyingErr(c, "缺少服务地址 / 邮箱 / 密码")
		return
	}
	conn := loadRelayConn()
	cur := conn.effectiveAccount()
	if secret == "****" {
		secret = cur.Password
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 20*time.Second)
	defer cancel()
	if err := loginAccount(ctx, UserLogin{BaseURL: base, Email: email, Password: secret}, false); err != nil {
		RecordAudit(consoleUser(c), "connect.account", email, "", "失败: 登录失败")
		jiyingErr(c, "登录失败: "+err.Error())
		return
	}
	if err := SyncAccountConn(AccountConn{BaseURL: base, Email: email, Secret: secret, CodeLogin: false}); err != nil {
		RecordAudit(consoleUser(c), "connect.account", email, "", "失败: "+err.Error())
		jiyingErr(c, err.Error())
		return
	}
	RecordAudit(consoleUser(c), "connect.account", email, "", "成功")
	jiyingJSON(c, gin.H{"logged_in": true, "status": UserSessionStatus()})
}

func connectTestAccount(c *gin.Context) {
	cfg := CurrentAccountConfig()
	if base := strings.TrimSpace(c.PostForm("base_url")); base != "" && !maskedSentinel(base) {
		cfg.BaseURL = strings.TrimRight(base, "/")
	}
	if email := strings.TrimSpace(c.PostForm("email")); email != "" {
		cfg.Email = email
	}
	check := accountReadiness(cfg)
	if check.Status != "configured" {
		jiyingErr(c, "账号配置无效")
		return
	}
	if !userLoggedIn() {
		jiyingErr(c, "账号通道未登录，请先完成登录")
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()
	if _, err := UserCall(ctx, "GET", "/api/client/me", nil); err != nil {
		jiyingErr(c, "账号连接测试失败: "+err.Error())
		return
	}
	jiyingJSON(c, gin.H{"tested": true, "status": "connected", "account": cfg.Email})
}

func connectAccountLogout(c *gin.Context) {
	email := CurrentAccountConn().Email
	RecordAudit(consoleUser(c), "connect.account.logout", email, "", "成功")
	SetUserSession("", "")
	clearUserToken(email)
	jiyingJSON(c, "ok")
}

func connectAccountHealth(c *gin.Context) {
	jiyingJSON(c, accountHealthSnapshot())
}

func connectAccountHealthCheck(c *gin.Context) {
	health := checkAccountHealth(c.Request.Context())
	RecordAudit(consoleUser(c), "account.health", health.Email, health.Status, health.Message)
	jiyingJSON(c, health)
}

func connectRequestCode(c *gin.Context) {
	email := strings.TrimSpace(c.PostForm("email"))
	base := resolveAccountBaseURL(c.PostForm("base_url"))
	if base == "" || email == "" {
		jiyingErr(c, "缺少服务地址 / 邮箱")
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 20*time.Second)
	defer cancel()
	expires, retry, err := RequestLoginCode(ctx, base, email)
	if err != nil {
		jiyingErr(c, "验证码发送失败: "+err.Error())
		return
	}
	jiyingJSON(c, gin.H{"expires_in_seconds": expires, "retry_after_seconds": retry})
}

func connectLoginCode(c *gin.Context) {
	email := strings.TrimSpace(c.PostForm("email"))
	code := strings.TrimSpace(c.PostForm("code"))
	base := resolveAccountBaseURL(c.PostForm("base_url"))
	if base == "" || email == "" || code == "" {
		jiyingErr(c, "缺少服务地址 / 邮箱 / 验证码")
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 20*time.Second)
	defer cancel()
	if err := loginAccount(ctx, UserLogin{BaseURL: base, Email: email, Password: code}, true); err != nil {
		RecordAudit(consoleUser(c), "connect.account.code", email, "", "失败: 登录失败")
		jiyingErr(c, "登录失败: "+err.Error())
		return
	}
	if err := SyncAccountConn(AccountConn{BaseURL: base, Email: email, Secret: "", CodeLogin: true}); err != nil {
		RecordAudit(consoleUser(c), "connect.account.code", email, "", "失败: "+err.Error())
		jiyingErr(c, err.Error())
		return
	}
	RecordAudit(consoleUser(c), "connect.account.code", email, "", "成功")
	jiyingJSON(c, gin.H{"logged_in": true, "status": UserSessionStatus()})
}
