package core

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"
)

type fakeSock struct {
	in  chan jiyingEnvelope
	out chan jiyingEnvelope
}

func (f *fakeSock) WriteJSON(v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	var env jiyingEnvelope
	if err := json.Unmarshal(b, &env); err != nil {
		return err
	}
	select {
	case f.out <- env:
		return nil
	case <-time.After(2 * time.Second):
		return errors.New("write timeout")
	}
}

func (f *fakeSock) ReadJSON(v any) error {
	env, ok := <-f.in
	if !ok {
		return errors.New("closed")
	}
	b, _ := json.Marshal(env)
	return json.Unmarshal(b, v)
}

func (f *fakeSock) Close() error { return nil }

func TestJiyingMessageRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("JIYING_DATA", dir)
	BucketApp, _ = MakeBucket("jiying-test", 6)
	BucketApp.Set("masters", "user-1")

	sock := &fakeSock{in: make(chan jiyingEnvelope, 8), out: make(chan jiyingEnvelope, 8)}
	client := NewJiyingClient(JiyingConfig{WSURL: "ws://example", AppID: "app", Secret: "sec"}, nil)
	client.conn = sock
	activeJiying = client
	defer func() { activeJiying = nil }()

	old := Senders
	Senders = make(chan Sender, 1)
	defer func() { Senders = old }()

	// 记录中间件发出的请求，并对每个请求立即回响应。
	var reqMu sync.Mutex
	var reqs []jiyingEnvelope
	go func() {
		for req := range sock.out {
			reqMu.Lock()
			reqs = append(reqs, req)
			reqMu.Unlock()
			_ = client.HandleInbound(context.Background(), jiyingEnvelope{
				Kind: "response", ReplyTo: req.ID, OK: true, Payload: json.RawMessage(`{}`),
			})
		}
	}()
	gotMethods := func() []string {
		reqMu.Lock()
		defer reqMu.Unlock()
		out := make([]string, 0, len(reqs))
		for _, r := range reqs {
			out = append(out, r.Method)
		}
		return out
	}

	payload, _ := json.Marshal(map[string]any{
		"conversation": map[string]any{"id": "conv-1", "type": "app"},
		"sender":       map[string]any{"id": "user-1", "name": "Ada"},
		"message":      map[string]any{"id": "m1", "summary": "帮助", "body": map[string]any{"type": "text", "content": "帮助"}},
	})
	err := client.HandleInbound(context.Background(), jiyingEnvelope{
		Kind: "event", Event: "message.created", Cursor: 7, Payload: payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	var s Sender
	select {
	case s = <-Senders:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}
	if s.GetMsg() != "帮助" {
		t.Fatalf("msg=%s", s.GetMsg())
	}
	if !s.Reply("pong") {
		t.Fatal("reply failed")
	}
	// 业务处理结束后才确认事件。
	finishSender(s)

	// 确认时序：先回复消息，再确认事件（ack 不得早于业务完成）。
	deadline := time.Now().Add(2 * time.Second)
	for {
		methods := gotMethods()
		if len(methods) >= 2 {
			if methods[0] != "message.send" || methods[1] != "events.ack" {
				t.Fatalf("methods=%v", methods)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("timeout waiting ack, methods=%v", methods)
		}
		time.Sleep(5 * time.Millisecond)
	}
	reqMu.Lock()
	var acked int64
	for _, r := range reqs {
		if r.Method == "events.ack" {
			var p struct {
				Cursor int64 `json:"cursor"`
			}
			_ = json.Unmarshal(r.Payload, &p)
			acked = p.Cursor
		}
	}
	reqMu.Unlock()
	if acked != 7 {
		t.Fatalf("acked cursor=%d", acked)
	}

	err = client.HandleInbound(context.Background(), jiyingEnvelope{
		Kind: "event", Event: "message.created", Cursor: 7, Payload: payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-Senders:
		t.Fatal("duplicate event delivered")
	case <-time.After(50 * time.Millisecond):
	}
}

func TestStatusIgnoresAck(t *testing.T) {
	client := NewJiyingClient(JiyingConfig{AppID: "app"}, nil)
	err := client.HandleInbound(context.Background(), jiyingEnvelope{Kind: "event", Event: "conversation.status"})
	if err != nil {
		t.Fatal(err)
	}
}

func TestCallRetriesWithSameRequestID(t *testing.T) {
	sock := &fakeSock{in: make(chan jiyingEnvelope, 8), out: make(chan jiyingEnvelope, 8)}
	client := NewJiyingClient(JiyingConfig{AppID: "app"}, nil)
	client.conn = sock
	go func() {
		first := <-sock.out
		_ = client.HandleInbound(context.Background(), jiyingEnvelope{
			Kind: "response", ReplyTo: first.ID, OK: false,
			Error: &jiyingError{Code: "internal_error", Message: "boom"},
		})
		second := <-sock.out
		if second.ID != first.ID {
			t.Errorf("retry must reuse request id: %s != %s", second.ID, first.ID)
		}
		_ = client.HandleInbound(context.Background(), jiyingEnvelope{
			Kind: "response", ReplyTo: second.ID, OK: true, Payload: json.RawMessage(`{"ok":true}`),
		})
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	raw, err := client.Call(ctx, "message.send", map[string]any{"x": 1})
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != `{"ok":true}` {
		t.Fatalf("payload=%s", raw)
	}
}

func TestTopicClosedFailsPendingTasks(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("JIYING_DATA", dir)
	BucketApp, _ = MakeBucket("jiying-topic", 6)
	task, err := EnqueueTask(Task{Feature: "帮助", UserID: "u1", ConversationID: "topic-1", Confirm: true})
	if err != nil {
		t.Fatal(err)
	}
	failTasksForConversation("topic-1")
	if !strings.Contains(ListTasks(5), task.ID+" 帮助 failed") {
		t.Fatalf("task not finished: %s", ListTasks(5))
	}
}

func TestStartSenderStatusPingsAppConversation(t *testing.T) {
	sock := &fakeSock{in: make(chan jiyingEnvelope, 8), out: make(chan jiyingEnvelope, 8)}
	client := NewJiyingClient(JiyingConfig{AppID: "app"}, nil)
	client.conn = sock
	activeJiying = client
	defer func() { activeJiying = nil }()

	got := make(chan jiyingEnvelope, 1)
	go func() {
		req := <-sock.out
		got <- req
		_ = client.HandleInbound(context.Background(), jiyingEnvelope{
			Kind: "response", ReplyTo: req.ID, OK: true, Payload: json.RawMessage(`{}`),
		})
	}()
	s := &JiyingSender{client: client, ChatID: "conv-1", ConvType: "app"}
	stop := startSenderStatus(s)
	defer stop()

	select {
	case req := <-got:
		if req.Method != "conversation.status" {
			t.Fatalf("method=%s", req.Method)
		}
		var p struct {
			ConversationID string `json:"conversation_id"`
			Status         string `json:"status"`
		}
		_ = json.Unmarshal(req.Payload, &p)
		if p.ConversationID != "conv-1" || p.Status == "" {
			t.Fatalf("payload=%s", req.Payload)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no status ping")
	}
}

func TestCommandPanicStillAcks(t *testing.T) {
	sock := &fakeSock{in: make(chan jiyingEnvelope, 8), out: make(chan jiyingEnvelope, 8)}
	client := NewJiyingClient(JiyingConfig{AppID: "app"}, nil)
	client.conn = sock
	activeJiying = client
	defer func() { activeJiying = nil }()

	cmdRouter.Store("panic-test", func() { panic("boom") })
	defer cmdRouter.Delete("panic-test")
	snapshot := Functions
	Functions = append([]*Function(nil), snapshot...)
	Functions = append(Functions, &Function{Title: "panic-test", Handle: "panic-test", Origin: true})
	defer func() { Functions = snapshot }()

	got := make(chan jiyingEnvelope, 1)
	go func() {
		for req := range sock.out {
			_ = client.HandleInbound(context.Background(), jiyingEnvelope{
				Kind: "response", ReplyTo: req.ID, OK: true, Payload: json.RawMessage(`{}`),
			})
			if req.Method == "events.ack" {
				got <- req
				return
			}
		}
	}()
	s := &JiyingSender{client: client, ChatID: "conv-1", MsgID: "m1", Text: "panic-test", ackCursor: 9}
	if !HandleMessage(s) {
		t.Fatal("command not matched")
	}
	select {
	case req := <-got:
		if req.Method != "events.ack" {
			t.Fatalf("method=%s", req.Method)
		}
		var p struct {
			Cursor int64 `json:"cursor"`
		}
		_ = json.Unmarshal(req.Payload, &p)
		if p.Cursor != 9 {
			t.Fatalf("cursor=%d", p.Cursor)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("panic path must still ack")
	}
}

func TestFeatureInstallAndTask(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("JIYING_DATA", dir)
	BucketApp, _ = MakeBucket("jiying-test2", 6)
	BucketApp.Set("masters", "u1")
	initFeatures()
	if err := UpsertFeature(Feature{Name: "回声", Kind: "builtin", Target: "help", Scope: "public", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	got, err := executeFeature(context.Background(), "回声", "", "u1", "")
	if err != nil || got == "" {
		t.Fatalf("exec %q %v", got, err)
	}
	t1, err := EnqueueTask(Task{Feature: "帮助", UserID: "u1"})
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(40 * time.Millisecond)
	kickTaskLoop()
	time.Sleep(80 * time.Millisecond)
	if !strings.Contains(ListTasks(5), t1.ID) {
		t.Fatalf("missing task %s", ListTasks(5))
	}
}

func TestInstallScriptRejectsEscape(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("JIYING_RES", dir)
	os.MkdirAll(filepath.Join(dir, "scripts"), 0o755)
	if err := InstallScript("../x.js", "result='no'"); err == nil {
		t.Fatal("path escape should fail")
	}
	if _, err := os.Stat(filepath.Join(dir, "x.js")); err == nil {
		t.Fatal("escaped file written")
	}
	if err := InstallScript("ok.js", "result='yes'"); err != nil {
		t.Fatal(err)
	}
}

func TestParseAgent(t *testing.T) {
	features.items["状态"] = &Feature{Name: "状态", Enabled: true, Kind: "builtin", Target: "status"}
	name, args, ok := ParseAgent("跑 状态 现在")
	if !ok || name != "状态" || args != "现在" {
		t.Fatalf("%s %s %v", name, args, ok)
	}
}

func TestMessageCreatedEnrichesSender(t *testing.T) {
	client := NewJiyingClient(JiyingConfig{AppID: "app"}, nil)

	old := Senders
	Senders = make(chan Sender, 1)
	defer func() { Senders = old }()

	payload, _ := json.Marshal(map[string]any{
		"conversation": map[string]any{"id": "conv-9", "type": "group", "name": "研发群"},
		"sender":       map[string]any{"id": "u-9", "name": "ada", "nickname": "Ada", "email": "ada@example.com", "type": "user"},
		"message": map[string]any{
			"id": "m-9", "seq": 42, "summary": "图片", "reply_to_message_id": "m-1",
			"body": map[string]any{"type": "image", "file_id": "file-1", "caption": "看这张图"},
		},
	})
	deferred, err := client.dispatchEvent(context.Background(), jiyingEnvelope{
		Kind: "event", Event: "message.created", Cursor: 3, Payload: payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !deferred {
		t.Fatal("message.created should defer ack")
	}

	var s Sender
	select {
	case s = <-Senders:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}
	js, ok := s.(*JiyingSender)
	if !ok {
		t.Fatalf("sender type %T", s)
	}
	if js.GetMsg() != "看这张图" {
		t.Fatalf("msg=%q", js.GetMsg())
	}
	if js.DisplayName() != "Ada" {
		t.Fatalf("name=%q", js.DisplayName())
	}
	if js.SenderEmail != "ada@example.com" || js.ConvName != "研发群" {
		t.Fatalf("meta=%+v", js)
	}
	if js.ReplyToMsgID != "m-1" || !js.IsReply() {
		t.Fatal("reply meta missing")
	}
	if !js.IsMedia() || js.Body.FileID != "file-1" {
		t.Fatalf("media=%+v", js.Body)
	}
	if !js.IsGroup() {
		t.Fatal("expected group conversation")
	}
	if js.GetImtype() != "jiying" {
		t.Fatalf("imtype=%q", js.GetImtype())
	}
	if js.Seq != 42 {
		t.Fatalf("seq=%d", js.Seq)
	}
}

func TestMessageBodyPlainText(t *testing.T) {
	cases := []struct {
		name string
		body jiyingMessageBody
		want string
	}{
		{"text", jiyingMessageBody{Type: "text", Content: "帮助"}, "帮助"},
		{"image-caption", jiyingMessageBody{Type: "image", FileID: "f1", Caption: "看这张"}, "看这张"},
		{"image-plain", jiyingMessageBody{Type: "image", FileID: "f1"}, "[图片]"},
		{"video", jiyingMessageBody{Type: "video", FileID: "f1"}, "[视频]"},
		{"voice-transcript", jiyingMessageBody{Type: "voice", FileID: "f1", Transcript: "你好"}, "你好"},
		{"voice-plain", jiyingMessageBody{Type: "voice", FileID: "f1"}, "[语音]"},
		{"file", jiyingMessageBody{Type: "file", FileID: "f1", Name: "report.txt"}, "report.txt"},
		{"file-plain", jiyingMessageBody{Type: "file", FileID: "f1"}, "[文件]"},
		{"link-title", jiyingMessageBody{Type: "link", URL: "https://x", Title: "标题"}, "标题"},
		{"link-url", jiyingMessageBody{Type: "link", URL: "https://x"}, "https://x"},
		{"card", jiyingMessageBody{Type: "card", Title: "卡片"}, "卡片"},
		{"empty", jiyingMessageBody{}, ""},
	}
	for _, tc := range cases {
		if got := tc.body.plainText(); got != tc.want {
			t.Errorf("%s: got %q want %q", tc.name, got, tc.want)
		}
	}
}

func TestReactionReceiptOnCommand(t *testing.T) {
	sock := &fakeSock{in: make(chan jiyingEnvelope, 8), out: make(chan jiyingEnvelope, 8)}
	client := NewJiyingClient(JiyingConfig{AppID: "app"}, nil)
	client.conn = sock
	activeJiying = client
	defer func() { activeJiying = nil }()

	cmdRouter.Store("receipt-test", func() {})
	defer cmdRouter.Delete("receipt-test")
	snapshot := Functions
	Functions = append([]*Function(nil), snapshot...)
	Functions = append(Functions, &Function{Title: "receipt-test", Handle: "receipt-test", Origin: true})
	defer func() { Functions = snapshot }()

	var mu sync.Mutex
	var reqs []jiyingEnvelope
	go func() {
		for req := range sock.out {
			mu.Lock()
			reqs = append(reqs, req)
			mu.Unlock()
			_ = client.HandleInbound(context.Background(), jiyingEnvelope{
				Kind: "response", ReplyTo: req.ID, OK: true, Payload: json.RawMessage(`{}`),
			})
		}
	}()

	s := &JiyingSender{client: client, ChatID: "conv-1", MsgID: "m1", Text: "receipt-test"}
	if !HandleMessage(s) {
		t.Fatal("command not matched")
	}
	deadline := time.Now().Add(2 * time.Second)
	for {
		mu.Lock()
		var reaction *jiyingEnvelope
		for i := range reqs {
			if reqs[i].Method == "message.reaction.set" {
				reaction = &reqs[i]
			}
		}
		mu.Unlock()
		if reaction != nil {
			var p struct {
				ConversationID string `json:"conversation_id"`
				MessageID      string `json:"message_id"`
				Text           string `json:"text"`
				Reacted        bool   `json:"reacted"`
			}
			_ = json.Unmarshal(reaction.Payload, &p)
			if p.ConversationID != "conv-1" || p.MessageID != "m1" {
				t.Fatalf("reaction target=%s", reaction.Payload)
			}
			if p.Text != receiptReaction || !p.Reacted {
				t.Fatalf("reaction content=%s", reaction.Payload)
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("no reaction receipt sent")
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestRateLimitedRetriesWithSameRequestID(t *testing.T) {
	sock := &fakeSock{in: make(chan jiyingEnvelope, 8), out: make(chan jiyingEnvelope, 8)}
	client := NewJiyingClient(JiyingConfig{AppID: "app"}, nil)
	client.conn = sock
	before := JiyingRateLimitedTotal()
	go func() {
		first := <-sock.out
		_ = client.HandleInbound(context.Background(), jiyingEnvelope{
			Kind: "response", ReplyTo: first.ID, OK: false,
			Error: &jiyingError{Code: "rate_limited", Message: "too many"},
		})
		second := <-sock.out
		if second.ID != first.ID {
			t.Errorf("retry must reuse request id: %s != %s", second.ID, first.ID)
		}
		_ = client.HandleInbound(context.Background(), jiyingEnvelope{
			Kind: "response", ReplyTo: second.ID, OK: true, Payload: json.RawMessage(`{}`),
		})
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := client.Call(ctx, "message.send", map[string]any{"x": 1}); err != nil {
		t.Fatal(err)
	}
	if got := JiyingRateLimitedTotal(); got != before+1 {
		t.Fatalf("rate limited counter=%d want=%d", got, before+1)
	}
}

func TestMessageCreatedTopicContext(t *testing.T) {
	client := NewJiyingClient(JiyingConfig{AppID: "app"}, nil)
	old := Senders
	Senders = make(chan Sender, 1)
	defer func() { Senders = old }()

	payload, _ := json.Marshal(map[string]any{
		"conversation": map[string]any{
			"id": "topic-1", "type": "topic", "name": "发布计划",
			"created_by_app_id": "app-owner",
			"parent":            map[string]any{"id": "group-1", "type": "group", "name": "产品讨论组"},
			"source_message":    map[string]any{"id": "src-1", "seq": 42},
		},
		"sender":  map[string]any{"id": "u1", "name": "Ada"},
		"message": map[string]any{"id": "m1", "summary": "hi", "body": map[string]any{"type": "text", "content": "hi"}},
	})
	if _, err := client.dispatchEvent(context.Background(), jiyingEnvelope{
		Kind: "event", Event: "message.created", Cursor: 5, Payload: payload,
	}); err != nil {
		t.Fatal(err)
	}
	var s Sender
	select {
	case s = <-Senders:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}
	js, ok := s.(*JiyingSender)
	if !ok {
		t.Fatalf("sender type %T", s)
	}
	if !js.IsTopic() {
		t.Fatal("expected topic conversation")
	}
	if js.ParentConvID != "group-1" || js.ParentConvType != "group" || js.ParentConvName != "产品讨论组" {
		t.Fatalf("parent meta=%+v", js)
	}
	if js.SourceMessageID != "src-1" || js.CreatedByAppID != "app-owner" {
		t.Fatalf("source meta=%+v", js)
	}
}

func TestValidateChoice(t *testing.T) {
	valid := []map[string]string{{"id": "today", "label": "今天"}, {"id": "tomorrow", "label": "明天"}}
	if err := validateChoice("请选择部署时间", valid); err != nil {
		t.Fatalf("valid choice rejected: %v", err)
	}
	bad := []struct {
		name    string
		content string
		opts    []map[string]string
	}{
		{"empty-content", "   ", valid},
		{"too-few-options", "x", []map[string]string{{"id": "a", "label": "A"}}},
		{"bad-option-id", "x", []map[string]string{{"id": "a b", "label": "A"}, {"id": "c", "label": "C"}}},
		{"duplicate-option-id", "x", []map[string]string{{"id": "a", "label": "A"}, {"id": "a", "label": "B"}}},
		{"missing-label", "x", []map[string]string{{"id": "a", "label": " "}, {"id": "c", "label": "C"}}},
		{"newline-label", "x", []map[string]string{{"id": "a", "label": "A\nB"}, {"id": "c", "label": "C"}}},
	}
	for _, tc := range bad {
		if err := validateChoice(tc.content, tc.opts); err == nil {
			t.Errorf("%s: expected error", tc.name)
		}
	}
}

func TestJiyingDialErrorMessage(t *testing.T) {
	cases := map[int]string{
		400: "应用 ID 格式错误",
		401: "连接密钥错误",
		403: "应用已停用",
		500: "HTTP 500",
	}
	for status, want := range cases {
		got := (&jiyingDialError{Status: status}).Error()
		if !strings.Contains(got, want) {
			t.Errorf("status %d: got %q want contains %q", status, got, want)
		}
	}
}

func TestClampOutboundText(t *testing.T) {
	if got := clampOutboundText("short"); got != "short" {
		t.Fatalf("short text changed: %q", got)
	}
	long := strings.Repeat("字", maxOutboundTextBytes)
	got := clampOutboundText(long)
	if len(got) > maxOutboundTextBytes {
		t.Fatalf("clamped length=%d exceeds limit=%d", len(got), maxOutboundTextBytes)
	}
	if !utf8.ValidString(got) {
		t.Fatal("clamped text is not valid UTF-8")
	}
	if !strings.HasSuffix(got, "（内容过长已截断）") {
		t.Fatalf("missing truncation note: %q", got[len(got)-32:])
	}
}

func TestChoiceResume(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("JIYING_DATA", dir)
	BucketApp, _ = MakeBucket("jiying-choice", 6)
	initFeatures()
	t1, err := EnqueueTask(Task{Feature: "帮助", UserID: "u1", Confirm: true})
	if err != nil {
		t.Fatal(err)
	}
	bindChoice("choice-1", t1.ID)
	if err := resumeChoice(context.Background(), "choice-1", "u1", "c1", []string{"no"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(ListTasks(5), "failed") {
		t.Fatalf("cancel not failed: %s", ListTasks(5))
	}
}
