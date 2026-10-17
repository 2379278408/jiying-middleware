package core

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

var sendUserCall = UserCall

var (
	errAccountNotLoggedIn  = fmt.Errorf("账号通道未登录")
	errIdentityUnavailable = fmt.Errorf("无法确定发送身份")
)

// SendOptions 统一描述一次控制台发送请求。
type SendOptions struct {
	ConversationID string
	Text           string
	Identity       string // auto | user | app
	ReplyTo        string
	DryRun         bool
}

// SendResult 回传实际使用的身份与发送结果，dry-run 时不产生副作用。
type SendResult struct {
	ConversationID string `json:"conversation_id"`
	Identity       string `json:"identity"`
	Text           string `json:"text"`
	ReplyTo        string `json:"reply_to,omitempty"`
	DryRun         bool   `json:"dry_run"`
	Sent           bool   `json:"sent"`
	MessageID      string `json:"message_id,omitempty"`
	Reason         string `json:"reason,omitempty"`
}

// ConversationCapabilities 描述指定会话可以使用的发送身份。
type ConversationCapabilities struct {
	ConversationID string `json:"conversation_id"`
	CanSendAsUser  bool   `json:"can_send_as_user"`
	CanSendAsApp   bool   `json:"can_send_as_app"`
	Reason         string `json:"reason"`
}

func ownedAppForConv(conversationID string) (string, bool) {
	appConnPool.mu.Lock()
	defer appConnPool.mu.Unlock()
	appID, ok := appConnPool.convApp[conversationID]
	if !ok || appID == "" {
		return "", false
	}
	rt, ok := appConnPool.items[appID]
	if !ok || rt == nil || rt.client == nil || !rt.client.Connected() {
		return appID, false
	}
	return appID, true
}

func conversationCapabilities(conversationID string) ConversationCapabilities {
	appID, appOnline := ownedAppForConv(conversationID)
	appReason := "会话不属于当前应用"
	if appID != "" && !appOnline {
		appReason = "应用通道离线"
	}
	if appOnline {
		appReason = "应用通道可用"
	}
	userOK := userLoggedIn()
	userReason := "账号通道可用"
	if !userOK {
		userReason = "账号通道未登录"
	}
	reason := appReason + "；" + userReason
	return ConversationCapabilities{
		ConversationID: conversationID,
		CanSendAsUser:  userOK,
		CanSendAsApp:   appOnline,
		Reason:         reason,
	}
}

func resolveSendIdentity(conversationID, identity string) (string, error) {
	identity = strings.ToLower(strings.TrimSpace(identity))
	if identity == "" || identity == "auto" {
		cap := conversationCapabilities(conversationID)
		if cap.CanSendAsApp {
			return "app", nil
		}
		if cap.CanSendAsUser {
			return "user", nil
		}
		return "", errIdentityUnavailable
	}
	switch identity {
	case "app":
		if _, ok := ownedAppForConv(conversationID); !ok {
			return "", fmt.Errorf("会话不属于当前应用，无法以应用身份发送")
		}
		return "app", nil
	case "user":
		if !userLoggedIn() {
			return "", errAccountNotLoggedIn
		}
		return "user", nil
	default:
		return "", fmt.Errorf("不支持的发送身份: %s", identity)
	}
}

func newClientMessageID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("relay-%d", time.Now().UnixNano())
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	h := hex.EncodeToString(b[:])
	return h[0:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:32]
}

// SendTextAsUser 以当前账号身份向上游会话发送文本，返回服务端消息 ID。
func SendTextAsUser(ctx context.Context, conversationID, replyTo, text string) (string, error) {
	if !validConversationID(conversationID) {
		return "", fmt.Errorf("会话 ID 不合法")
	}
	if strings.TrimSpace(text) == "" {
		return "", fmt.Errorf("内容为空")
	}
	if !userLoggedIn() {
		return "", errAccountNotLoggedIn
	}
	body := map[string]any{
		"client_message_id": newClientMessageID(),
		"body":              map[string]any{"type": "text", "content": text},
	}
	if replyTo != "" {
		body["reply_to_message_id"] = replyTo
	}
	raw, err := sendUserCall(ctx, http.MethodPost, "/api/client/conversations/"+conversationID+"/messages", body)
	if err != nil {
		snippet := truncateText(string(raw), 200)
		if snippet != "" {
			return "", fmt.Errorf("%v: %s", err, snippet)
		}
		return "", err
	}
	var parsed struct {
		Data struct {
			Message struct {
				ID string `json:"id"`
			} `json:"message"`
		} `json:"data"`
	}
	_ = json.Unmarshal(raw, &parsed)
	return parsed.Data.Message.ID, nil
}

// SendMessage 按指定或自动身份发送文本，dry-run 只做校验与身份解析。
func SendMessage(ctx context.Context, opts SendOptions) (SendResult, error) {
	convID := strings.TrimSpace(opts.ConversationID)
	text := strings.TrimSpace(opts.Text)
	if !validConversationID(convID) {
		return SendResult{}, fmt.Errorf("会话 ID 不合法")
	}
	if text == "" {
		return SendResult{}, fmt.Errorf("内容为空")
	}
	identity, err := resolveSendIdentity(convID, opts.Identity)
	if err != nil {
		return SendResult{}, err
	}
	out := SendResult{
		ConversationID: convID,
		Identity:       identity,
		Text:           text,
		ReplyTo:        strings.TrimSpace(opts.ReplyTo),
		DryRun:         opts.DryRun,
	}
	if opts.DryRun {
		out.Reason = "预演模式：未发起真实发送"
		return out, nil
	}
	if identity == "user" {
		ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
		defer cancel()
		id, err := SendTextAsUser(ctx, convID, out.ReplyTo, text)
		if err != nil {
			return out, err
		}
		out.Sent = true
		out.MessageID = id
		return out, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	if err := SendTextToConv(ctx, convID, out.ReplyTo, text); err != nil {
		return out, err
	}
	out.Sent = true
	return out, nil
}
