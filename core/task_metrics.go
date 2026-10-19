package core

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// taskDurationBuckets 与 taskWaitBuckets 覆盖从毫秒级脚本到 2 分钟超时的常见区间。
var (
	taskDurationBuckets = []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60, 120}
	taskWaitBuckets     = []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60, 120}
)

// metricHistogram 是一个不依赖第三方库的 Prometheus 直方图，只统计累计值。
type metricHistogram struct {
	buckets []float64
	counts  []uint64
	count   uint64
	sum     float64
}

func newMetricHistogram(buckets []float64) *metricHistogram {
	cp := append([]float64(nil), buckets...)
	sort.Float64s(cp)
	return &metricHistogram{buckets: cp, counts: make([]uint64, len(cp))}
}

func (h *metricHistogram) observe(v float64) {
	if v < 0 {
		v = 0
	}
	h.count++
	h.sum += v
	for i, le := range h.buckets {
		if v <= le {
			h.counts[i]++
		}
	}
}

func (h *metricHistogram) snapshot() ([]float64, []uint64, uint64, float64) {
	return append([]float64(nil), h.buckets...), append([]uint64(nil), h.counts...), h.count, h.sum
}

func formatMetricFloat(v float64) string {
	return fmt.Sprintf("%g", v)
}

type taskMetricStore struct {
	mu        sync.Mutex
	duration  *metricHistogram
	wait      *metricHistogram
	completed map[string]uint64
}

var taskMetrics = &taskMetricStore{
	duration:  newMetricHistogram(taskDurationBuckets),
	wait:      newMetricHistogram(taskWaitBuckets),
	completed: map[string]uint64{},
}

// observeTaskPicked 记录从入队到开始执行的等待时间；恢复任务没有入队时间时跳过。
func observeTaskPicked(t *Task) {
	if t == nil || t.enqueuedAt.IsZero() {
		return
	}
	wait := time.Since(t.enqueuedAt).Seconds()
	taskMetrics.mu.Lock()
	taskMetrics.wait.observe(wait)
	taskMetrics.mu.Unlock()
}

// observeTaskFinished 记录执行耗时与终态计数。
func observeTaskFinished(status string, duration time.Duration) {
	if status == "" {
		return
	}
	taskMetrics.mu.Lock()
	taskMetrics.duration.observe(duration.Seconds())
	taskMetrics.completed[status]++
	taskMetrics.mu.Unlock()
}

func writeTaskMetrics(b *strings.Builder) {
	taskMetrics.mu.Lock()
	durationBuckets, durationCounts, durationCount, durationSum := taskMetrics.duration.snapshot()
	waitBuckets, waitCounts, waitCount, waitSum := taskMetrics.wait.snapshot()
	completed := make(map[string]uint64, len(taskMetrics.completed))
	for k, v := range taskMetrics.completed {
		completed[k] = v
	}
	taskMetrics.mu.Unlock()

	writeHistogram(b, "jiying_task_duration_seconds", "任务执行耗时（秒）", durationBuckets, durationCounts, durationCount, durationSum)
	writeHistogram(b, "jiying_task_queue_wait_seconds", "任务从入队到开始执行的等待时间（秒）", waitBuckets, waitCounts, waitCount, waitSum)

	fmt.Fprintf(b, "# HELP jiying_task_completed_total 已完成任务数，按终态拆分\n")
	fmt.Fprintf(b, "# TYPE jiying_task_completed_total counter\n")
	statuses := make([]string, 0, len(completed))
	for status := range completed {
		statuses = append(statuses, status)
	}
	sort.Strings(statuses)
	for _, status := range statuses {
		fmt.Fprintf(b, "jiying_task_completed_total{status=%q} %d\n", status, completed[status])
	}
}

func writeHistogram(b *strings.Builder, name, help string, buckets []float64, counts []uint64, count uint64, sum float64) {
	fmt.Fprintf(b, "# HELP %s %s\n", name, help)
	fmt.Fprintf(b, "# TYPE %s histogram\n", name)
	for i, le := range buckets {
		fmt.Fprintf(b, "%s_bucket{le=%q} %d\n", name, formatMetricFloat(le), counts[i])
	}
	fmt.Fprintf(b, "%s_bucket{le=\"+Inf\"} %d\n", name, count)
	fmt.Fprintf(b, "%s_sum %s\n", name, formatMetricFloat(sum))
	fmt.Fprintf(b, "%s_count %d\n", name, count)
}
