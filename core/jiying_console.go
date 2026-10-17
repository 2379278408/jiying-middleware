package core

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func init() {
	registerJiyingConsole()
	registerJiyingAPI()
	registerConnectAPI()
	registerIconAPI()
	registerHealthAPI()
	registerAuditAPI()
	registerRolesAPI()
	registerConnectionsAPI()
	registerAPIKeysAPI()
	registerAIDocs()
	registerSkillsAPI()
	registerAgentAPI()
	registerMarketAPI()
}

func registerJiyingConsole() {
	Server.GET("/jiying/ping", requireLogin, jiyingConsolePing)
	Server.GET("/jiying/overview", requireLogin, jiyingConsoleOverview)
	Server.GET("/jiying/status", requireLogin, jiyingConsoleStatus)
}

func jiyingConsolePing(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"ws":      JiyingState(),
		"account": JiyingAccountStatus(),
	})
}

func jiyingConsoleOverview(c *gin.Context) {
	features := []string{}
	for _, s := range strings.Split(ListFeatures(), "\n") {
		if v := strings.TrimSpace(s); v != "" {
			features = append(features, v)
		}
	}
	crons := []string{}
	for _, s := range strings.Split(ListCronTitles(), "\n") {
		if v := strings.TrimSpace(s); v != "" {
			crons = append(crons, v)
		}
	}
	tasks := []string{}
	for _, s := range strings.Split(ListTasks(30), "\n") {
		if v := strings.TrimSpace(s); v != "" {
			tasks = append(tasks, v)
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"ws":       JiyingState(),
		"account":  JiyingAccountStatus(),
		"uptime":   JiyingUptime(),
		"features": features,
		"crons":    crons,
		"tasks":    tasks,
		"time":     time.Now().Format("2006-01-02 15:04:05"),
	})
}

func jiyingConsoleStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"features": ListFeatures(),
			"tasks":    ListTasks(20),
			"user":     JiyingAccountStatus(),
		},
	})
}
