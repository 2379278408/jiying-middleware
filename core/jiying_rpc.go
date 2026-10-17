package core

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/dop251/goja"
)

// 本文件按《第三方应用开发指南》（协议 v1）实现应用通道的完整 RPC 面：
// 目标类型化发送、查询、话题与群管理，以及向脚本运行时暴露的只读/回执能力。
// 未在指南中开放的方法（runas、message.send_as_user、entity_card 等）一律不实现。

// rpcTimeout 是单次应用通道 RPC 的等待上限。
const rpcTimeout = 15 * time.Second

// rpcTarget 描述 message.send 的目标对象。
type rpcTarget struct {
	Type           string
	UserID         string
	ConversationID string
}

// userTarget 构造一对一向用户发送的目标。
func userTarget(userID string) rpcTarget { return rpcTarget{Type: "user", UserID: userID} }

// convTarget 构造按会话发送的目标，kind 取 app/group/topic/conversation。
func convTarget(kind, conversationID string) rpcTarget {
	if kind == "" {
		kind = "conversation"
	}
	return rpcTarget{Type: kind, ConversationID: conversationID}
}

func (t rpcTarget) payload() (map[string]any, error) {
	switch t.Type {
	case "user":
		if strings.TrimSpace(t.UserID) == "" {
			return nil, fmt.Errorf("缺少用户 ID")
		}
		return map[string]any{"type": "user", "user_id": t.UserID}, nil
	case "app", "group", "topic", "conversation":
		if strings.TrimSpace(t.ConversationID) == "" {
			return nil, fmt.Errorf("缺少会话 ID")
		}
		return map[string]any{"type": t.Type, "conversation_id": t.ConversationID}, nil
	default:
		return nil, fmt.Errorf("不支持的目标类型 %q", t.Type)
	}
}

// SendTo 以应用身份向指定目标发送任意合法消息体，返回服务端消息 ID。
func (c *JiyingClient) SendTo(ctx context.Context, target rpcTarget, replyTo string, message map[string]any) (string, error) {
	if len(message) == 0 {
		return "", fmt.Errorf("消息内容为空")
	}
	tp, err := target.payload()
	if err != nil {
		return "", err
	}
	body := map[string]any{"target": tp, "message": message}
	if replyTo != "" {
		body["reply_to_message_id"] = replyTo
	}
	raw, err := c.Call(ctx, "message.send", body)
	if err != nil {
		return "", err
	}
	var resp struct {
		Message struct {
			ID string `json:"id"`
		} `json:"message"`
	}
	_ = json.Unmarshal(raw, &resp)
	return resp.Message.ID, nil
}

// SendToConv 按会话归属应用通道向指定目标发送消息。
func SendToConv(ctx context.Context, conversationID string, target rpcTarget, replyTo string, message map[string]any) (string, error) {
	c := clientForConv(conversationID)
	if c == nil {
		return "", errJiyingNotConnected
	}
	return c.SendTo(ctx, target, replyTo, message)
}

// ---- 查询类 ----

// JiyingUser 是 users.get 返回的用户资料。
type JiyingUser struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Nickname string `json:"nickname"`
	Email    string `json:"email"`
	Avatar   string `json:"avatar"`
}

// UsersGet 查询应用可见用户。
func (c *JiyingClient) UsersGet(ctx context.Context, userID string) (JiyingUser, error) {
	var out struct {
		User JiyingUser `json:"user"`
	}
	if strings.TrimSpace(userID) == "" {
		return out.User, fmt.Errorf("缺少用户 ID")
	}
	raw, err := c.Call(ctx, "users.get", map[string]any{"user_id": userID})
	if err != nil {
		return out.User, err
	}
	_ = json.Unmarshal(raw, &out)
	return out.User, nil
}

// JiyingApp 是 apps.get 返回的应用资料（不含连接密钥）。
type JiyingApp struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Avatar      string `json:"avatar"`
	Enabled     bool   `json:"enabled"`
	Online      bool   `json:"online"`
	Visibility  string `json:"visibility"`
}

// AppsGet 查询应用基本资料。
func (c *JiyingClient) AppsGet(ctx context.Context, appID string) (JiyingApp, error) {
	var out struct {
		App JiyingApp `json:"app"`
	}
	if strings.TrimSpace(appID) == "" {
		return out.App, fmt.Errorf("缺少应用 ID")
	}
	raw, err := c.Call(ctx, "apps.get", map[string]any{"app_id": appID})
	if err != nil {
		return out.App, err
	}
	_ = json.Unmarshal(raw, &out)
	return out.App, nil
}

