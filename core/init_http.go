package core

import (
	"net"
	"net/http"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

var httpLocalOnce sync.Once

func initHttpServer() {
	if Server == nil {
		return
	}
	httpLocalOnce.Do(func() {
		if Server == nil {
			GetMyLog().Warn("initHttpServer: Server 为空, 本地路由未注册")
			return
		}
		local := Server.Group("/local")
		local.Use(localOnly())
		local.POST("/msg", httpMsgHook)
		GetMyLog().Warn("本地 HTTP 管理路由 /local/msg 已注册")
	})
}

func localOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !isLoopbackRemote(c.Request.RemoteAddr) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"code": 403, "message": "仅限本机访问"})
			return
		}
		c.Next()
	}
}

// isLoopbackRemote 判断 TCP 直连对端是否回环。只信任连接来源地址，
// 不使用 c.ClientIP()——后者在默认信任所有代理时会采信 X-Forwarded-For，可被伪造绕过本机限制。
func isLoopbackRemote(remoteAddr string) bool {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	ip := net.ParseIP(strings.Trim(host, "[]"))
	return ip != nil && ip.IsLoopback()
}

func httpMsgHook(c *gin.Context) {
	msg := c.PostForm("msg")
	if strings.TrimSpace(msg) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "缺少 msg"})
		return
	}
	// 身份取真实对端地址，避免用可伪造的 X-Forwarded-For 冒充来源。
	peer, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err != nil {
		peer = c.Request.RemoteAddr
	}
	f := &Faker{
		Message: msg,
		Type:    2,
		UserID:  peer,
		ChatID:  peer,
	}
	select {
	case Senders <- f:
		c.JSON(http.StatusOK, gin.H{"code": 200})
	default:
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 1, "message": "消息通道已满"})
	}
}

func sockCmdHandler(c *gin.Context) {
	cmd := strings.TrimSpace(c.PostForm("cmd"))
	if cmd == "" {
		cmd = c.Param("cmd")
	}
	if cmd == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "缺少 cmd"})
		return
	}
	resp := dispatchLocalCommand(cmd)
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": resp})
}

func dispatchLocalCommand(cmd string) string {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return "空命令"
	}
	handle := parts[0]
	handler, ok := cmdRouter.Load(handle)
	if !ok {
		return "命令未注册: " + handle
	}
	fn, ok := handler.(func())
	if !ok {
		return "命令处理器类型错误: " + handle
	}
	fc := &Faker{
		Message: cmd,
		Type:    2,
		UserID:  "system",
		ChatID:  "local",
	}
	fc.FromGroupID = "local"
	withSender(fc, fn)
	return "命令已执行: " + handle
}

func dispatchChatCommand(chatID, cmd string) string {
	s := &Faker{Message: cmd, Type: 2, UserID: "system", ChatID: chatID}
	s.FromGroupID = chatID
	withSender(s, func() {
		if handled, _ := dispatchMiddleware(s, s.GetMsg()); handled {
			return
		}
		_ = HandleMessage(s)
	})
	return "命令已派发: " + cmd
}
