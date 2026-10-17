package core

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// AgentConfig 是 OpenAI 兼容的对话模型配置；密钥单独加密存储。
type AgentConfig struct {
	Enabled    bool     `json:"enabled"`
	BaseURL    string   `json:"base_url"`
	APIKey     string   `json:"api_key,omitempty"`
	Model      string   `json:"model"`
	Fallback   []string `json:"fallback_models,omitempty"`
	System     string   `json:"system"`
	MaxTurns   int      `json:"max_turns"`
	MaxHistory int      `json:"max_history"`
}

const agentConfigKey = "agent_config"

func LoadAgentConfig() AgentConfig {
	cfg := AgentConfig{Model: "gpt-4o-mini", MaxTurns: 6, MaxHistory: 12}
	cfg.BaseURL = strings.TrimRight(strings.TrimSpace(os.Getenv("USER_LLM_BASE_URL")), "/")
	cfg.APIKey = strings.TrimSpace(os.Getenv("USER_LLM_API_KEY"))
	if m := strings.TrimSpace(os.Getenv("USER_LLM_MODEL")); m != "" {
		cfg.Model = m
	}
	cfg.Fallback = parseModelList(os.Getenv("USER_LLM_FALLBACK_MODELS"))
	if BucketApp != nil {
		if raw := BucketApp.GetString(agentConfigKey); raw != "" {
			var stored AgentConfig
			if json.Unmarshal([]byte(raw), &stored) == nil {
				if stored.BaseURL != "" {
					cfg.BaseURL = stored.BaseURL
				}
				if stored.Model != "" {
					cfg.Model = stored.Model
				}
				if stored.System != "" {
					cfg.System = stored.System
				}
				// 已保存的配置是权威来源，允许管理员清空后备模型列表。
				cfg.Fallback = stored.Fallback
				if stored.MaxTurns > 0 {
					cfg.MaxTurns = stored.MaxTurns
				}
				if stored.MaxHistory > 0 {
					cfg.MaxHistory = stored.MaxHistory
				}
				cfg.Enabled = stored.Enabled
				if key := openSecret(stored.APIKey); key != "" {
					cfg.APIKey = key
				}
			}
		}
	}
	return cfg
}

func SaveAgentConfig(cfg AgentConfig) {
	if BucketApp == nil {
		return
	}
	stored := cfg
	stored.APIKey = sealSecret(cfg.APIKey)
	raw, _ := json.Marshal(stored)
	BucketApp.Set(agentConfigKey, string(raw))
}

func (c AgentConfig) ready() bool {
	return c.Enabled && c.BaseURL != "" && c.APIKey != "" && c.Model != ""
}

// parseModelList 解析逗号、分号或换行分隔的模型列表，去空去重并保持顺序。
func parseModelList(raw string) []string {
	fields := strings.FieldsFunc(raw, func(r rune) bool {
		switch r {
		case ',', ';', '\n', '\r', '，', '；', '、':
			return true
		}
		return false
	})
	out := make([]string, 0, len(fields))
	seen := map[string]bool{}
	for _, f := range fields {
		f = strings.TrimSpace(f)
		if f == "" || seen[f] {
			continue
		}
		seen[f] = true
		out = append(out, f)
	}
	return out
}

// candidateModels 返回主模型加后备模型的去重顺序列表，主模型始终优先。
func candidateModels(cfg AgentConfig) []string {
	out := make([]string, 0, len(cfg.Fallback)+1)
	seen := map[string]bool{}
	primary := strings.TrimSpace(cfg.Model)
	if primary != "" {
		seen[primary] = true
		out = append(out, primary)
	}
	for _, m := range cfg.Fallback {
		m = strings.TrimSpace(m)
		if m == "" || seen[m] {
			continue
		}
		seen[m] = true
		out = append(out, m)
	}
	return out
}

// agentTool 把功能映射成模型可调用的工具。
type agentTool struct {
	Name    string
	Feature string
}

func agentToolName(i int) string { return fmt.Sprintf("tool_%d", i) }

// buildAgentTools 为已启用且当前用户有权执行的功能生成工具定义与反查表。
func buildAgentTools(userID string) ([]map[string]any, map[string]string, []agentTool) {
	defs := make([]map[string]any, 0)
	index := map[string]string{}
	names := make([]agentTool, 0)
	for _, c := range BuildToolContracts(userID) {
		index[c.Tool] = c.Feature
		names = append(names, agentTool{Name: c.Tool, Feature: c.Feature})
		defs = append(defs, map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        c.Tool,
				"description": c.Description,
				"parameters":  c.Schema,
			},
		})
	}
	defs = append(defs, askChoiceToolDef())
	return defs, index, names
}