// JiyingConversation 是 conversations.list 返回的会话摘要。
type JiyingConversation struct {
	ID           string `json:"conversation_id"`
	Name         string `json:"name"`
	Type         string `json:"type"`
	MemberCount  int    `json:"member_count"`
	LastActiveAt string `json:"last_active_at"`
}

// ListConversations 列出应用参与的会话；limit 默认 20、最大 100。
func (c *JiyingClient) ListConversations(ctx context.Context, keyword string, limit int) ([]JiyingConversation, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	payload := map[string]any{"limit": limit}
	if strings.TrimSpace(keyword) != "" {
		payload["keyword"] = keyword
	}
	raw, err := c.Call(ctx, "conversations.list", payload)
	if err != nil {
		return nil, err
	}
	var out struct {
		Conversations []JiyingConversation `json:"conversations"`
	}
	_ = json.Unmarshal(raw, &out)
	return out.Conversations, nil
}

// JiyingMessage 是 conversation.messages.list 返回的消息摘要。
type JiyingMessage struct {
	ID               string          `json:"id"`
	Seq              int64           `json:"seq"`
	ReplyToMessageID string          `json:"reply_to_message_id"`
	Summary          string          `json:"summary"`
	CreatedAt        string          `json:"created_at"`
	Body             json.RawMessage `json:"body"`
	Sender           struct {
		ID   string `json:"id"`
		Type string `json:"type"`
	} `json:"sender"`
}

// plainText 返回消息可用于上下文的纯文本：优先消息体，其次摘要。
func (m JiyingMessage) plainText() string {
	var b jiyingMessageBody
	if len(m.Body) > 0 && json.Unmarshal(m.Body, &b) == nil {
		if t := strings.TrimSpace(b.plainText()); t != "" {
			return t
		}
	}
	return strings.TrimSpace(m.Summary)
}

// messagesToChatHistory 把服务端历史转成 Agent 上下文：跳过当前消息（作为 prompt
// 单独传入），按发送方类型标注角色，应用发出的消息视为 assistant。
func messagesToChatHistory(msgs []JiyingMessage, selfAppID string, uptoSeq int64) []chatMessage {
	out := make([]chatMessage, 0, len(msgs))
	for _, m := range msgs {
		if uptoSeq > 0 && m.Seq >= uptoSeq {
			continue
		}
		text := m.plainText()
		if text == "" {
			continue
		}
		role := "user"
		if m.Sender.Type == "app" || (selfAppID != "" && m.Sender.ID == selfAppID) {
			role = "assistant"
		}
		out = append(out, trimAgentMessage(chatMessage{Role: role, Content: text}))
	}
	return out
}

// ListMessages 读取应用可见的会话历史，按 seq 升序返回。
// beforeOrEqualSeq 必填且为正整数；limit 默认 30、最大 100。
func (c *JiyingClient) ListMessages(ctx context.Context, conversationID string, beforeOrEqualSeq int64, limit int) ([]JiyingMessage, error) {
	if strings.TrimSpace(conversationID) == "" {
		return nil, fmt.Errorf("缺少会话 ID")
	}
	if beforeOrEqualSeq <= 0 {
		return nil, fmt.Errorf("before_or_equal_seq 必须为正整数")
	}
	if limit <= 0 {
		limit = 30
	}
	if limit > 100 {
		limit = 100
	}
	raw, err := c.Call(ctx, "conversation.messages.list", map[string]any{
		"conversation_id":     conversationID,
		"before_or_equal_seq": beforeOrEqualSeq,
		"limit":               limit,
	})
	if err != nil {
		return nil, err
	}
	var out struct {
		Messages []JiyingMessage `json:"messages"`
	}
	_ = json.Unmarshal(raw, &out)
	return out.Messages, nil
}

// ---- 话题类 ----

// JiyingTopic 是话题创建/查询返回的状态。
type JiyingTopic struct {
	ConversationID       string `json:"-"`
	ParentConversationID string `json:"parent_conversation_id"`
	SourceMessageID      string `json:"source_message_id"`
	LastMessageSeq       int64  `json:"last_message_seq"`
	Archived             bool   `json:"archived"`
}

