package core

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"
)

// DingTalkConfig 是钉钉连接器配置；认证信息由本机 dws 持有，面板不保存任何钉钉密钥。
type DingTalkConfig struct {
	Enabled bool   `json:"enabled"`
	Bin     string `json:"bin"`
	// AutoPush 控制是否把命中的钉钉事件推送到即应。
	AutoPush bool `json:"auto_push"`
	// TopicMode 为 true 时按钉钉会话复用即应话题，避免所有通知挤在应用单聊。
	TopicMode          bool   `json:"topic_mode"`
	PushAppID          string `json:"push_app_id"`
	PushConversationID string `json:"push_conversation_id"`
}

const (
	dingTalkConfigKey = "dingtalk_config"
	dingTalkEventKeep = 200
	// dingTalkToolPrefix 是暴露给模型的钉钉工具名前缀，执行时按此路由到连接器。
	dingTalkToolPrefix = "dingtalk_"
)

// DingTalkEvent 是钉钉事件订阅的可展示摘要。
type DingTalkEvent struct {
	ID               string `json:"id"`
	Type             string `json:"type"`
	Reason           string `json:"reason"`
	ConversationID   string `json:"conversation_id,omitempty"`
	ConversationName string `json:"conversation_name,omitempty"`
	Sender           string `json:"sender,omitempty"`
	SenderOpenID     string `json:"sender_open_dingtalk_id,omitempty"`
	Content          string `json:"content,omitempty"`
	Time             string `json:"time,omitempty"`
	ReceivedAt       string `json:"received_at"`
	AtAll            bool   `json:"at_all"`
	Raw              string `json:"raw,omitempty"`
}

type dingTalkConnector struct {
	mu          sync.Mutex
	cfg         DingTalkConfig
	ctx         context.Context
	cancel      context.CancelFunc
	done        chan struct{}
	running     bool
	supervising bool
	pid         int
	startedAt   time.Time
	lastError   string
	lastExit    string
	retryAt     time.Time
	events      []DingTalkEvent
	seen        map[string]time.Time
	stderrLine  string
}

var dingTalk = &dingTalkConnector{seen: map[string]time.Time{}}

func loadDingTalkConfig() DingTalkConfig {
	cfg := DingTalkConfig{Bin: "dws"}
	raw := ""
	if BucketApp != nil {
		raw = BucketApp.GetString(dingTalkConfigKey)
	}
	if raw != "" {
		var stored DingTalkConfig
		if json.Unmarshal([]byte(raw), &stored) == nil {
			cfg = stored
		}
	} else {
		// 新部署默认按会话分话题，避免所有通知挤在应用单聊；用户可显式关闭。
		cfg.TopicMode = true
	}
	cfg.Bin = strings.TrimSpace(cfg.Bin)
	if cfg.Bin == "" {
		cfg.Bin = "dws"
	}
	cfg.PushAppID = strings.TrimSpace(cfg.PushAppID)
	cfg.PushConversationID = strings.TrimSpace(cfg.PushConversationID)
	return cfg
}

func saveDingTalkConfig(cfg DingTalkConfig) error {
	cfg.Bin = strings.TrimSpace(cfg.Bin)
	if cfg.Bin == "" {
		cfg.Bin = "dws"
	}
	if strings.ContainsAny(cfg.Bin, "\x00\n\r") {
		return fmt.Errorf("dws 路径不合法")
	}
	cfg.PushAppID = strings.TrimSpace(cfg.PushAppID)
	cfg.PushConversationID = strings.TrimSpace(cfg.PushConversationID)
	if cfg.AutoPush && (cfg.PushAppID == "" || cfg.PushConversationID == "") {
		return fmt.Errorf("开启自动推送需要填写推送应用与会话")
	}
	if BucketApp == nil {
		return fmt.Errorf("存储未就绪")
	}
	raw, _ := json.Marshal(cfg)
	BucketApp.Set(dingTalkConfigKey, string(raw))
	dingTalk.mu.Lock()
	dingTalk.cfg = cfg
	dingTalk.mu.Unlock()
	return nil
}

func initDingTalkConnector() {
	cfg := loadDingTalkConfig()
	dingTalk.mu.Lock()
	dingTalk.cfg = cfg
	dingTalk.mu.Unlock()
	if cfg.Enabled {
		startDingTalkConnector()
	}
}

// dingTalkActive 报告连接器是否已启用并持有运行上下文，用于决定是否暴露工具。
func dingTalkActive() bool {
	dingTalk.mu.Lock()
	defer dingTalk.mu.Unlock()
	return dingTalk.cfg.Enabled
}

