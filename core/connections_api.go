package core

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// AccountProfile 是个人账号对外展示的必要信息，不含任何凭据。
type AccountProfile struct {
	Email     string `json:"email"`
	Nickname  string `json:"nickname,omitempty"`
	AvatarURL string `json:"avatar_url,omitempty"`
}

// registerConnectionsAPI 注册连接池与账号状态接口。
func registerConnectionsAPI() {
	g := Server.Group("/jiying/api", requireLogin)
	g.GET("/connections", connectionsSnapshot)
	g.POST("/connections/app", requireRole(RoleOps), connectionsAppUpsert)
	g.POST("/connections/app/delete", requireRole(RoleOps), connectionsAppDelete)
	g.POST("/connections/app/default", requireRole(RoleOps), connectionsAppDefault)
	g.POST("/connections/account/select", requireRole(RoleOps), connectionsAccountSelect)
	g.POST("/connections/account/delete", requireRole(RoleOps), connectionsAccountDelete)
	g.POST("/connections/account/profile", connectionsAccountProfile)
}

func connectionsSnapshot(c *gin.Context) {
	apps := AppPoolSnapshot()
	acct := CurrentAccountConfig()
	acctConn := CurrentAccountConn()
	profile := accountProfile(c.Request.Context())
	jiyingJSON(c, gin.H{
		"apps":     apps,
		"accounts": AccountPoolSnapshot(),
		"account": gin.H{
			"configured": acctConn.configured(),
			"code_login": acctConn.CodeLogin,
			"email":      firstNonEmpty(profile.Email, acct.Email),
			"nickname":   profile.Nickname,
			"avatar_url": profile.AvatarURL,
			"logged_in":  userLoggedIn(),
			"status":     UserSessionStatus(),
		},
	})
}

func connectionsAccountSelect(c *gin.Context) {
	account, err := SelectAccountConn(c.PostForm("email"))
	if err != nil {
		jiyingErr(c, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 20*time.Second)
	defer cancel()
	if token, _, ok := loadUserToken(account.Email); ok {
		SetUserSession(account.Email, token)
	} else if account.CodeLogin {
		jiyingErr(c, "该账号使用邮箱验证码登录，请重新获取验证码")
		return
	} else if err := loginAccount(ctx, UserLogin{BaseURL: account.BaseURL, Email: account.Email, Password: account.Secret}, false); err != nil {
		jiyingErr(c, "账号切换登录失败: "+err.Error())
		return
	}
	setActiveAccountEmail(account.Email)
	RecordAudit(consoleUser(c), "account.select", account.Email, "", "成功")
	jiyingJSON(c, gin.H{"email": account.Email})
}

func connectionsAccountDelete(c *gin.Context) {
	email := strings.TrimSpace(c.PostForm("email"))
	if err := RemoveAccountConn(email); err != nil {
		jiyingErr(c, err.Error())
		return
	}
	RecordAudit(consoleUser(c), "account.delete", email, "", "成功")
	jiyingJSON(c, "ok")
}

// connectionsAccountProfile 拉取个人账号资料以刷新头像与昵称。
func connectionsAccountProfile(c *gin.Context) {
	if !userLoggedIn() {
		jiyingErr(c, "账号通道未登录")
		return
	}
	profile := accountProfile(c.Request.Context())
	if profile.Email == "" {
		jiyingErr(c, "账号资料获取失败")
		return
	}
	jiyingJSON(c, profile)
}

func connectionsAppUpsert(c *gin.Context) {
	secret := c.PostForm("app_secret")
	appID := strings.TrimSpace(c.PostForm("app_id"))
	if appID != "" && !isUUID(appID) {
		RecordAudit(consoleUser(c), "app.upsert", appID, "", "失败: 应用 ID 不是 UUID")
		jiyingErr(c, "应用 ID 必须是 UUID（8-4-4-4-12），请从 MagicChat 应用详情复制")
		return
	}
	if secret == "****" {
		for _, a := range loadAppConns() {
			if a.AppID == appID {
				secret = a.AppSecret
				break
			}
		}
	}
	cfg := AppConn{
		WSURL:     resolveAppWSURL(c.PostForm("ws_url"), appID),
		AppID:     appID,
		AppSecret: secret,
	}
	if err := UpsertAppConn(cfg); err != nil {
		RecordAudit(consoleUser(c), "app.upsert", appID, "", "失败: "+err.Error())
		jiyingErr(c, err.Error())
		return
	}
	RecordAudit(consoleUser(c), "app.upsert", appID, "", "成功")
	jiyingJSON(c, "ok")
}

func connectionsAppDelete(c *gin.Context) {
	appID := strings.TrimSpace(c.PostForm("app_id"))
	if err := RemoveAppConn(appID); err != nil {
		RecordAudit(consoleUser(c), "app.delete", appID, "", "失败: "+err.Error())
		jiyingErr(c, err.Error())
		return
	}
	RecordAudit(consoleUser(c), "app.delete", appID, "", "成功")
	jiyingJSON(c, "ok")
}

func connectionsAppDefault(c *gin.Context) {
	appID := strings.TrimSpace(c.PostForm("app_id"))
	if err := SetDefaultAppConn(appID); err != nil {
		RecordAudit(consoleUser(c), "app.default", appID, "", "失败: "+err.Error())
		jiyingErr(c, err.Error())
		return
	}
	RecordAudit(consoleUser(c), "app.default", appID, "", "成功")
	jiyingJSON(c, gin.H{"default_app_id": appID})
}

// accountProfile 尽力读取个人账号资料，失败时返回零值而非报错。
func accountProfile(ctx context.Context) AccountProfile {
	if !userLoggedIn() {
		return AccountProfile{}
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	raw, err := UserCall(ctx, http.MethodGet, "/api/client/me", nil)
	if err != nil {
		return AccountProfile{}
	}
	return parseAccountProfile(raw)
}

// parseAccountProfile 解析 /api/client/me 响应；用户对象位于 data.user 一层。
func parseAccountProfile(raw []byte) AccountProfile {
	var parsed struct {
		Data struct {
			User struct {
				Email     string `json:"email"`
				Name      string `json:"name"`
				Nickname  string `json:"nickname"`
				Avatar    string `json:"avatar"`
				AvatarURL string `json:"avatar_url"`
			} `json:"user"`
		} `json:"data"`
	}
	if json.Unmarshal(raw, &parsed) != nil {
		return AccountProfile{}
	}
	user := parsed.Data.User
	return AccountProfile{
		Email:     user.Email,
		Nickname:  firstNonEmpty(user.Nickname, user.Name),
		AvatarURL: firstNonEmpty(user.Avatar, user.AvatarURL),
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