func parseTopic(raw json.RawMessage) JiyingTopic {
	var out struct {
		Conversation struct {
			ID string `json:"id"`
		} `json:"conversation"`
		ParentConversationID string `json:"parent_conversation_id"`
		SourceMessageID      string `json:"source_message_id"`
		LastMessageSeq       int64  `json:"last_message_seq"`
		Archived             bool   `json:"archived"`
	}
	_ = json.Unmarshal(raw, &out)
	return JiyingTopic{
		ConversationID:       out.Conversation.ID,
		ParentConversationID: out.ParentConversationID,
		SourceMessageID:      out.SourceMessageID,
		LastMessageSeq:       out.LastMessageSeq,
		Archived:             out.Archived,
	}
}

// TopicCreate 基于一条可见历史消息创建或复用话题。
func (c *JiyingClient) TopicCreate(ctx context.Context, conversationID, sourceMessageID string) (JiyingTopic, error) {
	if strings.TrimSpace(conversationID) == "" || strings.TrimSpace(sourceMessageID) == "" {
		return JiyingTopic{}, fmt.Errorf("缺少父会话或来源消息 ID")
	}
	raw, err := c.Call(ctx, "conversation.topic.create", map[string]any{
		"conversation_id":   conversationID,
		"source_message_id": sourceMessageID,
	})
	if err != nil {
		return JiyingTopic{}, err
	}
	return parseTopic(raw), nil
}

// TopicGet 查询当前应用已参与的话题状态。
func (c *JiyingClient) TopicGet(ctx context.Context, conversationID string) (JiyingTopic, error) {
	if strings.TrimSpace(conversationID) == "" {
		return JiyingTopic{}, fmt.Errorf("缺少话题会话 ID")
	}
	raw, err := c.Call(ctx, "conversation.topic.get", map[string]any{"conversation_id": conversationID})
	if err != nil {
		return JiyingTopic{}, err
	}
	return parseTopic(raw), nil
}

// TopicClose 关闭由当前应用创建的话题；expectedLastSeq 来自 TopicGet。
func (c *JiyingClient) TopicClose(ctx context.Context, conversationID string, expectedLastSeq int64) error {
	if strings.TrimSpace(conversationID) == "" {
		return fmt.Errorf("缺少话题会话 ID")
	}
	if expectedLastSeq < 0 {
		return fmt.Errorf("expected_last_message_seq 不能为负")
	}
	_, err := c.Call(ctx, "conversation.topic.close", map[string]any{
		"conversation_id":           conversationID,
		"expected_last_message_seq": expectedLastSeq,
	})
	return err
}

// ---- 群聊类 ----

// validateGroupName 校验群名称（1~120 字符）。
func validateGroupName(name string) error {
	if n := len([]rune(strings.TrimSpace(name))); n == 0 || n > 120 {
		return fmt.Errorf("群名称需为 1~120 字符")
	}
	return nil
}

// GroupCreate 创建由当前应用担任群主的群聊。
func (c *JiyingClient) GroupCreate(ctx context.Context, name string, memberIDs, appIDs []string) (json.RawMessage, error) {
	if err := validateGroupName(name); err != nil {
		return nil, err
	}
	if len(memberIDs) == 0 {
		return nil, fmt.Errorf("至少需要一名群成员")
	}
	payload := map[string]any{"name": name, "member_ids": memberIDs}
	if len(appIDs) > 0 {
		payload["app_ids"] = appIDs
	}
	raw, err := c.Call(ctx, "group_conversations.create", payload)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(raw), nil
}

// GroupGet 查询当前应用已加入的群聊详情。
func (c *JiyingClient) GroupGet(ctx context.Context, conversationID string) (json.RawMessage, error) {
	if strings.TrimSpace(conversationID) == "" {
		return nil, fmt.Errorf("缺少群聊 ID")
	}
	raw, err := c.Call(ctx, "group_conversations.get", map[string]any{"conversation_id": conversationID})
	if err != nil {
		return nil, err
	}
	return json.RawMessage(raw), nil
}

// GroupUpdate 修改群名称或群公告；两者只能其一。
func (c *JiyingClient) GroupUpdate(ctx context.Context, conversationID, name, announcement string) (json.RawMessage, error) {
	if strings.TrimSpace(conversationID) == "" {
		return nil, fmt.Errorf("缺少群聊 ID")
	}
	hasName := strings.TrimSpace(name) != ""
	hasAnnounce := announcement != ""
	if hasName == hasAnnounce {
		return nil, fmt.Errorf("名称与群公告只能修改其一")
	}
	payload := map[string]any{"conversation_id": conversationID}
	if hasName {
		if err := validateGroupName(name); err != nil {
			return nil, err
		}
		payload["name"] = name
	} else {
		if n := len([]rune(strings.TrimSpace(announcement))); n > 200 {
			return nil, fmt.Errorf("群公告最多 200 字符")
		}
		payload["announcement"] = announcement
	}
	raw, err := c.Call(ctx, "group_conversations.update", payload)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(raw), nil
}