// askChoiceTool 是内置工具名，用于向用户发送单选集卡片并挂起后续回答。
const askChoiceTool = "ask_choice"

func askChoiceToolDef() map[string]any {
	return map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        askChoiceTool,
			"description": "向当前会话发送单选集卡片，请用户从 2~20 个选项中选一个。调用后直接结束本轮回复，用户的选择会作为新消息继续对话。",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"question": map[string]any{"type": "string", "description": "卡片正文，1~5000 字符"},
					"options": map[string]any{
						"type": "array",
						"items": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"id":    map[string]any{"type": "string", "description": "选项 ID，1~64 位字母/数字/-/_"},
								"label": map[string]any{"type": "string", "description": "展示文本，1~200 字符且不含换行"},
							},
							"required": []string{"id", "label"},
						},
					},
				},
				"required": []string{"question", "options"},
			},
		},
	}
}

type chatMessage struct {
	Role       string         `json:"role"`
	Content    string         `json:"content"`
	ToolCalls  []chatToolCall `json:"tool_calls,omitempty"`
	ToolCallID string         `json:"tool_call_id,omitempty"`
}

type chatToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// agentHistory 保存每个会话的短上下文，按会话落盘并限容，避免无限增长与重启丢失。
const (
	agentHistKeyPrefix = "agent_hist_"
	agentMaxConvs      = 256
	agentMaxMsgLen     = 4000
)

var agentHistory = struct {
	sync.Mutex
	items map[string][]chatMessage
	order []string
}{items: map[string][]chatMessage{}}

func agentHistKey(convID string) string { return agentHistKeyPrefix + convID }

func trimAgentMessage(m chatMessage) chatMessage {
	if len(m.Content) > agentMaxMsgLen {
		m.Content = m.Content[:agentMaxMsgLen]
	}
	for i := range m.ToolCalls {
		if len(m.ToolCalls[i].Function.Arguments) > agentMaxMsgLen {
			m.ToolCalls[i].Function.Arguments = m.ToolCalls[i].Function.Arguments[:agentMaxMsgLen]
		}
	}
	return m
}

func loadAgentHistory(convID string) []chatMessage {
	if BucketApp == nil {
		return nil
	}
	raw := BucketApp.GetString(agentHistKey(convID))
	if raw == "" {
		return nil
	}
	var list []chatMessage
	if json.Unmarshal([]byte(raw), &list) != nil {
		return nil
	}
	return list
}

func getAgentHistory(convID string) []chatMessage {
	if convID == "" {
		return nil
	}
	agentHistory.Lock()
	list, ok := agentHistory.items[convID]
	if !ok {
		list = loadAgentHistory(convID)
		if list != nil {
			agentHistory.items[convID] = list
			agentHistory.order = append(agentHistory.order, convID)
		}
	} else {
		touchAgentConvLocked(convID)
	}
	cp := append([]chatMessage(nil), list...)
	agentHistory.Unlock()
	return cp
}

// touchAgentConvLocked 把会话移到 LRU 队尾，保证淘汰的是最久未用而非最早创建。
func touchAgentConvLocked(convID string) {
	for i, id := range agentHistory.order {
		if id != convID {
			continue
		}
		if i == len(agentHistory.order)-1 {
			return
		}
		agentHistory.order = append(agentHistory.order[:i], agentHistory.order[i+1:]...)
		agentHistory.order = append(agentHistory.order, convID)
		return
	}
}

func appendAgentHistory(convID string, msgs ...chatMessage) {
	if convID == "" {
		return
	}
	agentHistory.Lock()
	list, ok := agentHistory.items[convID]
	if !ok {
		list = loadAgentHistory(convID)
		if list == nil {
			list = []chatMessage{}
		}
		agentHistory.items[convID] = list
		agentHistory.order = append(agentHistory.order, convID)
	} else {
		touchAgentConvLocked(convID)
	}
	limit := LoadAgentConfig().MaxHistory
	if limit <= 0 {
		limit = 12
	}
	for _, m := range msgs {
		list = append(list, trimAgentMessage(m))
	}
	if len(list) > limit {
		list = append([]chatMessage(nil), list[len(list)-limit:]...)
	}
	agentHistory.items[convID] = list
	evictAgentConvsLocked()
	saveAgentHistoryLocked(convID, list)
	agentHistory.Unlock()
}