func startDingTalkConnector() error {
	deadline := time.Now().Add(10 * time.Second)
	for {
		dingTalk.mu.Lock()
		if !dingTalk.supervising {
			break
		}
		if dingTalk.cancel != nil {
			// 已有一个运行中的监督协程，重复启动保持幂等。
			dingTalk.mu.Unlock()
			return nil
		}
		done := dingTalk.done
		dingTalk.mu.Unlock()
		if done == nil {
			return fmt.Errorf("连接器正在停止，请稍后重试")
		}
		wait := time.Until(deadline)
		if wait <= 0 {
			return fmt.Errorf("等待连接器停止超时")
		}
		select {
		case <-done:
		case <-time.After(wait):
			return fmt.Errorf("等待连接器停止超时")
		}
	}
	if dingTalk.cfg.PushAppID != "" && !appConnExists(dingTalk.cfg.PushAppID) {
		dingTalk.mu.Unlock()
		return fmt.Errorf("推送应用不在连接池中")
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	dingTalk.ctx = ctx
	dingTalk.cancel = cancel
	dingTalk.done = done
	dingTalk.running = true
	dingTalk.supervising = true
	dingTalk.startedAt = time.Now()
	dingTalk.lastError = ""
	dingTalk.lastExit = ""
	dingTalk.retryAt = time.Time{}
	dingTalk.mu.Unlock()
	go func() {
		defer close(done)
		runDingTalkSupervisor(ctx)
	}()
	return nil
}

func stopDingTalkConnector() {
	dingTalk.mu.Lock()
	cancel := dingTalk.cancel
	dingTalk.cancel = nil
	dingTalk.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func runDingTalkSupervisor(ctx context.Context) {
	backoff := 2 * time.Second
	for {
		if ctx.Err() != nil {
			dingTalk.markStopped("", false)
			return
		}
		err := consumeDingTalkEvents(ctx)
		if ctx.Err() != nil {
			dingTalk.markStopped("", false)
			return
		}
		msg := "事件订阅已退出"
		if err != nil {
			msg = err.Error()
		}
		dingTalk.mu.Lock()
		dingTalk.lastExit = truncateText(msg, 300)
		dingTalk.lastError = truncateText(msg, 300)
		dingTalk.retryAt = time.Now().Add(backoff)
		dingTalk.running = false
		dingTalk.pid = 0
		dingTalk.mu.Unlock()
		if !sleepCtx(ctx, backoff) {
			dingTalk.markStopped(msg, false)
			return
		}
		if backoff < time.Minute {
			backoff *= 2
		}
		dingTalk.mu.Lock()
		dingTalk.running = true
		dingTalk.retryAt = time.Time{}
		dingTalk.mu.Unlock()
	}
}

func (d *dingTalkConnector) markStopped(msg string, errorState bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.running = false
	d.supervising = false
	d.pid = 0
	d.retryAt = time.Time{}
	if msg != "" {
		if errorState {
			d.lastError = truncateText(msg, 300)
		}
		d.lastExit = truncateText(msg, 300)
	}
}

func consumeDingTalkEvents(ctx context.Context) error {
	dingTalk.mu.Lock()
	bin := dingTalk.cfg.Bin
	dingTalk.mu.Unlock()
	if bin == "" {
		bin = "dws"
	}
	cmd := exec.Command(bin, "event", "consume",
		"user_im_message_receive_o2o_all",
		"user_im_message_receive_at",
		"user_im_message_receive_group_all",
		"--flatten", "-f", "ndjson", "-y")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动 dws 失败: %w", err)
	}
	dingTalk.mu.Lock()
	dingTalk.pid = cmd.Process.Pid
	dingTalk.mu.Unlock()

	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			terminateProcessGroup(cmd, done)
		case <-done:
		}
	}()

	stderrDone := make(chan struct{})
	go func() {
		defer close(stderrDone)
		sc := bufio.NewScanner(stderr)
		sc.Buffer(make([]byte, 64*1024), 256*1024)
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if line == "" {
				continue
			}
			dingTalk.mu.Lock()
			dingTalk.stderrLine = truncateText(line, 300)
			dingTalk.mu.Unlock()
			if strings.Contains(line, "[event] ready") {
				dingTalk.mu.Lock()
				dingTalk.lastError = ""
				dingTalk.retryAt = time.Time{}
				dingTalk.mu.Unlock()
			}
		}
	}()

	sc := bufio.NewScanner(stdout)
	sc.Buffer(make([]byte, 256*1024), 4*1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		if ev, ok := parseDingTalkEvent([]byte(line)); ok {
			dingTalk.ingestEvent(ev)
		}
	}
	scanErr := sc.Err()
	<-stderrDone
	waitErr := cmd.Wait()
	close(done)
	dingTalk.mu.Lock()
	dingTalk.pid = 0
	dingTalk.mu.Unlock()
	if scanErr != nil {
		return scanErr
	}
	return waitErr
}

