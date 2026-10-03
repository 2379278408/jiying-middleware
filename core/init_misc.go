package core

import (
	"os"
	"path/filepath"
	"time"

	"jiying/middleware/utils"
)

func reboot(reason string) {
	GetMyLog().Warn("重启原因: %s", reason)
	utils.Daemon(0)
	time.Sleep(500 * time.Millisecond)
	os.Exit(0)
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