func saveAgentHistoryLocked(convID string, list []chatMessage) {
	if BucketApp == nil {
		return
	}
	raw, err := json.Marshal(list)
	if err != nil {
		return
	}
	BucketApp.Set(agentHistKey(convID), string(raw))
}

// evictAgentConvsLocked 在内存会话数超限时淘汰最久未用的会话，并清理对应持久层。
func evictAgentConvsLocked() {
	for len(agentHistory.order) > agentMaxConvs {
		oldest := agentHistory.order[0]
		agentHistory.order = agentHistory.order[1:]
		delete(agentHistory.items, oldest)
		if BucketApp != nil {
			BucketApp.Delete(agentHistKey(oldest))
		}
	}
}

func resetAgentHistory(convID string) {
	if convID == "" {
		return
	}
	agentHistory.Lock()
	delete(agentHistory.items, convID)
	out := agentHistory.order[:0]
	for _, c := range agentHistory.order {
		if c != convID {
			out = append(out, c)
		}
	}
	agentHistory.order = out
	if BucketApp != nil {
		BucketApp.Delete(agentHistKey(convID))
	}
	agentHistory.Unlock()
}

// loadServerHistory 读取服务端会话历史作为 Agent 上下文；seq 缺失或读取失败时返回 nil，
// 由本地历史回退。当前消息（seq）作为 prompt 单独传入，故从历史中剔除以免重复。
func loadServerHistory(ctx context.Context, convID string, seq int64) []chatMessage {
	if convID == "" || seq <= 0 {
		return nil
	}
	client := clientForConv(convID)
	if client == nil {
		return nil
	}
	limit := LoadAgentConfig().MaxHistory
	if limit <= 0 {
		limit = 30
	}
	if limit > 100 {
		limit = 100
	}
	msgs, err := client.ListMessages(ctx, convID, seq, limit)
	if err != nil {
		GetMyLog().Debug("读取服务端历史失败 conv=%s seq=%d: %v", convID, seq, err)
		return nil
	}
	return messagesToChatHistory(msgs, strings.TrimSpace(CurrentAppConfig().AppID), seq)
}

// agentContext 组装 Agent 的既有上下文：优先服务端可见历史，不可用时回退本地短历史。
func agentContext(ctx context.Context, convID string, seq int64) []chatMessage {
	if h := loadServerHistory(ctx, convID, seq); len(h) > 0 {
		return h
	}
	return getAgentHistory(convID)
}

// RunAgent 执行一轮薄 Agent：模型决策、调用功能、回灌结果，直到出最终答复。
// seq 为触发消息的会话序号，用于读取服务端历史；无序号时回退本地历史。
func RunAgent(ctx context.Context, convID, userID, prompt string, seq int64) (string, error) {
	cfg := LoadAgentConfig()
	if !cfg.ready() {
		return "", fmt.Errorf("Agent 未配置：请设置 USER_LLM_BASE_URL / USER_LLM_API_KEY / USER_LLM_MODEL")
	}
	defs, index, _ := buildAgentTools(userID)
	runtime := newAgentRuntime(cfg)
	msgs := []chatMessage{}
	if cfg.System != "" {
		msgs = append(msgs, chatMessage{Role: "system", Content: cfg.System})
	}
	msgs = append(msgs, agentContext(ctx, convID, seq)...)
	msgs = append(msgs, chatMessage{Role: "user", Content: prompt})

	turns := cfg.MaxTurns
	if turns <= 0 {
		turns = 6
	}
	for i := 0; i < turns; i++ {
		resp, err := runtime.Complete(ctx, msgs, defs)
		if err != nil {
			return "", err
		}
		if len(resp.Choices) == 0 {
			return "", fmt.Errorf("模型无响应")
		}
		msg := resp.Choices[0].Message
		if len(msg.ToolCalls) == 0 {
			appendAgentHistory(convID, chatMessage{Role: "user", Content: prompt}, chatMessage{Role: "assistant", Content: msg.Content})
			return msg.Content, nil
		}
		msgs = append(msgs, msg)
		for _, tc := range msg.ToolCalls {
			if tc.Function.Name == askChoiceTool {
				out := runAgentAskChoice(ctx, convID, userID, tc.Function.Arguments)
				msgs = append(msgs, chatMessage{Role: "tool", ToolCallID: tc.ID, Content: out})
				continue
			}
			feature := index[tc.Function.Name]
			out := runAgentTool(ctx, feature, tc.Function.Arguments, userID, convID)
			msgs = append(msgs, chatMessage{Role: "tool", ToolCallID: tc.ID, Content: out})
		}
	}
	return "", fmt.Errorf("Agent 超过最大轮次")
}

