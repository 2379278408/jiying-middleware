package core

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"strings"
	"sync"
	"time"

	"jiying/middleware/utils"
)

func HandleMessage(sender Sender) bool {
	if sender == nil {
		return false
	}
	RecordMessage(sender)
	msg := utils.TrimHiddenCharacter(sender.GetMsg())
	if msg == "" {
		return false
	}

	for _, f := range FunctionsSnapshot() {
		if f == nil || f.Disable {
			continue
		}
		if matchFunctionCommand(f, msg) {
			RecordCommand(f.Title)
			// 命中处理即回接收回执，不等脚本/Agent 执行完成。
			sendReceipt(sender)
			dispatch := cloneFunction(f)
			go func() {
				if !tryAcquireMsgExec() {
					GetMyLog().Warn("命令执行并发已满，暂缓执行: %s", dispatch.Title)
					sender.Reply("系统繁忙，请稍后再试")
					finishSender(sender)
					return
				}
				defer releaseMsgExec()
				stopStatus := startSenderStatus(sender)
				defer func() {
					stopStatus()
					if r := recover(); r != nil {
						GetMyLog().Warn("命令 %s 执行异常: %v", dispatch.Title, r)
					}
					// 脚本异常是确定性失败，重放不会成功；照常确认以避免毒消息循环。
					finishSender(sender)
				}()
				withSender(sender, func() { executeFunction(dispatch) })
			}()
			return true
		}
	}
	if js, ok := sender.(*JiyingSender); ok {
		if prompt, ok := agentWakePrompt(js); ok {
			sendReceipt(js)
			dispatchAgentReply(js, prompt)
			return true
		}
	}
	return false
}

// appReplyBudgetPerConv 限制单会话每日应用间自动回复总条数，防止无限对刷消耗 token。
// 配额按天持久化，进程重启后继续生效；跨天自动清零。
const (
	appReplyBudgetPerConv = 40
	appReplyBudgetKey     = "relay_app_reply_budget"
)

// appReplyBurstPerMin 限制单会话每分钟的应用间自动回复，避免短时间互相刷屏；
// 每日总量仍由持久化配额兜底。变量形式便于测试覆盖阈值。
var appReplyBurstPerMin = 12

type appReplyBudgetStore struct {
	Day    string         `json:"day"`
	Counts map[string]int `json:"counts"`
}

var appReplyBudget = struct {
	sync.Mutex
	day    string
	minute string
	counts map[string]int
	burst  map[string]int
	bucket string
	loaded bool
}{counts: map[string]int{}, burst: map[string]int{}}

// ensureAppReplyBudgetLocked 按天加载配额；存储桶或日期变化时重建缓存。
func ensureAppReplyBudgetLocked(today string) {
	key := ""
	if BucketApp != nil {
		key = BucketApp.String()
	}
	if appReplyBudget.loaded && appReplyBudget.bucket == key && appReplyBudget.day == today {
		return
	}
	appReplyBudget.loaded = true
	appReplyBudget.bucket = key
	appReplyBudget.day = today
	appReplyBudget.counts = map[string]int{}
	if BucketApp == nil {
		return
	}
	raw := BucketApp.GetString(appReplyBudgetKey)
	if raw == "" {
		return
	}
	var stored appReplyBudgetStore
	if json.Unmarshal([]byte(raw), &stored) != nil || strings.TrimSpace(stored.Day) != today {
		return
	}
	for convID, count := range stored.Counts {
		if convID == "" || count <= 0 {
			continue
		}
		appReplyBudget.counts[convID] = count
	}
}

// allowAppReplyBudget 为每个会话发放有限的每日应用间回复配额。
func allowAppReplyBudget(convID string) bool {
	convID = strings.TrimSpace(convID)
	if convID == "" {
		return false
	}
	now := time.Now()
	today := now.Format("2006-01-02")
	minute := now.Format("2006-01-02T15:04")
	appReplyBudget.Lock()
	defer appReplyBudget.Unlock()
	ensureAppReplyBudgetLocked(today)
	if appReplyBudget.minute != minute {
		appReplyBudget.minute = minute
		appReplyBudget.burst = map[string]int{}
	}
	if appReplyBudget.counts[convID] >= appReplyBudgetPerConv {
		return false
	}
	if appReplyBudget.burst[convID] >= appReplyBurstPerMin {
		return false
	}
	appReplyBudget.burst[convID]++
	appReplyBudget.counts[convID]++
	if BucketApp != nil {
		raw, err := json.Marshal(appReplyBudgetStore{Day: today, Counts: appReplyBudget.counts})
		if err == nil {
			BucketApp.Set(appReplyBudgetKey, string(raw))
		}
	}
	return true
}

