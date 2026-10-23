package core

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

type choiceWait struct {
	TaskID         string
	UserID         string
	ConversationID string
	CreatedAt      time.Time
}

// choiceWaitTTL 限制确认等待的最长时间，避免无人响应的任务永久占用内存。
const choiceWaitTTL = 24 * time.Hour

func pruneChoiceWaits() {
	now := time.Now()
	choiceWaits.Range(func(k, v any) bool {
		if w, ok := v.(choiceWait); ok && !w.CreatedAt.IsZero() && now.Sub(w.CreatedAt) > choiceWaitTTL {
			choiceWaits.Delete(k)
		}
		return true
	})
}

var choiceWaits sync.Map

func registerJiyingCommands() {
	registerSysCommand("帮助", "jy_help", 20, cmdHelpPublic)
	registerSysCommand("我的ID", "jy_my_id", 20, cmdMyID)
	registerSysCommand("状态", "jy_status", 20, cmdJiyingStatus)
	registerSysCommand("功能列表", "feature_list", 20, cmdFeatureList)
	registerSysCommand("任务列表", "task_list", 20, cmdTaskList)
	registerSysCommand("定时列表", "cron_list", 20, cmdCronList)
	registerSysCommand("插件列表", "plugin_list", 20, cmdPluginList)
	registerSysCommand("安装功能", "feature_install", 20, cmdFeatureInstall)
	registerSysCommand("安装脚本", "script_install", 20, cmdScriptInstall)
	registerSysCommand("启用功能", "feature_on", 20, cmdFeatureToggle(true))
	registerSysCommand("停用功能", "feature_off", 20, cmdFeatureToggle(false))
	registerSysCommand("增加定时", "cron_add", 20, cmdCronAdd)
	registerSysCommand("删除定时", "cron_del", 20, cmdCronDel)
	registerSysCommand("审计", "audit_list", 20, cmdAuditList)
	registerSysCommand("技能列表", "skill_list", 20, cmdSkillList)
	registerSysCommand("问", "agent_ask", 20, cmdAskAgent)
	registerSysCommand("市场列表", "market_list", 20, cmdMarketList)
	registerSysCommand("安装插件", "market_install", 20, cmdMarketInstall)
}

func cmdMyID() {
	userID := senderUserID(CurrentSender())
	if userID == "" {
		ReplyCurrent("无法识别当前账号 ID")
		return
	}
	ReplyCurrent("你的用户 ID：" + userID)
}

func senderUserID(s Sender) string {
	if js, ok := s.(*JiyingSender); ok {
		return js.UserID
	}
	if f, ok := s.(*Faker); ok && f.UserID != "" {
		return f.UserID
	}
	return ""
}

// senderSeq 返回触发消息的会话序号；非即应消息为 0（表示无服务端历史锚点）。
func senderSeq(s Sender) int64 {
	if js, ok := s.(*JiyingSender); ok {
		return js.Seq
	}
	return 0
}

func requireMaster() bool {
	s := CurrentSender()
	if s == nil {
		ReplyCurrent("无会话")
		return false
	}
	userID := senderUserID(s)
	if privilegedUser(userID) {
		return true
	}
	ReplyCurrent("无权限")
	return false
}

func cmdHelpPublic() {
	ReplyCurrent(runBuiltin("help", ""))
}

func cmdJiyingStatus() {
	ReplyCurrent(runBuiltin("status", ""))
}

func cmdFeatureList() {
	ReplyCurrent(ListFeatures())
}

func cmdAuditList() {
	if !requireMaster() {
		return
	}
	ReplyCurrent(AuditLogSummary(20))
}

func cmdSkillList() {
	skills := ListSkills()
	if len(skills) == 0 {
		ReplyCurrent("没有技能")
		return
	}
	names := make([]string, 0, len(skills))
	for _, s := range skills {
		names = append(names, s.Name)
	}
	ReplyCurrent("技能: " + strings.Join(names, "、"))
}

func cmdAskAgent() {
	s := CurrentSender()
	if s == nil {
		return
	}
	prompt := strings.TrimSpace(strings.TrimPrefix(s.GetMsg(), "问"))
	if prompt == "" {
		ReplyCurrent("用法: 问 <内容>")
		return
	}
	convID := fmt.Sprint(s.GetParentID())
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	out, err := RunAgent(ctx, convID, senderUserID(s), prompt, senderSeq(s))
	if err != nil {
		ReplyCurrent("Agent 执行失败: " + err.Error())
		return
	}
	ReplyCurrent(out)
}