// agentChoiceWait 记录一次 ask_choice 挂起，用户选择后据此恢复 Agent 对话。
type agentChoiceWait struct {
	ConversationID string
	UserID         string
	Labels         map[string]string
}

var agentChoiceWaits sync.Map

// runAgentAskChoice 发送单选集卡片并挂起，用户选择后由 resumeAgentChoice 恢复。
func runAgentAskChoice(ctx context.Context, convID, userID, rawArgs string) string {
	var req struct {
		Question string `json:"question"`
		Options  []struct {
			ID    string `json:"id"`
			Label string `json:"label"`
		} `json:"options"`
	}
	if err := json.Unmarshal([]byte(rawArgs), &req); err != nil {
		return "参数解析失败：" + err.Error()
	}
	options := make([]map[string]string, 0, len(req.Options))
	labels := make(map[string]string, len(req.Options))
	for _, opt := range req.Options {
		options = append(options, map[string]string{"id": opt.ID, "label": opt.Label})
		labels[opt.ID] = opt.Label
	}
	if err := validateChoice(req.Question, options); err != nil {
		return "选择卡片参数不合法：" + err.Error()
	}
	client := clientForConv(convID)
	if client == nil {
		return "即应未连接，无法发送选择卡片"
	}
	id, err := client.SendChoice(ctx, convID, req.Question, options)
	if err != nil {
		return "发送选择卡片失败：" + err.Error()
	}
	agentChoiceWaits.Store(id, agentChoiceWait{ConversationID: convID, UserID: userID, Labels: labels})
	return "已发送选择卡片 message_id=" + id + "，请等待用户选择，本轮不要再重复提问。"
}

// resumeAgentChoice 在用户选择后异步恢复 Agent，并把最终答复发回会话。
func resumeAgentChoice(choiceMessageID, conversationID string, optionIDs []string) bool {
	v, ok := agentChoiceWaits.LoadAndDelete(choiceMessageID)
	if !ok {
		return false
	}
	w := v.(agentChoiceWait)
	if w.ConversationID != "" {
		conversationID = w.ConversationID
	}
	chosen := make([]string, 0, len(optionIDs))
	for _, id := range optionIDs {
		if label, ok := w.Labels[id]; ok && label != "" {
			chosen = append(chosen, label)
		} else {
			chosen = append(chosen, id)
		}
	}
	prompt := "用户已完成选择"
	if len(chosen) > 0 {
		prompt = "用户选择了：" + strings.Join(chosen, "、")
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		out, err := RunAgent(ctx, conversationID, w.UserID, prompt, 0)
		if err != nil {
			GetMyLog().Warn("Agent 处理选择结果失败 conv=%s: %v", conversationID, err)
			return
		}
		client := clientForConv(conversationID)
		if client == nil {
			return
		}
		if err := client.SendText(ctx, conversationID, "", out); err != nil {
			GetMyLog().Warn("Agent 回复选择结果失败 conv=%s: %v", conversationID, err)
		}
	}()
	return true
}

// buildArgsForTool 把模型的工具调用参数转换成功能输入字符串。
// 兼容两种形态：`{"args":"..."}` 直传，或按清单声明的命名参数依次取值拼接。
func buildArgsForTool(rawArgs string, params []string) string {
	var envelope struct {
		Args string `json:"args"`
	}
	if json.Unmarshal([]byte(rawArgs), &envelope) == nil && envelope.Args != "" {
		return envelope.Args
	}
	var obj map[string]any
	if json.Unmarshal([]byte(rawArgs), &obj) != nil {
		return rawArgs
	}
	parts := make([]string, 0, len(params))
	for _, p := range params {
		name, _ := splitParam(p)
		if name == "" {
			continue
		}
		if v, ok := obj[name]; ok && v != nil {
			switch val := v.(type) {
			case string:
				if val != "" {
					parts = append(parts, val)
				}
			case float64:
				parts = append(parts, strconv.FormatFloat(val, 'f', -1, 64))
			default:
				parts = append(parts, fmt.Sprintf("%v", val))
			}
		}
	}
	return strings.Join(parts, " ")
}

