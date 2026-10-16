package core

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/dop251/goja"
)

// rpcRecorder 记录应用通道发出的请求，供测试断言。
type rpcRecorder struct {
	mu   sync.Mutex
	reqs []jiyingEnvelope
}

func (r *rpcRecorder) add(env jiyingEnvelope) {
	r.mu.Lock()
	r.reqs = append(r.reqs, env)
	r.mu.Unlock()
}

func (r *rpcRecorder) methods() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, 0, len(r.reqs))
	for _, e := range r.reqs {
		out = append(out, e.Method)
	}
	return out
}

func (r *rpcRecorder) payload(method string) json.RawMessage {
	r.mu.Lock()
	defer r.mu.Unlock()
	var last json.RawMessage
	for _, e := range r.reqs {
		if e.Method == method {
			last = e.Payload
		}
	}
	return last
}

// rpcTestClient 创建一个带假连接、按 respond 规则自动应答的客户端。
func rpcTestClient(t *testing.T, respond func(method string, payload json.RawMessage) (bool, string, json.RawMessage)) (*JiyingClient, *rpcRecorder) {
	t.Helper()
	sock := &fakeSock{in: make(chan jiyingEnvelope, 16), out: make(chan jiyingEnvelope, 16)}
	client := NewJiyingClient(JiyingConfig{WSURL: "ws://example", AppID: "app", Secret: "sec"}, nil)
	client.conn = sock
	rec := &rpcRecorder{}
	go func() {
		for req := range sock.out {
			rec.add(req)
			ok, code, payload := respond(req.Method, req.Payload)
			env := jiyingEnvelope{Kind: "response", ReplyTo: req.ID, OK: ok}
			if ok {
				env.Payload = payload
			} else {
				env.Error = &jiyingError{Code: code, Message: code}
			}
			_ = client.HandleInbound(context.Background(), env)
		}
	}()
	return client, rec
}