func cmdMarketList() {
	items := ListMarket()
	if len(items) == 0 {
		ReplyCurrent("市场暂无插件")
		return
	}
	parts := make([]string, 0, len(items))
	for _, it := range items {
		state := "未安装"
		if it.Installed {
			state = "已安装"
		}
		parts = append(parts, fmt.Sprintf("%s@%s(%s/%s,%s)", it.Name, it.Version, it.Kind, it.Identity, state))
	}
	ReplyCurrent("市场: " + strings.Join(parts, "、"))
}

func cmdMarketInstall() {
	if !requireMaster() {
		return
	}
	s := CurrentSender()
	name := strings.TrimSpace(strings.TrimPrefix(s.GetMsg(), "安装插件"))
	if name == "" {
		ReplyCurrent("用法: 安装插件 <名字>")
		return
	}
	if err := InstallMarketPlugin(name, false); err != nil {
		ReplyCurrent("安装失败: " + err.Error())
		return
	}
	ReplyCurrent("已安装插件 " + name)
}

func cmdTaskList() {
	if !requireMaster() {
		return
	}
	ReplyCurrent(ListTasks(20))
}

func cmdCronList() {
	if !requireMaster() {
		return
	}
	ReplyCurrent(listCronTitles())
}

func cmdPluginList() {
	if !requireMaster() {
		return
	}
	ReplyCurrent(listPluginTitles())
}

func cmdFeatureInstall() {
	if !requireMaster() {
		return
	}
	s := CurrentSender()
	rest := strings.TrimSpace(strings.TrimPrefix(s.GetMsg(), "安装功能"))
	parts := strings.Fields(rest)
	if len(parts) < 2 {
		ReplyCurrent("用法: 安装功能 <名字> <http地址>")
		return
	}
	if err := validateOutboundURL(parts[1], false); err != nil {
		ReplyCurrent(err.Error())
		return
	}
	if err := UpsertFeature(Feature{Name: parts[0], Kind: "http", Target: parts[1], Scope: "master", Identity: "app", Enabled: true, Confirm: true}); err != nil {
		ReplyCurrent(err.Error())
		return
	}
	ReplyCurrent("已安装功能 " + parts[0])
}

func cmdScriptInstall() {
	if !requireMaster() {
		return
	}
	s := CurrentSender()
	rest := strings.TrimSpace(strings.TrimPrefix(s.GetMsg(), "安装脚本"))
	parts := strings.SplitN(rest, " ", 2)
	if len(parts) < 2 {
		ReplyCurrent("用法: 安装脚本 <文件名.js> <代码>")
		return
	}
	if err := InstallScript(parts[0], parts[1]); err != nil {
		ReplyCurrent("安装失败: " + err.Error())
		return
	}
	name := strings.TrimSuffix(filepathBase(parts[0]), ".js")
	_ = UpsertFeature(Feature{Name: name, Kind: "script", Target: filepathBase(parts[0]), Scope: "master", Identity: "app", Enabled: true})
	ReplyCurrent("脚本已测试并安装 " + filepathBase(parts[0]))
}

func filepathBase(name string) string {
	i := strings.LastIndex(name, "/")
	if i >= 0 {
		name = name[i+1:]
	}
	return name
}

func cmdFeatureToggle(on bool) func() {
	word := "停用功能"
	if on {
		word = "启用功能"
	}
	return func() {
		if !requireMaster() {
			return
		}
		s := CurrentSender()
		name := strings.TrimSpace(strings.TrimPrefix(s.GetMsg(), word))
		if name == "" {
			ReplyCurrent("用法: " + word + " <名字>")
			return
		}
		if err := SetFeatureEnabled(name, on); err != nil {
			ReplyCurrent(err.Error())
			return
		}
		if on {
			ReplyCurrent("已启用 " + name)
			return
		}
		ReplyCurrent("已停用 " + name)
	}
}

func cmdCronAdd() {
	if !requireMaster() {
		return
	}
	s := CurrentSender()
	rest := strings.TrimSpace(strings.TrimPrefix(s.GetMsg(), "增加定时"))
	tokens := strings.Fields(rest)
	if len(tokens) < 3 {
		ReplyCurrent("用法: 增加定时 <名字> <cron> <命令>")
		return
	}
	name := tokens[0]
	var cronSpec, command string
	if tokens[1] == "@every" {
		if len(tokens) < 4 {
			ReplyCurrent("用法: 增加定时 <名字> <cron> <命令>")
			return
		}
		cronSpec = tokens[1] + " " + tokens[2]
		command = strings.Join(tokens[3:], " ")
	} else {
		cronSpec = tokens[1]
		command = strings.Join(tokens[2:], " ")
	}
	chat := fmt.Sprint(s.GetParentID())
	appID := ""
	if js, ok := s.(*JiyingSender); ok {
		appID = senderAppID(js)
	}
	id := ccs.add(&CronCmd{
		Name: name, Cron: cronSpec, Command: command, Status: true,
		GroupManager: []string{chat}, OwnerRole: RoleMaster, Owner: "system",
		AppID: appID,
	})
	ReplyCurrent("已增加定时 " + id + " " + name)
}

