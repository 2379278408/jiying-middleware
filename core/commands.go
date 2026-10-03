package core

import (
	"sort"
	"sync"

	"github.com/robfig/cron/v3"
)

var (
	funcLock sync.RWMutex
)

// appendFunction 在锁保护下向命令表追加脚本命令。
func appendFunction(f *Function) {
	if f == nil {
		return
	}
	funcLock.Lock()
	Functions = append(Functions, f)
	funcLock.Unlock()
}

func AddCommand(version byte, fs ...*Function) error {
	funcLock.Lock()
	defer funcLock.Unlock()

	for _, f := range fs {
		if f == nil {
			continue
		}
		if f.Title == "" {
			GetMyLog().Warn("命令标题为空，跳过注册")
			continue
		}
		if hasFunctionByTitle(f.Title) {
			GetMyLog().Warn("命令 %s 已存在，跳过", f.Title)
			continue
		}
		if version != 0 && f.Priority == 0 {
			f.Priority = int(version)
		}
		f.Origin = true
		if f.Cron != "" && ScriptCron != nil {
			id, err := ScriptCron.AddFunc(f.Cron, func() {
				runCronCommand(f)
			})
			if err != nil {
				GetMyLog().Warn("命令 %s 挂载定时失败: %v", f.Title, err)
			} else {
				f.CronId = int(id)
			}
		}
		Functions = append(Functions, f)
	}
	sortFunctionsByPriority()
	return nil
}

func AddOriginCommand(f *Function) {
	if f == nil {
		return
	}
	_ = AddCommand(0, f)
}

func hasFunctionByTitle(title string) bool {
	for _, f := range Functions {
		if f == nil {
			continue
		}
		if f.Title == title {
			return true
		}
	}
	return false
}

func sortFunctionsByPriority() {
	sort.SliceStable(Functions, func(i, j int) bool {
		li, lj := Functions[i], Functions[j]
		if li == nil || lj == nil {
			return false
		}
		return li.Priority > lj.Priority
	})
}

func runCronCommand(f *Function) {
	defer func() {
		if r := recover(); r != nil {
			GetMyLog().Warn("定时命令 %s 执行异常: %v", f.Title, r)
		}
	}()
	if f == nil || f.Handle == "" {
		return
	}
	GetMyLog().Info("定时执行命令 %s", f.Title)
	if handler, ok := cmdRouter.Load(f.Handle); ok {
		if fn, ok := handler.(func()); ok && fn != nil {
			fn()
		}
	}
}

var _ = cron.EntryID(0)
