package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Task struct {
	ID             string `json:"id"`
	Feature        string `json:"feature"`
	Args           string `json:"args"`
	ConversationID string `json:"conversation_id"`
	UserID         string `json:"user_id"`
	ReplyTo        string `json:"reply_to"`
	Status         string `json:"status"`
	Result         string `json:"result"`
	Confirm        bool   `json:"confirm"`
	Silent         bool   `json:"silent"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at,omitempty"`
	StartedAt      string `json:"started_at,omitempty"`
	FinishedAt     string `json:"finished_at,omitempty"`
	DurationMS     int64  `json:"duration_ms,omitempty"`
	RetryCount     int    `json:"retry_count,omitempty"`
	ErrorCode      string `json:"error_code,omitempty"`
}

const (
	taskPending             = "pending"
	taskWaitingConfirmation = "waiting_confirmation"
	taskRunning             = "running"
	taskDone                = "done"
	taskFailed              = "failed"
	taskCanceled            = "canceled"
	taskTimeout             = "timeout"
)

const taskRunTimeout = 2 * time.Minute

type taskQueue struct {
	mu sync.Mutex
	// bucket 在初始化时固定为当前存储；后台循环通过原子值读取，避免测试或热切换时与全局变量竞态。
	bucket atomic.Value
	items  map[string]*Task
	order  []string
	wake   chan struct{}
	// waits 保存静默任务（如 Agent 工具调用）的完成通知；值 chan 容量 1，非阻塞发送。
	waits map[string]chan *Task
}

var tasks = &taskQueue{items: map[string]*Task{}, wake: make(chan struct{}, 1), waits: map[string]chan *Task{}}

var taskLoopStarted atomic.Bool

func initTaskQueue() {
	if BucketApp != nil {
		tasks.bucket.Store(BucketApp)
	}
	tasks.load()
	if !taskLoopStarted.Swap(true) {
		go tasks.loop()
	}
}

