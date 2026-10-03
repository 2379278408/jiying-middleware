package core

import (
	"context"
	"testing"
)

func TestEventLedgerSurvivesRestart(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("JIYING_DATA", dir)
	BucketApp, _ = MakeBucket("jiying-ledger-restart", 6)

	c1 := NewJiyingClient(JiyingConfig{AppID: "app1"}, nil)
	c1.markProcessed(7)

	// 模拟进程重启：同一份本地存储，新的客户端实例。
	c2 := NewJiyingClient(JiyingConfig{AppID: "app1"}, nil)
	c2.loadAcks()
	if !c2.seenCursor(7) {
		t.Fatal("重启后应识别已处理事件")
	}
	if c2.seenCursor(6) {
		t.Fatal("未处理的 cursor 不应被误判为已处理")
	}
}

func TestAckSkipsAtOrBelowWatermark(t *testing.T) {
	c := NewJiyingClient(JiyingConfig{AppID: "app"}, nil)
	c.ackWatermark = 10
	if err := c.Ack(context.Background(), 5); err != nil {
		t.Fatal(err)
	}
}

func TestEventLedgerPrunesToBound(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("JIYING_DATA", dir)
	BucketApp, _ = MakeBucket("jiying-ledger-prune", 6)

	c := NewJiyingClient(JiyingConfig{AppID: "app"}, nil)
	for i := int64(1); i <= eventLedgerMax+10; i++ {
		c.markProcessed(i)
	}
	list := parseCursors(BucketApp.GetString("jiying_evt_app"))
	if len(list) != eventLedgerMax {
		t.Fatalf("台账长度=%d 期望 %d", len(list), eventLedgerMax)
	}
	if list[0] != 11 {
		t.Fatalf("最旧 cursor=%d 期望 11", list[0])
	}
}
