package core

import (
	"testing"
	"time"
)

func TestIngestEventDeduplicatesWithoutServerID(t *testing.T) {
	d := &dingTalkConnector{seen: map[string]time.Time{}}
	base := DingTalkEvent{Type: "message", ConversationID: "c1", Sender: "张三", Content: "第一条", ReceivedAt: "2026-10-23T10:00:00Z"}
	d.ingestEvent(base)
	d.ingestEvent(base)
	if len(d.events) != 1 {
		t.Fatalf("same event should be deduplicated, got %d", len(d.events))
	}
	other := base
	other.Content = "同一秒的第二条"
	d.ingestEvent(other)
	if len(d.events) != 2 {
		t.Fatalf("different content in the same second must not be dropped, got %d", len(d.events))
	}
	withID := base
	withID.ID = "ev-1"
	d.ingestEvent(withID)
	d.ingestEvent(withID)
	if len(d.events) != 3 {
		t.Fatalf("event with server id should also deduplicate, got %d", len(d.events))
	}
}