func (q *taskQueue) load() {
	if q.bucket.Load() == nil {
		return
	}
	bucket := q.bucket.Load().(Bucket)
	raw := bucket.GetString("jiying_tasks")
	if raw == "" {
		return
	}
	var list []*Task
	if err := json.Unmarshal([]byte(raw), &list); err != nil {
		GetMyLog().Warn("任务队列解析失败")
		return
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	for _, t := range list {
		if t.Status == taskRunning {
			t.Status = taskPending
		}
		if t.Confirm && (t.Status == "" || t.Status == taskPending) {
			t.Status = taskWaitingConfirmation
		}
		q.items[t.ID] = t
		q.order = append(q.order, t.ID)
	}
}

func (q *taskQueue) saveLocked() {
	if q.bucket.Load() == nil {
		return
	}
	bucket := q.bucket.Load().(Bucket)
	// 内存中保留全部任务但限制规模；持久层只落非静默任务，避免 Agent 工具调用冲刷队列。
	if len(q.order) > 200 {
		drop := q.order[:len(q.order)-200]
		for _, id := range drop {
			delete(q.items, id)
		}
		q.order = append([]string(nil), q.order[len(q.order)-200:]...)
	}
	list := make([]*Task, 0, len(q.order))
	for _, id := range q.order {
		if t := q.items[id]; t != nil && !t.Silent {
			list = append(list, t)
		}
	}
	if len(list) > 200 {
		list = list[len(list)-200:]
	}
	raw, _ := json.Marshal(list)
	bucket.Set("jiying_tasks", string(raw))
}

func EnqueueTask(t Task) (*Task, error) {
	if strings.TrimSpace(t.Feature) == "" {
		return nil, fmt.Errorf("缺少功能名")
	}
	if err := checkQuota(t); err != nil {
		rateLimitRejected.Add(1)
		return nil, err
	}
	if t.ID == "" {
		t.ID = fmt.Sprintf("task-%d", time.Now().UnixNano())
	}
	if t.Status == "" {
		if t.Confirm {
			t.Status = taskWaitingConfirmation
		} else {
			t.Status = taskPending
		}
	}
	now := time.Now().UTC()
	if t.CreatedAt == "" {
		t.CreatedAt = now.Format(time.RFC3339)
	}
	t.UpdatedAt = now.Format(time.RFC3339)
	tasks.mu.Lock()
	cp := t
	tasks.items[t.ID] = &cp
	tasks.order = append(tasks.order, t.ID)
	tasks.saveLocked()
	tasks.mu.Unlock()
	if !t.Confirm {
		kickTaskLoop()
	}
	return &cp, nil
}

func (q *taskQueue) loop() {
	for range q.wake {
		for {
			t := q.next()
			if t == nil {
				break
			}
			q.run(t)
		}
	}
}

func (q *taskQueue) next() *Task {
	q.mu.Lock()
	defer q.mu.Unlock()
	for _, id := range q.order {
		t := q.items[id]
		if t != nil && t.Status == taskPending && !t.Confirm {
			t.Status = taskRunning
			t.StartedAt = time.Now().UTC().Format(time.RFC3339)
			t.UpdatedAt = t.StartedAt
			t.ErrorCode = ""
			q.saveLocked()
			cp := *t
			return &cp
		}
	}
	return nil
}

func (q *taskQueue) run(t *Task) {
	started := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), taskRunTimeout)
	result, err := safeExecute(func() (string, error) {
		return executeFeature(ctx, t.Feature, t.Args, t.UserID, t.ConversationID)
	})
	cancel()
	q.mu.Lock()
	cur := q.items[t.ID]
	text := result
	if cur != nil {
		finished := time.Now().UTC()
		cur.FinishedAt = finished.Format(time.RFC3339)
		cur.UpdatedAt = cur.FinishedAt
		cur.DurationMS = time.Since(started).Milliseconds()
		if err != nil {
			cur.Status = taskFailed
			cur.Result = err.Error()
			cur.ErrorCode = "EXECUTION_FAILED"
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				cur.Status = taskTimeout
				cur.ErrorCode = "EXECUTION_TIMEOUT"
			}
			text = "失败: " + err.Error()
		} else {
			cur.Status = taskDone
			cur.Result = result
		}
		q.saveLocked()
	}
	conv := t.ConversationID
	reply := t.ReplyTo
	waiter := q.waits[t.ID]
	q.mu.Unlock()
	if t.Silent {
		if waiter != nil {
			q.mu.Lock()
			fin := q.items[t.ID]
			var cp *Task
			if fin != nil {
				c := *fin
				cp = &c
			}
			delete(q.waits, t.ID)
			q.mu.Unlock()
			select {
			case waiter <- cp:
			default:
			}
		}
		// 静默任务报完结果即从队列清除，不进入任务列表与持久层。
		DeleteTask(t.ID)
		return
	}
	if conv != "" && clientForConv(conv) != nil && text != "" && !topicClosed(conv) {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		_ = SendTextToConv(ctx, conv, reply, text)
		cancel()
	}
}

// safeExecute 兜底 recover：任务消费是单条长驻协程，单个功能 panic
// 若放任上抛会终止整个队列，后续任务将永远不再被消费。
func safeExecute(run func() (string, error)) (result string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("功能执行异常: %v", r)
		}
	}()
	return run()
}

// WaitForTaskDone 等待任务进入终态并返回最终副本；仅用于静默（Agent 工具）任务。
// 任务已完成则立即返回，否则注册完成通知，超时后返回当前状态快照。
func WaitForTaskDone(id string, timeout time.Duration) *Task {
	ch := make(chan *Task, 1)
	tasks.mu.Lock()
	t := tasks.items[id]
	if t != nil && (t.Status == taskDone || t.Status == taskFailed) {
		cp := *t
		tasks.mu.Unlock()
		return &cp
	}
	tasks.waits[id] = ch
	tasks.mu.Unlock()
	select {
	case cp := <-ch:
		return cp
	case <-time.After(timeout):
		tasks.mu.Lock()
		delete(tasks.waits, id)
		t = tasks.items[id]
		var cp *Task
		if t != nil {
			c := *t
			cp = &c
		}
		tasks.mu.Unlock()
		return cp
	}
}

