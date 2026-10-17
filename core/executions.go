package core

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	executionRetention  = 500
	executionKindCron   = "cron"
	executionKindFlow   = "flow"
	executionKindManual = "manual"

	executionSuccess = "success"
	executionFailed  = "failed"
	executionTimeout = "timeout"
	executionSkipped = "skipped"
	executionRunning = "running"
)

// Execution 是一次定时、流程或手动任务的执行摘要，供面板排查使用。
type Execution struct {
	ID         string `json:"id"`
	Kind       string `json:"kind"`
	Name       string `json:"name"`
	Command    string `json:"command,omitempty"`
	Status     string `json:"status"`
	Detail     string `json:"detail,omitempty"`
	Error      string `json:"error,omitempty"`
	StartedAt  string `json:"started_at"`
	FinishedAt string `json:"finished_at,omitempty"`
	DurationMS int64  `json:"duration_ms"`
	TraceID    string `json:"trace_id,omitempty"`
	Owner      string `json:"owner,omitempty"`
}

var executionLedger = struct {
	sync.Mutex
	items []*Execution
}{}

var executionPersist = make(chan struct{}, 1)

// initExecutions 加载历史执行记录并启动异步落盘协程，避免阻塞请求与定时线程。
func initExecutions() {
	loadExecutions()
	go func() {
		for range executionPersist {
			persistExecutions()
		}
	}()
}

func executionsPath() string {
	return filepath.Join(DataHome, "executions.json")
}

func loadExecutions() {
	raw, err := os.ReadFile(executionsPath())
	if err != nil {
		return
	}
	var items []*Execution
	if json.Unmarshal(raw, &items) != nil {
		return
	}
	if len(items) > executionRetention {
		items = items[:executionRetention]
	}
	executionLedger.Lock()
	executionLedger.items = items
	executionLedger.Unlock()
}

func scheduleExecutionPersist() {
	select {
	case executionPersist <- struct{}{}:
	default:
	}
}

func persistExecutions() {
	executionLedger.Lock()
	items := make([]Execution, 0, len(executionLedger.items))
	for _, item := range executionLedger.items {
		if item != nil {
			items = append(items, *item)
		}
	}
	executionLedger.Unlock()
	raw, err := json.Marshal(items)
	if err != nil {
		return
	}
	dir := filepath.Dir(executionsPath())
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return
	}
	tmp, err := os.CreateTemp(dir, ".executions-*.tmp")
	if err != nil {
		return
	}
	name := tmp.Name()
	defer os.Remove(name)
	_ = tmp.Chmod(0o600)
	if _, err := tmp.Write(raw); err != nil {
		tmp.Close()
		return
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return
	}
	if err := tmp.Close(); err != nil {
		return
	}
	_ = os.Rename(name, executionsPath())
}

func newExecutionID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "exec-" + time.Now().Format("20060102150405.000000000")
	}
	return "exec-" + hex.EncodeToString(b[:])
}

func beginExecution(kind, name, command, owner string) string {
	id := newExecutionID()
	e := &Execution{
		ID:        id,
		Kind:      kind,
		Name:      name,
		Command:   command,
		Status:    executionRunning,
		StartedAt: time.Now().Format("2006-01-02 15:04:05"),
		TraceID:   newTraceID(),
		Owner:     owner,
	}
	executionLedger.Lock()
	executionLedger.items = append([]*Execution{e}, executionLedger.items...)
	if len(executionLedger.items) > executionRetention {
		executionLedger.items = executionLedger.items[:executionRetention]
	}
	executionLedger.Unlock()
	scheduleExecutionPersist()
	return id
}

func finishExecution(id, status, detail, errText string) {
	now := time.Now()
	executionLedger.Lock()
	for _, e := range executionLedger.items {
		if e == nil || e.ID != id {
			continue
		}
		e.Status = status
		e.Detail = truncateText(detail, 2000)
		e.Error = truncateText(errText, 500)
		e.FinishedAt = now.Format("2006-01-02 15:04:05")
		if started, err := time.ParseInLocation("2006-01-02 15:04:05", e.StartedAt, time.Local); err == nil {
			e.DurationMS = now.Sub(started).Milliseconds()
		}
		break
	}
	executionLedger.Unlock()
	scheduleExecutionPersist()
}

func truncateText(s string, max int) string {
	s = strings.TrimSpace(s)
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

// ExecutionFilter 限定执行记录的查询范围。
type ExecutionFilter struct {
	Kind   string
	Name   string
	Status string
	Limit  int
}

func listExecutions(f ExecutionFilter) []Execution {
	if f.Limit <= 0 || f.Limit > executionRetention {
		f.Limit = 100
	}
	executionLedger.Lock()
	defer executionLedger.Unlock()
	out := make([]Execution, 0, f.Limit)
	for _, e := range executionLedger.items {
		if e == nil {
			continue
		}
		if f.Kind != "" && e.Kind != f.Kind {
			continue
		}
		if f.Status != "" && e.Status != f.Status {
			continue
		}
		if f.Name != "" && !strings.Contains(e.Name, f.Name) && !strings.Contains(e.Command, f.Name) {
			continue
		}
		out = append(out, *e)
		if len(out) >= f.Limit {
			break
		}
	}
	return out
}

func executionSummary() (total, failed int) {
	executionLedger.Lock()
	defer executionLedger.Unlock()
	for _, e := range executionLedger.items {
		if e == nil {
			continue
		}
		total++
		if e.Status == executionFailed || e.Status == executionTimeout {
			failed++
		}
	}
	return total, failed
}

// runTrackedExecution 统一记录一次执行的开始、结果与耗时。
func runTrackedExecution(kind, name, command, owner string, fn func() (string, error)) (string, error) {
	id := beginExecution(kind, name, command, owner)
	out, err := fn()
	if err != nil {
		finishExecution(id, executionFailed, out, err.Error())
		return out, err
	}
	finishExecution(id, executionSuccess, out, "")
	return out, nil
}
