package core

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

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
	// 浏览器发起的跨站表单会携带 Origin/Referer/Sec-Fetch-*，
	// 本地命令行工具不会；据此阻断浏览器借回环地址触发命令。
	if c.GetHeader("Origin") != "" || c.GetHeader("Referer") != "" ||
		c.GetHeader("Sec-Fetch-Site") != "" || c.GetHeader("Sec-Fetch-Mode") != "" {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"code": 403, "message": "拒绝浏览器上下文请求"})
		return
	}
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
	out, err := dispatchLocalCommandFor(cmd, "system")
	if err != nil {
		return err.Error()
	}
	return out
}

// cronFeatureCommand 描述定时任务中以 JSON 声明的功能调用。
type cronFeatureCommand struct {
	Type           string `json:"type"`
	Command        string `json:"command"`
	Feature        string `json:"feature"`
	Args           string `json:"args"`
	ConversationID string `json:"conversation_id"`
}

func parseCronFeatureCommand(raw string) (cronFeatureCommand, bool, error) {
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(raw, "{") {
		return cronFeatureCommand{}, false, nil
	}
	var spec cronFeatureCommand
	if err := json.Unmarshal([]byte(raw), &spec); err != nil {
		return spec, true, fmt.Errorf("命令 JSON 解析失败: %v", err)
	}
	if spec.Type == "" {
		spec.Type = "feature"
	}
	if spec.Type != "feature" && spec.Type != "command" {
		return spec, true, fmt.Errorf("不支持的命令类型: %s", spec.Type)
	}
	return spec, true, nil
}

// validateCronFeatureCommand 校验定时命令中的功能存在性与会话 ID 合法性。
// 纯系统命令与命令包装不做功能存在性检查，交由执行阶段按注册表校验。
func validateCronFeatureCommand(command string) error {
	parsed, isJSON, err := parseCronFeatureCommand(command)
	if err != nil {
		return err
	}
	if !isJSON || parsed.Type != "feature" {
		return nil
	}
	name := strings.TrimSpace(parsed.Feature)
	if name == "" {
		return fmt.Errorf("缺少功能名")
	}
	if _, ok := GetFeature(name); !ok {
		return fmt.Errorf("功能不存在: %s", name)
	}
	if conv := strings.TrimSpace(parsed.ConversationID); conv != "" && !validConversationID(conv) {
		return fmt.Errorf("会话 ID 不合法")
	}
	return nil
}

// dispatchLocalCommandFor 执行系统命令或 JSON 功能命令，并按 userID 校验敏感命令权限。
// 纯命令名维持原有系统命令语义，供 cron 与 socket 调用方复用。
func dispatchLocalCommandFor(cmd, userID string) (string, error) {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return "", fmt.Errorf("空命令")
	}
	spec, isJSON, err := parseCronFeatureCommand(cmd)
	if err != nil {
		return "", err
	}
	if isJSON && spec.Type == "feature" {
		name := strings.TrimSpace(spec.Feature)
		if name == "" {
			return "", fmt.Errorf("缺少功能名")
		}
		convID := strings.TrimSpace(spec.ConversationID)
		if convID != "" && !validConversationID(convID) {
			return "", fmt.Errorf("会话 ID 不合法")
		}
		if userID == "" {
			userID = "system"
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		out, err := executeFeature(ctx, name, spec.Args, userID, convID)
		if err != nil {
			return out, err
		}
		return out, nil
	}
	if isJSON && spec.Type == "command" {
		cmd = strings.TrimSpace(spec.Command)
		if cmd == "" {
			return "", fmt.Errorf("缺少命令")
		}
	}
	parts := strings.Fields(cmd)
	handle := parts[0]
	handler, ok := cmdRouter.Load(handle)
	if !ok {
		return "", fmt.Errorf("命令未注册: %s", handle)
	}
	if f := functionByHandle(handle); f != nil && f.Admin && !privilegedUser(userID) {
		return "", fmt.Errorf("权限不足: %s", handle)
	}
	fn, ok := handler.(func())
	if !ok {
		return "", fmt.Errorf("命令处理器类型错误: %s", handle)
	}
	fc := &Faker{
		Message: cmd,
		Type:    2,
		UserID:  userID,
		ChatID:  "local",
	}
	fc.FromGroupID = "local"
	withSender(fc, fn)
	return "命令已执行: " + handle, nil
}

func dispatchChatCommand(chatID, cmd, userID string) string {
	if strings.TrimSpace(userID) == "" {
		userID = "system"
	}
	s := &Faker{Message: cmd, Type: 2, UserID: userID, ChatID: chatID}
	s.FromGroupID = chatID
	withSender(s, func() {
		if handled, _ := dispatchMiddleware(s, s.GetMsg()); handled {
			return
		}
		_ = HandleMessage(s)
	})
	return "命令已派发: " + cmd
}