func cmdCronDel() {
	if !requireMaster() {
		return
	}
	s := CurrentSender()
	name := strings.TrimSpace(strings.TrimPrefix(s.GetMsg(), "删除定时"))
	if name == "" {
		ReplyCurrent("用法: 删除定时 <名字或ID>")
		return
	}
	if !deleteCronByName(name) {
		ReplyCurrent("没有定时 " + name)
		return
	}
	ReplyCurrent("已删除定时 " + name)
}

func deleteCronByName(name string) bool {
	ccs.Lock()
	defer ccs.Unlock()
	out := ccs.Pts[:0]
	found := false
	for _, p := range ccs.Pts {
		if p != nil && (p.Name == name || p.ID == name) {
			if ccs.cronSched != nil && p.CronEntityId > 0 {
				ccs.cronSched.Remove(p.CronEntityId)
			}
			if ccs.Bucket != nil {
				ccs.Bucket.Delete(p.ID)
			}
			found = true
			continue
		}
		out = append(out, p)
	}
	ccs.Pts = out
	return found
}

func dispatchMiddleware(sender Sender, msg string) (handled bool, ok bool) {
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return false, false
	}
	for _, p := range []string{"安装功能", "安装脚本", "启用功能", "停用功能", "增加定时", "删除定时"} {
		if strings.HasPrefix(msg, p) {
			return false, false
		}
	}
	name, args, hit := ParseAgent(msg)
	if !hit {
		return false, false
	}
	f, exists := GetFeature(name)
	if !exists {
		sender.Reply("没有功能 " + name)
		return true, true
	}
	userID := senderUserID(sender)
	chatID := fmt.Sprint(sender.GetParentID())
	if !featureAllowedFor(userID, f, appIDForConversation(chatID)) {
		sender.Reply("无权限")
		return true, true
	}
	replyTo := ""
	if js, ok := sender.(*JiyingSender); ok {
		replyTo = js.MsgID
	}
	t, err := EnqueueTask(Task{
		Feature: name, Args: args, ConversationID: chatID, UserID: userID, ReplyTo: replyTo, Confirm: f.Confirm,
	})
	if err != nil {
		sender.Reply(err.Error())
		return true, true
	}
	if f.Confirm {
		var client *JiyingClient
		if js, ok := sender.(*JiyingSender); ok {
			client = js.client
		}
		if client == nil {
			client = clientForConv(chatID)
		}
		if client == nil {
			sender.Reply("需要确认，但即应未连接")
			return true, true
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		choiceID, err := client.SendChoice(ctx, chatID, "确认执行 "+name+" ?", []map[string]string{
			{"id": "yes", "label": "确认"},
			{"id": "no", "label": "取消"},
		})
		if err != nil {
			sender.Reply("确认卡片发送失败")
			return true, true
		}
		bindChoice(choiceID, t)
		sender.Reply("已挂起，等待确认")
		return true, true
	}
	sender.Reply("已入队 " + t.ID)
	return true, true
}

func resumeChoice(ctx context.Context, choiceMessageID, userID, conversationID string, optionIDs []string) error {
	if resumeAgentChoice(choiceMessageID, conversationID, userID, optionIDs) {
		return nil
	}
	_ = ctx
	yes := false
	for _, id := range optionIDs {
		if id == "yes" {
			yes = true
		}
	}
	pruneChoiceWaits()
	v, ok := choiceWaits.Load(choiceMessageID)
	if !ok {
		return nil
	}
	w := v.(choiceWait)
	if w.UserID != "" && userID != "" && w.UserID != userID {
		return fmt.Errorf("只有发起人才能确认该操作")
	}
	if w.ConversationID != "" && conversationID != "" && w.ConversationID != conversationID {
		return fmt.Errorf("确认来源与会话不匹配")
	}
	choiceWaits.Delete(choiceMessageID)
	return ConfirmTask(w.TaskID, yes)
}