// agentWakePrompt applies the conversation wake policy after command matching:
// private conversations accept ordinary text, while groups require an explicit
// mention of the connected app. App senders follow the same mention rule, so a
// group cannot wake every connected app just by posting ordinary text.
func agentWakePrompt(sender Sender) (string, bool) {
	js, ok := sender.(*JiyingSender)
	if !ok || js == nil {
		return "", false
	}
	appSender := js.SenderType == "app"
	fromApp := ""
	if appSender {
		fromApp = strings.TrimSpace(js.UserID)
		toApp := senderAppID(js)
		// 应用不能唤醒自己；来源缺失时不放行，避免协议异常导致回环。
		if fromApp == "" || (toApp != "" && fromApp == toApp) {
			GetMyLog().Info("应用消息未唤醒 conv=%s type=%s from=%s to=%s reason=self_or_empty",
				js.ChatID, js.SenderType, fromApp, toApp)
			return "", false
		}
	}
	prompt := utils.TrimHiddenCharacter(js.GetMsg())
	if prompt == "" {
		GetMyLog().Info("消息未唤醒 conv=%s type=%s reason=empty", js.ChatID, js.SenderType)
		return "", false
	}
	if !js.IsGroup() {
		if appSender && !allowAppReplyBudget(js.ChatID) {
			GetMyLog().Info("应用间对话达到上限 conv=%s from=%s", js.ChatID, fromApp)
			return "", false
		}
		if appSender {
			GetMyLog().Info("应用消息唤醒 conv=%s from=%s", js.ChatID, fromApp)
		}
		return prompt, true
	}
	appID := senderAppID(js)
	if appID == "" {
		GetMyLog().Info("消息未唤醒 conv=%s type=%s reason=no_app", js.ChatID, js.SenderType)
		return "", false
	}
	mention := "{(@app/" + appID + ")}"
	if !strings.Contains(prompt, mention) {
		GetMyLog().Info("群消息未唤醒 conv=%s type=%s reason=no_mention", js.ChatID, js.SenderType)
		return "", false
	}
	prompt = strings.TrimSpace(strings.ReplaceAll(prompt, mention, ""))
	if prompt == "" {
		// 只 @ 不带正文时也要唤醒；否则群聊里单独 @ 应用会被静默忽略。
		prompt = "对方@了你，请简短回应"
	}
	if appSender && !allowAppReplyBudget(js.ChatID) {
		GetMyLog().Info("应用间对话达到上限 conv=%s from=%s", js.ChatID, fromApp)
		return "", false
	}
	if appSender {
		GetMyLog().Info("应用消息唤醒 conv=%s from=%s", js.ChatID, fromApp)
	}
	return prompt, true
}

// senderAppID 优先取消息来源连接的应用；缺失时回退会话归属与默认应用。
func senderAppID(sender *JiyingSender) string {
	if sender != nil && sender.client != nil {
		if appID := strings.TrimSpace(sender.client.cfg.AppID); appID != "" {
			return appID
		}
	}
	if sender != nil {
		return appIDForConversation(sender.ChatID)
	}
	return strings.TrimSpace(CurrentAppConfig().AppID)
}

func dispatchAgentReply(sender *JiyingSender, prompt string) {
	go func() {
		if !tryAcquireMsgExec() {
			GetMyLog().Warn("Agent 回复并发已满，暂缓回复 conv=%s", sender.ChatID)
			_ = sender.Reply("系统繁忙，请稍后再试")
			finishSender(sender)
			return
		}
		defer releaseMsgExec()
		started := time.Now()
		stopStatus := startSenderStatus(sender)
		defer func() {
			stopStatus()
			if r := recover(); r != nil {
				GetMyLog().Warn("Agent 自动回复异常: %v", r)
			}
			finishSender(sender)
		}()
		ctx, cancel := context.WithTimeout(context.Background(), agentRunTimeout(LoadAgentConfig()))
		defer cancel()
		out, err := RunAgent(ctx, sender.ChatID, sender.UserID, prompt, sender.Seq)
		if err != nil {
			GetMyLog().Warn("Agent 自动回复失败 conv=%s user=%s 耗时=%s 错误=%v",
				sender.ChatID, sender.UserID, time.Since(started).Round(time.Millisecond), err)
			_ = sender.Reply(agentFailureMessage(err))
			return
		}
		if strings.TrimSpace(out) == "" {
			// 模型偶尔只回工具调用、正文为空；补一句兜底，避免用户看到"已回复但空白"。
			out = "我在，但这轮没生成有效回复，你再说一次？"
		}
		GetMyLog().Info("Agent 自动回复成功 conv=%s user=%s 提示=%d字 回复=%d字 耗时=%s",
			sender.ChatID, sender.UserID, len([]rune(prompt)), len([]rune(out)), time.Since(started).Round(time.Millisecond))
		// 回复应用时带上结构化 mention，让对端应用能被唤醒，形成双向对话。
		reply := out
		if sender.SenderType == "app" && strings.TrimSpace(sender.UserID) != "" {
			reply = "{(@app/" + strings.TrimSpace(sender.UserID) + ")} " + out
		}
		// Agent 输出常含标题、列表与表格，优先按 Markdown 渲染；发送失败再回退纯文本。
		if !sender.ReplyMarkdown(reply) {
			_ = sender.Reply(reply)
		}
	}()
}

