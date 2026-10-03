package core

import "testing"

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
