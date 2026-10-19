package core

import (
	"strings"
	"testing"
	"time"
)

func TestMetricHistogramBucketsAndSum(t *testing.T) {
	h := newMetricHistogram([]float64{0.1, 1})
	h.observe(0.05)
	h.observe(0.5)
	h.observe(2)

	buckets, counts, count, sum := h.snapshot()
	if len(buckets) != 2 || counts[0] != 1 || counts[1] != 2 || count != 3 {
		t.Fatalf("直方图累计桶不正确: buckets=%v counts=%v count=%d", buckets, counts, count)
	}
	if sum < 2.54 || sum > 2.56 {
		t.Fatalf("直方图总和异常: %f", sum)
	}
}

func TestTaskMetricsRender(t *testing.T) {
	observeTaskFinished(taskDone, 1500*time.Millisecond)
	var b strings.Builder
	writeTaskMetrics(&b)
	out := b.String()
	for _, want := range []string{
		"jiying_task_duration_seconds_bucket{le=\"1\"}",
		"jiying_task_duration_seconds_sum",
		"jiying_task_queue_wait_seconds_bucket",
		"jiying_task_completed_total{status=\"done\"}",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("指标缺少 %q:\n%s", want, out)
		}
	}
}
