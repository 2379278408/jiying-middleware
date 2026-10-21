package core

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

const dingTalkTopicsKey = "dingtalk_topics"

// dingTalkTopic 记录一个钉钉会话与即应话题的稳定映射。
type dingTalkTopic struct {
	TopicID             string `json:"topic_id"`
	ConversationID      string `json:"conversation_id"`
	Name                string `json:"name,omitempty"`
	Kind                string `json:"kind,omitempty"`
	OpenDingTalkID      string `json:"open_dingtalk_id,omitempty"`
	AppID               string `json:"app_id,omitempty"`
	ParentConversation  string `json:"parent_conversation_id,omitempty"`
	LastDingTalkMessage string `json:"last_message_id,omitempty"`
	CreatedAt           string `json:"created_at,omitempty"`
}

type dingTalkTopicStore struct {
	ParentConversationID string                   `json:"parent_conversation_id,omitempty"`
	ByDingTalk           map[string]dingTalkTopic `json:"by_dingtalk"`
	ByTopic              map[string]string        `json:"by_topic,omitempty"`
}

var (
	dingTalkTopicMu    sync.Mutex
	dingTalkTopicLocks = map[string]*sync.Mutex{}
	dingTalkNameMu     sync.Mutex
	dingTalkNameCache  = map[string]dingTalkCachedName{}
	dingTalkMute       = map[string]dingTalkCachedMute{}
)

type dingTalkCachedName struct {
	name string
	at   time.Time
}

type dingTalkCachedMute struct {
	muted bool
	at    time.Time
}

func loadDingTalkTopics() dingTalkTopicStore {
	store := dingTalkTopicStore{ByDingTalk: map[string]dingTalkTopic{}, ByTopic: map[string]string{}}
	if BucketApp == nil {
		return store
	}
	raw := BucketApp.GetString(dingTalkTopicsKey)
	if raw == "" {
		return store
	}
	_ = json.Unmarshal([]byte(raw), &store)
	if store.ByDingTalk == nil {
		store.ByDingTalk = map[string]dingTalkTopic{}
	}
	if store.ByTopic == nil {
		store.ByTopic = map[string]string{}
		for convID, rec := range store.ByDingTalk {
			if rec.TopicID != "" {
				store.ByTopic[rec.TopicID] = convID
			}
		}
	}
	return store
}

func saveDingTalkTopics(store dingTalkTopicStore) error {
	if BucketApp == nil {
		return fmt.Errorf("存储未就绪")
	}
	if store.ByDingTalk == nil {
		store.ByDingTalk = map[string]dingTalkTopic{}
	}
	raw, err := json.Marshal(store)
	if err != nil {
		return err
	}
	BucketApp.Set(dingTalkTopicsKey, string(raw))
	return nil
}

func saveDingTalkTopic(rec dingTalkTopic) error {
	dingTalkTopicMu.Lock()
	defer dingTalkTopicMu.Unlock()
	return saveDingTalkTopicLocked(rec)
}

func saveDingTalkTopicLocked(rec dingTalkTopic) error {
	store := loadDingTalkTopics()
	store.ByDingTalk[rec.ConversationID] = rec
	if store.ByTopic == nil {
		store.ByTopic = map[string]string{}
	}
	store.ByTopic[rec.TopicID] = rec.ConversationID
	return saveDingTalkTopics(store)
}

func forgetDingTalkTopic(convID string) {
	dingTalkTopicMu.Lock()
	defer dingTalkTopicMu.Unlock()
	store := loadDingTalkTopics()
	rec, ok := store.ByDingTalk[convID]
	if !ok {
		return
	}
	delete(store.ByDingTalk, convID)
	delete(store.ByTopic, rec.TopicID)
	_ = saveDingTalkTopics(store)
}

// dingTalkTopicByConversation 按钉钉会话查找已创建的话题。
func dingTalkTopicByConversation(convID string) (dingTalkTopic, bool) {
	dingTalkTopicMu.Lock()
	defer dingTalkTopicMu.Unlock()
	rec, ok := loadDingTalkTopics().ByDingTalk[strings.TrimSpace(convID)]
	return rec, ok
}

// dingTalkTopicByTopicID 按即应话题 ID 反查钉钉会话，供话题回复转回钉钉。
func dingTalkTopicByTopicID(topicID string) (dingTalkTopic, bool) {
	dingTalkTopicMu.Lock()
	defer dingTalkTopicMu.Unlock()
	store := loadDingTalkTopics()
	convID, ok := store.ByTopic[strings.TrimSpace(topicID)]
	if !ok {
		return dingTalkTopic{}, false
	}
	rec, ok := store.ByDingTalk[convID]
	return rec, ok
}

