package core

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	"jiying/middleware/utils"
)

type AutWatcher struct {
	watcher *fsnotify.Watcher
	once    sync.Once
}

func NewAutWatcher() *AutWatcher {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		GetMyLog().Error("创建目录监控器失败: %v", err)
		return nil
	}
	return &AutWatcher{watcher: w}
}

var watchExts = []string{".js", ".py", ".ts", ".mjs"}

func (a *AutWatcher) AddWatchDir(dir string) {
	if a == nil || a.watcher == nil {
		return
	}
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil
			}
			return err
		}
		if !info.IsDir() {
			return nil
		}
		if isExcludeDir(path) {
			return filepath.SkipDir
		}
		abs, abserr := filepath.Abs(path)
		if abserr != nil {
			return nil
		}
		if err := a.watcher.Add(abs); err != nil {
			GetMyLog().Warn("监控目录添加失败 %s: %v", abs, err)
		}
		return nil
	})
}

func (a *AutWatcher) WatchFileEvents(handle func(string)) {
	if a == nil || a.watcher == nil {
		return
	}
	for {
		select {
		case ev, ok := <-a.watcher.Events:
			if !ok {
				return
			}
			if isExcludeDir(ev.Name) {
				continue
			}
			if !hasWatchExt(ev.Name) {
				continue
			}
			if ev.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename|fsnotify.Remove) != 0 {
				GetMyLog().Debug("watch event %s", ev.String())
				handle(ev.Name)
			}
		case err, ok := <-a.watcher.Errors:
			if !ok {
				return
			}
			GetMyLog().Warn("目录监控错误: %v", err)
		}
	}
}

func isExcludeDir(path string) bool {
	for _, seg := range strings.Split(path, string(filepath.Separator)) {
		if strings.HasPrefix(seg, ".") || seg == "cache" || seg == "node_modules" || seg == "@tmpfs" {
			return true
		}
	}
	return false
}

func hasWatchExt(p string) bool {
	ext := strings.ToLower(filepath.Ext(p))
	return utils.Contains(watchExts, ext)
}
