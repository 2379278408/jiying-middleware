package core

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func setupMsgStatsBucket(t *testing.T, name string) {
	t.Helper()
	t.Setenv("JIYING_DATA", t.TempDir())
	stopMsgStatsFlush()
	b, err := MakeBucket(name, 6)
	if err != nil {
		t.Fatal(err)
	}
	BucketApp = b
	BucketApp.Delete(jiyingMsgStatsKey)
	resetMsgStatsForTest()
	t.Cleanup(func() {
		stopMsgStatsFlush()
		BucketApp.Delete(jiyingMsgStatsKey)
		resetMsgStatsForTest()
	})
}

func resetMsgStatsForTest() {
	jiyingMsgStats.Lock()
	jiyingMsgStats.day = ""
	jiyingMsgStats.inbound = 0
	jiyingMsgStats.out = 0
	jiyingMsgStats.Unlock()
}

func TestMsgStatsPersistAcrossReload(t *testing.T) {
	setupMsgStatsBucket(t, "msgstats-persist")

	jiyingCountInbound()
	jiyingCountOutbound()
	jiyingCountOutbound()
	if in, out := jiyingMsgCounts(); in != 1 || out != 2 {
		t.Fatalf("unexpected live counts: in=%d out=%d", in, out)
	}

	// 批量落盘由定时器触发；测试显式 flush 后再模拟进程重启。
	flushMsgStats()
	resetMsgStatsForTest()
	if in, out := jiyingMsgCounts(); in != 1 || out != 2 {
		t.Fatalf("counts not restored after reload: in=%d out=%d", in, out)
	}
}

func TestOutboundTransportsCountOnce(t *testing.T) {
	setupMsgStatsBucket(t, "msgstats-outbound-transports")
	client, _ := rpcTestClient(t, func(method string, _ json.RawMessage) (bool, string, json.RawMessage) {
		return true, "", json.RawMessage(`{"message":{"id":"m-1"}}`)
	})

	if _, err := client.SendMessage(rpcCtx(t), "conv-1", "", map[string]any{"type": "text", "content": "hi"}); err != nil {
		t.Fatal(err)
	}
	if err := client.SendText(rpcCtx(t), "conv-1", "", "hi"); err != nil {
		t.Fatal(err)
	}
	if _, err := client.SendChoice(rpcCtx(t), "conv-1", "选一个", []map[string]string{{"id": "a", "label": "A"}, {"id": "b", "label": "B"}}); err != nil {
		t.Fatal(err)
	}
	SetUserSession("user@example.com", "tok")
	defer SetUserSession("", "")
	oldSend := sendUserCall
	defer func() { sendUserCall = oldSend }()
	sendUserCall = func(ctx context.Context, method, path string, body any) ([]byte, error) {
		return []byte(`{"data":{"message":{"id":"m-user"}}}`), nil
	}
	if _, err := SendTextAsUser(context.Background(), "conv-user", "", "hi"); err != nil {
		t.Fatal(err)
	}
	if in, out := jiyingMsgCounts(); in != 0 || out != 4 {
		t.Fatalf("each successful transport must count once: in=%d out=%d", in, out)
	}
}

func TestMsgStatsResetOnNewDay(t *testing.T) {
	setupMsgStatsBucket(t, "msgstats-day")

	jiyingMsgStats.Lock()
	jiyingMsgStats.day = time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	jiyingMsgStats.inbound = 9
	jiyingMsgStats.out = 7
	saveJiyingMsgStatsLocked()
	jiyingMsgStats.Unlock()
	resetMsgStatsForTest()

	if in, out := jiyingMsgCounts(); in != 0 || out != 0 {
		t.Fatalf("stale day should read as zero: in=%d out=%d", in, out)
	}
	jiyingCountInbound()
	if in, out := jiyingMsgCounts(); in != 1 || out != 0 {
		t.Fatalf("new day should reset counters: in=%d out=%d", in, out)
	}
}

func TestUnifiedSendMessageCountsOutbound(t *testing.T) {
	setupMsgStatsBucket(t, "msgstats-unified-send")
	client, _ := rpcTestClient(t, func(method string, _ json.RawMessage) (bool, string, json.RawMessage) {
		return true, "", json.RawMessage(`{"message":{"id":"m-app"}}`)
	})
	appConnPool.mu.Lock()
	appConnPool.items[client.cfg.AppID] = &appRuntime{cfg: AppConn{AppID: client.cfg.AppID}, client: client}
	appConnPool.defaultID = client.cfg.AppID
	appConnPool.syncDefaultLocked()
	appConnPool.mu.Unlock()
	t.Cleanup(stopAppPool)
	rememberConvApp("conv-app", client.cfg.AppID)

	if _, err := SendMessage(context.Background(), SendOptions{
		ConversationID: "conv-app", Text: "hi", Identity: "app",
	}); err != nil {
		t.Fatal(err)
	}

	SetUserSession("user@example.com", "tok")
	defer SetUserSession("", "")
	oldSend := sendUserCall
	defer func() { sendUserCall = oldSend }()
	sendUserCall = func(ctx context.Context, method, path string, body any) ([]byte, error) {
		return []byte(`{"data":{"message":{"id":"m-user"}}}`), nil
	}
	if _, err := SendMessage(context.Background(), SendOptions{
		ConversationID: "conv-user", Text: "hi", Identity: "user",
	}); err != nil {
		t.Fatal(err)
	}

	if in, out := jiyingMsgCounts(); in != 0 || out != 2 {
		t.Fatalf("unified send should count once per identity: in=%d out=%d", in, out)
	}
}
