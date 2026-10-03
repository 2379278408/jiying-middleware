package core

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"jiying/middleware/utils"
)

var cmdRouter = sync.Map{}

func registerSysCommand(title, handle string, priority int, fn func()) {
	registerSysCommandWithAdmin(title, handle, priority, false, fn)
}

// registerAdminSysCommand 注册仅限主控触发的系统命令，用于重启、日志、审计等敏感操作。
func registerAdminSysCommand(title, handle string, priority int, fn func()) {
	registerSysCommandWithAdmin(title, handle, priority, true, fn)
}

func registerSysCommandWithAdmin(title, handle string, priority int, admin bool, fn func()) {
	cmdRouter.Store(handle, fn)
	_ = AddCommand(0, &Function{
		Title:    title,
		Handle:   handle,
		Priority: priority,
		Origin:   true,
		Admin:    admin,
	})
}

func initSys() {
	registerAdminSysCommand("重启运行环境", "restart", 10, sysCmdRestart)
	registerAdminSysCommand("升级运行环境", "upgrade", 10, sysCmdUpgrade)
	registerAdminSysCommand("查看系统状态", "status", 10, sysCmdStatus)
	registerAdminSysCommand("运行日志", "logs", 10, sysCmdLogs)
	registerAdminSysCommand("查看统计", "statistics", 10, sysCmdStatistics)
	registerAdminSysCommand("命令帮助", "help", 10, sysCmdHelp)
	registerSysCommand("消息测试", "echo", 10, sysCmdEcho)
	registerAdminSysCommand("用户管理", "users", 10, sysCmdUsers)
	registerAdminSysCommand("插件管理", "plugins", 10, sysCmdPlugins)
	registerAdminSysCommand("定时任务", "cron", 10, sysCmdCron)
	registerSysCommand("版本信息", "version", 10, sysCmdVersion)
	registerSysCommand("心跳检测", "ping", 10, sysCmdPing)
	registerSysCommand("关于", "about", 10, sysCmdAbout)
	registerAdminSysCommand("审计日志", "audit", 10, sysCmdAudit)
}

func sysCmdRestart()    { reboot("命令触发重启") }
func sysCmdUpgrade()    { ReplyCurrent("升级需在本机执行") }
func sysCmdStatus()     { reportStatus() }
func sysCmdLogs()       { showLogsHelper() }
func sysCmdStatistics() { reportStatistics() }
func sysCmdHelp()       { listSysCommands() }
func sysCmdEcho()       { GetMyLog().Info("消息测试 OK") }
func sysCmdUsers()      { GetMyLog().Info("用户管理（Web 端 /users 处理）") }
func sysCmdPlugins()    { GetMyLog().Info("插件管理（Web 端插件页处理）") }
func sysCmdCron()       { GetMyLog().Info("定时任务（Web 端 /croncmds 处理）") }
func sysCmdVersion()    { reportVersion() }
func sysCmdPing()       { ReplyCurrent("pong") }
func sysCmdAbout()      { reportVersion() }
func sysCmdAudit()      { ReplyCurrent(AuditLogSummary(20)) }

func reportStatus() {
	status := fmt.Sprintf("运行环境 %s | Goroutine %d | 数据目录 %s | %s",
		utils.GetExecPathDir(), runtime.NumGoroutine(), utils.GetDataHome(), jiyingState())
	ReplyCurrent(status)
}

func reportStatistics() {
	buf := strings.Builder{}
	buf.WriteString("系统统计: ")
	buf.WriteString(fmt.Sprintf("命令 %d | goroutine %d | %s",
		len(FunctionsSnapshot()), runtime.NumGoroutine(), runtime.Version()))
	GetMyLog().Info("%s", buf.String())
}

func reportVersion() {
	GetMyLog().Info("即应中间件(%s/%s) | %s", runtime.GOOS, runtime.GOARCH, runtime.Version())
}

func listSysCommands() {
	funcs := FunctionsSnapshot()
	titles := make([]string, 0, len(funcs))
	for _, f := range funcs {
		if f != nil && f.Origin {
			titles = append(titles, f.Title)
		}
	}
	sort.Strings(titles)
	ReplyCurrent("系统命令: " + strings.Join(titles, "、"))
}

func showLogsHelper() {
	dir := filepath.Join(utils.GetDataHome(), "logs")
	entries, err := os.ReadDir(dir)
	if err != nil {
		ReplyCurrent("日志目录不可用 " + dir)
		return
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)
	ReplyCurrent(fmt.Sprintf("日志文件: %s（最新见 logs/%s）", strings.Join(names, ", "),
		time.Now().Format("20060102")+".log"))
}