// GroupDissolve 解散群聊，仅应用群主可调用。
func (c *JiyingClient) GroupDissolve(ctx context.Context, conversationID string) error {
	if strings.TrimSpace(conversationID) == "" {
		return fmt.Errorf("缺少群聊 ID")
	}
	_, err := c.Call(ctx, "group_conversations.dissolve", map[string]any{"conversation_id": conversationID})
	return err
}

// JiyingGroupMember 是群成员分页查询返回的成员。
type JiyingGroupMember struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Name     string `json:"name"`
	Nickname string `json:"nickname"`
	Role     string `json:"role"`
	JoinedAt string `json:"joined_at"`
}

// GroupMembers 分页查询群成员；page 默认 1，pageSize 默认 100、最大 500。
func (c *JiyingClient) GroupMembers(ctx context.Context, conversationID string, page, pageSize int) ([]JiyingGroupMember, int, error) {
	if strings.TrimSpace(conversationID) == "" {
		return nil, 0, fmt.Errorf("缺少群聊 ID")
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 100
	}
	if pageSize > 500 {
		pageSize = 500
	}
	raw, err := c.Call(ctx, "group_conversations.members.list", map[string]any{
		"conversation_id": conversationID,
		"page":            page,
		"page_size":       pageSize,
	})
	if err != nil {
		return nil, 0, err
	}
	var out struct {
		Members []JiyingGroupMember `json:"members"`
		Total   int                 `json:"total"`
	}
	_ = json.Unmarshal(raw, &out)
	return out.Members, out.Total, nil
}

// GroupAddMembers 向群聊添加用户或应用，两者至少其一非空。
func (c *JiyingClient) GroupAddMembers(ctx context.Context, conversationID string, memberIDs, appIDs []string) error {
	if strings.TrimSpace(conversationID) == "" {
		return fmt.Errorf("缺少群聊 ID")
	}
	if len(memberIDs) == 0 && len(appIDs) == 0 {
		return fmt.Errorf("至少提供一名用户或应用")
	}
	payload := map[string]any{"conversation_id": conversationID}
	if len(memberIDs) > 0 {
		payload["member_ids"] = memberIDs
	}
	if len(appIDs) > 0 {
		payload["app_ids"] = appIDs
	}
	_, err := c.Call(ctx, "group_conversations.members.add", payload)
	return err
}

// GroupRemoveMember 移除群成员；memberType 取 user 或 app。
func (c *JiyingClient) GroupRemoveMember(ctx context.Context, conversationID, memberType, memberID string) error {
	if strings.TrimSpace(conversationID) == "" || strings.TrimSpace(memberID) == "" {
		return fmt.Errorf("缺少群聊或成员 ID")
	}
	if memberType != "user" && memberType != "app" {
		return fmt.Errorf("member_type 只能是 user 或 app")
	}
	_, err := c.Call(ctx, "group_conversations.members.remove", map[string]any{
		"conversation_id": conversationID,
		"member_type":     memberType,
		"member_id":       memberID,
	})
	return err
}

// GroupSetRole 设置或取消群管理员；role 取 admin 或 member。
func (c *JiyingClient) GroupSetRole(ctx context.Context, conversationID, memberType, memberID, role string) error {
	if strings.TrimSpace(conversationID) == "" || strings.TrimSpace(memberID) == "" {
		return fmt.Errorf("缺少群聊或成员 ID")
	}
	if memberType != "user" && memberType != "app" {
		return fmt.Errorf("member_type 只能是 user 或 app")
	}
	if role != "admin" && role != "member" {
		return fmt.Errorf("role 只能是 admin 或 member")
	}
	_, err := c.Call(ctx, "group_conversations.members.set_role", map[string]any{
		"conversation_id": conversationID,
		"member_type":     memberType,
		"member_id":       memberID,
		"role":            role,
	})
	return err
}

// ---- 脚本运行时暴露 ----

// appClientForScript 返回脚本可用的应用通道客户端：优先按会话归属，否则默认应用。
func appClientForScript(convID string) *JiyingClient {
	if convID != "" {
		if c := clientForConv(convID); c != nil {
			return c
		}
	}
	return currentJiying()
}

