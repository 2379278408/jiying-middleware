package core

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var dingTalkMediaIDRe = regexp.MustCompile(`mediaId=([^)\s]+)`)
var dingTalkImageBlockRe = regexp.MustCompile(`\[[^\]]*图片消息[^\]]*\]\(mediaId=[^)]+\)`)
var dingTalkImageTagRe = regexp.MustCompile(`\[图片消息\]\(mediaId=[^)]+\)`)

func dingTalkExtractMediaIDs(text string) []string {
	out := make([]string, 0)
	seen := map[string]bool{}
	for _, m := range dingTalkMediaIDRe.FindAllStringSubmatch(text, -1) {
		id := strings.TrimSpace(m[1])
		if decoded, err := url.PathUnescape(id); err == nil {
			id = decoded
		}
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}

func dingTalkIsImageContent(text string) bool {
	return strings.Contains(text, "图片消息") || len(dingTalkExtractMediaIDs(text)) > 0
}

func dingTalkTextWithoutImages(text string) string {
	text = dingTalkImageBlockRe.ReplaceAllString(text, "")
	text = dingTalkImageTagRe.ReplaceAllString(text, "")
	text = strings.ReplaceAll(text, "注意：如需下载使用dws chat message download-media命令下载", "")
	lines := strings.Split(text, "\n")
	out := make([]string, 0, len(lines))
	blank := 0
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			blank++
			if blank > 1 {
				continue
			}
		} else {
			blank = 0
		}
		out = append(out, line)
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}

func dingTalkImageMediaIDs(ctx context.Context, ev DingTalkEvent) []string {
	raw := ev.Content
	if !dingTalkIsImageContent(raw) {
		return nil
	}
	ids := []string{}
	if msgID := strings.TrimSpace(ev.ID); msgID != "" {
		if payload, err := runDwsData(ctx, "chat", "+messages-mget", "--msg-ids", msgID, "--no-reactions", "--no-threads"); err == nil {
			ids = dingTalkImageResourceIDs(payload)
		}
	}
	if len(ids) == 0 {
		ids = dingTalkExtractMediaIDs(raw)
	}
	if len(ids) > 4 {
		ids = ids[:4]
	}
	return ids
}

