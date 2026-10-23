package core

import (
	"testing"
	"time"
)

func TestMentionedAppIDs(t *testing.T) {
	got := mentionedAppIDs("{(@app/app-b)} 你好 {(@app/app-c)} 和 {(@app/app-b)}")
	if len(got) != 2 || got[0] != "app-b" || got[1] != "app-c" {
		t.Fatalf("mentions=%v", got)
	}
	if ids := mentionedAppIDs("普通文本"); len(ids) != 0 {
		t.Fatalf("should not parse mentions: %v", ids)
	}
}

func TestNotifyAppMentionsDeliversOnce(t *testing.T) {
	setupAppPoolBucket(t, "app-mention")
	t.Cleanup(stopAppPool)

	source := NewJiyingClient(JiyingConfig{AppID: "app-a"}, nil)
	target := NewJiyingClient(JiyingConfig{AppID: "app-b"}, nil)
	appConnPool.mu.Lock()
	appConnPool.items["app-a"] = &appRuntime{cfg: AppConn{AppID: "app-a"}, client: source}
	appConnPool.items["app-b"] = &appRuntime{cfg: AppConn{AppID: "app-b"}, client: target}
	appConnPool.defaultID = "app-a"
	appConnPool.syncDefaultLocked()
	appConnPool.mu.Unlock()

	oldSenders := Senders
	Senders = make(chan Sender, 4)
	defer func() { Senders = oldSenders }()

	source.notifyAppMentions("conv-1", "m-1", "{(@app/app-b)} 请处理")
	select {
	case s := <-Senders:
		js, ok := s.(*JiyingSender)
		if !ok || js.ChatID != "conv-1" || js.MsgID != "m-1" || js.UserID != "app-a" || js.SenderType != "app" {
			t.Fatalf("unexpected sender: %+v", s)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("target app was not woken")
	}

	// 同一条消息再次投递时必须被去重，避免本地唤醒与轮询重复触发。
	source.notifyAppMentions("conv-1", "m-1", "{(@app/app-b)} 请处理")
	select {
	case s := <-Senders:
		t.Fatalf("duplicate delivery: %+v", s)
	case <-time.After(300 * time.Millisecond):
	}
}
