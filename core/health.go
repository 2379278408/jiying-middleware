package core

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// registerHealthAPI 暴露匿名探针与指标；探针不返回任何凭据或业务数据。
func registerHealthAPI() {
	Server.GET("/healthz", healthz)
	Server.GET("/readyz", readyz)
	Server.GET("/metrics", metrics)
}

func healthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func readyz(c *gin.Context) {
	checks := map[string]bool{
		"storage":     BucketApp != nil,
		"task_queue":  taskLoopReady(),
		"app_channel": AppChannelConnected(),
	}
	ready := true
	for _, ok := range checks {
		if !ok {
			ready = false
		}
	}
	code := http.StatusOK
	status := "ready"
	if !ready {
		code = http.StatusServiceUnavailable
		status = "not_ready"
	}
	c.JSON(code, gin.H{"status": status, "checks": checks})
}

func metrics(c *gin.Context) {
	pending, running, done, failed := TaskCounts()
	statusCounts := TaskStatusCounts()
	enabled := 0
	for _, f := range ListFeatureItems() {
		if f != nil && f.Enabled {
			enabled++
		}
	}
	scripts := 0
	for _, f := range FunctionsSnapshot() {
		if f != nil && !f.Origin {
			scripts++
		}
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# HELP jiying_uptime_seconds 进程运行时长\n")
	fmt.Fprintf(&b, "# TYPE jiying_uptime_seconds gauge\n")
	fmt.Fprintf(&b, "jiying_uptime_seconds %d\n", int64(time.Since(StartedAt()).Seconds()))
	fmt.Fprintf(&b, "# HELP jiying_messages_total 已接收消息总数\n")
	fmt.Fprintf(&b, "# TYPE jiying_messages_total counter\n")
	_, _, _, msgTotal := StatsSnapshot()
	fmt.Fprintf(&b, "jiying_messages_total %d\n", msgTotal)
	fmt.Fprintf(&b, "# HELP jiying_tasks 任务队列计数\n")
	fmt.Fprintf(&b, "# TYPE jiying_tasks gauge\n")
	fmt.Fprintf(&b, "jiying_tasks{status=\"pending\"} %d\n", pending)
	fmt.Fprintf(&b, "jiying_tasks{status=\"running\"} %d\n", running)
	fmt.Fprintf(&b, "jiying_tasks{status=\"done\"} %d\n", done)
	fmt.Fprintf(&b, "jiying_tasks{status=\"failed\"} %d\n", failed)
	for _, status := range []string{taskWaitingConfirmation, taskCanceled, taskTimeout} {
		fmt.Fprintf(&b, "jiying_tasks{status=\"%s\"} %d\n", status, statusCounts[status])
	}
	writeTaskMetrics(&b)
	fmt.Fprintf(&b, "# HELP jiying_app_connections_configured 已配置应用连接数\n")
	fmt.Fprintf(&b, "# TYPE jiying_app_connections_configured gauge\n")
	fmt.Fprintf(&b, "jiying_app_connections_configured %d\n", len(loadAppConns()))
	fmt.Fprintf(&b, "# HELP jiying_features_enabled 已启用功能数\n")
	fmt.Fprintf(&b, "# TYPE jiying_features_enabled gauge\n")
	fmt.Fprintf(&b, "jiying_features_enabled %d\n", enabled)
	fmt.Fprintf(&b, "# HELP jiying_scripts_loaded 已加载脚本数\n")
	fmt.Fprintf(&b, "# TYPE jiying_scripts_loaded gauge\n")
	fmt.Fprintf(&b, "jiying_scripts_loaded %d\n", scripts)
	fmt.Fprintf(&b, "# HELP jiying_channel_up 通道状态 1 在线 0 离线\n")
	fmt.Fprintf(&b, "# TYPE jiying_channel_up gauge\n")
	fmt.Fprintf(&b, "jiying_channel_up{channel=\"app\"} %d\n", boolToFloat(AppChannelConnected()))
	fmt.Fprintf(&b, "jiying_channel_up{channel=\"account\"} %d\n", boolToFloat(strings.HasPrefix(UserSessionStatus(), "账号已登录")))
	fmt.Fprintf(&b, "# HELP jiying_ratelimit_rejected_total 因限流或配额被拒绝的任务数\n")
	fmt.Fprintf(&b, "# TYPE jiying_ratelimit_rejected_total counter\n")
	fmt.Fprintf(&b, "jiying_ratelimit_rejected_total %d\n", RateLimitRejected())
	fmt.Fprintf(&b, "# HELP jiying_app_ratelimited_total 应用通道被服务端限流(rate_limited)的调用次数\n")
	fmt.Fprintf(&b, "# TYPE jiying_app_ratelimited_total counter\n")
	fmt.Fprintf(&b, "jiying_app_ratelimited_total %d\n", JiyingRateLimitedTotal())
	calls := JiyingCallStats()
	if len(calls) > 0 {
		fmt.Fprintf(&b, "# HELP jiying_app_calls_total 应用通道 RPC 调用次数\n")
		fmt.Fprintf(&b, "# TYPE jiying_app_calls_total counter\n")
		for _, st := range calls {
			fmt.Fprintf(&b, "jiying_app_calls_total{method=%q,outcome=\"ok\"} %d\n", st.Method, st.OK)
			fmt.Fprintf(&b, "jiying_app_calls_total{method=%q,outcome=\"error\"} %d\n", st.Method, st.Failed)
		}
		fmt.Fprintf(&b, "# HELP jiying_app_call_errors_total 应用通道 RPC 失败次数，按错误码拆分\n")
		fmt.Fprintf(&b, "# TYPE jiying_app_call_errors_total counter\n")
		for _, st := range calls {
			codes := make([]string, 0, len(st.Codes))
			for code := range st.Codes {
				codes = append(codes, code)
			}
			sort.Strings(codes)
			for _, code := range codes {
				fmt.Fprintf(&b, "jiying_app_call_errors_total{method=%q,code=%q} %d\n", st.Method, code, st.Codes[code])
			}
		}
		fmt.Fprintf(&b, "# HELP jiying_app_call_duration_milliseconds_avg 应用通道 RPC 平均耗时（毫秒）\n")
		fmt.Fprintf(&b, "# TYPE jiying_app_call_duration_milliseconds_avg gauge\n")
		for _, st := range calls {
			fmt.Fprintf(&b, "jiying_app_call_duration_milliseconds_avg{method=%q} %.3f\n", st.Method, st.AvgMs)
		}
	}
	c.Data(http.StatusOK, "text/plain; version=0.0.4; charset=utf-8", []byte(b.String()))
}

func boolToFloat(v bool) int {
	if v {
		return 1
	}
	return 0
}
