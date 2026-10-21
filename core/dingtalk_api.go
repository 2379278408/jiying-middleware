package core

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// registerDingTalkAPI 注册钉钉连接器配置、状态与事件接口。
func registerDingTalkAPI() {
	g := Server.Group("/jiying/api", requireLogin)
	g.GET("/dingtalk/status", requireRole(RoleOps), dingTalkAPIStatus)
	g.GET("/dingtalk/events", requireRole(RoleOps), dingTalkAPIEvents)
	g.POST("/dingtalk/config", requireRole(RoleMaster), dingTalkAPIConfig)
	g.POST("/dingtalk/start", requireRole(RoleMaster), dingTalkAPIStart)
	g.POST("/dingtalk/stop", requireRole(RoleMaster), dingTalkAPIStop)
	g.POST("/dingtalk/probe", requireRole(RoleMaster), dingTalkAPIProbe)
	g.POST("/dingtalk/tool", requireRole(RoleMaster), dingTalkAPITool)
}

func dingTalkAPIStatus(c *gin.Context) {
	jiyingJSON(c, dingTalkStatusSnapshot(50))
}

func dingTalkAPIEvents(c *gin.Context) {
	limit, _ := strconv.Atoi(c.Query("limit"))
	jiyingJSON(c, gin.H{"events": dingTalkEventsSnapshot(limit)})
}

func dingTalkAPIConfig(c *gin.Context) {
	old := loadDingTalkConfig()
	cfg := loadDingTalkConfig()
	cfg.Enabled = c.PostForm("enabled") == "true" || c.PostForm("enabled") == "1"
	cfg.AutoPush = c.PostForm("auto_push") == "true" || c.PostForm("auto_push") == "1"
	if v := strings.TrimSpace(c.PostForm("bin")); v != "" {
		cfg.Bin = v
	}
	if v, ok := c.GetPostForm("push_app_id"); ok {
		cfg.PushAppID = strings.TrimSpace(v)
	}
	if v, ok := c.GetPostForm("push_conversation_id"); ok {
		cfg.PushConversationID = strings.TrimSpace(v)
	}
	if err := saveDingTalkConfig(cfg); err != nil {
		RecordAudit(consoleUser(c), "dingtalk.config", cfg.Bin, "", "失败: "+err.Error())
		jiyingErr(c, err.Error())
		return
	}
	if cfg.Enabled {
		if old.Enabled && strings.TrimSpace(old.Bin) != strings.TrimSpace(cfg.Bin) {
			stopDingTalkConnector()
		}
		if err := startDingTalkConnector(); err != nil {
			RecordAudit(consoleUser(c), "dingtalk.config", cfg.Bin, "", "失败: "+err.Error())
			jiyingErr(c, err.Error())
			return
		}
	} else {
		stopDingTalkConnector()
	}
	RecordAudit(consoleUser(c), "dingtalk.config", cfg.Bin, "enabled="+strconv.FormatBool(cfg.Enabled), "成功")
	jiyingJSON(c, dingTalkStatusSnapshot(50))
}

func dingTalkAPIStart(c *gin.Context) {
	cfg := loadDingTalkConfig()
	cfg.Enabled = true
	if err := saveDingTalkConfig(cfg); err != nil {
		jiyingErr(c, err.Error())
		return
	}
	if err := startDingTalkConnector(); err != nil {
		jiyingErr(c, err.Error())
		return
	}
	RecordAudit(consoleUser(c), "dingtalk.start", cfg.Bin, "", "成功")
	jiyingJSON(c, dingTalkStatusSnapshot(50))
}

func dingTalkAPIStop(c *gin.Context) {
	cfg := loadDingTalkConfig()
	cfg.Enabled = false
	if err := saveDingTalkConfig(cfg); err != nil {
		jiyingErr(c, err.Error())
		return
	}
	stopDingTalkConnector()
	RecordAudit(consoleUser(c), "dingtalk.stop", cfg.Bin, "", "成功")
	jiyingJSON(c, "ok")
}

func dingTalkAPIProbe(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 20*time.Second)
	defer cancel()
	out, err := runDwsCommand(ctx, "auth", "status")
	if err != nil {
		jiyingErr(c, err.Error())
		return
	}
	jiyingJSON(c, gin.H{"auth": out})
}

func dingTalkAPITool(c *gin.Context) {
	name := strings.TrimSpace(c.PostForm("name"))
	args := strings.TrimSpace(c.PostForm("args"))
	if name == "" {
		jiyingErr(c, "缺少工具名")
		return
	}
	if args == "" {
		args = "{}"
	}
	if !json.Valid([]byte(args)) {
		jiyingErr(c, "参数必须是 JSON 对象")
		return
	}
	out := runDingTalkTool(name, args)
	if strings.HasPrefix(out, "FAILED:") {
		RecordAudit(consoleUser(c), "dingtalk.tool", name, args, "失败: "+strings.TrimPrefix(out, "FAILED: "))
		jiyingErr(c, strings.TrimSpace(strings.TrimPrefix(out, "FAILED:")))
		return
	}
	RecordAudit(consoleUser(c), "dingtalk.tool", name, args, "成功")
	jiyingJSON(c, gin.H{"result": out})
}
