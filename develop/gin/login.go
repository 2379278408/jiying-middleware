package gin

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"jiying/middleware/core"
	"jiying/middleware/develop/users"
	"jiying/middleware/utils"
)

const (
	cookieName = "jiying_session"
)

func registerOpen() bool {
	return core.BucketApp.GetBool("register_open", "web")
}

func IssueUserSession(c *gin.Context, u *users.User) {
	s, err := core.SessionStore.Get(c.Request, cookieName)
	if err != nil {
		return
	}
	s.Values["user"] = u.ID
	if remember, _ := strconv.ParseBool(c.PostForm("rememberme")); remember {
		s.Options.MaxAge = 24 * 3600 * 30
	} else {
		s.Options.MaxAge = 24 * 3600 * 7
	}
	_ = s.Save(c.Request, c.Writer)
}

func CurrentUser(c *gin.Context) *users.User {
	s, err := core.SessionStore.Get(c.Request, cookieName)
	if err != nil {
		return nil
	}
	id, ok := s.Values["user"].(string)
	if !ok || id == "" {
		return nil
	}
	return users.GetUser(id)
}

func jsonOK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": data})
}

func jsonErr(c *gin.Context, msg string) {
	c.JSON(http.StatusOK, gin.H{"code": 1, "message": msg, "msg": msg})
}

func PostLogin(c *gin.Context) {
	username := strings.TrimSpace(c.PostForm("username"))
	password := c.PostForm("password")
	if username == "" || password == "" {
		jsonErr(c, "用户名或密码不能为空")
		return
	}
	// 以“来源 + 用户名”为键，避免共享出口 IP 下不同账号相互牵连，
	// 同时仍能限制针对同一账号的密码猜测。
	guardKey := utils.GetClientIP(c) + "|" + strings.ToLower(username)
	if core.LoginBlocked(guardKey) {
		core.GetMyLog().Warn("登录已被临时锁定 %s", guardKey)
		core.RecordAudit(username, "login.blocked", username, "来源 "+utils.GetClientIP(c), "失败: 账号临时锁定")
		jsonErr(c, "失败次数过多，账号已临时锁定，请约 5 分钟后重试")
		return
	}
	u := users.VerifyPassword(username, password)
	if u == nil {
		core.NoteLoginFailure(guardKey)
		core.GetMyLog().Warn("登录失败 %s，%s", username, utils.GetClientIP(c))
		core.RecordAudit(username, "login.fail", username, "来源 "+utils.GetClientIP(c), "失败: 用户名或密码错误")
		jsonErr(c, "用户名或密码错误")
		return
	}
	core.ResetLoginFailures(guardKey)
	IssueUserSession(c, u)
	// 管理员已取回一次性引导口令，登录成功后清理该文件。
	users.ClearInitialPasswordFile()
	core.GetMyLog().Info("用户 %s 登录成功 %s", username, utils.GetClientIP(c))
	core.RecordAudit(username, "login.success", username, "来源 "+utils.GetClientIP(c), "成功")
	jsonOK(c, nil)
}

func GetLogin(c *gin.Context) {
	if CurrentUser(c) != nil {
		jsonOK(c, gin.H{"user": CurrentUser(c).ID})
		return
	}
	clearSessionCookie(c)
	c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未登录"})
}

func PostLogout(c *gin.Context) {
	s, err := core.SessionStore.Get(c.Request, cookieName)
	if err == nil {
		s.Values = map[interface{}]interface{}{}
		s.Options.MaxAge = -1
		_ = s.Save(c.Request, c.Writer)
	}
	c.JSON(http.StatusOK, gin.H{"code": 200})
}

func GetICPP(c *gin.Context) {
	icp := core.BucketApp.GetString("icp")
	if icp == "" {
		icp = core.BucketApp.GetString("filing", "icp")
	}
	jsonOK(c, gin.H{
		"icp":            icp,
		"cloud_username": "",
		"machine_id":     "",
	})
}

func SendCode2(c *gin.Context) {
	username := strings.TrimSpace(c.PostForm("username"))
	if username == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "请输入用户名"})
		return
	}
	u := users.GetUser(username)
	if u == nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "用户不存在"})
		return
	}
	if u.Email == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "该用户未绑定邮箱"})
		return
	}
	code := genVerifyCode()
	s, err := core.SessionStore.Get(c.Request, cookieName)
	if err == nil {
		s.Values["verify_code"] = code
		s.Values["recover_user"] = u.ID
		_ = s.Save(c.Request, c.Writer)
	}
	// 不回写验证码明文：日志可见即等于任意账号可被重置。
	core.GetMyLog().Info("已向 %s 发送找回验证码", u.Email)
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "验证码已发送"})
}

func ForgetPassword(c *gin.Context) {
	code := strings.TrimSpace(c.PostForm("code"))
	password := c.PostForm("password")
	if code == "" || password == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "请填写验证码与新密码"})
		return
	}
	// 验证码只有 6 位，必须限速，避免在线暴力枚举。
	guardKey := "reset|" + utils.GetClientIP(c)
	if core.LoginBlocked(guardKey) {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "尝试次数过多，请约 5 分钟后重试"})
		return
	}
	s, err := core.SessionStore.Get(c.Request, cookieName)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "会话异常"})
		return
	}
	got, _ := s.Values["verify_code"].(string)
	if got == "" || got != code {
		core.NoteLoginFailure(guardKey)
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "验证码错误"})
		return
	}
	targetID, _ := s.Values["recover_user"].(string)
	for k := range s.Values {
		if k == "verify_code" || k == "recover_user" {
			delete(s.Values, k)
		}
	}
	_ = s.Save(c.Request, c.Writer)

	target := users.GetUser(targetID)
	if target == nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "用户异常"})
		return
	}
	if err := target.SetPassword(password); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "密码设置失败"})
		return
	}
	if err := target.Save(); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "保存失败"})
		return
	}
	core.ResetLoginFailures(guardKey)
	core.GetMyLog().Info("用户 %s 通过验证码重置密码", target.ID)
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "密码已重置"})
}

func genVerifyCode() string {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil || n == nil {
		return fmt.Sprintf("%06d", time.Now().UnixNano()%1000000)
	}
	return fmt.Sprintf("%06d", n.Int64())
}

func clearSessionCookie(c *gin.Context) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		if CurrentUser(c) == nil {
			clearSessionCookie(c)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "请先登录"})
			return
		}
		c.Next()
	}
}

// requireConsoleMaster 仅允许控制台主控角色访问遗留的高权限接口（如任意桶读取）。
func requireConsoleMaster() gin.HandlerFunc {
	return requireConsoleRole(core.RoleMaster)
}

// requireConsoleRole 仅允许达到指定控制台角色等级的账号访问遗留接口。
func requireConsoleRole(min string) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := CurrentUser(c)
		if u == nil || !core.HasRole(u.ID, min) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"code": 403, "message": "权限不足"})
			return
		}
		c.Next()
	}
}

func CheckSession(c *gin.Context) bool {
	return CurrentUser(c) != nil
}
