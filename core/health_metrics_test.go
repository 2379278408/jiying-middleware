package core

import (
	"testing"
	"time"
)

func TestTaskStatusCountsIncludesLifecycleStates(t *testing.T) {
	tasks.mu.Lock()
	oldItems, oldOrder := tasks.items, tasks.order
	tasks.items = map[string]*Task{
		"confirm": {ID: "confirm", Status: taskWaitingConfirmation},
		"cancel":  {ID: "cancel", Status: taskCanceled},
		"timeout": {ID: "timeout", Status: taskTimeout},
	}
	tasks.order = []string{"confirm", "cancel", "timeout"}
	tasks.mu.Unlock()
	t.Cleanup(func() {
		tasks.mu.Lock()
		tasks.items, tasks.order = oldItems, oldOrder
		tasks.mu.Unlock()
	})
	counts := TaskStatusCounts()
	if counts[taskWaitingConfirmation] != 1 || counts[taskCanceled] != 1 || counts[taskTimeout] != 1 {
		t.Fatalf("unexpected lifecycle counts: %#v", counts)
	}
}

func TestJiyingCallStatsAggregatesByMethod(t *testing.T) {
	method := "test.stats.method"
	jiyingObserveCall(method, 10*time.Millisecond, nil)
	jiyingObserveCall(method, 30*time.Millisecond, &jiyingCallError{Code: "response_too_large", Message: "too big"})
	jiyingObserveCall(method, 20*time.Millisecond, errJiyingNotConnected)

	var got *JiyingCallSnapshot
	for _, st := range JiyingCallStats() {
		if st.Method == method {
			snapshot := st
			got = &snapshot
		}
	}
	if got == nil {
		t.Fatalf("method %q missing from stats", method)
	}
	if got.OK != 1 || got.Failed != 2 {
		t.Fatalf("unexpected counters: %+v", got)
	}
	if got.Codes["response_too_large"] != 1 || got.Codes["not_connected"] != 1 {
		t.Fatalf("unexpected error codes: %+v", got.Codes)
	}
	if got.AvgMs < 19 || got.AvgMs > 21 {
		t.Fatalf("unexpected average: %f", got.AvgMs)
	}
}

func TestReconnectBackoffStaysWithinHalfToFull(t *testing.T) {
	base := 8 * time.Second
	for i := 0; i < 200; i++ {
		d := reconnectBackoff(base)
		if d < base/2 || d > base {
			t.Fatalf("backoff %v outside [%v, %v]", d, base/2, base)
		}
	}
	if reconnectBackoff(0) != 0 {
		t.Fatal("zero base must stay zero")
	}
}
