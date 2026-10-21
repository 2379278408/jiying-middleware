package core

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"
)

// dingTalkToolContracts 返回连接器启用时暴露给模型的钉钉工具。
// 钉钉工具会以宿主账号发消息，只向该应用内达到 ops 的用户开放。
func dingTalkToolContracts(userID, appID string) []ToolContract {
	if !dingTalkActive() || !HasRoleForApp(userID, appID, RoleOps) {
		return nil
	}
	schema := func(props map[string]any, required ...string) map[string]any {
		out := map[string]any{"type": "object", "properties": props}
		if len(required) > 0 {
			out["required"] = required
		}
		return out
	}
	str := func(desc string) map[string]any {
		return map[string]any{"type": "string", "description": desc}
	}
	num := func(desc string) map[string]any {
		return map[string]any{"type": "integer", "description": desc}
	}
	mk := func(name, desc string, s map[string]any) ToolContract {
		return ToolContract{
			Tool:        dingTalkToolPrefix + name,
			Feature:     "dingtalk:" + name,
			Description: desc,
			Identity:    "user",
			Scope:       "public",
			Schema:      s,
		}
	}
	return []ToolContract{
		mk("list_unread", "列出当前钉钉未读会话，包含未读数与免打扰状态。",
			schema(map[string]any{"count": num("返回数量，默认 30")})),
		mk("list_mentions", "查询最近 @我 的钉钉消息。",
			schema(map[string]any{"days": num("最近天数，默认 7"), "limit": num("返回条数，默认 20")})),
		mk("search_messages", "按关键词、发送人或群搜索钉钉消息，至少提供一个条件。",
			schema(map[string]any{"query": str("关键词"), "sender": str("发送人"), "group": str("群"), "days": num("最近天数")})),
		mk("read_messages", "读取一个钉钉会话的最近消息，群聊传 conversation_id，单聊传 open_dingtalk_id。",
			schema(map[string]any{"conversation_id": str("群会话 ID"), "open_dingtalk_id": str("单聊 openDingTalkId"), "limit": num("返回条数，默认 20")})),
		mk("find_target", "按姓名找人，或按群名找群，拿到后续发送要用的稳定 ID。",
			schema(map[string]any{"kind": map[string]any{"type": "string", "enum": []string{"user", "group"}, "description": "user 或 group"}, "query": str("姓名或群名")}, "kind", "query")),
		mk("send_message", "以当前钉钉用户身份发送私聊或群消息，调用后直接发出。",
			schema(map[string]any{"text": str("消息正文"), "conversation_id": str("群会话 ID"), "group_name": str("群名"), "open_dingtalk_id": str("单聊 openDingtalkId"), "user_id": str("用户 ID"), "user_name": str("用户姓名")}, "text")),
		mk("reply_message", "引用回复一条钉钉消息，调用后直接发出。",
			schema(map[string]any{"message_id": str("被回复消息 ID"), "text": str("回复正文"), "conversation_id": str("群会话 ID")}, "message_id", "text")),
	}
}

// runDingTalkTool 执行一次钉钉工具调用，返回可直接回灌给模型的文本。
func runDingTalkTool(name, rawArgs string) string {
	name = strings.TrimPrefix(strings.TrimSpace(name), dingTalkToolPrefix)
	var input map[string]any
	if strings.TrimSpace(rawArgs) != "" {
		if json.Unmarshal([]byte(rawArgs), &input) != nil {
			return "FAILED: 工具参数不是合法 JSON"
		}
	}
	if input == nil {
		input = map[string]any{}
	}
	args, err := dingTalkToolArgs(name, input)
	if err != nil {
		return "FAILED: " + err.Error()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Second)
	defer cancel()
	out, err := runDwsCommand(ctx, args...)
	if err != nil {
		return "FAILED: " + redactSecretText(err.Error())
	}
	return redactSecretText(out)
}

