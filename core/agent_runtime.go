package core

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// AgentRuntime 抽象对话模型后端。内置实现走 OpenAI 兼容的 /chat/completions，
// 后续可替换为外部 agent SDK，而不影响工具契约与薄循环逻辑。
type AgentRuntime interface {
	Complete(ctx context.Context, msgs []chatMessage, tools []map[string]any) (*chatResponse, error)
}

// openAIRuntime 是 AgentRuntime 的内置 OpenAI 兼容实现。
type openAIRuntime struct {
	cfg AgentConfig
}

func (r openAIRuntime) Complete(ctx context.Context, msgs []chatMessage, tools []map[string]any) (*chatResponse, error) {
	models := candidateModels(r.cfg)
	return completeWithFallback(ctx, models, func(ctx context.Context, model string) (*chatResponse, error) {
		return callChat(ctx, r.cfg, model, msgs, tools)
	})
}

// completeWithFallback 按顺序尝试主模型与后备模型：首个成功者返回，全部失败返回最后一次错误。
// call 注入后可脱离网络单测降级链与审计语义。
func completeWithFallback(ctx context.Context, models []string, call func(ctx context.Context, model string) (*chatResponse, error)) (*chatResponse, error) {
	if len(models) == 0 {
		return nil, fmt.Errorf("未配置模型")
	}
	var lastErr error
	var failed []string
	for i, model := range models {
		resp, err := call(ctx, model)
		if err == nil {
			if i > 0 {
				GetMyLog().Warn("模型 %s 不可用，已降级到 %s", strings.Join(failed, ", "), model)
				RecordAudit("system", "agent.fallback", model, "降级使用；不可用: "+strings.Join(failed, ", "), "降级成功")
			}
			return resp, nil
		}
		lastErr = err
		failed = append(failed, model)
		if ctx.Err() != nil {
			return nil, err
		}
		if i < len(models)-1 {
			GetMyLog().Warn("模型 %s 调用失败，尝试后备模型: %v", model, err)
		}
	}
	if len(models) > 1 {
		GetMyLog().Warn("主模型与全部后备模型均失败: %v", lastErr)
		RecordAudit("system", "agent.fallback", models[0], "全部后备不可用: "+strings.Join(failed, ", "), "全部失败: "+lastErr.Error())
	}
	return nil, lastErr
}

// newAgentRuntime 依据配置构造运行时；当前只有内置实现。
func newAgentRuntime(cfg AgentConfig) AgentRuntime {
	return openAIRuntime{cfg: cfg}
}

var _ AgentRuntime = openAIRuntime{}

// ToolContract 是暴露给模型的工具契约，由功能与插件清单推导。
type ToolContract struct {
	Tool        string         `json:"tool"`
	Feature     string         `json:"feature"`
	Description string         `json:"description"`
	Identity    string         `json:"identity"`
	Scope       string         `json:"scope"`
	Params      []string       `json:"params"`
	Schema      map[string]any `json:"schema"`
}

// BuildToolContracts 为已启用且当前用户有权执行的功能生成工具契约（含 JSON Schema 参数）。
// 按功能名排序，保证同一环境下工具编号稳定，便于在线预览与复现。
func BuildToolContracts(userID string) []ToolContract {
	items := ListFeatureItems()
	sort.SliceStable(items, func(i, j int) bool {
		if items[i] == nil || items[j] == nil {
			return false
		}
		return items[i].Name < items[j].Name
	})
	out := make([]ToolContract, 0, len(items))
	for i, f := range items {
		if f == nil || !f.Enabled || !featureAllowedFor(userID, f) || !featureAllowedForApp(f, CurrentAppConfig().AppID) {
			continue
		}
		out = append(out, buildToolContract(agentToolName(i), f))
	}
	return out
}

func buildToolContract(tool string, f *Feature) ToolContract {
	if f == nil {
		return ToolContract{Tool: tool}
	}
	desc := fmt.Sprintf("执行中间件功能「%s」（类型 %s）", f.Name, f.Kind)
	params := []string{}
	if fn := functionByTitle(f.Name); fn != nil {
		if fn.Description != "" {
			desc = fn.Description
		}
		params = append(params, fn.Params...)
	}
	props := map[string]any{}
	valid := make([]string, 0, len(params))
	for _, p := range params {
		name, pdesc := splitParam(p)
		if name == "" {
			continue
		}
		valid = append(valid, p)
		props[name] = map[string]any{"type": "string", "description": pdesc}
	}
	if len(valid) == 0 {
		props["args"] = map[string]any{"type": "string", "description": "传给功能的参数，可为空"}
	}
	return ToolContract{
		Tool:        tool,
		Feature:     f.Name,
		Description: desc,
		Identity:    f.Identity,
		Scope:       f.Scope,
		Params:      valid,
		Schema: map[string]any{
			"type":       "object",
			"properties": props,
		},
	}
}