func rpcCtx(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func TestRpcTargetValidation(t *testing.T) {
	if _, err := (rpcTarget{Type: "user"}).payload(); err == nil {
		t.Fatal("user target without id must fail")
	}
	if _, err := (rpcTarget{Type: "group"}).payload(); err == nil {
		t.Fatal("group target without conversation must fail")
	}
	if _, err := (rpcTarget{Type: "bogus", ConversationID: "c"}).payload(); err == nil {
		t.Fatal("unknown target type must fail")
	}
	p, err := userTarget("u-9").payload()
	if err != nil {
		t.Fatal(err)
	}
	if p["type"] != "user" || p["user_id"] != "u-9" {
		t.Fatalf("payload=%v", p)
	}
	// convTarget 空类型默认 conversation。
	p, err = convTarget("", "conv-1").payload()
	if err != nil {
		t.Fatal(err)
	}
	if p["type"] != "conversation" || p["conversation_id"] != "conv-1" {
		t.Fatalf("payload=%v", p)
	}
}

func TestSendToUserAndConversation(t *testing.T) {
	client, rec := rpcTestClient(t, func(string, json.RawMessage) (bool, string, json.RawMessage) {
		return true, "", json.RawMessage(`{"message":{"id":"m-1"}}`)
	})

	id, err := client.SendTo(rpcCtx(t), userTarget("u-9"), "", map[string]any{"type": "text", "content": "hi"})
	if err != nil {
		t.Fatal(err)
	}
	if id != "m-1" {
		t.Fatalf("id=%s", id)
	}
	var body struct {
		Target map[string]any `json:"target"`
	}
	if err := json.Unmarshal(rec.payload("message.send"), &body); err != nil {
		t.Fatal(err)
	}
	if body.Target["type"] != "user" || body.Target["user_id"] != "u-9" {
		t.Fatalf("target=%v", body.Target)
	}

	if _, err := client.SendTo(rpcCtx(t), convTarget("group", "g-2"), "m-0", map[string]any{"type": "text", "content": "yo"}); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(rec.payload("message.send"), &body); err != nil {
		t.Fatal(err)
	}
	if body.Target["type"] != "group" || body.Target["conversation_id"] != "g-2" {
		t.Fatalf("target=%v", body.Target)
	}

	if _, err := client.SendTo(rpcCtx(t), userTarget("u-9"), "", nil); err == nil {
		t.Fatal("empty message must fail")
	}
}

func TestUsersGetAndAppsGet(t *testing.T) {
	client, _ := rpcTestClient(t, func(method string, _ json.RawMessage) (bool, string, json.RawMessage) {
		switch method {
		case "users.get":
			return true, "", json.RawMessage(`{"user":{"id":"u-1","name":"Ada","nickname":"ada"}}`)
		case "apps.get":
			return true, "", json.RawMessage(`{"app":{"id":"app-1","name":"Bot","online":true}}`)
		}
		return false, "not_found", nil
	})
	u, err := client.UsersGet(rpcCtx(t), "u-1")
	if err != nil {
		t.Fatal(err)
	}
	if u.ID != "u-1" || u.Name != "Ada" {
		t.Fatalf("user=%+v", u)
	}
	a, err := client.AppsGet(rpcCtx(t), "app-1")
	if err != nil {
		t.Fatal(err)
	}
	if a.ID != "app-1" || !a.Online {
		t.Fatalf("app=%+v", a)
	}
	if _, err := client.UsersGet(rpcCtx(t), ""); err == nil {
		t.Fatal("empty user id must fail")
	}
}

func TestListMessagesValidatesSeq(t *testing.T) {
	client, _ := rpcTestClient(t, func(string, json.RawMessage) (bool, string, json.RawMessage) {
		return true, "", json.RawMessage(`{"messages":[{"id":"m2","seq":5}]}`)
	})
	if _, err := client.ListMessages(rpcCtx(t), "c-1", 0, 10); err == nil {
		t.Fatal("zero seq must fail")
	}
	msgs, err := client.ListMessages(rpcCtx(t), "c-1", 5, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 || msgs[0].ID != "m2" || msgs[0].Seq != 5 {
		t.Fatalf("msgs=%+v", msgs)
	}
}

func TestTopicCreateAndParse(t *testing.T) {
	client, rec := rpcTestClient(t, func(string, json.RawMessage) (bool, string, json.RawMessage) {
		return true, "", json.RawMessage(`{"conversation":{"id":"topic-7"},"parent_conversation_id":"c-1","source_message_id":"m-3","last_message_seq":9}`)
	})
	topic, err := client.TopicCreate(rpcCtx(t), "c-1", "m-3")
	if err != nil {
		t.Fatal(err)
	}
	if topic.ConversationID != "topic-7" || topic.ParentConversationID != "c-1" || topic.SourceMessageID != "m-3" || topic.LastMessageSeq != 9 {
		t.Fatalf("topic=%+v", topic)
	}
	if rec.payload("conversation.topic.create") == nil {
		t.Fatal("create request not sent")
	}
	if _, err := client.TopicCreate(rpcCtx(t), "c-1", ""); err == nil {
		t.Fatal("missing source message must fail")
	}
}

func TestGroupValidations(t *testing.T) {
	client, _ := rpcTestClient(t, func(string, json.RawMessage) (bool, string, json.RawMessage) {
		return true, "", json.RawMessage(`{}`)
	})
	if _, err := client.GroupCreate(rpcCtx(t), "群", nil, nil); err == nil {
		t.Fatal("create without members must fail")
	}
	if _, err := client.GroupCreate(rpcCtx(t), "", []string{"u1"}, nil); err == nil {
		t.Fatal("create without name must fail")
	}
	if _, err := client.GroupUpdate(rpcCtx(t), "g-1", "名字", "公告"); err == nil {
		t.Fatal("name+announcement must fail")
	}
	if _, err := client.GroupUpdate(rpcCtx(t), "g-1", "", ""); err == nil {
		t.Fatal("no change must fail")
	}
	if err := client.GroupSetRole(rpcCtx(t), "g-1", "user", "u1", "boss"); err == nil {
		t.Fatal("bad role must fail")
	}
	if err := client.GroupRemoveMember(rpcCtx(t), "g-1", "robot", "x"); err == nil {
		t.Fatal("bad member type must fail")
	}
}

func TestMessagesToChatHistory(t *testing.T) {
	body := func(v map[string]any) json.RawMessage {
		b, _ := json.Marshal(v)
		return b
	}
	msg := func(seq int64, typ, id string, b map[string]any) JiyingMessage {
		m := JiyingMessage{Seq: seq, Body: body(b)}
		m.Sender.ID = id
		m.Sender.Type = typ
		return m
	}
	msgs := []JiyingMessage{
		msg(1, "user", "u-1", map[string]any{"type": "text", "content": "你好"}),
		msg(2, "app", "app-1", map[string]any{"type": "text", "content": "在的"}),
		msg(3, "user", "u-2", map[string]any{"type": "image", "file_id": "f-1", "caption": "看这个"}),
		msg(4, "user", "u-1", map[string]any{"type": "text", "content": "当前问题"}),
	}
	// uptoSeq=4：最后一条作为 prompt 传入，必须从历史中剔除。
	got := messagesToChatHistory(msgs, "app-1", 4)
	if len(got) != 3 {
		t.Fatalf("len=%d, want 3", len(got))
	}
	if got[0].Role != "user" || got[0].Content != "你好" {
		t.Fatalf("got[0]=%+v", got[0])
	}
	if got[1].Role != "assistant" || got[1].Content != "在的" {
		t.Fatalf("got[1]=%+v", got[1])
	}
	if got[2].Role != "user" || got[2].Content != "看这个" {
		t.Fatalf("got[2]=%+v", got[2])
	}
	// 没有 seq 锚点时保留全部（含最后一条），供调用方自行决定。
	if all := messagesToChatHistory(msgs, "app-1", 0); len(all) != 4 {
		t.Fatalf("len=%d, want 4", len(all))
	}
}

func TestJiyingMessagePlainTextFallsBackToSummary(t *testing.T) {
	m := JiyingMessage{Summary: "摘要兜底"}
	if m.plainText() != "摘要兜底" {
		t.Fatalf("plainText=%q", m.plainText())
	}
	m2 := JiyingMessage{Summary: "忽略", Body: json.RawMessage(`{"type":"file","name":"a.pdf"}`)}
	if m2.plainText() != "a.pdf" {
		t.Fatalf("plainText=%q", m2.plainText())
	}
}

func TestScriptAppApis(t *testing.T) {
	client, rec := rpcTestClient(t, func(method string, _ json.RawMessage) (bool, string, json.RawMessage) {
		switch method {
		case "message.reaction.set":
			return true, "", json.RawMessage(`{}`)
		case "users.get":
			return false, "not_found", nil
		}
		return true, "", json.RawMessage(`{"user":{"id":"u-1"}}`)
	})
	old := activeJiying
	activeJiying = client
	t.Cleanup(func() { activeJiying = old })

	rt := goja.New()
	registerAppApis(rt, "")
	if v, err := rt.RunString(`appReaction("c-1","m-2","🔥")`); err != nil {
		t.Fatal(err)
	} else if !v.ToBoolean() {
		t.Fatal("appReaction must report success")
	}
	var p struct {
		ConversationID string `json:"conversation_id"`
		MessageID      string `json:"message_id"`
		Text           string `json:"text"`
		Reacted        bool   `json:"reacted"`
	}
	if err := json.Unmarshal(rec.payload("message.reaction.set"), &p); err != nil {
		t.Fatal(err)
	}
	if p.ConversationID != "c-1" || p.MessageID != "m-2" || p.Text != "🔥" || !p.Reacted {
		t.Fatalf("reaction payload=%+v", p)
	}
	// 失败时查询类接口返回 null，脚本可安全判空。
	if v, err := rt.RunString(`appUser("u-1") === null`); err != nil {
		t.Fatal(err)
	} else if !v.ToBoolean() {
		t.Fatal("appUser must return null on error")
	}
}