func dingTalkToolArgs(name string, input map[string]any) ([]string, error) {
	text := strings.TrimSpace(strInput(input, "text"))
	switch name {
	case "list_unread":
		return []string{"chat", "message", "list-unread-conversations", "--count", strconv.Itoa(intInput(input, "count", 30))}, nil
	case "list_mentions":
		return []string{"chat", "+at-me", "--days", strconv.Itoa(intInput(input, "days", 7)), "--limit", strconv.Itoa(intInput(input, "limit", 20))}, nil
	case "search_messages":
		args := []string{"chat", "+search-msg"}
		added := false
		for _, key := range []string{"query", "sender", "group"} {
			if v := strings.TrimSpace(strInput(input, key)); v != "" {
				args = append(args, "--"+key, v)
				added = true
			}
		}
		if days := intInput(input, "days", 0); days > 0 {
			args = append(args, "--days", strconv.Itoa(days))
			added = true
		}
		if !added {
			return nil, fmt.Errorf("搜索至少要有一个条件")
		}
		return args, nil
	case "read_messages":
		limit := strconv.Itoa(intInput(input, "limit", 20))
		if v := strings.TrimSpace(strInput(input, "conversation_id")); v != "" {
			return []string{"chat", "+chat-messages", "--group", v, "--limit", limit}, nil
		}
		if v := strings.TrimSpace(strInput(input, "open_dingtalk_id")); v != "" {
			return []string{"chat", "+chat-messages", "--open-dingtalk-id", v, "--limit", limit}, nil
		}
		return nil, fmt.Errorf("读取会话需要 conversation_id 或 open_dingtalk_id")
	case "find_target":
		kind := strings.TrimSpace(strInput(input, "kind"))
		query := strings.TrimSpace(strInput(input, "query"))
		if query == "" {
			return nil, fmt.Errorf("缺少查询内容")
		}
		if kind == "group" {
			return []string{"chat", "+chat-search", "--query", query}, nil
		}
		if kind == "user" {
			return []string{"contact", "user", "search", "--query", query}, nil
		}
		return nil, fmt.Errorf("kind 只能是 user 或 group")
	case "send_message":
		if text == "" {
			return nil, fmt.Errorf("缺少消息正文")
		}
		args := []string{"chat", "+messages-send", "--as", "user"}
		switch {
		case strings.TrimSpace(strInput(input, "conversation_id")) != "":
			args = append(args, "--group", strings.TrimSpace(strInput(input, "conversation_id")))
		case strings.TrimSpace(strInput(input, "group_name")) != "":
			args = append(args, "--chat-query", strings.TrimSpace(strInput(input, "group_name")))
		case strings.TrimSpace(strInput(input, "open_dingtalk_id")) != "":
			args = append(args, "--open-dingtalk-id", strings.TrimSpace(strInput(input, "open_dingtalk_id")))
		case strings.TrimSpace(strInput(input, "user_id")) != "":
			args = append(args, "--user", strings.TrimSpace(strInput(input, "user_id")))
		case strings.TrimSpace(strInput(input, "user_name")) != "":
			args = append(args, "--user-query", strings.TrimSpace(strInput(input, "user_name")))
		default:
			return nil, fmt.Errorf("发送消息需要群或人的目标")
		}
		args = append(args, "--text", text, "--idempotency-key", dingTalkToolKey(name, input))
		return args, nil
	case "reply_message":
		messageID := strings.TrimSpace(strInput(input, "message_id"))
		if messageID == "" || text == "" {
			return nil, fmt.Errorf("回复需要 message_id 与 text")
		}
		args := []string{"chat", "+messages-reply", "--as", "user", "--message-id", messageID, "--content", text}
		if conv := strings.TrimSpace(strInput(input, "conversation_id")); conv != "" {
			args = append(args, "--group", conv)
		}
		args = append(args, "--idempotency-key", dingTalkToolKey(name, input))
		return args, nil
	}
	return nil, fmt.Errorf("未知钉钉工具 %s", name)
}

func dingTalkToolKey(name string, input map[string]any) string {
	raw, _ := json.Marshal(sortedAny(input))
	sum := sha256.Sum256(append([]byte(name+":"), raw...))
	return hex.EncodeToString(sum[:])[:32]
}

func sortedAny(value any) any {
	switch v := value.(type) {
	case map[string]any:
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		out := make(map[string]any, len(v))
		for _, k := range keys {
			out[k] = sortedAny(v[k])
		}
		return out
	case []any:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = sortedAny(item)
		}
		return out
	default:
		return value
	}
}

func strInput(input map[string]any, key string) string {
	if v, ok := input[key].(string); ok {
		return v
	}
	return ""
}

func intInput(input map[string]any, key string, fallback int) int {
	switch v := input[key].(type) {
	case float64:
		return int(v)
	case string:
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			return n
		}
	}
	return fallback
}

// runDwsCommand 调用本机 dws 并按 JSON 优先解析输出，返回统一的结果文本。
func runDwsCommand(ctx context.Context, args ...string) (string, error) {
	dingTalk.mu.Lock()
	bin := dingTalk.cfg.Bin
	dingTalk.mu.Unlock()
	if strings.TrimSpace(bin) == "" {
		bin = "dws"
	}
	full := append(append([]string{}, args...), "-f", "json", "-y", "--timeout", "30")
	cmd := exec.CommandContext(ctx, bin, full...)
	raw, err := cmd.Output()
	text := strings.TrimSpace(string(raw))
	if err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			stderr := strings.TrimSpace(string(exit.Stderr))
			if stderr != "" {
				return "", fmt.Errorf("dws 退出 %d: %s", exit.ExitCode(), truncateText(stderr, 500))
			}
		}
		if ctx.Err() != nil {
			return "", fmt.Errorf("dws 超时")
		}
		return "", fmt.Errorf("dws 调用失败: %w", err)
	}
	if text == "" {
		return "{}", nil
	}
	var parsed any
	if json.Unmarshal([]byte(text), &parsed) == nil {
		pretty, _ := json.Marshal(parsed)
		return truncateText(string(pretty), 64*1024), nil
	}
	return truncateText(text, 64*1024), nil
}
