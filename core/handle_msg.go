package core

import (
	"context"
	"strings"
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

// agentWakePrompt applies the conversation wake policy after command matching:
// private app conversations accept ordinary text, while groups require an
// explicit mention of the connected app.
func agentWakePrompt(sender Sender) (string, bool) {
	js, ok := sender.(*JiyingSender)
	if !ok || js == nil || js.SenderType == "app" {
		return "", false
	}
	prompt := utils.TrimHiddenCharacter(js.GetMsg())
	if prompt == "" {
		return "", false
	}
	if !js.IsGroup() {
		return prompt, true
	}
	appID := strings.TrimSpace(CurrentAppConfig().AppID)
	mention := "{(@app/" + appID + ")}"
	if appID == "" || !strings.Contains(prompt, mention) {
		return "", false
	}
	prompt = strings.TrimSpace(strings.ReplaceAll(prompt, mention, ""))
	return prompt, prompt != ""
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
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()
		out, err := RunAgent(ctx, sender.ChatID, sender.UserID, prompt, sender.Seq)
		if err != nil {
			GetMyLog().Warn("Agent 自动回复失败 conv=%s user=%s 耗时=%s 错误=%v",
				sender.ChatID, sender.UserID, time.Since(started).Round(time.Millisecond), err)
			_ = sender.Reply("Agent 执行失败，请稍后重试。")
			return
		}
		GetMyLog().Info("Agent 自动回复成功 conv=%s user=%s 提示=%d字 回复=%d字 耗时=%s",
			sender.ChatID, sender.UserID, len([]rune(prompt)), len([]rune(out)), time.Since(started).Round(time.Millisecond))
		// Agent 输出常含标题、列表与表格，优先按 Markdown 渲染；发送失败再回退纯文本。
		if !sender.ReplyMarkdown(out) {
			_ = sender.Reply(out)
		}
	}()
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
