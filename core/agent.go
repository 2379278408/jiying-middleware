package core

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

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
	// MaxContextBytes 限制单次请求组装后的上下文总量（含系统提示与当前输入）。
	MaxContextBytes int `json:"max_context_bytes"`
	// MaxTools 限制单次运行实际执行的工具调用总数。
	MaxTools int `json:"max_tools"`
	// TimeoutSeconds 是单次模型请求的超时时间；0 表示使用默认值。
	TimeoutSeconds int `json:"timeout_seconds,omitempty"`
	// AppSystems 按应用覆盖系统提示；未配置或为空时使用全局 System。
	AppSystems map[string]string `json:"app_systems,omitempty"`
	// Temperature 是采样温度；0 表示不显式传给上游。
	Temperature float64 `json:"temperature,omitempty"`
	// AppOverrides 按应用覆盖模型、轮次与温度，未配置的字段继承全局值。
	AppOverrides map[string]AppAgentOverride `json:"app_overrides,omitempty"`
}

// AppAgentOverride 描述单个应用对 Agent 运行参数的覆盖。
type AppAgentOverride struct {
	Model       string  `json:"model,omitempty"`
	MaxTurns    int     `json:"max_turns,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
}

const agentConfigKey = "agent_config"

const (
	agentMinTurns        = 1
	agentMaxTurns        = 12
	agentMinHistory      = 1
	agentMaxHistory      = 50
	agentMinContextBytes = 32 * 1024
	agentMaxContextBytes = 1024 * 1024
	agentDefaultContext  = 128 * 1024
	agentMinTools        = 1
	agentMaxTools        = 32
	agentDefaultTools    = 8
	agentMinTimeout      = 30
	agentMaxTimeout      = 600
	agentDefaultTimeout  = 150
	agentToolResultCap   = 12 * 1024
	// agentToolResultTotalCap 限制单次运行回灌给模型的全部工具结果总量。
	agentToolResultTotalCap = 64 * 1024
	agentMaxPromptRunes     = 8000
)

func clampInt(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func normalizeAgentLimits(cfg *AgentConfig) {
	if cfg.MaxTurns == 0 {
		cfg.MaxTurns = 6
	}
	if cfg.MaxHistory == 0 {
		cfg.MaxHistory = 12
	}
	if cfg.MaxContextBytes == 0 {
		cfg.MaxContextBytes = agentDefaultContext
	}
	if cfg.MaxTools == 0 {
		cfg.MaxTools = agentDefaultTools
	}
	if cfg.TimeoutSeconds == 0 {
		cfg.TimeoutSeconds = agentDefaultTimeout
	}
	cfg.MaxTurns = clampInt(cfg.MaxTurns, agentMinTurns, agentMaxTurns)
	cfg.MaxHistory = clampInt(cfg.MaxHistory, agentMinHistory, agentMaxHistory)
	cfg.MaxContextBytes = clampInt(cfg.MaxContextBytes, agentMinContextBytes, agentMaxContextBytes)
	cfg.MaxTools = clampInt(cfg.MaxTools, agentMinTools, agentMaxTools)
	cfg.TimeoutSeconds = clampInt(cfg.TimeoutSeconds, agentMinTimeout, agentMaxTimeout)
}

func LoadAgentConfig() AgentConfig {
	cfg := AgentConfig{Model: "gpt-4o-mini", MaxTurns: 6, MaxHistory: 12, MaxContextBytes: agentDefaultContext, MaxTools: agentDefaultTools, TimeoutSeconds: agentDefaultTimeout}
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
				if len(stored.AppSystems) > 0 {
					cfg.AppSystems = make(map[string]string, len(stored.AppSystems))
					for id, prompt := range stored.AppSystems {
						if id = strings.TrimSpace(id); id != "" {
							cfg.AppSystems[id] = prompt
						}
					}
				}
				cfg.Temperature = normalizeTemperature(stored.Temperature)
				if len(stored.AppOverrides) > 0 {
					cfg.AppOverrides = make(map[string]AppAgentOverride, len(stored.AppOverrides))
					for id, ov := range stored.AppOverrides {
						if id = strings.TrimSpace(id); id != "" {
							ov.Model = strings.TrimSpace(ov.Model)
							ov.MaxTurns = clampInt(ov.MaxTurns, 0, agentMaxTurns)
							ov.Temperature = normalizeTemperature(ov.Temperature)
							cfg.AppOverrides[id] = ov
						}
					}
				}
				// 已保存的配置是权威来源，允许管理员清空后备模型列表。
				cfg.Fallback = stored.Fallback
				if stored.MaxTurns > 0 {
					cfg.MaxTurns = stored.MaxTurns
				}
				if stored.MaxHistory > 0 {
					cfg.MaxHistory = stored.MaxHistory
				}
				if stored.MaxContextBytes > 0 {
					cfg.MaxContextBytes = stored.MaxContextBytes
				}
				if stored.MaxTools > 0 {
					cfg.MaxTools = stored.MaxTools
				}
				if stored.TimeoutSeconds > 0 {
					cfg.TimeoutSeconds = stored.TimeoutSeconds
				}
				cfg.Enabled = stored.Enabled
				if key := openSecret(stored.APIKey); key != "" {
					cfg.APIKey = key
				}
			}
		}
	}
	normalizeAgentLimits(&cfg)
	return cfg
}

func SaveAgentConfig(cfg AgentConfig) {
	normalizeAgentLimits(&cfg)
	if BucketApp == nil {
		return
	}
	stored := cfg
	sealed, err := sealSecret(cfg.APIKey)
	if err != nil {
		GetMyLog().Error("模型密钥加密失败，已取消保存: %v", err)
		return
	}
	stored.APIKey = sealed
	raw, _ := json.Marshal(stored)
	BucketApp.Set(agentConfigKey, string(raw))
}

func (c AgentConfig) ready() bool {
	return c.Enabled && c.BaseURL != "" && c.APIKey != "" && c.Model != ""
}

// agentRunTimeout 为一次 Agent 运行预留总预算：至少覆盖单次模型请求的
// 配置超时并留出工具执行时间，同时保留原有 240 秒下限。
func agentRunTimeout(cfg AgentConfig) time.Duration {
	per := time.Duration(clampInt(cfg.TimeoutSeconds, agentMinTimeout, agentMaxTimeout)+60) * time.Second
	if per < 240*time.Second {
		return 240 * time.Second
	}
	return per
}

func normalizeTemperature(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 2 {
		return 2
	}
	return v
}

// ForApp 返回应用级运行参数：模型、轮次与温度按应用覆盖，其余字段继承全局。
func (c AgentConfig) ForApp(appID string) AgentConfig {
	appID = strings.TrimSpace(appID)
	if appID == "" || len(c.AppOverrides) == 0 {
		return c
	}
	ov, ok := c.AppOverrides[appID]
	if !ok {
		return c
	}
	if ov.Model != "" {
		c.Model = ov.Model
	}
	if ov.MaxTurns > 0 {
		c.MaxTurns = clampInt(ov.MaxTurns, agentMinTurns, agentMaxTurns)
	}
	if ov.Temperature > 0 {
		c.Temperature = ov.Temperature
	}
	return c
}

// SetAppOverride 写入或清除应用级运行参数；空值表示恢复继承全局配置。
func SetAppOverride(appID string, ov AppAgentOverride) error {
	appID = strings.TrimSpace(appID)
	if appID == "" {
		return fmt.Errorf("缺少应用 ID")
	}
	ov.Model = strings.TrimSpace(ov.Model)
	if len(ov.Model) > 200 {
		return fmt.Errorf("模型名过长")
	}
	ov.MaxTurns = clampInt(ov.MaxTurns, 0, agentMaxTurns)
	ov.Temperature = normalizeTemperature(ov.Temperature)
	cfg := LoadAgentConfig()
	if ov.Model == "" && ov.MaxTurns == 0 && ov.Temperature == 0 {
		if cfg.AppOverrides != nil {
			delete(cfg.AppOverrides, appID)
		}
	} else {
		if cfg.AppOverrides == nil {
			cfg.AppOverrides = map[string]AppAgentOverride{}
		}
		cfg.AppOverrides[appID] = ov
	}
	SaveAgentConfig(cfg)
	return nil
}

// SystemForApp 返回指定应用的系统提示；未配置应用级提示时回退全局提示。
func (c AgentConfig) SystemForApp(appID string) string {
	appID = strings.TrimSpace(appID)
	if appID != "" {
		if prompt, ok := c.AppSystems[appID]; ok && strings.TrimSpace(prompt) != "" {
			return prompt
		}
	}
	return c.System
}

// SetAppSystem 写入或清除应用级系统提示；空字符串表示恢复继承全局提示。
func SetAppSystem(appID, prompt string) error {
	appID = strings.TrimSpace(appID)
	if appID == "" {
		return fmt.Errorf("缺少应用 ID")
	}
	prompt = strings.TrimSpace(prompt)
	if len(prompt) > 20000 {
		return fmt.Errorf("系统提示过长，最多 20000 字符")
	}
	cfg := LoadAgentConfig()
	if cfg.AppSystems == nil {
		cfg.AppSystems = map[string]string{}
	}
	if prompt == "" {
		delete(cfg.AppSystems, appID)
	} else {
		cfg.AppSystems[appID] = prompt
	}
	SaveAgentConfig(cfg)
	return nil
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

// agentToolName 用功能名哈希生成稳定工具名，避免增删插件导致编号漂移。
func agentToolName(feature string) string {
	sum := sha1.Sum([]byte(strings.TrimSpace(feature)))
	return "tool_" + hex.EncodeToString(sum[:5])
}

// buildAgentTools 为已启用且当前用户有权执行的功能生成工具定义与反查表。
func buildAgentTools(userID string) ([]map[string]any, map[string]string, []agentTool) {
	return buildAgentToolsForApp(userID, CurrentAppConfig().AppID)
}

// buildAgentToolsForApp 按会话所属应用生成工具定义，避免把未授权插件暴露给模型。
func buildAgentToolsForApp(userID, appID string) ([]map[string]any, map[string]string, []agentTool) {
	defs := make([]map[string]any, 0)
	index := map[string]string{}
	names := make([]agentTool, 0)
	contracts := BuildToolContractsForApp(userID, appID)
	contracts = append(contracts, dingTalkToolContracts(userID, appID)...)
	for _, c := range contracts {
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

// deleteAgentHistory 清除指定会话的内存与落盘历史，供应用删除级联清理。
func deleteAgentHistory(convID string) { resetAgentHistory(convID) }

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
func loadServerHistory(ctx context.Context, convID string, seq int64, appID string) []chatMessage {
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
	return messagesToChatHistory(msgs, strings.TrimSpace(appID), seq)
}

// agentContext 组装 Agent 的既有上下文：优先服务端可见历史，不可用时回退本地短历史。
func agentContext(ctx context.Context, convID string, seq int64, appID string) []chatMessage {
	if h := loadServerHistory(ctx, convID, seq, appID); len(h) > 0 {
		return h
	}
	return getAgentHistory(convID)
}

// agentGroundRule 是每次对话都追加的可信度约束：实时数据先调工具，失败就如实说明。
const agentGroundRule = "回答实时数据前必须先调用工具；工具失败或没有结果时，如实说明无法获取，不要编造数据或图片链接。"

// agentSystemPrompt 把当前用户身份与可信度约束注入系统提示。
// 用户可在控制台自定义 system，身份与约束始终追加在末尾，避免被覆盖。
func agentSystemPrompt(base, userID string) string {
	parts := make([]string, 0, 3)
	if base = strings.TrimSpace(base); base != "" {
		parts = append(parts, base)
	}
	if userID != "" {
		role := "普通用户"
		if privilegedUser(userID) {
			role = "主控权限"
		}
		parts = append(parts, "当前用户ID："+userID+"（"+role+"）")
	}
	parts = append(parts, agentGroundRule)
	return strings.Join(parts, "\n\n")
}

// agentRealtimeKeywords 是实时数据类意图的关键词；命中且存在同名/同描述工具时，
// 首轮强制模型选择工具，减少「不调用工具直接编造数据」的概率。
var agentRealtimeKeywords = []string{
	"天气", "气温", "降雨", "空气质量", "新闻", "热搜", "股价", "股票", "基金",
	"汇率", "油价", "金价", "比分", "赛程", "航班", "快递", "物流", "彩票",
	"限行", "实时", "现在几点", "当前时间",
}

// agentToolChoice 决定本轮 tool_choice：命中实时意图且有对应工具时返回 required。
func agentToolChoice(prompt string, defs []map[string]any) string {
	p := strings.ToLower(prompt)
	hit := ""
	for _, k := range agentRealtimeKeywords {
		if strings.Contains(p, k) {
			hit = k
			break
		}
	}
	if hit == "" {
		return ""
	}
	for _, d := range defs {
		fn, ok := d["function"].(map[string]any)
		if !ok {
			continue
		}
		name := fmt.Sprint(fn["name"])
		desc := fmt.Sprint(fn["description"])
		if strings.Contains(name, hit) || strings.Contains(desc, hit) {
			return "required"
		}
	}
	return ""
}

// agentMarkdownImageRE 匹配 Markdown 图片语法；Agent 未真正发送图片时降级为普通链接。
var agentMarkdownImageRE = regexp.MustCompile(`!\[([^\]]*)\]\(([^)\s]+)\)`)

// sanitizeAgentReply 阻止模型用 Markdown 图片语法伪造「已发送图片」。
// Agent 需要发图时必须调用对应工具，最终文本里的图片语法统一降级为链接。
func sanitizeAgentReply(out string) string {
	if strings.TrimSpace(out) == "" {
		return out
	}
	return agentMarkdownImageRE.ReplaceAllString(out, "[图片：$1]($2)")
}

// agentMessagesBytes 估算一组消息占用的上下文字节数，用于全局预算控制。
func agentMessagesBytes(msgs []chatMessage) int {
	total := 0
	for _, m := range msgs {
		total += len(m.Content) + len(m.ToolCallID)
		for _, tc := range m.ToolCalls {
			total += len(tc.Function.Name) + len(tc.Function.Arguments)
		}
	}
	return total
}

// trimAgentHistory 从最旧的消息开始丢弃，直到历史总量不超过预算。
// 历史只含普通 user/assistant 消息，不会破坏工具调用配对。
func trimAgentHistory(msgs []chatMessage, budget int) []chatMessage {
	if budget < 0 {
		budget = 0
	}
	start := 0
	total := agentMessagesBytes(msgs)
	for start < len(msgs) && total > budget {
		total -= agentMessagesBytes(msgs[start : start+1])
		start++
	}
	return msgs[start:]
}

// capAgentToolOutput 在单次运行的总结果预算内截断工具输出。
func capAgentToolOutput(s string, remaining int) (string, int) {
	if remaining <= 0 {
		return "工具结果预算已用尽，请直接回答用户。", 0
	}
	if len(s) <= remaining {
		return s, len(s)
	}
	cut := remaining
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	if cut == 0 {
		return "工具结果预算已用尽，请直接回答用户。", 0
	}
	return s[:cut] + "\n...[结果已按预算截断]", cut
}

// RunAgent 执行一轮薄 Agent：模型决策、调用功能、回灌结果，直到出最终答复。
// seq 为触发消息的会话序号，用于读取服务端历史；无序号时回退本地历史。
func RunAgent(ctx context.Context, convID, userID, prompt string, seq int64) (string, error) {
	cfg := LoadAgentConfig()
	if !cfg.ready() {
		return "", fmt.Errorf("Agent 未配置：请设置 USER_LLM_BASE_URL / USER_LLM_API_KEY / USER_LLM_MODEL")
	}
	if utf8.RuneCountInString(prompt) > agentMaxPromptRunes {
		return "", fmt.Errorf("输入过长，最多 %d 字符", agentMaxPromptRunes)
	}
	appID := appIDForConversation(convID)
	cfg = cfg.ForApp(appID)
	defs, index, _ := buildAgentToolsForApp(userID, appID)
	runtime := newAgentRuntime(cfg)
	msgs := []chatMessage{}
	system := agentSystemPrompt(cfg.SystemForApp(appID), userID)
	if system != "" {
		msgs = append(msgs, chatMessage{Role: "system", Content: system})
	}
	remaining := cfg.MaxContextBytes - agentMessagesBytes(msgs) - len(prompt)
	if remaining < 0 {
		return "", fmt.Errorf("上下文预算不足，请缩短系统提示或输入")
	}
	msgs = append(msgs, trimAgentHistory(agentContext(ctx, convID, seq, appID), remaining)...)
	msgs = append(msgs, chatMessage{Role: "user", Content: prompt})

	turns := cfg.MaxTurns
	if turns <= 0 {
		turns = 6
	}
	maxTools := cfg.MaxTools
	if maxTools <= 0 {
		maxTools = agentDefaultTools
	}
	usedTools := 0
	toolResultBytes := 0
	for i := 0; i < turns; i++ {
		if agentMessagesBytes(msgs) > cfg.MaxContextBytes {
			return "", fmt.Errorf("上下文超过预算，请缩短输入或减少工具输出")
		}
		// 首轮命中实时类意图时强制选择工具，避免模型直接编造实时数据。
		// 工具预算用尽后不再下发工具定义，让模型基于已有结果直接作答。
		callDefs := defs
		choice := ""
		if usedTools >= maxTools {
			callDefs = nil
		} else if i == 0 {
			choice = agentToolChoice(prompt, defs)
		}
		resp, err := runtime.Complete(ctx, msgs, callDefs, choice)
		if err != nil {
			return "", err
		}
		if len(resp.Choices) == 0 {
			return "", fmt.Errorf("模型无响应")
		}
		msg := resp.Choices[0].Message
		if len(msg.ToolCalls) == 0 {
			out := sanitizeAgentReply(msg.Content)
			appendAgentHistory(convID, chatMessage{Role: "user", Content: prompt}, chatMessage{Role: "assistant", Content: out})
			return out, nil
		}
		msgs = append(msgs, msg)
		for _, tc := range msg.ToolCalls {
			if usedTools >= maxTools {
				msgs = append(msgs, chatMessage{Role: "tool", ToolCallID: tc.ID, Content: "工具调用预算已用尽，请直接回答用户。"})
				continue
			}
			usedTools++
			if tc.Function.Name == askChoiceTool {
				out := runAgentAskChoice(ctx, convID, userID, tc.Function.Arguments)
				msgs = append(msgs, chatMessage{Role: "tool", ToolCallID: tc.ID, Content: out})
				continue
			}
			feature := index[tc.Function.Name]
			out := runAgentTool(ctx, feature, tc.Function.Arguments, userID, convID)
			out = truncateAgentToolResult(out)
			out, used := capAgentToolOutput(out, agentToolResultTotalCap-toolResultBytes)
			toolResultBytes += used
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
func resumeAgentChoice(choiceMessageID, conversationID, userID string, optionIDs []string) bool {
	v, ok := agentChoiceWaits.Load(choiceMessageID)
	if !ok {
		return false
	}
	w := v.(agentChoiceWait)
	if w.ConversationID != "" && conversationID != "" && w.ConversationID != conversationID {
		return false
	}
	if w.UserID != "" && userID != "" && w.UserID != userID {
		return false
	}
	agentChoiceWaits.Delete(choiceMessageID)
	if w.ConversationID != "" {
		conversationID = w.ConversationID
	}
	if w.UserID != "" {
		userID = w.UserID
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
		return "FAILED: 未知工具"
	}
	appID := appIDForConversation(convID)
	if strings.HasPrefix(feature, "dingtalk:") {
		if !dingTalkToolAllowed(userID, appID) {
			RecordAuditApp(appID, userID, "dingtalk.tool", strings.TrimPrefix(feature, "dingtalk:"), rawArgs, "拒绝: 无权限或应用未绑定")
			return "FAILED: 无权限使用钉钉工具"
		}
		out := runDingTalkTool(strings.TrimPrefix(feature, "dingtalk:"), rawArgs)
		result := "成功"
		if strings.HasPrefix(out, "FAILED:") {
			result = "失败: " + strings.TrimPrefix(out, "FAILED: ")
		}
		RecordAuditApp(appID, userID, "dingtalk.tool", strings.TrimPrefix(feature, "dingtalk:"), rawArgs, result)
		return out
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
		AppID:          appID,
		Silent:         true,
	})
	if err != nil {
		RecordAuditApp(appID, userID, "agent.tool", feature, args, "拒绝: "+err.Error())
		return "FAILED: 执行被拒绝: " + err.Error()
	}
	// 等待窗口覆盖任务执行上限，并随 Agent 上下文取消，避免任务超时后
	// 把仍在 running 的快照误判为成功结果。
	done := WaitForTaskDoneContext(ctx, t.ID, taskRunTimeout+10*time.Second)
	if done == nil {
		RecordAuditApp(appID, userID, "agent.tool", feature, args, "执行超时")
		return "FAILED: 执行超时\n工具执行超时，请如实告知用户，不要编造结果。"
	}
	if done.Status != taskDone {
		reason := done.Result
		if reason == "" {
			reason = done.Status
		}
		RecordAuditApp(appID, userID, "agent.tool", feature, args, "失败: "+redactSecretText(reason))
		return "FAILED: " + redactSecretText(reason) + "\n工具执行失败，请如实告知用户，不要编造结果。"
	}
	RecordAuditApp(appID, userID, "agent.tool", feature, args, "成功")
	return redactSecretText(done.Result)
}

func callChat(ctx context.Context, cfg AgentConfig, model string, msgs []chatMessage, tools []map[string]any, toolChoice string) (*chatResponse, error) {
	body := map[string]any{"model": model, "messages": msgs}
	if cfg.Temperature > 0 {
		body["temperature"] = cfg.Temperature
	}
	if len(tools) > 0 {
		body["tools"] = tools
		if toolChoice == "" {
			toolChoice = "auto"
		}
		body["tool_choice"] = toolChoice
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
	timeout := time.Duration(clampInt(cfg.TimeoutSeconds, agentMinTimeout, agentMaxTimeout)) * time.Second
	client := &http.Client{Timeout: timeout}
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
			"enabled":           cfg.Enabled,
			"base_url":          maskURL(cfg.BaseURL),
			"model":             cfg.Model,
			"fallback_models":   cfg.Fallback,
			"system":            cfg.System,
			"app_systems":       cfg.AppSystems,
			"temperature":       cfg.Temperature,
			"app_overrides":     cfg.AppOverrides,
			"max_turns":         cfg.MaxTurns,
			"max_history":       cfg.MaxHistory,
			"max_context_bytes": cfg.MaxContextBytes,
			"max_context_kb":    cfg.MaxContextBytes / 1024,
			"max_tools":         cfg.MaxTools,
			"timeout_seconds":   cfg.TimeoutSeconds,
			"has_key":           cfg.APIKey != "",
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
		if v := strings.TrimSpace(c.PostForm("temperature")); v != "" {
			var temp float64
			if _, err := fmt.Sscan(v, &temp); err == nil {
				cfg.Temperature = normalizeTemperature(temp)
			}
		}
		if v := strings.TrimSpace(c.PostForm("api_key")); v != "" && v != "****" {
			cfg.APIKey = v
		}
		if v := c.PostForm("max_turns"); v != "" {
			fmt.Sscan(v, &cfg.MaxTurns)
		}
		if v := c.PostForm("max_history"); v != "" {
			fmt.Sscan(v, &cfg.MaxHistory)
		}
		if v := strings.TrimSpace(c.PostForm("max_context_kb")); v != "" {
			var kb int
			if _, err := fmt.Sscan(v, &kb); err == nil {
				cfg.MaxContextBytes = kb * 1024
			}
		}
		if v := strings.TrimSpace(c.PostForm("max_context_bytes")); v != "" {
			var raw int
			if _, err := fmt.Sscan(v, &raw); err == nil {
				cfg.MaxContextBytes = raw
			}
		}
		if v := c.PostForm("max_tools"); v != "" {
			fmt.Sscan(v, &cfg.MaxTools)
		}
		if v := strings.TrimSpace(c.PostForm("timeout_seconds")); v != "" {
			var timeout int
			if _, err := fmt.Sscan(v, &timeout); err == nil {
				cfg.TimeoutSeconds = timeout
			}
		}
		normalizeAgentLimits(&cfg)
		SaveAgentConfig(cfg)
		RecordAudit(consoleUser(c), "agent.config", cfg.Model, fmt.Sprintf("enabled=%v", cfg.Enabled), "成功")
		jiyingJSON(c, "ok")
	})
	g.POST("/agent/app-system", requireRole(RoleMaster), func(c *gin.Context) {
		appID := strings.TrimSpace(c.PostForm("app_id"))
		prompt := c.PostForm("system")
		if err := SetAppSystem(appID, prompt); err != nil {
			jiyingErr(c, err.Error())
			return
		}
		RecordAudit(consoleUser(c), "agent.app_system", appID, fmt.Sprintf("%d 字符", len([]rune(strings.TrimSpace(prompt)))), "成功")
		jiyingJSON(c, "ok")
	})
	g.POST("/agent/app-override", requireRole(RoleMaster), func(c *gin.Context) {
		appID := strings.TrimSpace(c.PostForm("app_id"))
		var ov AppAgentOverride
		ov.Model = c.PostForm("model")
		if v := strings.TrimSpace(c.PostForm("max_turns")); v != "" {
			fmt.Sscan(v, &ov.MaxTurns)
		}
		if v := strings.TrimSpace(c.PostForm("temperature")); v != "" {
			fmt.Sscan(v, &ov.Temperature)
		}
		if err := SetAppOverride(appID, ov); err != nil {
			jiyingErr(c, err.Error())
			return
		}
		RecordAudit(consoleUser(c), "agent.app_override", appID, fmt.Sprintf("model=%s", ov.Model), "成功")
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

// truncateAgentToolResult 限制回灌给模型的工具结果大小，避免超长输出挤占上下文。
func truncateAgentToolResult(s string) string {
	if len(s) <= agentToolResultCap {
		return s
	}
	cut := agentToolResultCap
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut] + "\n...[结果已截断]"
}