// runAgentTool 执行一次 Agent 工具调用：先按功能推导参数，再走任务队列（含限流配额），同步等待结果。
func runAgentTool(ctx context.Context, feature, rawArgs, userID, convID string) string {
	if feature == "" {
		return "未知工具"
	}
	params := []string{}
	if fn := functionByTitle(feature); fn != nil {
		params = fn.Params
	}
	args := buildArgsForTool(rawArgs, params)
	t, err := EnqueueTask(Task{
		Feature:        feature,
		Args:           args,
		UserID:         userID,
		ConversationID: convID,
		Silent:         true,
	})
	if err != nil {
		RecordAudit(userID, "agent.tool", feature, args, "拒绝: "+err.Error())
		return "执行被拒绝: " + err.Error()
	}
	done := WaitForTaskDone(t.ID, 60*time.Second)
	if done == nil {
		RecordAudit(userID, "agent.tool", feature, args, "执行超时")
		return "执行超时"
	}
	if done.Status == taskFailed {
		RecordAudit(userID, "agent.tool", feature, args, "失败: "+done.Result)
		return "执行失败: " + done.Result
	}
	RecordAudit(userID, "agent.tool", feature, args, "成功")
	return done.Result
}

func callChat(ctx context.Context, cfg AgentConfig, model string, msgs []chatMessage, tools []map[string]any) (*chatResponse, error) {
	body := map[string]any{"model": model, "messages": msgs}
	if len(tools) > 0 {
		body["tools"] = tools
		body["tool_choice"] = "auto"
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.BaseURL+"/chat/completions", bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	out, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	var parsed chatResponse
	if json.Unmarshal(out, &parsed) != nil {
		return nil, fmt.Errorf("模型响应解析失败")
	}
	if parsed.Error != nil && parsed.Error.Message != "" {
		return nil, fmt.Errorf("模型错误: %s", parsed.Error.Message)
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("模型 HTTP %d", resp.StatusCode)
	}
	return &parsed, nil
}

func registerAgentAPI() {
	g := Server.Group("/jiying/api", requireLogin)
	g.GET("/agent/config", requireRole(RoleMaster), func(c *gin.Context) {
		cfg := LoadAgentConfig()
		jiyingJSON(c, gin.H{
			"enabled":         cfg.Enabled,
			"base_url":        maskURL(cfg.BaseURL),
			"model":           cfg.Model,
			"fallback_models": cfg.Fallback,
			"system":          cfg.System,
			"max_turns":       cfg.MaxTurns,
			"max_history":     cfg.MaxHistory,
			"has_key":         cfg.APIKey != "",
		})
	})
	g.POST("/agent/config", requireRole(RoleMaster), func(c *gin.Context) {
		cfg := LoadAgentConfig()
		cfg.Enabled = c.PostForm("enabled") == "true" || c.PostForm("enabled") == "1"
		// 掩码值（如 http://********）表示前端未修改，不能当作真实地址落库。
		if v := strings.TrimRight(strings.TrimSpace(c.PostForm("base_url")), "/"); v != "" && !maskedSentinel(v) {
			cfg.BaseURL = v
		}
		if v := strings.TrimSpace(c.PostForm("model")); v != "" {
			cfg.Model = v
		}
		if raw, ok := c.GetPostForm("fallback_models"); ok {
			cfg.Fallback = parseModelList(raw)
		}
		cfg.System = c.PostForm("system")
		if v := strings.TrimSpace(c.PostForm("api_key")); v != "" && v != "****" {
			cfg.APIKey = v
		}
		if v := c.PostForm("max_turns"); v != "" {
			fmt.Sscan(v, &cfg.MaxTurns)
		}
		if v := c.PostForm("max_history"); v != "" {
			fmt.Sscan(v, &cfg.MaxHistory)
		}
		SaveAgentConfig(cfg)
		RecordAudit(consoleUser(c), "agent.config", cfg.Model, fmt.Sprintf("enabled=%v", cfg.Enabled), "成功")
		jiyingJSON(c, "ok")
	})
	g.GET("/agent/tools", requireRole(RoleOps), func(c *gin.Context) {
		jiyingJSON(c, BuildToolContracts(consoleUser(c)))
	})
	g.POST("/agent/run", requireRole(RoleOps), func(c *gin.Context) {
		convID := strings.TrimSpace(c.PostForm("conversation_id"))
		prompt := strings.TrimSpace(c.PostForm("prompt"))
		if prompt == "" {
			jiyingErr(c, "缺少内容")
			return
		}
		if c.PostForm("reset") == "true" {
			resetAgentHistory(convID)
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 120*time.Second)
		defer cancel()
		out, err := RunAgent(ctx, convID, consoleUser(c), prompt, 0)
		if err != nil {
			RecordAudit(consoleUser(c), "agent.run", convID, prompt, "失败: "+err.Error())
			jiyingErr(c, err.Error())
			return
		}
		RecordAudit(consoleUser(c), "agent.run", convID, prompt, "成功")
		jiyingJSON(c, out)
	})
}