func ConfirmTask(id string, ok bool) error {
	tasks.mu.Lock()
	t := tasks.items[id]
	if t == nil {
		tasks.mu.Unlock()
		return fmt.Errorf("任务不存在")
	}
	conv := t.ConversationID
	reply := t.ReplyTo
	if !ok {
		t.Status = taskCanceled
		t.Result = "已取消"
		t.Confirm = false
		t.ErrorCode = "TASK_CANCELED"
		t.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		tasks.saveLocked()
		tasks.mu.Unlock()
		if conv != "" && clientForConv(conv) != nil && !topicClosed(conv) {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			_ = SendTextToConv(ctx, conv, reply, "已取消")
			cancel()
		}
		return nil
	}
	t.Confirm = false
	t.Status = taskPending
	t.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	tasks.saveLocked()
	tasks.mu.Unlock()
	kickTaskLoop()
	return nil
}

func ListTasks(limit int) string {
	tasks.mu.Lock()
	defer tasks.mu.Unlock()
	if limit <= 0 || limit > len(tasks.order) {
		limit = len(tasks.order)
	}
	start := len(tasks.order) - limit
	if start < 0 {
		start = 0
	}
	var b strings.Builder
	for _, id := range tasks.order[start:] {
		t := tasks.items[id]
		if t == nil || t.Silent {
			continue
		}
		fmt.Fprintf(&b, "%s %s %s\n", t.ID, t.Feature, t.Status)
	}
	if b.Len() == 0 {
		return "没有任务"
	}
	return strings.TrimSpace(b.String())
}

// TaskCounts 按状态统计任务数量，供指标与就绪探针使用。
func TaskCounts() (pending, running, done, failed int) {
	tasks.mu.Lock()
	defer tasks.mu.Unlock()
	for _, id := range tasks.order {
		t := tasks.items[id]
		if t == nil {
			continue
		}
		switch t.Status {
		case taskPending:
			pending++
		case taskRunning:
			running++
		case taskDone:
			done++
		case taskFailed:
			failed++
		}
	}
	return
}

// TaskStatusCounts 返回所有任务状态，供观测端点保持与任务列表一致。
func TaskStatusCounts() map[string]int {
	tasks.mu.Lock()
	defer tasks.mu.Unlock()
	counts := map[string]int{}
	for _, id := range tasks.order {
		t := tasks.items[id]
		if t != nil {
			counts[t.Status]++
		}
	}
	return counts
}

// ActiveTaskCount 返回排队中与执行中的任务总数。
func ActiveTaskCount() int {
	tasks.mu.Lock()
	defer tasks.mu.Unlock()
	return countActiveLocked("", "")
}

// ActiveCountByUser 返回某用户排队中与执行中的任务数。
func ActiveCountByUser(userID string) int {
	tasks.mu.Lock()
	defer tasks.mu.Unlock()
	return countActiveLocked("user", userID)
}

// ActiveCountByFeature 返回某功能排队中与执行中的任务数。
func ActiveCountByFeature(feature string) int {
	tasks.mu.Lock()
	defer tasks.mu.Unlock()
	return countActiveLocked("feature", feature)
}

// countActiveLocked 在已持有锁的前提下统计活动任务；field 为空表示全部。
func countActiveLocked(field, value string) int {
	n := 0
	for _, id := range tasks.order {
		t := tasks.items[id]
		if t == nil || (t.Status != taskPending && t.Status != taskWaitingConfirmation && t.Status != taskRunning) {
			continue
		}
		switch field {
		case "user":
			if t.UserID != value {
				continue
			}
		case "feature":
			if t.Feature != value {
				continue
			}
		}
		n++
	}
	return n
}

// taskLoopReady 表示任务队列已初始化并开始消费。
func taskLoopReady() bool {
	return taskLoopStarted.Load()
}

func ListTaskItems(limit int) []*Task {
	return listTaskItems(TaskFilter{Limit: limit})
}

