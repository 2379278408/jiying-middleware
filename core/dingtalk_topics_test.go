package core

import (
	"strings"
	"testing"
)

func TestDingTalkMediaExtraction(t *testing.T) {
	text := "[图片消息](mediaId=$iwEcAqNwbmcDAQTRCswF0QVW) 注意：如需下载使用dws chat message download-media命令下载"
	ids := dingTalkExtractMediaIDs(text)
	if len(ids) != 1 || ids[0] != "$iwEcAqNwbmcDAQTRCswF0QVW" {
		t.Fatalf("mediaId 提取错误: %v", ids)
	}
	if got := dingTalkTextWithoutImages(text); got != "" {
		t.Fatalf("图片标记应被移除，got %q", got)
	}
	if got := dingTalkTextWithoutImages("看这个\n\n[图片消息](mediaId=$x)\n\n继续"); got != "看这个\n\n继续" {
		t.Fatalf("正文保留错误: %q", got)
	}
	if !dingTalkIsImageContent(text) {
		t.Fatal("图片消息应被识别")
	}
	encoded := dingTalkExtractMediaIDs("[图片消息](mediaId=%24abc%2Fdef)")
	if len(encoded) != 1 || encoded[0] != "$abc/def" {
		t.Fatalf("mediaId 应解码: %v", encoded)
	}
}

func TestDingTalkImageResourceIDsPrefersImageContent(t *testing.T) {
	payload := map[string]any{
		"messages": []any{map[string]any{"resourceRefs": []any{
			map[string]any{"resourceId": "$thumb", "resourceIdType": "mediaId", "type": "mediaId"},
			map[string]any{"resourceId": "@image", "resourceIdType": "mediaId", "type": "mediaId", "contentType": "image"},
		}}},
	}
	ids := dingTalkImageResourceIDs(payload)
	if len(ids) != 1 || ids[0] != "@image" {
		t.Fatalf("应优先取图片资源: %v", ids)
	}
}

func TestDingTalkMutedFromPayload(t *testing.T) {
	payload := map[string]any{"result": map[string]any{"settings": []any{
		map[string]any{"notificationOff": float64(1), "openConversationId": "muted", "success": true},
		map[string]any{"notificationOff": float64(0), "openConversationId": "open", "success": true},
	}}}
	if muted, ok := dingTalkMutedFromPayload(payload, "muted"); !ok || !muted {
		t.Fatalf("应识别免打扰: %v %v", muted, ok)
	}
	if muted, ok := dingTalkMutedFromPayload(payload, "open"); !ok || muted {
		t.Fatalf("应识别未免打扰: %v %v", muted, ok)
	}
	if _, ok := dingTalkMutedFromPayload(payload, "missing"); ok {
		t.Fatal("缺失会话不应给出结论")
	}
}

func TestDingTalkTopicStoreAndCleanup(t *testing.T) {
	setupAgentBucket(t, "dingtalk-topics")
	rec := dingTalkTopic{TopicID: "topic-1", ConversationID: "cid-1", Name: "销售群", AppID: "app-a"}
	if err := saveDingTalkTopic(rec); err != nil {
		t.Fatal(err)
	}
	if got, ok := dingTalkTopicByConversation("cid-1"); !ok || got.TopicID != "topic-1" {
		t.Fatalf("话题映射未保存: %+v %v", got, ok)
	}
	if got, ok := dingTalkTopicByTopicID("topic-1"); !ok || got.ConversationID != "cid-1" {
		t.Fatalf("话题反查失败: %+v %v", got, ok)
	}
	removeDingTalkTopicsForApp("app-b")
	if _, ok := dingTalkTopicByConversation("cid-1"); !ok {
		t.Fatal("其他应用清理不应删除本应用话题")
	}
	if err := saveDingTalkTopic(dingTalkTopic{TopicID: "topic-2", ConversationID: "cid-2", AppID: "app-a"}); err != nil {
		t.Fatal(err)
	}
	forgetDingTalkTopic("cid-2")
	if _, ok := dingTalkTopicByTopicID("topic-2"); ok {
		t.Fatal("清理会话时应同时删除反查索引")
	}
	removeDingTalkTopicsForApp("app-a")
	if _, ok := dingTalkTopicByConversation("cid-1"); ok {
		t.Fatal("应用删除后话题映射应清理")
	}
	if _, ok := dingTalkTopicByTopicID("topic-1"); ok {
		t.Fatal("应用删除后话题反查应清理")
	}
}

func TestDingTalkTopicTextFormat(t *testing.T) {
	dm := DingTalkEvent{Reason: "dm", Sender: "赵然"}
	if got := formatDingTalkTopicText(dm, "在吗"); got != "在吗" {
		t.Fatalf("私聊应只发正文: %q", got)
	}
	group := DingTalkEvent{Reason: "mention", Sender: "陈佳"}
	if got := formatDingTalkTopicText(group, "今晚发布"); got != "> **陈佳**：\n\n今晚发布" {
		t.Fatalf("群聊格式错误: %q", got)
	}
	if got := formatDingTalkTopicText(group, "@所有人 发布"); strings.Contains(got, "未命名") {
		t.Fatalf("群聊不应插入占位名: %q", got)
	}
}

func TestDingTalkTopicReplyRequiresOps(t *testing.T) {
	setupAgentBucket(t, "dingtalk-topic-auth")
	if err := saveDingTalkTopic(dingTalkTopic{TopicID: "topic-auth", ConversationID: "cid-auth", AppID: "app-auth"}); err != nil {
		t.Fatal(err)
	}
	dingTalk.mu.Lock()
	oldCfg := dingTalk.cfg
	dingTalk.cfg.Enabled = true
	dingTalk.mu.Unlock()
	defer func() {
		dingTalk.mu.Lock()
		dingTalk.cfg = oldCfg
		dingTalk.mu.Unlock()
	}()
	s := &JiyingSender{ConvType: "topic", ChatID: "topic-auth", SenderType: "user", UserID: "intruder", Text: "hi"}
	if !relayDingTalkTopicReply(s) {
		t.Fatal("话题回复应在鉴权后拦截")
	}
}

func TestDingTalkTextSplit(t *testing.T) {
	text := strings.Repeat("钉", 3501)
	chunks := splitDingTalkText(text, 3500)
	if len(chunks) != 2 || len([]rune(chunks[0])) != 3500 || len([]rune(chunks[1])) != 1 {
		t.Fatalf("长消息分片错误: %d", len(chunks))
	}
	if got := splitDingTalkText("", 3500); len(got) != 1 || got[0] != "" {
		t.Fatalf("空文本分片错误: %v", got)
	}
}

func TestDingTalkReplyKeyStable(t *testing.T) {
	a := dingTalkReplyKey("message-1", "hello")
	b := dingTalkReplyKey("message-1", "changed")
	if a != b {
		t.Fatal("相同消息应得到稳定幂等键")
	}
	long := dingTalkReplyKey(strings.Repeat("x", 80), "")
	if len(long) != 32 {
		t.Fatalf("超长键应哈希为 32 位: %d", len(long))
	}
	if dingTalkReplyKey("", "same") != dingTalkReplyKey("", "same") {
		t.Fatal("无消息 ID 时应由正文稳定派生")
	}
}