func removeDingTalkTopicsForApp(appID string) {
	dingTalkTopicMu.Lock()
	defer dingTalkTopicMu.Unlock()
	store := loadDingTalkTopics()
	changed := false
	for convID, rec := range store.ByDingTalk {
		if !strings.EqualFold(rec.AppID, appID) {
			continue
		}
		delete(store.ByDingTalk, convID)
		delete(store.ByTopic, rec.TopicID)
		changed = true
	}
	if changed {
		_ = saveDingTalkTopics(store)
	}
}

func dingTalkTopicLock(convID string) *sync.Mutex {
	dingTalkTopicMu.Lock()
	defer dingTalkTopicMu.Unlock()
	lock := dingTalkTopicLocks[convID]
	if lock == nil {
		lock = &sync.Mutex{}
		dingTalkTopicLocks[convID] = lock
	}
	return lock
}

// pushEventToTopic 把事件写入对应钉钉会话的话题；必要时先创建话题。
func (d *dingTalkConnector) pushEventToTopic(ctx context.Context, cfg DingTalkConfig, ev DingTalkEvent) error {
	rec, err := d.ensureTopic(ctx, cfg, ev)
	if err != nil {
		return err
	}
	client := clientForApp(rec.AppID)
	if client == nil {
		return fmt.Errorf("推送应用通道离线")
	}
	images := dingTalkImageMediaIDs(ctx, ev)
	body := dingTalkMessageBody(ev)
	text := dingTalkTextWithoutImages(body)
	if len(images) == 0 {
		return sendDingTalkTopicText(ctx, client, rec.TopicID, formatDingTalkTopicText(ev, body))
	}
	caption := ""
	if ev.Reason != "dm" {
		caption = "> **" + dingTalkSenderName(ev) + "**："
	}
	if text != "" {
		if err := sendDingTalkTopicText(ctx, client, rec.TopicID, formatDingTalkTopicText(ev, text)); err != nil {
			return err
		}
		caption = ""
	}
	sentImage := false
	for _, mediaID := range images {
		url, err := downloadDingTalkImage(ctx, ev, mediaID)
		if err != nil {
			GetMyLog().Warn("钉钉图片下载失败: %v", err)
			continue
		}
		if _, err := client.SendTo(ctx, rpcTarget{Type: "topic", ConversationID: rec.TopicID}, "", ImageMessage(url, caption, "markdown")); err != nil {
			return err
		}
		sentImage = true
		caption = ""
	}
	if !sentImage {
		if text == "" {
			text = "（图片下载失败，原消息也没有可显示的正文）"
		}
		return sendDingTalkTopicText(ctx, client, rec.TopicID, formatDingTalkTopicText(ev, text))
	}
	return nil
}

func (d *dingTalkConnector) ensureTopic(ctx context.Context, cfg DingTalkConfig, ev DingTalkEvent) (dingTalkTopic, error) {
	convID := strings.TrimSpace(ev.ConversationID)
	if convID == "" {
		return dingTalkTopic{}, fmt.Errorf("事件缺少钉钉会话 ID")
	}
	lock := dingTalkTopicLock(convID)
	lock.Lock()
	defer lock.Unlock()
	if rec, ok := dingTalkTopicByConversation(convID); ok && rec.TopicID != "" {
		if rec.AppID == cfg.PushAppID {
			return rec, nil
		}
		// 推送应用变了，旧话题对新应用不可用；清掉映射后按新应用重建。
		forgetDingTalkTopic(convID)
	}
	client := clientForApp(cfg.PushAppID)
	if client == nil {
		return dingTalkTopic{}, fmt.Errorf("推送应用通道离线")
	}
	name := d.eventTopicName(ctx, ev)
	parent := strings.TrimSpace(cfg.PushConversationID)
	title := name
	if title == "" {
		title = "钉钉会话"
	}
	msgID, err := client.SendTo(ctx, convTarget("conversation", parent), "", MarkdownMessage(title))
	if err != nil {
		return dingTalkTopic{}, fmt.Errorf("创建话题的起始消息发送失败: %w", err)
	}
	if strings.TrimSpace(msgID) == "" {
		return dingTalkTopic{}, fmt.Errorf("创建话题的起始消息没有返回消息 ID")
	}
	topic, err := client.TopicCreate(ctx, parent, msgID)
	if err != nil {
		return dingTalkTopic{}, fmt.Errorf("创建即应话题失败: %w", err)
	}
	if strings.TrimSpace(topic.ConversationID) == "" {
		return dingTalkTopic{}, fmt.Errorf("创建即应话题没有返回话题 ID")
	}
	rec := dingTalkTopic{
		TopicID:            topic.ConversationID,
		ConversationID:     convID,
		Name:               name,
		Kind:               "group",
		OpenDingTalkID:     ev.SenderOpenID,
		AppID:              cfg.PushAppID,
		ParentConversation: parent,
		CreatedAt:          time.Now().Format(time.RFC3339),
	}
	if ev.Reason == "dm" {
		rec.Kind = "dm"
	}
	if err := saveDingTalkTopic(rec); err != nil {
		return dingTalkTopic{}, err
	}
	rememberConvApp(rec.TopicID, rec.AppID)
	return rec, nil
}

