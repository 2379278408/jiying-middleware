package core

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

func TestPollInboundUsesVisibleConversationsAndMentions(t *testing.T) {
	const selfApp = "app-self"
	const peerApp = "app-peer"
	oldAt := time.Now().Add(-time.Minute).Format(time.RFC3339)
	newAt := time.Now().Format(time.RFC3339)
	mention := "{(@app/" + selfApp + ")}"

	client, _ := rpcTestClient(t, func(method string, _ json.RawMessage) (bool, string, json.RawMessage) {
		switch method {
		case "conversations.list":
			return true, "", json.RawMessage(`{"conversations":[{"conversation_id":"g1","name":"发布群","type":"group","last_active_at":"2026-10-21T10:00:00Z"}]}`)
		case "conversation.messages.list":
			return true, "", json.RawMessage(fmt.Sprintf(`{"messages":[
				{"id":"m-old","seq":6,"created_at":%q,"summary":"%s 旧消息","sender":{"id":%q,"type":"app"}},
				{"id":"m-1","seq":10,"created_at":%q,"summary":"%s 请回复","sender":{"id":%q,"type":"app"}},
				{"id":"m-2","seq":11,"created_at":%q,"summary":"普通消息","sender":{"id":%q,"type":"app"}},
				{"id":"m-self","seq":12,"created_at":%q,"summary":"%s 自己","sender":{"id":%q,"type":"app"}}
			]}`, oldAt, mention, peerApp, newAt, mention, peerApp, newAt, peerApp, newAt, mention, selfApp))
		default:
			return false, "unexpected", nil
		}
	})
	client.cfg.AppID = selfApp

	oldSenders := Senders
	Senders = make(chan Sender, 4)
	defer func() { Senders = oldSenders }()

	states := map[string]*pollConvState{
		"g1": {lastActive: "2026-10-21T09:00:00Z", lastSeq: 5},
	}
	client.pollInboundOnce(rpcCtx(t), states, time.Now().Add(-time.Minute), false)

	if got := len(Senders); got != 1 {
		t.Fatalf("只应入队一条本应用被 @ 的新消息，got %d", got)
	}
	sender, ok := (<-Senders).(*JiyingSender)
	if !ok || sender.ChatID != "g1" || sender.MsgID != "m-1" || sender.UserID != peerApp {
		t.Fatalf("轮询消息字段错误: %+v", sender)
	}
	if sender.ConvType != "group" || sender.SenderType != "app" {
		t.Fatalf("轮询消息类型错误: %+v", sender)
	}
	if states["g1"].lastSeq != 12 {
		t.Fatalf("应推进到最新序号，got %d", states["g1"].lastSeq)
	}

	// 活跃时间未变化时不再重复读取，避免同一消息重复入队。
	client.pollInboundOnce(rpcCtx(t), states, time.Now().Add(-time.Minute), false)
	if got := len(Senders); got != 0 {
		t.Fatalf("无变化时不应重复入队，got %d", got)
	}
}

func TestInboundPollInterval(t *testing.T) {
	t.Setenv("JIYING_POLL_INTERVAL_SECONDS", "")
	if got := inboundPollInterval(); got != 12*time.Second {
		t.Fatalf("default interval=%s", got)
	}
	t.Setenv("JIYING_POLL_INTERVAL_SECONDS", "1")
	if got := inboundPollInterval(); got != 5*time.Second {
		t.Fatalf("min interval=%s", got)
	}
	t.Setenv("JIYING_POLL_INTERVAL_SECONDS", "999")
	if got := inboundPollInterval(); got != 120*time.Second {
		t.Fatalf("max interval=%s", got)
	}
	t.Setenv("JIYING_POLL_INTERVAL_SECONDS", "bad")
	if got := inboundPollInterval(); got != 12*time.Second {
		t.Fatalf("invalid interval=%s", got)
	}
}
