package core

import (
	"os"
	"path/filepath"
	"syscall"

	"jiying/middleware/utils"
)

// RequestRestart 请求进程原地重启；供 main 的异常监控复用。
func RequestRestart(reason string) { reboot(reason) }

func reboot(reason string) {
	GetMyLog().Warn("重启原因: %s", reason)
	log := GetMyLog()
	log.Sync()
	// 优先用 exec 替换当前进程，避免旧进程继续存活导致双实例与端口冲突。
	if exe := utils.GetExecPath(); exe != "" {
		argv := append([]string{exe}, os.Args[1:]...)
		if err := syscall.Exec(exe, argv, os.Environ()); err == nil {
			return
		} else {
			GetMyLog().Error("进程原地重启失败: %v", err)
		}
	}
	os.Exit(1)
}

func initUpdate() {
	registerSysCommand("检查更新", "check_update", 5, func() {
		GetMyLog().Info("在线更新（本重建版为离线交付，跳过网络检查）")
	})
}

func initPluginsCmd() {
	registerSysCommand("插件管理", "plugins_manage", 5, func() {
		listPlugins()
	})
}

func listPlugins() {
	dir := filepath.Join(utils.GetResHome(), "scripts")
	entries, err := os.ReadDir(dir)
	if err != nil {
		GetMyLog().Warn("插件目录不可用 %s: %v", dir, err)
		return
	}
	n := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		n++
	}
	GetMyLog().Info("已加载插件 %d 个（scripts/）：%s", n, dir)
}