// TaskFilter 控制任务列表的筛选条件，空字段表示不过滤。
type TaskFilter struct {
	Status  string
	Feature string
	UserID  string
	Limit   int
}

func listTaskItems(filter TaskFilter) []*Task {
	tasks.mu.Lock()
	defer tasks.mu.Unlock()
	limit := filter.Limit
	if limit <= 0 || limit > len(tasks.order) {
		limit = len(tasks.order)
	}
	out := make([]*Task, 0, limit)
	for i := len(tasks.order) - 1; i >= 0 && len(out) < limit; i-- {
		t := tasks.items[tasks.order[i]]
		if t == nil || t.Silent {
			continue
		}
		if filter.Status != "" && t.Status != filter.Status {
			continue
		}
		if filter.Feature != "" && t.Feature != filter.Feature {
			continue
		}
		if filter.UserID != "" && t.UserID != filter.UserID {
			continue
		}
		cp := *t
		out = append(out, &cp)
	}
	for left, right := 0, len(out)-1; left < right; left, right = left+1, right-1 {
		out[left], out[right] = out[right], out[left]
	}
	return out
}

func RetryTask(id string) error {
	tasks.mu.Lock()
	t := tasks.items[id]
	if t == nil {
		tasks.mu.Unlock()
		return fmt.Errorf("任务不存在")
	}
	if t.Status == taskRunning || t.Status == taskPending {
		tasks.mu.Unlock()
		return fmt.Errorf("任务正在等待或执行中")
	}
	t.Status = taskPending
	t.Confirm = false
	t.RetryCount++
	t.ErrorCode = ""
	t.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	tasks.saveLocked()
	tasks.mu.Unlock()
	kickTaskLoop()
	return nil
}

func DeleteTask(id string) bool {
	tasks.mu.Lock()
	defer tasks.mu.Unlock()
	if tasks.items[id] == nil {
		return false
	}
	delete(tasks.items, id)
	out := tasks.order[:0]
	for _, oid := range tasks.order {
		if oid != id {
			out = append(out, oid)
		}
	}
	tasks.order = out
	tasks.saveLocked()
	return true
}

func kickTaskLoop() {
	select {
	case tasks.wake <- struct{}{}:
	default:
	}
}

// resetTaskQueue 清空内存任务队列，供测试隔离使用；不会停止已启动的消费协程。
func resetTaskQueue() {
	tasks.mu.Lock()
	tasks.items = map[string]*Task{}
	tasks.order = nil
	tasks.waits = map[string]chan *Task{}
	tasks.mu.Unlock()
}

// failTasksForConversation 结束某会话下等待执行或待确认的任务，用于话题关闭后的收尾。
func failTasksForConversation(convID string) {
	if convID == "" {
		return
	}
	tasks.mu.Lock()
	defer tasks.mu.Unlock()
	changed := false
	for _, id := range tasks.order {
		t := tasks.items[id]
		if t == nil || t.ConversationID != convID {
			continue
		}
		if t.Status == taskPending || t.Confirm {
			t.Status = taskFailed
			t.Result = "话题已关闭"
			t.Confirm = false
			changed = true
		}
	}
	if changed {
		tasks.saveLocked()
	}
}

func enqueueCron(cmd *CronCmd) {
	if cmd == nil {
		return
	}
	chat := ""
	if len(cmd.GroupManager) > 0 {
		chat = cmd.GroupManager[0]
	}
	_, err := EnqueueTask(Task{
		Feature:        "cron:" + cmd.Name,
		Args:           cmd.Command,
		ConversationID: chat,
		UserID:         cronIdentity(cmd),
	})
	if err != nil {
		GetMyLog().Warn("定时入队失败 %s", cmd.Name)
	}
}

func bindChoice(choiceMessageID string, task *Task) {
	if choiceMessageID == "" || task == nil || task.ID == "" {
		return
	}
	choiceWaits.Store(choiceMessageID, choiceWait{
		TaskID:         task.ID,
		UserID:         task.UserID,
		ConversationID: task.ConversationID,
	})
}