func terminateProcessGroup(cmd *exec.Cmd, done <-chan struct{}) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	pgid, err := syscall.Getpgid(cmd.Process.Pid)
	if err == nil {
		_ = syscall.Kill(-pgid, syscall.SIGTERM)
	} else {
		_ = cmd.Process.Signal(syscall.SIGTERM)
	}
	time.AfterFunc(5*time.Second, func() {
		select {
		case <-done:
			return
		default:
		}
		if pgid, err := syscall.Getpgid(cmd.Process.Pid); err == nil {
			_ = syscall.Kill(-pgid, syscall.SIGKILL)
		} else {
			_ = cmd.Process.Kill()
		}
	})
}

func parseDingTalkEvent(raw []byte) (DingTalkEvent, bool) {
	var obj map[string]any
	if json.Unmarshal(raw, &obj) != nil {
		return DingTalkEvent{}, false
	}
	ev := DingTalkEvent{
		Type:             strAny(obj, "type"),
		ConversationID:   firstAny(obj, "conversation_id", "conversationId", "open_conversation_id", "openConversationId"),
		ConversationName: firstAny(obj, "conversation_name", "conversationName", "chat_name", "chatName"),
		Sender:           firstAny(obj, "sender", "sender_name", "senderName", "from"),
		SenderOpenID:     firstAny(obj, "sender_open_dingtalk_id", "senderOpenDingTalkId", "open_dingtalk_id", "openDingTalkId"),
		Content:          firstAny(obj, "content", "text", "message"),
		ID:               firstAny(obj, "message_id", "msg_id", "id"),
		Time:             firstAny(obj, "create_time", "event_time", "timestamp"),
		AtAll:            isDingTalkAtAll(obj),
		ReceivedAt:       time.Now().Format(time.RFC3339),
		Raw:              truncateText(string(raw), 4000),
	}
	ev.Reason = classifyDingTalkEvent(ev.Type, ev.AtAll)
	if ev.Reason == "" {
		return DingTalkEvent{}, false
	}
	if ev.ConversationID == "" {
		ev.ConversationID = firstAny(obj, "chat_id", "chatId")
	}
	if ev.Content == "" {
		ev.Content = "（没有可显示的正文，可能是图片、文件或卡片）"
	}
	return ev, true
}

func classifyDingTalkEvent(typ string, atAll bool) string {
	switch typ {
	case "user_im_message_receive_o2o_all":
		return "dm"
	case "user_im_message_receive_at":
		if atAll {
			return "at-all"
		}
		return "mention"
	case "user_im_message_receive_group_all":
		if atAll {
			return "at-all"
		}
	}
	return ""
}

func isDingTalkAtAll(obj map[string]any) bool {
	text := firstAny(obj, "content", "text")
	if strings.Contains(text, "@所有人") || strings.Contains(text, "@全体") {
		return true
	}
	for _, key := range []string{"atAll", "at_all", "isAtAll", "is_at_all"} {
		if v, ok := obj[key].(bool); ok && v {
			return true
		}
	}
	raw, _ := json.Marshal(obj)
	return strings.Contains(string(raw), `"atAll":true`) || strings.Contains(string(raw), `"at_all":true`)
}

func firstAny(obj map[string]any, keys ...string) string {
	for _, key := range keys {
		if v, ok := obj[key]; ok {
			switch val := v.(type) {
			case string:
				if strings.TrimSpace(val) != "" {
					return strings.TrimSpace(val)
				}
			case float64:
				return fmt.Sprintf("%v", val)
			case map[string]any:
				if s := firstAny(val, "name", "nickname", "id", "content"); s != "" {
					return s
				}
			}
		}
	}
	return ""
}

