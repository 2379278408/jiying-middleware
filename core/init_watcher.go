package core

import (
	"os"
	"path/filepath"

	"github.com/robfig/cron/v3"
	"jiying/middleware/utils"
)

func initWatcher() {
	wMain := NewAutWatcher()
	wExt := NewAutWatcher()

	dirs := []string{
		filepath.Join(utils.GetResHome(), "plugin"),
		filepath.Join(utils.GetResHome(), "plugin", "scripts"),
		filepath.Join(utils.GetResHome(), "plugin", "encrypted_ext"),
		filepath.Join(utils.GetResHome(), "data"),
	}
	for _, dir := range dirs {
		mode := os.FileMode(0o750)
		if filepath.Base(dir) == "data" {
			mode = 0o700
		}
		if mkErr := os.MkdirAll(dir, mode); mkErr != nil {
			GetMyLog().Error("创建监控目录失败 %s: %v", dir, mkErr)
			continue
		}
		if chErr := os.Chmod(dir, mode); chErr != nil {
			GetMyLog().Error("收紧监控目录权限失败 %s: %v", dir, chErr)
		}
		GetMyLog().Warn("监控目录就绪 %s", dir)
	}

	if wMain != nil {
		wMain.AddWatchDir(dirs[0])
		wMain.AddWatchDir(dirs[1])
		watchOnce(wMain)
	}
	if wExt != nil {
		wExt.AddWatchDir(dirs[2])
		wExt.AddWatchDir(dirs[3])
		watchOnce(wExt)
	}
	select {}
}

func watchOnce(w *AutWatcher) {
	go w.WatchFileEvents(func(path string) {
		onWatchEvent(path)
	})
}

func onWatchEvent(path string) {
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		return
	}
	if scriptLangOf(path) == "" {
		GetMyLog().Debug("watch 忽略非脚本文件 %s", path)
		return
	}
	reloadScript(path)
}

func reloadScript(path string) {
	lang := scriptLangOf(path)
	if lang == "" {
		return
	}
	removed := false
	funcLock.Lock()
	kept := Functions[:0]
	for _, f := range Functions {
		if f != nil && f.PluginPath == path {
			if f.CronId > 0 && ScriptCron != nil {
				ScriptCron.Remove(cron.EntryID(f.CronId))
			}
			removed = true
			continue
		}
		kept = append(kept, f)
	}
	Functions = kept
	funcLock.Unlock()

	data, err := os.ReadFile(path)
	if err != nil {
		if removed {
			GetMyLog().Warn("脚本移除 %s", path)
		} else {
			GetMyLog().Debug("重载跳过不可读文件 %s: %v", path, err)
		}
		return
	}
	fn := &Function{
		PluginPath:    path,
		PluginContent: string(data),
		PluginLang:    lang,
		Title:         scriptTitle(path),
		Origin:        false,
	}
	appendFunction(fn)
	if removed {
		GetMyLog().Warn("脚本已更新重载 %s", path)
	} else {
		GetMyLog().Warn("脚本新增加载 %s", path)
	}
	_ = runScript(fn)
}