func dingTalkImageResourceIDs(payload any) []string {
	refs := make([]map[string]any, 0)
	var walk func(any)
	walk = func(v any) {
		switch value := v.(type) {
		case map[string]any:
			if id, _ := value["resourceId"].(string); id != "" {
				typ, _ := value["resourceIdType"].(string)
				kind, _ := value["type"].(string)
				if typ == "mediaId" || kind == "mediaId" {
					refs = append(refs, value)
				}
			}
			for _, child := range value {
				walk(child)
			}
		case []any:
			for _, child := range value {
				walk(child)
			}
		}
	}
	walk(payload)
	chosen := make([]map[string]any, 0)
	for _, ref := range refs {
		if kind, _ := ref["contentType"].(string); kind == "image" {
			chosen = append(chosen, ref)
		}
	}
	if len(chosen) == 0 {
		chosen = refs
	}
	out := make([]string, 0, len(chosen))
	seen := map[string]bool{}
	for _, ref := range chosen {
		id, _ := ref["resourceId"].(string)
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}

func dingTalkConversationName(ctx context.Context, convID string) string {
	convID = strings.TrimSpace(convID)
	if convID == "" {
		return ""
	}
	dingTalkNameMu.Lock()
	cached, ok := dingTalkNameCache[convID]
	dingTalkNameMu.Unlock()
	if ok && time.Since(cached.at) < 10*time.Minute {
		return cached.name
	}
	name := ""
	if payload, err := runDwsData(ctx, "chat", "+conversation-info", "--group", convID); err == nil {
		name = dingTalkFirstTitle(payload)
	}
	if name != "" {
		dingTalkNameMu.Lock()
		dingTalkNameCache[convID] = dingTalkCachedName{name: name, at: time.Now()}
		dingTalkNameMu.Unlock()
	}
	return name
}

func dingTalkFirstTitle(payload any) string {
	var walk func(any, int) string
	walk = func(v any, depth int) string {
		if depth > 4 || v == nil {
			return ""
		}
		switch value := v.(type) {
		case map[string]any:
			for _, key := range []string{"title", "conversationTitle", "name"} {
				if s, ok := value[key].(string); ok {
					if s = dingTalkCleanName(s); s != "" {
						return s
					}
				}
			}
			for _, key := range []string{"conversationInfo", "result", "conversation"} {
				if child, ok := value[key]; ok {
					if s := walk(child, depth+1); s != "" {
						return s
					}
				}
			}
		}
		return ""
	}
	return walk(payload, 0)
}

func dingTalkCleanName(value string) string {
	value = strings.TrimSpace(value)
	if value == "null" {
		return ""
	}
	return value
}

// dingTalkMuted 查询群免打扰状态；查询失败时按免打扰处理，避免打扰用户。
func dingTalkMuted(ctx context.Context, convID string) bool {
	convID = strings.TrimSpace(convID)
	if convID == "" {
		return true
	}
	dingTalkNameMu.Lock()
	cached, ok := dingTalkMute[convID]
	dingTalkNameMu.Unlock()
	if ok && time.Since(cached.at) < 10*time.Minute {
		return cached.muted
	}
	if payload, err := runDwsData(ctx, "chat", "group", "user-settings", "query", "--groups", convID); err == nil {
		if value, known := dingTalkMutedFromPayload(payload, convID); known {
			dingTalkNameMu.Lock()
			dingTalkMute[convID] = dingTalkCachedMute{muted: value, at: time.Now()}
			dingTalkNameMu.Unlock()
			return value
		}
	}
	return true
}

// dingTalkDirectMuted 查询单聊免打扰状态；单聊没有群设置接口，改为从最近会话列表读取。
// 只读第一页以控制输出体积；查不到会话或查询失败时按未免打扰处理，避免误吞私聊消息。
func dingTalkDirectMuted(ctx context.Context, convID string) bool {
	convID = strings.TrimSpace(convID)
	if convID == "" {
		return false
	}
	key := "direct:" + convID
	dingTalkNameMu.Lock()
	cached, ok := dingTalkMute[key]
	dingTalkNameMu.Unlock()
	if ok && time.Since(cached.at) < 10*time.Minute {
		return cached.muted
	}
	checkCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	if payload, err := runDwsData(checkCtx, "chat", "+conversation-list", "--limit", "100"); err == nil {
		if value, known := dingTalkNotificationOffFromPayload(payload, convID); known {
			dingTalkNameMu.Lock()
			dingTalkMute[key] = dingTalkCachedMute{muted: value, at: time.Now()}
			dingTalkNameMu.Unlock()
			return value
		}
	}
	return false
}

// dingTalkNotificationOffFromPayload 从会话列表响应中递归查找指定会话的免打扰开关。
func dingTalkNotificationOffFromPayload(payload any, convID string) (bool, bool) {
	switch value := payload.(type) {
	case []any:
		for _, child := range value {
			if muted, ok := dingTalkNotificationOffFromPayload(child, convID); ok {
				return muted, true
			}
		}
	case map[string]any:
		if id, _ := value["openConversationId"].(string); id == convID {
			if off, ok := value["notificationOff"]; ok {
				switch v := off.(type) {
				case bool:
					return v, true
				case float64:
					return v == 1, true
				case string:
					return strings.TrimSpace(v) == "1", true
				}
			}
		}
		for _, child := range value {
			if muted, ok := dingTalkNotificationOffFromPayload(child, convID); ok {
				return muted, true
			}
		}
	}
	return false, false
}

func dingTalkMutedFromPayload(payload any, convID string) (bool, bool) {
	var settings []any
	var walk func(any)
	walk = func(v any) {
		obj, ok := v.(map[string]any)
		if !ok {
			if list, ok := v.([]any); ok {
				for _, child := range list {
					walk(child)
				}
			}
			return
		}
		if list, ok := obj["settings"].([]any); ok {
			settings = list
			return
		}
		if res, ok := obj["result"]; ok {
			walk(res)
		}
	}
	walk(payload)
	for _, item := range settings {
		row, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if id, _ := row["openConversationId"].(string); id != convID {
			continue
		}
		if success, ok := row["success"].(bool); ok && !success {
			return true, true
		}
		off, ok := row["notificationOff"]
		if !ok || off == nil {
			return true, true
		}
		switch value := off.(type) {
		case float64:
			return value == 1, true
		case string:
			return strings.TrimSpace(value) == "1", true
		}
		return true, true
	}
	return true, false
}

func downloadDingTalkImage(ctx context.Context, ev DingTalkEvent, mediaID string) (string, error) {
	msgID := strings.TrimSpace(ev.ID)
	convID := strings.TrimSpace(ev.ConversationID)
	if msgID == "" || convID == "" || mediaID == "" {
		return "", fmt.Errorf("缺少图片下载所需的会话或消息 ID")
	}
	path, err := dingTalkMediaPath(msgID, mediaID)
	if err != nil {
		return "", err
	}
	defer func() { _ = removeDingTalkTemp(path) }()
	payload, err := runDwsData(ctx, "chat", "message", "download-media",
		"--type", "mediaId",
		"--resource-id", mediaID,
		"--message-id", msgID,
		"--open-conversation-id", convID,
		"--output", path)
	if err != nil {
		return "", err
	}
	url := dingTalkFirstHTTPURL(payload)
	if url == "" {
		return "", fmt.Errorf("下载图片没有返回可访问地址")
	}
	return dingTalkPublicImageURL(url), nil
}

func dingTalkFirstHTTPURL(payload any) string {
	var walk func(any) string
	walk = func(v any) string {
		switch value := v.(type) {
		case map[string]any:
			for key, child := range value {
				if s, ok := child.(string); ok && strings.Contains(strings.ToLower(key), "url") && (strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")) {
					return s
				}
			}
			for _, child := range value {
				if s := walk(child); s != "" {
					return s
				}
			}
		case []any:
			for _, child := range value {
				if s := walk(child); s != "" {
					return s
				}
			}
		}
		return ""
	}
	return walk(payload)
}

func dingTalkPublicImageURL(url string) string {
	if strings.HasPrefix(url, "http://") && strings.Contains(url, ".aliyuncs.com/") {
		return "https://" + strings.TrimPrefix(url, "http://")
	}
	return url
}

func dingTalkMediaPath(msgID, mediaID string) (string, error) {
	dir := filepath.Join(os.TempDir(), "jiying-dingtalk-media")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	name := dingTalkHashShort(msgID + ":" + mediaID)
	return filepath.Join(dir, name+".bin"), nil
}

func removeDingTalkTemp(path string) error {
	if path == "" {
		return nil
	}
	return os.Remove(path)
}