func strAny(obj map[string]any, key string) string {
	if v, ok := obj[key].(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
}

func (d *dingTalkConnector) ingestEvent(ev DingTalkEvent) {
	key := ev.Type + "|" + ev.ID + "|" + ev.ConversationID + "|" + ev.ReceivedAt
	if ev.ID != "" {
		key = ev.Type + "|" + ev.ID + "|" + ev.ConversationID
	}
	d.mu.Lock()
	if ev.ID != "" {
		for k, at := range d.seen {
			if time.Since(at) > 10*time.Minute {
				delete(d.seen, k)
			}
		}
		if _, dup := d.seen[key]; dup {
			d.mu.Unlock()
			return
		}
		d.seen[key] = time.Now()
	}
	d.events = append(d.events, ev)
	if len(d.events) > dingTalkEventKeep {
		d.events = d.events[len(d.events)-dingTalkEventKeep:]
	}
	cfg := d.cfg
	d.mu.Unlock()
	if cfg.AutoPush && cfg.PushAppID != "" && cfg.PushConversationID != "" {
		go d.pushEvent(ev, cfg)
	}
}

// pushEvent 把一条事件推送到即应；话题模式按钉钉会话复用话题，失败时回退到父会话。
func (d *dingTalkConnector) pushEvent(ev DingTalkEvent, cfg DingTalkConfig) {
	ctx, cancel := context.WithTimeout(context.Background(), 75*time.Second)
	defer cancel()
	if ev.Reason == "dm" {
		if dingTalkDirectMuted(ctx, ev.ConversationID) {
			return
		}
	} else if dingTalkMuted(ctx, ev.ConversationID) {
		return
	}
	var err error
	if cfg.TopicMode {
		err = d.pushEventToTopic(ctx, cfg, ev)
	} else {
		err = SendTextToConvVia(ctx, cfg.PushAppID, cfg.PushConversationID, "", formatDingTalkPush(ev))
	}
	if err == nil {
		return
	}
	d.mu.Lock()
	d.lastError = truncateText("自动推送失败: "+err.Error(), 300)
	d.mu.Unlock()
	GetMyLog().Warn("钉钉事件自动推送失败: %v", err)
	if cfg.TopicMode {
		fallback := formatDingTalkPush(ev)
		if sendErr := SendTextToConvVia(ctx, cfg.PushAppID, cfg.PushConversationID, "", fallback); sendErr == nil {
			d.mu.Lock()
			d.lastError = truncateText("话题推送失败，已回退父会话: "+err.Error(), 300)
			d.mu.Unlock()
		}
	}
}

func formatDingTalkPush(ev DingTalkEvent) string {
	reason := map[string]string{"dm": "私聊", "mention": "@我", "at-all": "@所有人"}[ev.Reason]
	if reason == "" {
		reason = "相关消息"
	}
	sender := ev.Sender
	if sender == "" {
		sender = "未知发送人"
	}
	return fmt.Sprintf("**钉钉通知 · %s**\n**%s**\n%s\n\n%s", reason, sender, ev.Time, ev.Content)
}

// DingTalkStatus 是连接器对外状态快照。
type DingTalkStatus struct {
	Config     DingTalkConfig  `json:"config"`
	Running    bool            `json:"running"`
	PID        int             `json:"pid,omitempty"`
	StartedAt  string          `json:"started_at,omitempty"`
	LastError  string          `json:"last_error,omitempty"`
	LastExit   string          `json:"last_exit,omitempty"`
	RetryAt    string          `json:"retry_at,omitempty"`
	StderrLine string          `json:"stderr,omitempty"`
	Events     []DingTalkEvent `json:"events"`
	BinFound   bool            `json:"bin_found"`
}

func dingTalkStatusSnapshot(limit int) DingTalkStatus {
	dingTalk.mu.Lock()
	defer dingTalk.mu.Unlock()
	if limit <= 0 || limit > dingTalkEventKeep {
		limit = 50
	}
	start := len(dingTalk.events) - limit
	if start < 0 {
		start = 0
	}
	events := append([]DingTalkEvent(nil), dingTalk.events[start:]...)
	out := DingTalkStatus{
		Config:     dingTalk.cfg,
		Running:    dingTalk.running,
		PID:        dingTalk.pid,
		LastError:  dingTalk.lastError,
		LastExit:   dingTalk.lastExit,
		StderrLine: dingTalk.stderrLine,
		Events:     events,
		BinFound:   dingTalkBinAvailable(dingTalk.cfg.Bin),
	}
	if !dingTalk.startedAt.IsZero() {
		out.StartedAt = dingTalk.startedAt.Format(time.RFC3339)
	}
	if !dingTalk.retryAt.IsZero() {
		out.RetryAt = dingTalk.retryAt.Format(time.RFC3339)
	}
	return out
}

func dingTalkBinAvailable(bin string) bool {
	bin = strings.TrimSpace(bin)
	if bin == "" {
		bin = "dws"
	}
	_, err := exec.LookPath(bin)
	return err == nil
}

func dingTalkEventsSnapshot(limit int) []DingTalkEvent {
	return dingTalkStatusSnapshot(limit).Events
}