// registerAppApis 向 goja 运行时注入应用通道的查询、历史、回执、话题、临时状态与群管理能力。
// 只暴露指南对普通应用开放的 20 个方法，不含 runas / 代用户发送 / entity_card。
func registerAppApis(rt *goja.Runtime, convID string) {
	rpc := func(method string, payload map[string]any) (goja.Value, error) {
		c := appClientForScript(convID)
		if c == nil {
			return goja.Undefined(), errJiyingNotConnected
		}
		ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
		defer cancel()
		raw, err := c.Call(ctx, method, payload)
		if err != nil {
			return goja.Undefined(), err
		}
		var v any
		if json.Unmarshal(raw, &v) != nil {
			return rt.ToValue(string(raw)), nil
		}
		return rt.ToValue(v), nil
	}
	// appUser(userId) -> {user:{...}}；失败返回 null。
	_ = rt.Set("appUser", func(call goja.FunctionCall) goja.Value {
		v, err := rpc("users.get", map[string]any{"user_id": gojaArgString(call, 0)})
		if err != nil {
			return goja.Null()
		}
		return v
	})
	// appSelf() -> {app:{...}}；未配置应用 ID 或失败返回 null。
	_ = rt.Set("appSelf", func(call goja.FunctionCall) goja.Value {
		appID := strings.TrimSpace(CurrentAppConfig().AppID)
		if appID == "" {
			return goja.Null()
		}
		v, err := rpc("apps.get", map[string]any{"app_id": appID})
		if err != nil {
			return goja.Null()
		}
		return v
	})
	// appConversations([keyword[, limit]]) -> {conversations:[...]}；失败返回 null。
	_ = rt.Set("appConversations", func(call goja.FunctionCall) goja.Value {
		limit := int(call.Argument(1).ToInteger())
		v, err := rpc("conversations.list", map[string]any{"keyword": gojaArgString(call, 0), "limit": limit})
		if err != nil {
			return goja.Null()
		}
		return v
	})
	// appHistory(conversationId, beforeOrEqualSeq[, limit]) -> {messages:[...]}；失败返回 null。
	_ = rt.Set("appHistory", func(call goja.FunctionCall) goja.Value {
		payload := map[string]any{
			"conversation_id":     gojaArgString(call, 0),
			"before_or_equal_seq": call.Argument(1).ToInteger(),
			"limit":               int(call.Argument(2).ToInteger()),
		}
		v, err := rpc("conversation.messages.list", payload)
		if err != nil {
			return goja.Null()
		}
		return v
	})
	// appReaction(conversationId, messageId, text[, reacted]) -> bool。
	_ = rt.Set("appReaction", func(call goja.FunctionCall) goja.Value {
		reacted := true
		if len(call.Arguments) >= 4 {
			reacted = call.Argument(3).ToBoolean()
		}
		payload := map[string]any{
			"conversation_id": gojaArgString(call, 0),
			"message_id":      gojaArgString(call, 1),
			"text":            gojaArgString(call, 2),
			"reacted":         reacted,
		}
		if _, err := rpc("message.reaction.set", payload); err != nil {
			return rt.ToValue(false)
		}
		return rt.ToValue(true)
	})
	// appTopic(conversationId) -> 话题状态；失败返回 null。
	_ = rt.Set("appTopic", func(call goja.FunctionCall) goja.Value {
		v, err := rpc("conversation.topic.get", map[string]any{"conversation_id": gojaArgString(call, 0)})
		if err != nil {
			return goja.Null()
		}
		return v
	})
	// appTopicCreate(parentConversationId, sourceMessageId) -> 话题；失败返回 null。
	_ = rt.Set("appTopicCreate", func(call goja.FunctionCall) goja.Value {
		v, err := rpc("conversation.topic.create", map[string]any{
			"conversation_id":   gojaArgString(call, 0),
			"source_message_id": gojaArgString(call, 1),
		})
		if err != nil {
			return goja.Null()
		}
		return v
	})
	// appTopicClose(conversationId[, expectedLastSeq]) -> bool。
	_ = rt.Set("appTopicClose", func(call goja.FunctionCall) goja.Value {
		_, err := rpc("conversation.topic.close", map[string]any{
			"conversation_id":           gojaArgString(call, 0),
			"expected_last_message_seq": call.Argument(1).ToInteger(),
		})
		return rt.ToValue(err == nil)
	})
	// appStatus(conversationId, status) -> bool；向一对一会话同步临时状态。
	_ = rt.Set("appStatus", func(call goja.FunctionCall) goja.Value {
		_, err := rpc("conversation.status", map[string]any{
			"conversation_id": gojaArgString(call, 0),
			"status":          gojaArgString(call, 1),
		})
		return rt.ToValue(err == nil)
	})
	// groupCreate(name, memberIds[, appIds]) -> {conversation,message}；失败返回 null。
	_ = rt.Set("groupCreate", func(call goja.FunctionCall) goja.Value {
		payload := map[string]any{
			"name":       gojaArgString(call, 0),
			"member_ids": gojaArgStrings(call, 1),
		}
		if appIDs := gojaArgStrings(call, 2); len(appIDs) > 0 {
			payload["app_ids"] = appIDs
		}
		v, err := rpc("group_conversations.create", payload)
		if err != nil {
			return goja.Null()
		}
		return v
	})
	// groupGet(conversationId) -> 群详情；失败返回 null。
	_ = rt.Set("groupGet", func(call goja.FunctionCall) goja.Value {
		v, err := rpc("group_conversations.get", map[string]any{"conversation_id": gojaArgString(call, 0)})
		if err != nil {
			return goja.Null()
		}
		return v
	})
	// groupRename(conversationId, name) -> 群详情；失败返回 null。
	_ = rt.Set("groupRename", func(call goja.FunctionCall) goja.Value {
		v, err := rpc("group_conversations.update", map[string]any{
			"conversation_id": gojaArgString(call, 0),
			"name":            gojaArgString(call, 1),
		})
		if err != nil {
			return goja.Null()
		}
		return v
	})
	// groupAnnounce(conversationId, announcement) -> 群详情；传空串表示清空，失败返回 null。
	_ = rt.Set("groupAnnounce", func(call goja.FunctionCall) goja.Value {
		v, err := rpc("group_conversations.update", map[string]any{
			"conversation_id": gojaArgString(call, 0),
			"announcement":    gojaArgString(call, 1),
		})
		if err != nil {
			return goja.Null()
		}
		return v
	})
	// groupDissolve(conversationId) -> bool。
	_ = rt.Set("groupDissolve", func(call goja.FunctionCall) goja.Value {
		_, err := rpc("group_conversations.dissolve", map[string]any{"conversation_id": gojaArgString(call, 0)})
		return rt.ToValue(err == nil)
	})
	// groupMembers(conversationId[, page[, pageSize]]) -> {members,total}；失败返回 null。
	_ = rt.Set("groupMembers", func(call goja.FunctionCall) goja.Value {
		v, err := rpc("group_conversations.members.list", map[string]any{
			"conversation_id": gojaArgString(call, 0),
			"page":            int(call.Argument(1).ToInteger()),
			"page_size":       int(call.Argument(2).ToInteger()),
		})
		if err != nil {
			return goja.Null()
		}
		return v
	})
	// groupAddMembers(conversationId, memberIds[, appIds]) -> bool。
	_ = rt.Set("groupAddMembers", func(call goja.FunctionCall) goja.Value {
		payload := map[string]any{
			"conversation_id": gojaArgString(call, 0),
			"member_ids":      gojaArgStrings(call, 1),
		}
		if appIDs := gojaArgStrings(call, 2); len(appIDs) > 0 {
			payload["app_ids"] = appIDs
		}
		_, err := rpc("group_conversations.members.add", payload)
		return rt.ToValue(err == nil)
	})
	// groupRemoveMember(conversationId, memberType, memberId) -> bool。
	_ = rt.Set("groupRemoveMember", func(call goja.FunctionCall) goja.Value {
		_, err := rpc("group_conversations.members.remove", map[string]any{
			"conversation_id": gojaArgString(call, 0),
			"member_type":     gojaArgString(call, 1),
			"member_id":       gojaArgString(call, 2),
		})
		return rt.ToValue(err == nil)
	})
	// groupSetRole(conversationId, memberType, memberId, role) -> bool。
	_ = rt.Set("groupSetRole", func(call goja.FunctionCall) goja.Value {
		_, err := rpc("group_conversations.members.set_role", map[string]any{
			"conversation_id": gojaArgString(call, 0),
			"member_type":     gojaArgString(call, 1),
			"member_id":       gojaArgString(call, 2),
			"role":            gojaArgString(call, 3),
		})
		return rt.ToValue(err == nil)
	})
}
