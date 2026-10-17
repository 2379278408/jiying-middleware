package core

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestSendMessageDryRunAndIdentity(t *testing.T) {
	SetUserSession("user@example.com", "tok")
	defer SetUserSession("", "")
	old := sendUserCall
	defer func() { sendUserCall = old }()
	sendUserCall = func(ctx context.Context, method, path string, body any) ([]byte, error) {
		return []byte(`{"data":{"message":{"id":"m-1"}}}`), nil
	}
	conv := "47f5c179-5d06-42fd-8a06-736866d190e0"
	out, err := SendMessage(context.Background(), SendOptions{ConversationID: conv, Text: "重置", Identity: "auto", DryRun: true})
	if err != nil || out.Identity != "user" || out.Sent {
		t.Fatalf("auto 预演失败: %+v %v", out, err)
	}
	if _, err := SendMessage(context.Background(), SendOptions{ConversationID: conv, Text: "重置", Identity: "app", DryRun: true}); err == nil {
		t.Fatal("非归属会话不应允许应用身份")
	}
	if _, err := SendMessage(context.Background(), SendOptions{ConversationID: conv, Text: "重置", Identity: "auto"}); err != nil {
		t.Fatalf("发送不应失败: %v", err)
	}
}

func TestSendTextAsUserPayload(t *testing.T) {
	SetUserSession("user@example.com", "tok")
	defer SetUserSession("", "")
	old := sendUserCall
	defer func() { sendUserCall = old }()
	var gotPath string
	var gotBody map[string]any
	sendUserCall = func(ctx context.Context, method, path string, body any) ([]byte, error) {
		gotPath = path
		raw, _ := json.Marshal(body)
		_ = json.Unmarshal(raw, &gotBody)
		return []byte(`{"data":{"message":{"id":"m-1"}}}`), nil
	}
	id, err := SendTextAsUser(context.Background(), "47f5c179-5d06-42fd-8a06-736866d190e0", "", "重置")
	if err != nil || id != "m-1" {
		t.Fatalf("发送失败: %q %v", id, err)
	}
	if !strings.HasSuffix(gotPath, "/messages") {
		t.Fatalf("请求路径错误: %s", gotPath)
	}
	body, _ := gotBody["body"].(map[string]any)
	if body["type"] != "text" || body["content"] != "重置" {
		t.Fatalf("消息体错误: %+v", gotBody)
	}
	if cid, _ := gotBody["client_message_id"].(string); len(cid) != 36 {
		t.Fatalf("client_message_id 非法: %q", cid)
	}
}
