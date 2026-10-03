package core

import "testing"

func TestTouchAgentConvMovesToEnd(t *testing.T) {
	agentHistory.Lock()
	oldItems, oldOrder := agentHistory.items, agentHistory.order
	agentHistory.items = map[string][]chatMessage{"a": {}, "b": {}, "c": {}}
	agentHistory.order = []string{"a", "b", "c"}
	touchAgentConvLocked("a")
	got := append([]string(nil), agentHistory.order...)
	agentHistory.items, agentHistory.order = oldItems, oldOrder
	agentHistory.Unlock()

	want := []string{"b", "c", "a"}
	if len(got) != len(want) {
		t.Fatalf("order=%v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order=%v want %v", got, want)
		}
	}
}
