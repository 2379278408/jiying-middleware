package core

import (
	"sort"
	"sync"
	"time"
)

var (
	statsMu      sync.RWMutex
	msgsByImtype = map[string]int{}
	msgsByHour   = map[string]int{}
	cmdHits      = map[string]int{}
	msgTotal     int
	startedAt    = time.Now()
)

type imtypeExtractor interface {
	GetImtype() string
}

func RecordMessage(sender Sender) {
	im := "local"
	if ex, ok := sender.(imtypeExtractor); ok {
		if v := ex.GetImtype(); v != "" {
			im = v
		}
	}
	hour := time.Now().Format("15:04")
	statsMu.Lock()
	msgsByImtype[im]++
	msgsByHour[hour]++
	msgTotal++
	statsMu.Unlock()
}

func RecordCommand(title string) {
	statsMu.Lock()
	cmdHits[title]++
	statsMu.Unlock()
}

func StatsSnapshot() (imtypes, hours, cmds map[string]int, total int) {
	statsMu.RLock()
	defer statsMu.RUnlock()
	cp := func(m map[string]int) map[string]int {
		out := make(map[string]int, len(m))
		for k, v := range m {
			out[k] = v
		}
		return out
	}
	return cp(msgsByImtype), cp(msgsByHour), cp(cmdHits), msgTotal
}

func StartedAt() time.Time { return startedAt }

func CommandTotalAndFinished() (total, finished int) {
	for _, p := range CronCmdsSnapshot() {
		if p == nil {
			continue
		}
		total++
		if p.LastRun != "" {
			finished++
		}
	}
	return total, finished
}

func FunctionsSnapshot() []*Function {
	funcLock.RLock()
	defer funcLock.RUnlock()
	out := make([]*Function, len(Functions))
	copy(out, Functions)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i] == nil || out[j] == nil {
			return false
		}
		return out[i].Priority > out[j].Priority
	})
	return out
}

// functionByTitle 返回命令表里指定标题的函数，供权限与工具契约查询。
func functionByTitle(title string) *Function {
	funcLock.RLock()
	defer funcLock.RUnlock()
	for _, f := range Functions {
		if f != nil && f.Title == title {
			return f
		}
	}
	return nil
}