// agentFailureMessage 把模型与运行错误转成可行动的用户提示，避免所有失败都显示同一句话。
func agentFailureMessage(err error) string {
	if err == nil {
		return "Agent 执行失败，请稍后重试。"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "Agent 处理超时，请缩短问题或稍后重试。"
	}
	if errors.Is(err, context.Canceled) {
		return "Agent 执行已取消，请重试。"
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return "Agent 模型响应超时，请稍后重试。"
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "未配置"):
		return "Agent 尚未配置，请联系管理员。"
	case strings.Contains(msg, "输入过长"), strings.Contains(msg, "上下文"):
		return "问题或上下文过长，请缩短后重试。"
	case strings.Contains(msg, "模型 HTTP 401"), strings.Contains(msg, "模型 HTTP 403"):
		return "Agent 模型鉴权失败，请联系管理员。"
	case strings.Contains(msg, "模型 HTTP 429"):
		return "Agent 模型限流，请稍后重试。"
	case strings.Contains(msg, "模型 HTTP"):
		return "Agent 模型服务异常，请稍后重试。"
	case strings.Contains(msg, "模型响应解析失败"):
		return "Agent 模型返回格式异常，请稍后重试。"
	case strings.Contains(msg, "模型错误"):
		return "Agent 模型拒绝了请求，请调整问题后重试。"
	case strings.Contains(msg, "最大轮次"):
		return "Agent 轮次已用尽，请拆分为更小的问题。"
	case strings.Contains(msg, "模型无响应"):
		return "Agent 模型未返回内容，请稍后重试。"
	default:
		return "Agent 执行失败，请稍后重试。"
	}
}

func matchFunctionCommand(f *Function, msg string) bool {
	if f == nil {
		return false
	}
	if f.Handle != "" && firstWordEqual(msg, f.Handle) {
		return true
	}
	if f.Title != "" && strings.HasPrefix(msg, f.Title) {
		rest := strings.TrimPrefix(msg, f.Title)
		if rest == "" || isCommandBoundary(rest) {
			return true
		}
	}
	return matchManifestRules(f, msg)
}

func firstWordEqual(msg, target string) bool {
	if !strings.HasPrefix(msg, target) {
		return false
	}
	rest := strings.TrimPrefix(msg, target)
	return rest == "" || isCommandBoundary(rest)
}

func isCommandBoundary(rest string) bool {
	for _, sp := range utils.DefaultSeparators {
		if strings.HasPrefix(rest, sp) {
			return true
		}
	}
	return false
}

func cloneFunction(f *Function) *Function {
	if f == nil {
		return nil
	}
	c := *f
	c.ImTypeWhite = append([]string(nil), f.ImTypeWhite...)
	c.ImTypeBlack = append([]string(nil), f.ImTypeBlack...)
	c.UserIdWhite = append([]string(nil), f.UserIdWhite...)
	c.UserIdBlack = append([]string(nil), f.UserIdBlack...)
	c.GroupIdWhite = append([]string(nil), f.GroupIdWhite...)
	c.GroupIdBlack = append([]string(nil), f.GroupIdBlack...)
	c.Routers = append([]string(nil), f.Routers...)
	c.Methods = append([]string(nil), f.Methods...)
	return &c
}

func executeFunction(f *Function) {
	if f == nil {
		return
	}
	// 主控专属命令（含清单声明 admin 的脚本）必须由主控触发，
	// 防止会话内任意成员通过消息直接执行系统级操作。
	if f.adminOnly() && !requireMaster() {
		return
	}
	if f.Origin {
		if handler, ok := cmdRouter.Load(f.Handle); ok {
			if fn, ok := handler.(func()); ok && fn != nil {
				fn()
				return
			}
		}
		GetMyLog().Warn("系统命令处理器缺失: %s", f.Handle)
		return
	}
	GetMyLog().Debug("执行脚本命令 %s", f.Title)
	_ = runScript(f)
}

// adminOnly 报告命令是否仅限主控执行：内置系统命令标记 Function.Admin，
// 脚本则可由清单的 admin 字段声明。
func (f *Function) adminOnly() bool {
	if f == nil {
		return false
	}
	return f.Admin || (f.Manifest.Admin != nil && *f.Manifest.Admin)
}

func dispatchMessage(s Sender) bool {
	return HandleMessage(s)
}