func (d *dingTalkConnector) eventTopicName(ctx context.Context, ev DingTalkEvent) string {
	if ev.Reason == "dm" {
		if name := dingTalkCleanName(ev.Sender); name != "" {
			return name
		}
		return "私聊"
	}
	if name := dingTalkCleanName(ev.ConversationName); name != "" {
		return name
	}
	if name := dingTalkConversationName(ctx, ev.ConversationID); name != "" {
		return name
	}
	if ev.ConversationID != "" {
		return ev.ConversationID
	}
	return "群聊"
}

func formatDingTalkTopicText(ev DingTalkEvent, body string) string {
	body = strings.TrimSpace(body)
	if ev.Reason == "dm" {
		return body
	}
	return "> **" + dingTalkSenderName(ev) + "**：\n\n" + body
}

func dingTalkSenderName(ev DingTalkEvent) string {
	if name := dingTalkCleanName(ev.Sender); name != "" {
		return name
	}
	return "未知发送人"
}

func dingTalkMessageBody(ev DingTalkEvent) string {
	if body := strings.TrimSpace(ev.Content); body != "" {
		return body
	}
	return "（没有可显示的正文，可能是图片、文件或卡片）"
}

func sendDingTalkTopicText(ctx context.Context, client *JiyingClient, topicID, text string) error {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	for _, chunk := range splitDingTalkText(text, 3500) {
		if _, err := client.SendTo(ctx, rpcTarget{Type: "topic", ConversationID: topicID}, "", MarkdownMessage(clampOutboundText(chunk))); err != nil {
			return err
		}
	}
	return nil
}

// splitDingTalkText 与上游保持一致，按 3500 字符分片，避免长消息被平台截断。
func splitDingTalkText(text string, size int) []string {
	if size <= 0 {
		return []string{text}
	}
	runes := []rune(text)
	chunks := make([]string, 0, len(runes)/size+1)
	for len(runes) > 0 {
		n := size
		if len(runes) < n {
			n = len(runes)
		}
		chunks = append(chunks, string(runes[:n]))
		runes = runes[n:]
	}
	if len(chunks) == 0 {
		return []string{""}
	}
	return chunks
}

// relayDingTalkTopicReply 把话题中的用户文字直接转回对应钉钉会话，不经过 Agent。
func relayDingTalkTopicReply(s Sender) bool {
	js, ok := s.(*JiyingSender)
	if !ok || !js.IsTopic() || js.SenderType != "user" || js.UserID == "" {
		return false
	}
	rec, ok := dingTalkTopicByTopicID(js.ChatID)
	if !ok || rec.TopicID == "" {
		return false
	}
	if !dingTalkActive() {
		return false
	}
	if !HasRoleForApp(js.UserID, rec.AppID, RoleOps) {
		js.Reply("你没有该应用 ops 及以上角色，不能把话题回复转发到钉钉。")
		return true
	}
	text := strings.TrimSpace(js.GetMsg())
	if text == "" {
		js.Reply("这条消息没有可转发的文字。")
		return true
	}
	args := []string{"chat", "+messages-send", "--as", "user"}
	if rec.Kind == "dm" {
		if strings.TrimSpace(rec.OpenDingTalkID) == "" {
			js.Reply("缺少对方的钉钉 ID，无法转发。")
			return true
		}
		args = append(args, "--open-dingtalk-id", rec.OpenDingTalkID)
	} else {
		args = append(args, "--group", rec.ConversationID)
	}
	args = append(args, "--text", text, "--idempotency-key", dingTalkReplyKey(js.MsgID, text))
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	if _, err := runDwsCommand(ctx, args...); err != nil {
		js.Reply("转发到钉钉失败：" + redactSecretText(err.Error()))
		RecordAuditApp(rec.AppID, js.UserID, "dingtalk.topic_reply", rec.Name, text, "失败: "+err.Error())
		return true
	}
	RecordAuditApp(rec.AppID, js.UserID, "dingtalk.topic_reply", rec.Name, text, "成功")
	return true
}

func dingTalkReplyKey(messageID, text string) string {
	key := strings.TrimSpace(messageID)
	if key == "" {
		key = text
	}
	if len(key) > 32 {
		key = dingTalkHashShort(key)
	}
	if key == "" {
		key = time.Now().Format("20060102150405")
	}
	return key
}

func dingTalkHashShort(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:16])
}

// runDwsData 调用 dws 并把 JSON 输出解析成通用结构；解析失败时按原文返回。
func runDwsData(ctx context.Context, args ...string) (any, error) {
	out, err := runDwsCommand(ctx, args...)
	if err != nil {
		return nil, err
	}
	var parsed any
	if json.Unmarshal([]byte(out), &parsed) != nil {
		return out, nil
	}
	return parsed, nil
}
