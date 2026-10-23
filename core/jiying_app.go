package core

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/gorilla/websocket"
)

const jiyingProtoVer = 1

const (
	jiyingReadLimit = 1 << 20
	jiyingPongWait  = 60 * time.Second
	jiyingRetryWait = 500 * time.Millisecond
	statusPingEvery = 3 * time.Second
	// maxCallAttempts 是单次调用（含首次）的最大尝试次数。
	maxCallAttempts = 2
	// maxRateLimitWait 限制为等待服务端限流恢复而在调用内阻塞的上限。
	maxRateLimitWait = 10 * time.Second
	// maxOutboundTextBytes 是单条文本消息的字面上限，避免 Envelope 触顶 1 MiB。
	maxOutboundTextBytes = 256 << 10
	// receiptReaction 是收到待处理消息后立即回发的接收回执表情。
	receiptReaction = "🔥"
)

type JiyingConfig struct {
	WSURL  string
	AppID  string
	Secret string
}

func JiyingConfigFromEnv() JiyingConfig {
	return JiyingConfig{
		WSURL:  strings.TrimSpace(os.Getenv("JIYING_WS_URL")),
		AppID:  strings.TrimSpace(os.Getenv("JIYING_APP_ID")),
		Secret: strings.TrimSpace(os.Getenv("JIYING_APP_SECRET")),
	}
}

func (c JiyingConfig) enabled() bool {
	return c.WSURL != "" && c.AppID != "" && c.Secret != ""
}

type jiyingEnvelope struct {
	V       int             `json:"v"`
	Kind    string          `json:"kind"`
	ID      string          `json:"id,omitempty"`
	ReplyTo string          `json:"reply_to,omitempty"`
	Method  string          `json:"method,omitempty"`
	Event   string          `json:"event,omitempty"`
	Cursor  int64           `json:"cursor,omitempty"`
	OK      bool            `json:"ok,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty"`
	Error   *jiyingError    `json:"error,omitempty"`
}

type jiyingError struct {
	Code              string `json:"code"`
	Message           string `json:"message"`
	RetryAfterSeconds int    `json:"retry_after_seconds,omitempty"`
}

// jiyingDialError 承载应用通道握手失败时的 HTTP 状态码，供日志与就绪面板给出可读原因。
type jiyingDialError struct {
	Status int
}

func (e *jiyingDialError) Error() string {
	switch e.Status {
	case http.StatusBadRequest:
		return "应用 ID 格式错误（HTTP 400）"
	case http.StatusUnauthorized:
		return "应用不存在或连接密钥错误（HTTP 401）"
	case http.StatusForbidden:
		return "应用已停用（HTTP 403）"
	default:
		return fmt.Sprintf("应用握手失败 HTTP %d", e.Status)
	}
}

type jiyingPending struct {
	ch chan jiyingEnvelope
}

type JiyingClient struct {
	cfg     JiyingConfig
	dial    func(context.Context) (jiyingSocket, error)
	mu      sync.Mutex
	writeMu sync.Mutex
	conn    jiyingSocket
	pending map[string]jiyingPending
	seq     int64
	// acks 记录已处理（业务成功）的 cursor，来源包括本地台账与本次运行。
	acks map[int64]bool
	// acked 记录已成功向服务端确认的 cursor。
	acked map[int64]bool
	// ackWatermark 是已确认水位；ACK 某个 cursor 会同时确认不大于它的所有事件。
	ackWatermark int64
	serial       chan struct{}
	// eventSem 限制事件处理 goroutine 的并发排队上限，避免突发流量无限堆积。
	eventSem chan struct{}
	// lastErr 记录最近一次连接失败的原因，供就绪面板与排障读取。
	lastErr string
	// retryAt 是下一次重连的计划时间；连接成功后清空。
	retryAt time.Time
	// seenMsg 记录已投递的消息 ID，避免本地即时唤醒与轮询/补发重复触发。
	seenMsg map[string]time.Time
}

type jiyingSocket interface {
	WriteJSON(v any) error
	ReadJSON(v any) error
	Close() error
}

func NewJiyingClient(cfg JiyingConfig, dial func(context.Context) (jiyingSocket, error)) *JiyingClient {
	if dial == nil {
		dial = func(ctx context.Context) (jiyingSocket, error) {
			h := http.Header{}
			h.Set("X-MagicChat-App-ID", cfg.AppID)
			h.Set("Authorization", "Bearer "+cfg.Secret)
			d := websocket.Dialer{HandshakeTimeout: 10 * time.Second}
			c, resp, err := d.DialContext(ctx, cfg.WSURL, h)
			if err != nil {
				if resp != nil {
					return nil, &jiyingDialError{Status: resp.StatusCode}
				}
				return nil, err
			}
			return c, nil
		}
	}
	return &JiyingClient{
		cfg: cfg, dial: dial,
		pending:  map[string]jiyingPending{},
		acks:     map[int64]bool{},
		acked:    map[int64]bool{},
		serial:   make(chan struct{}, 1),
		eventSem: make(chan struct{}, 64),
		seenMsg:  map[string]time.Time{},
	}
}

func (c *JiyingClient) nextID() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.seq++
	return fmt.Sprintf("req-%d-%d", time.Now().UnixNano(), c.seq)
}

func (c *JiyingClient) Call(ctx context.Context, method string, payload any) (raw json.RawMessage, err error) {
	started := time.Now()
	defer func() { jiyingObserveCall(method, time.Since(started), err) }()
	raw, err = json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	// 同一次业务操作超时或瞬时失败时复用同一请求 ID 重试；
	// Server 按 (id, method, payload) 去重，重复请求返回首次结果。
	id := c.nextID()
	var lastErr error
	delay := time.Duration(0)
	for attempt := 0; attempt < maxCallAttempts; attempt++ {
		if attempt > 0 && !sleepCtx(ctx, delay) {
			return nil, ctx.Err()
		}
		resp, err := c.callOnce(ctx, id, method, raw)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		if ctx.Err() != nil {
			break
		}
		wait, retry := callRetryDelay(err)
		if !retry {
			logCallFailure(method, err)
			break
		}
		delay = wait
		GetMyLog().Warn("即应调用 %s 失败，复用请求 ID 重试：%v", method, err)
	}
	return nil, lastErr
}

// callOnce 使用固定请求 ID 发起一次调用。
func (c *JiyingClient) callOnce(ctx context.Context, id, method string, raw json.RawMessage) (json.RawMessage, error) {
	ch := make(chan jiyingEnvelope, 1)
	c.mu.Lock()
	if c.conn == nil {
		c.mu.Unlock()
		return nil, errJiyingNotConnected
	}
	c.pending[id] = jiyingPending{ch: ch}
	conn := c.conn
	c.mu.Unlock()
	env := jiyingEnvelope{V: jiyingProtoVer, Kind: "request", ID: id, Method: method, Payload: raw}
	c.writeMu.Lock()
	err := conn.WriteJSON(env)
	c.writeMu.Unlock()
	if err != nil {
		c.dropPending(id)
		return nil, err
	}
	select {
	case <-ctx.Done():
		c.dropPending(id)
		return nil, ctx.Err()
	case resp := <-ch:
		if !resp.OK {
			if resp.Error != nil {
				return nil, &jiyingCallError{Code: resp.Error.Code, Message: resp.Error.Message, RetryAfterSeconds: resp.Error.RetryAfterSeconds}
			}
			return nil, fmt.Errorf("即应调用失败")
		}
		return resp.Payload, nil
	}
}

type jiyingCallError struct {
	Code              string
	Message           string
	RetryAfterSeconds int
}

func (e *jiyingCallError) Error() string { return e.Code + ": " + e.Message }

var errJiyingNotConnected = errors.New("即应未连接")

// rateLimitWait 把服务端给出的 retry_after_seconds 折算为阻塞时长，并收敛到上限。
func rateLimitWait(seconds int) time.Duration {
	if seconds <= 0 {
		return jiyingRetryWait
	}
	d := time.Duration(seconds) * time.Second
	if d > maxRateLimitWait {
		return maxRateLimitWait
	}
	return d
}

// callRetryDelay 判断错误是否适合复用原请求 ID 重试，并给出建议等待时长。
// 传输层失败与 Server internal_error 可重试；rate_limited 按 retry_after_seconds 退避；
// 其余确定性失败（版本、请求 ID 冲突、体积/目标/会话不合法等）不重试。
func callRetryDelay(err error) (time.Duration, bool) {
	if err == nil {
		return 0, false
	}
	var ce *jiyingCallError
	if errors.As(err, &ce) {
		switch ce.Code {
		case "internal_error":
			return jiyingRetryWait, true
		case "rate_limited":
			jiyingCountRateLimited()
			return rateLimitWait(ce.RetryAfterSeconds), true
		default:
			return 0, false
		}
	}
	if errors.Is(err, errJiyingNotConnected) {
		return 0, false
	}
	return jiyingRetryWait, true
}

// jiyingFailureHints 为确定性失败给出排查方向；命中后不重试，只记录一次告警。
var jiyingFailureHints = map[string]string{
	"unsupported_version":  "协议版本不受服务端支持，需升级中间件",
	"request_id_conflict":  "请求 ID 与既有请求冲突，检查是否复用了固定 ID",
	"response_too_large":   "响应体超过 1 MiB，改用分页、缩短查询范围或减少返回字段",
	"request_too_large":    "请求体或媒体超过服务端限额，压缩或拆分后重发",
	"invalid_conversation": "会话不存在或类型不支持，仅可向应用参与的一对一会话发送",
	"not_found":            "目标资源不存在，确认会话、消息或文件 ID 仍然有效",
	"invalid_request":      "请求参数不符合协议要求，检查字段名、类型与长度限制",
	"invalid_target":       "消息目标不合法，确认 target.type 与 conversation_id",
	"permission_denied":    "应用无权执行该操作，检查应用授权与可见范围",
}

// logCallFailure 对需要人工关注的确定性失败打一条告警，便于从运行日志定位。
func logCallFailure(method string, err error) {
	var ce *jiyingCallError
	if !errors.As(err, &ce) {
		return
	}
	if hint, ok := jiyingFailureHints[ce.Code]; ok {
		GetMyLog().Warn("即应调用 %s 失败 code=%s: %s（%s）", method, ce.Code, ce.Message, hint)
	}
}

func (c *JiyingClient) dropPending(id string) {
	c.mu.Lock()
	delete(c.pending, id)
	c.mu.Unlock()
}

// failPending 在连接断开时唤醒所有等待响应的调用方，避免它们一直等到超时。
func (c *JiyingClient) failPending() {
	c.mu.Lock()
	pending := c.pending
	c.pending = map[string]jiyingPending{}
	c.mu.Unlock()
	for id, p := range pending {
		select {
		case p.ch <- jiyingEnvelope{
			Kind: "response", ID: id, OK: false,
			Error: &jiyingError{Code: "disconnected", Message: "即应连接已断开"},
		}:
		default:
		}
	}
}

// configureJiyingSocket 为客户端的读限制、心跳续期与 Pong 响应做统一设置。
func configureJiyingSocket(sock jiyingSocket) {
	ws, ok := sock.(*websocket.Conn)
	if !ok {
		return
	}
	ws.SetReadLimit(jiyingReadLimit)
	_ = ws.SetReadDeadline(time.Now().Add(jiyingPongWait))
	ws.SetPingHandler(func(msg string) error {
		_ = ws.SetReadDeadline(time.Now().Add(jiyingPongWait))
		return ws.WriteControl(websocket.PongMessage, []byte(msg), time.Now().Add(10*time.Second))
	})
	ws.SetPongHandler(func(string) error {
		return ws.SetReadDeadline(time.Now().Add(jiyingPongWait))
	})
}

// clampOutboundText 限制单条文本消息体积，避免入站/出站 Envelope 触顶 1 MiB。
// 超限时按字节边界安全截断并追加提示，保证不会切断 UTF-8 字符。
func clampOutboundText(text string) string {
	if len(text) <= maxOutboundTextBytes {
		return text
	}
	const note = "\n\n（内容过长已截断）"
	limit := maxOutboundTextBytes - len(note)
	if limit < 0 {
		limit = 0
	}
	for limit > 0 && !utf8.RuneStart(text[limit]) {
		limit--
	}
	return text[:limit] + note
}

func (c *JiyingClient) SendText(ctx context.Context, conversationID, replyTo, text string) error {
	text = clampOutboundText(text)
	body := map[string]any{
		"target":  map[string]any{"type": "conversation", "conversation_id": conversationID},
		"message": map[string]any{"type": "text", "content": text},
	}
	if replyTo != "" {
		body["reply_to_message_id"] = replyTo
	}
	raw, err := c.Call(ctx, "message.send", body)
	if err != nil {
		return err
	}
	jiyingCountOutbound()
	var resp struct {
		Message struct {
			ID string `json:"id"`
		} `json:"message"`
	}
	_ = json.Unmarshal(raw, &resp)
	// 即应不会把应用消息推给其他应用；发送成功后按 mention 在本地即时唤醒目标，
	// 避免只能依赖轮询导致最长 12 秒延迟。失败不影响发送结果，轮询仍会兜底。
	if resp.Message.ID != "" {
		c.notifyAppMentions(conversationID, resp.Message.ID, text)
	}
	return nil
}

// mentionedAppIDs 解析文本中的 {(@app/<id>)} 提法，按出现顺序去重。
func mentionedAppIDs(text string) []string {
	const prefix = "{(@app/"
	var out []string
	seen := map[string]bool{}
	for {
		idx := strings.Index(text, prefix)
		if idx < 0 {
			break
		}
		rest := text[idx+len(prefix):]
		end := strings.Index(rest, ")}")
		if end < 0 {
			break
		}
		id := strings.TrimSpace(rest[:end])
		if id != "" && !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
		text = rest[end+2:]
	}
	return out
}

// notifyAppMentions 把应用发送的 @ 消息即时投递给在线目标应用。
// 目标按 message.created 走同一套处理链路，因此权限、预算与回执行为保持一致。
func (c *JiyingClient) notifyAppMentions(conversationID, messageID, text string) {
	ids := mentionedAppIDs(text)
	if len(ids) == 0 {
		return
	}
	senderName := c.cfg.AppID
	appConnPool.mu.Lock()
	if rt := appConnPool.items[c.cfg.AppID]; rt != nil && rt.profile.Name != "" {
		senderName = rt.profile.Name
	}
	appConnPool.mu.Unlock()
	for _, id := range ids {
		if id == c.cfg.AppID {
			continue
		}
		target := clientForApp(id)
		if target == nil {
			continue
		}
		payload, err := json.Marshal(map[string]any{
			"conversation": map[string]any{"id": conversationID, "type": "group"},
			"sender":       map[string]any{"id": c.cfg.AppID, "name": senderName, "type": "app"},
			"message": map[string]any{
				"id": messageID, "body": map[string]any{"type": "text", "content": text},
			},
		})
		if err != nil {
			continue
		}
		go target.injectEvent(context.Background(), jiyingEnvelope{
			Kind: "event", Event: "message.created", Payload: payload,
		})
	}
}

// injectEvent 把本地生成的事件送入目标应用的串行处理链路。
func (c *JiyingClient) injectEvent(ctx context.Context, env jiyingEnvelope) {
	select {
	case <-ctx.Done():
		return
	case c.eventSem <- struct{}{}:
	}
	c.processEvent(ctx, env)
}

// duplicateMessage 标记消息 ID 是否已投递；空 ID 不参与去重。
func (c *JiyingClient) duplicateMessage(id string) bool {
	id = strings.TrimSpace(id)
	if id == "" {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.seenMsg == nil {
		c.seenMsg = map[string]time.Time{}
	}
	now := time.Now()
	if len(c.seenMsg) > 1024 {
		for k, at := range c.seenMsg {
			if now.Sub(at) > 30*time.Minute {
				delete(c.seenMsg, k)
			}
		}
	}
	if _, ok := c.seenMsg[id]; ok {
		return true
	}
	c.seenMsg[id] = now
	return false
}

// SendStatus 向一对一会话中的在线用户同步临时状态；尽力投递，失败不影响业务。
func (c *JiyingClient) SendStatus(ctx context.Context, conversationID, status string) error {
	if conversationID == "" || strings.TrimSpace(status) == "" {
		return nil
	}
	_, err := c.Call(ctx, "conversation.status", map[string]any{
		"conversation_id": conversationID,
		"status":          status,
	})
	return err
}

// SendReaction 以应用身份给指定消息添加或移除文本表情（message.reaction.set）。
// 文本去首尾空白后不得为空；这里用于接收回执，失败只记日志、不影响业务。
func (c *JiyingClient) SendReaction(ctx context.Context, conversationID, messageID, text string, reacted bool) error {
	if conversationID == "" || messageID == "" || strings.TrimSpace(text) == "" {
		return fmt.Errorf("缺少会话 / 消息 / 表情内容")
	}
	_, err := c.Call(ctx, "message.reaction.set", map[string]any{
		"conversation_id": conversationID,
		"message_id":      messageID,
		"text":            text,
		"reacted":         reacted,
	})
	return err
}

// startStatusPing 在处理业务期间每 3 秒续发一次临时状态，返回停止函数。
func startStatusPing(convType, convID, status string) func() {
	client := clientForConv(convID)
	if convType != "app" || convID == "" || client == nil {
		return func() {}
	}
	stopped := make(chan struct{})
	go func() {
		send := func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			_ = client.SendStatus(ctx, convID, status)
			cancel()
		}
		send()
		t := time.NewTicker(statusPingEvery)
		defer t.Stop()
		for {
			select {
			case <-stopped:
				return
			case <-t.C:
				send()
			}
		}
	}()
	var once sync.Once
	return func() { once.Do(func() { close(stopped) }) }
}

// startSenderStatus 为即应应用会话的发送方启动处理状态提示。
func startSenderStatus(s Sender) func() {
	js, ok := s.(*JiyingSender)
	if !ok || js == nil {
		return func() {}
	}
	return startStatusPing(js.ConvType, js.ChatID, "正在处理消息")
}

// choiceOptionIDRe 匹配选择项 ID 允许的字符集。
var choiceOptionIDRe = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// validateChoice 校验选择消息参数，避免把必然触发 invalid_request 的内容发出去。
// 规则来自应用开发指南：正文去空白后 1~5000 字符；选项 2~20 个，ID 1~64 字符且限
// 字母/数字/连字符/下划线，展示文本 1~200 字符且不含换行或控制字符。
func validateChoice(content string, options []map[string]string) error {
	content = strings.TrimSpace(content)
	if content == "" {
		return fmt.Errorf("选择消息正文不能为空")
	}
	if n := len([]rune(content)); n > 5000 {
		return fmt.Errorf("选择消息正文过长（%d > 5000 字符）", n)
	}
	if len(options) < 2 || len(options) > 20 {
		return fmt.Errorf("选择项数量必须是 2~20 个，当前 %d 个", len(options))
	}
	seen := map[string]bool{}
	for _, opt := range options {
		id := strings.TrimSpace(opt["id"])
		if id == "" || len([]rune(id)) > 64 || !choiceOptionIDRe.MatchString(id) {
			return fmt.Errorf("选择项 ID 非法：%q（1~64 位字母/数字/-/_）", id)
		}
		if seen[id] {
			return fmt.Errorf("选择项 ID 重复：%q", id)
		}
		seen[id] = true
		label := opt["label"]
		if strings.TrimSpace(label) == "" {
			return fmt.Errorf("选择项 %q 缺少展示文本", id)
		}
		if len([]rune(label)) > 200 {
			return fmt.Errorf("选择项 %q 展示文本过长（最多 200 字符）", id)
		}
		if strings.ContainsAny(label, "\r\n") {
			return fmt.Errorf("选择项 %q 展示文本不能包含换行", id)
		}
	}
	return nil
}

func (c *JiyingClient) SendChoice(ctx context.Context, conversationID, content string, options []map[string]string) (string, error) {
	if err := validateChoice(content, options); err != nil {
		return "", err
	}
	raw, err := c.Call(ctx, "message.send", map[string]any{
		"target": map[string]any{"type": "conversation", "conversation_id": conversationID},
		"message": map[string]any{
			"type": "choice", "content_type": "text", "content": content,
			"selection": "single", "options": options,
		},
	})
	if err != nil {
		return "", err
	}
	var resp struct {
		Message struct {
			ID string `json:"id"`
		} `json:"message"`
	}
	_ = json.Unmarshal(raw, &resp)
	jiyingCountOutbound()
	return resp.Message.ID, nil
}

func (c *JiyingClient) Ack(ctx context.Context, cursor int64) error {
	c.mu.Lock()
	if cursor <= c.ackWatermark || c.acked[cursor] {
		c.mu.Unlock()
		return nil
	}
	c.mu.Unlock()
	if _, err := c.Call(ctx, "events.ack", map[string]any{"cursor": cursor}); err != nil {
		// 业务已经成功，只是确认失败：先落台账，重放时不再重复执行业务。
		c.markProcessed(cursor)
		return err
	}
	c.mu.Lock()
	c.acked[cursor] = true
	c.acks[cursor] = true
	if cursor > c.ackWatermark {
		c.ackWatermark = cursor
	}
	// 水位线以下的 cursor 已由 ackWatermark 覆盖，及时清理避免长连接内存只增不减。
	for v := range c.acks {
		if v <= c.ackWatermark {
			delete(c.acks, v)
		}
	}
	for v := range c.acked {
		if v <= c.ackWatermark {
			delete(c.acked, v)
		}
	}
	c.persistAck(cursor)
	c.markProcessedLocked(cursor)
	c.mu.Unlock()
	return nil
}

func (c *JiyingClient) persistAck(cursor int64) {
	if BucketApp == nil || c.cfg.AppID == "" {
		return
	}
	key := "jiying_ack_" + c.cfg.AppID
	prev := BucketApp.GetString(key)
	var n int64
	fmt.Sscan(prev, &n)
	if cursor > n {
		BucketApp.Set(key, fmt.Sprintf("%d", cursor))
	}
}

func (c *JiyingClient) loadAcks() {
	var n int64
	var ledger []int64
	if BucketApp != nil && c.cfg.AppID != "" {
		fmt.Sscan(BucketApp.GetString("jiying_ack_"+c.cfg.AppID), &n)
		ledger = parseCursors(BucketApp.GetString("jiying_evt_" + c.cfg.AppID))
	}
	c.mu.Lock()
	if n > 0 {
		c.ackWatermark = n
		c.acks[n] = true
		c.acked[n] = true
	}
	for _, cur := range ledger {
		c.acks[cur] = true
	}
	c.mu.Unlock()
}

// markProcessed 把已成功处理的 cursor 写入本地台账，重启后据此跳过重复执行业务。
func (c *JiyingClient) markProcessed(cursor int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.markProcessedLocked(cursor)
}

// markProcessedLocked 与 markProcessed 相同，但调用方必须持有 c.mu。
func (c *JiyingClient) markProcessedLocked(cursor int64) {
	if cursor <= 0 {
		return
	}
	if c.acks[cursor] {
		return
	}
	c.acks[cursor] = true
	if BucketApp == nil || c.cfg.AppID == "" {
		return
	}
	key := "jiying_evt_" + c.cfg.AppID
	list := parseCursors(BucketApp.GetString(key))
	for _, v := range list {
		if v == cursor {
			return
		}
	}
	list = append(list, cursor)
	if len(list) > eventLedgerMax {
		list = list[len(list)-eventLedgerMax:]
	}
	BucketApp.Set(key, formatCursors(list))
}

// eventLedgerMax 是本地事件台账保留的最近 cursor 数量，用于限制存储增长。
const eventLedgerMax = 512

func parseCursors(raw string) []int64 {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]int64, 0, len(parts))
	for _, p := range parts {
		var v int64
		if _, err := fmt.Sscan(strings.TrimSpace(p), &v); err == nil && v > 0 {
			out = append(out, v)
		}
	}
	return out
}

func formatCursors(list []int64) string {
	parts := make([]string, len(list))
	for i, v := range list {
		parts[i] = fmt.Sprintf("%d", v)
	}
	return strings.Join(parts, ",")
}

func (c *JiyingClient) seenCursor(cursor int64) bool {
	if cursor <= 0 {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.acks[cursor] {
		return true
	}
	return cursor <= c.ackWatermark
}

func (c *JiyingClient) HandleInbound(ctx context.Context, env jiyingEnvelope) error {
	switch env.Kind {
	case "response":
		c.mu.Lock()
		p, ok := c.pending[env.ReplyTo]
		if ok {
			delete(c.pending, env.ReplyTo)
		}
		c.mu.Unlock()
		if ok {
			p.ch <- env
		}
		return nil
	case "event":
		if env.Event == "conversation.status" {
			return nil
		}
		if env.Cursor > 0 && c.seenCursor(env.Cursor) {
			// 已处理过的事件补一次确认：业务成功但确认失败时，避免服务端无限重放。
			return c.Ack(ctx, env.Cursor)
		}
		deferred, err := c.dispatchEvent(ctx, env)
		if err != nil {
			return err
		}
		// 消息类事件在业务处理结束后由 sender 触发确认；其余事件处理完即确认。
		if !deferred && env.Cursor > 0 {
			return c.Ack(ctx, env.Cursor)
		}
	}
	return nil
}

// jiyingMessageBody 对应即应消息体的联合结构，覆盖 text/markdown/link/card/
// entity_card/file/image/video/voice/choice 等类型；未出现的字段保持零值。
type jiyingMessageBody struct {
	Type        string `json:"type"`
	Content     string `json:"content"`
	Caption     string `json:"caption"`
	URL         string `json:"url"`
	Title       string `json:"title"`
	Description string `json:"description"`
	EntityID    string `json:"entity_id"`
	EntityType  string `json:"entity_type"`
	FileID      string `json:"file_id"`
	Name        string `json:"name"`
	SizeBytes   int64  `json:"size_bytes"`
	ContentType string `json:"content_type"`
	Transcript  string `json:"transcript"`
	DurationMS  int    `json:"duration_ms"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
}

// plainText 返回可作为命令文本的纯文本：优先正文，其次摘要性字段，最后退化为类型占位。
func (b jiyingMessageBody) plainText() string {
	switch b.Type {
	case "", "text", "markdown", "choice":
		return b.Content
	case "image":
		if b.Caption != "" {
			return b.Caption
		}
		return "[图片]"
	case "video":
		if b.Caption != "" {
			return b.Caption
		}
		return "[视频]"
	case "voice":
		if b.Transcript != "" {
			return b.Transcript
		}
		return "[语音]"
	case "file":
		if b.Name != "" {
			return b.Name
		}
		return "[文件]"
	case "link":
		if b.Title != "" {
			return b.Title
		}
		return b.URL
	case "card":
		if b.Title != "" {
			return b.Title
		}
		return b.Description
	case "entity_card":
		return "[名片]"
	case "forward_bundle":
		return "[合并转发]"
	}
	return b.Content
}

// isMedia 报告消息体是否携带可读取的临时文件（图片/文件/视频/语音）。
func (b jiyingMessageBody) isMedia() bool {
	switch b.Type {
	case "image", "file", "video", "voice":
		return b.FileID != ""
	}
	return false
}

// dispatchEvent 处理事件并返回是否将确认延后到业务完成之后。
func (c *JiyingClient) dispatchEvent(ctx context.Context, env jiyingEnvelope) (bool, error) {
	switch env.Event {
	case "message.created":
		var p struct {
			Conversation struct {
				ID             string `json:"id"`
				Type           string `json:"type"`
				Name           string `json:"name"`
				CreatedByAppID string `json:"created_by_app_id"`
				Parent         struct {
					ID   string `json:"id"`
					Type string `json:"type"`
					Name string `json:"name"`
				} `json:"parent"`
				SourceMessage struct {
					ID  string `json:"id"`
					Seq int64  `json:"seq"`
				} `json:"source_message"`
			} `json:"conversation"`
			Sender struct {
				ID       string `json:"id"`
				Name     string `json:"name"`
				Nickname string `json:"nickname"`
				Email    string `json:"email"`
				Type     string `json:"type"`
			} `json:"sender"`
			Message struct {
				ID               string            `json:"id"`
				Seq              int64             `json:"seq"`
				Summary          string            `json:"summary"`
				ReplyToMessageID string            `json:"reply_to_message_id"`
				Body             jiyingMessageBody `json:"body"`
			} `json:"message"`
		}
		if err := json.Unmarshal(env.Payload, &p); err != nil {
			return false, err
		}
		if c.duplicateMessage(p.Message.ID) {
			return false, nil
		}
		text := p.Message.Body.plainText()
		if text == "" {
			text = p.Message.Summary
		}
		rememberConvApp(p.Conversation.ID, c.cfg.AppID)
		if p.Conversation.Parent.ID != "" {
			rememberConvApp(p.Conversation.Parent.ID, c.cfg.AppID)
		}
		if p.Message.Body.isMedia() && p.Message.Body.FileID != "" {
			recordMediaCtx(p.Conversation.ID, p.Message.ID, []string{p.Message.Body.FileID})
		}
		s := &JiyingSender{
			client: c, UserID: p.Sender.ID, ChatID: p.Conversation.ID,
			MsgID: p.Message.ID, Seq: p.Message.Seq, ConvType: p.Conversation.Type, Text: text,
			SenderName: p.Sender.Name, SenderNickname: p.Sender.Nickname,
			SenderEmail: p.Sender.Email, SenderType: p.Sender.Type,
			ConvName: p.Conversation.Name, ReplyToMsgID: p.Message.ReplyToMessageID,
			Body:            p.Message.Body,
			ParentConvID:    p.Conversation.Parent.ID,
			ParentConvType:  p.Conversation.Parent.Type,
			ParentConvName:  p.Conversation.Parent.Name,
			SourceMessageID: p.Conversation.SourceMessage.ID,
			CreatedByAppID:  p.Conversation.CreatedByAppID,
			ackCursor:       env.Cursor,
		}
		if err := enqueueSender(ctx, s); err != nil {
			return false, err
		}
		return true, nil
	case "choice.response_created":
		var p struct {
			Conversation struct {
				ID string `json:"id"`
			} `json:"conversation"`
			Sender struct {
				ID string `json:"id"`
			} `json:"sender"`
			ChoiceMessage struct {
				ID string `json:"id"`
			} `json:"choice_message"`
			Response struct {
				OptionIDs []string `json:"option_ids"`
			} `json:"response"`
		}
		if err := json.Unmarshal(env.Payload, &p); err != nil {
			return false, err
		}
		rememberConvApp(p.Conversation.ID, c.cfg.AppID)
		return false, resumeChoice(ctx, p.ChoiceMessage.ID, p.Sender.ID, p.Conversation.ID, p.Response.OptionIDs)
	case "topic.closed":
		var p struct {
			ConversationID string `json:"conversation_id"`
		}
		_ = json.Unmarshal(env.Payload, &p)
		rememberConvApp(p.ConversationID, c.cfg.AppID)
		markTopicClosed(p.ConversationID)
		failTasksForConversation(p.ConversationID)
		return false, nil
	default:
		return false, nil
	}
}

func enqueueSender(ctx context.Context, s Sender) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case Senders <- s:
		return nil
	}
}

func (c *JiyingClient) processEvent(ctx context.Context, env jiyingEnvelope) {
	defer func() { <-c.eventSem }()
	c.serial <- struct{}{}
	defer func() { <-c.serial }()
	if err := c.HandleInbound(ctx, env); err != nil {
		GetMyLog().Warn("即应事件处理失败 cursor=%d", env.Cursor)
	}
}

// pollInbound 轮询群内其他应用发来的 @ 消息。
// 即应只把人类消息推给应用连接，应用间消息不会触发 WS 事件，
// 因此用轻量轮询补齐这条链路，让应用之间的 @ 也能被唤醒与回执。
// 轮询以应用自身可见的会话列表为准，不再依赖单值归属映射，
// 因此同一个群里的多个应用都能各自看到发给自己的 @。
func (c *JiyingClient) pollInbound(ctx context.Context) {
	const interval = 12 * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	states := map[string]*pollConvState{}
	since := time.Now().Add(-interval)
	lastFull := time.Now()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		force := time.Since(lastFull) >= time.Minute
		if force {
			lastFull = time.Now()
		}
		c.pollInboundOnce(ctx, states, since, force)
	}
}

type pollConvState struct {
	lastActive string
	lastSeq    int64
	// followup 在检测到活跃变化后多读一轮，兜住同一时间粒度内的后续消息。
	followup bool
}

// pollInboundOnce 刷新会话列表，优先读取活跃时间变化的会话；
// force 为 true 时补扫全部会话，兜住未更新 last_active_at 的应用间消息。
func (c *JiyingClient) pollInboundOnce(ctx context.Context, states map[string]*pollConvState, since time.Time, force bool) {
	if !c.Connected() {
		return
	}
	listCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	convs, err := c.ListConversations(listCtx, "", 100)
	cancel()
	if err != nil {
		GetMyLog().Warn("轮询会话列表失败: %v", err)
		return
	}
	type pollJob struct {
		conv  JiyingConversation
		state *pollConvState
	}
	var jobs []pollJob
	for _, conv := range convs {
		convID := strings.TrimSpace(conv.ID)
		if convID == "" {
			continue
		}
		conv.ID = convID
		st := states[convID]
		if st == nil {
			st = &pollConvState{lastActive: conv.LastActiveAt}
			states[convID] = st
			if !force {
				// 首次只记会话活跃水位，不回放历史消息。
				continue
			}
		}
		if !force && !st.followup && st.lastActive != "" && st.lastActive == conv.LastActiveAt {
			continue
		}
		st.followup = true
		jobs = append(jobs, pollJob{conv: conv, state: st})
	}
	if len(jobs) == 0 {
		return
	}
	// 多个活跃会话并行读取，避免一个慢会话拖住整轮轮询。
	sem := make(chan struct{}, 4)
	var wg sync.WaitGroup
	for _, job := range jobs {
		wg.Add(1)
		go func(job pollJob) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			c.pollConversation(ctx, job.conv, job.state, since)
		}(job)
	}
	wg.Wait()
}

// pollConversation 读取单个会话的最近消息，只入队本应用被显式 @ 的新消息。
func (c *JiyingClient) pollConversation(ctx context.Context, conv JiyingConversation, st *pollConvState, since time.Time) {
	callCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	msgs, err := c.RecentMessages(callCtx, conv.ID, 50)
	if err != nil {
		GetMyLog().Warn("轮询会话失败 conv=%s: %v", conv.ID, err)
		return
	}
	sort.Slice(msgs, func(i, j int) bool { return msgs[i].Seq < msgs[j].Seq })
	mention := "{(@app/" + c.cfg.AppID + ")}"
	for _, m := range msgs {
		if m.Seq <= st.lastSeq {
			continue
		}
		st.lastSeq = m.Seq
		if m.Sender.Type != "app" || m.Sender.ID == c.cfg.AppID {
			continue
		}
		text := m.plainText()
		if !strings.Contains(text, mention) {
			continue
		}
		if created, err := time.Parse(time.RFC3339, m.CreatedAt); err == nil && created.Before(since) {
			continue
		}
		if c.duplicateMessage(m.ID) {
			continue
		}
		convType := strings.TrimSpace(conv.Type)
		if convType == "" {
			convType = "group"
		}
		s := &JiyingSender{
			client: c, UserID: m.Sender.ID, ChatID: conv.ID,
			MsgID: m.ID, Seq: m.Seq, ConvType: convType, Text: text,
			SenderType: m.Sender.Type, ConvName: conv.Name,
		}
		if err := enqueueSender(ctx, s); err != nil {
			GetMyLog().Warn("轮询入队失败 conv=%s: %v", conv.ID, err)
		}
	}
	st.lastActive = conv.LastActiveAt
	st.followup = false
}

func (c *JiyingClient) Run(ctx context.Context) error {
	c.loadAcks()
	go c.pollInbound(ctx)
	backoff := time.Second
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		sock, err := c.dial(ctx)
		if err != nil {
			c.setLastError(err.Error())
			GetMyLog().Warn("即应连接失败: %v", err)
			wait := reconnectBackoff(backoff)
			c.setRetryAt(time.Now().Add(wait))
			if !sleepCtx(ctx, wait) {
				return ctx.Err()
			}
			if backoff < 30*time.Second {
				backoff *= 2
			}
			continue
		}
		backoff = time.Second
		c.setLastError("")
		c.setRetryAt(time.Time{})
		c.mu.Lock()
		c.conn = sock
		c.mu.Unlock()
		configureJiyingSocket(sock)
		GetMyLog().Info("即应应用通道已连接")
		err = c.readLoop(ctx, sock)
		c.mu.Lock()
		c.conn = nil
		c.mu.Unlock()
		c.failPending()
		_ = sock.Close()
		if ctx.Err() != nil {
			return ctx.Err()
		}
		GetMyLog().Warn("即应连接断开: %v", err)
		wait := reconnectBackoff(backoff)
		c.setRetryAt(time.Now().Add(wait))
		if !sleepCtx(ctx, wait) {
			return ctx.Err()
		}
		if backoff < 30*time.Second {
			backoff *= 2
		}
	}
}

func (c *JiyingClient) readLoop(ctx context.Context, sock jiyingSocket) error {
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		var env jiyingEnvelope
		if err := sock.ReadJSON(&env); err != nil {
			return err
		}
		if env.Kind == "response" {
			_ = c.HandleInbound(ctx, env)
			continue
		}
		select {
		case c.eventSem <- struct{}{}:
			go c.processEvent(ctx, env)
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func sleepCtx(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}

// reconnectBackoff 在指数退避基础上加入 50%~100% 抖动，避免多实例同时重连形成尖峰。
func reconnectBackoff(base time.Duration) time.Duration {
	if base <= 0 {
		return 0
	}
	half := base / 2
	if half <= 0 {
		return base
	}
	return half + time.Duration(rand.Int64N(int64(half)+1))
}

type JiyingSender struct {
	client         *JiyingClient
	UserID         string
	ChatID         string
	MsgID          string
	Seq            int64
	ConvType       string
	Text           string
	SenderName     string
	SenderNickname string
	SenderEmail    string
	SenderType     string
	ConvName       string
	ReplyToMsgID   string
	Body           jiyingMessageBody
	// 话题上下文：消息来自话题时记录父会话与来源消息，便于功能与 Agent 感知层级。
	ParentConvID    string
	ParentConvType  string
	ParentConvName  string
	SourceMessageID string
	CreatedByAppID  string
	ackCursor       int64
	ackOnce         sync.Once
	BaseSender
}

func (s *JiyingSender) GetMsg() string       { return s.Text }
func (s *JiyingSender) GetType() interface{} { return "jiying" }
func (s *JiyingSender) GetParentID() any     { return s.ChatID }
func (s *JiyingSender) SetMsg(msg string)    { s.Text = msg }
func (s *JiyingSender) SetParentID(id any)   { s.ChatID = fmt.Sprint(id) }

// GetImtype 供统计与下游按渠道归类消息。
func (s *JiyingSender) GetImtype() string { return "jiying" }

// DisplayName 返回发送者的展示名，优先昵称，其次用户名，最后回落到用户 ID。
func (s *JiyingSender) DisplayName() string {
	if s.SenderNickname != "" {
		return s.SenderNickname
	}
	if s.SenderName != "" {
		return s.SenderName
	}
	return s.UserID
}

// IsGroup 报告消息是否来自群聊会话。
func (s *JiyingSender) IsGroup() bool { return s.ConvType == "group" }

// IsMedia 报告本条消息是否携带可读取的临时文件。
func (s *JiyingSender) IsMedia() bool { return s.Body.isMedia() }

// IsReply 报告用户是否在回复某条更早的消息。
func (s *JiyingSender) IsReply() bool { return s.ReplyToMsgID != "" }

// IsTopic 报告消息是否来自话题会话。
func (s *JiyingSender) IsTopic() bool { return s.ConvType == "topic" }

// Receipt 在确认需要处理该消息后立即异步回接收回执；尽力投递，失败只记日志。
// 不阻塞消息处理，也不影响 events.ack。
func (s *JiyingSender) Receipt() {
	if s == nil || s.client == nil || s.ChatID == "" || s.MsgID == "" {
		return
	}
	client, convID, msgID := s.client, s.ChatID, s.MsgID
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		if err := client.SendReaction(ctx, convID, msgID, receiptReaction, true); err != nil {
			GetMyLog().Debug("接收回执发送失败 conv=%s msg=%s: %v", convID, msgID, err)
		}
	}()
}

// sendReceipt 触发 sender 自带的接收回执（非即应消息为 no-op）。
func sendReceipt(sender Sender) {
	if r, ok := sender.(interface{ Receipt() }); ok {
		r.Receipt()
	}
}

func (s *JiyingSender) Send(msg interface{}) bool {
	return s.send(fmt.Sprint(msg), "")
}

func (s *JiyingSender) Reply(msg interface{}) bool {
	return s.send(fmt.Sprint(msg), s.MsgID)
}

// ReplyMarkdown 以 markdown 富文本回复，客户端会渲染标题、列表与表格。
// 发送失败返回 false，调用方据此回退纯文本。
func (s *JiyingSender) ReplyMarkdown(msg interface{}) bool {
	text := clampOutboundText(strings.TrimSpace(fmt.Sprint(msg)))
	if s.client == nil || text == "" {
		return false
	}
	if topicClosed(s.ChatID) {
		GetMyLog().Warn("话题已关闭，拒绝发送")
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	// 直接复用当前发送者的客户端，避免多应用场景下按会话猜测到其他应用通道。
	if _, err := s.client.SendMessage(ctx, s.ChatID, s.MsgID, MarkdownMessage(text)); err != nil {
		GetMyLog().Warn("即应发送失败：%v", err)
		return false
	}
	return true
}

func (s *JiyingSender) send(text, replyTo string) bool {
	if s.client == nil || text == "" {
		return false
	}
	if topicClosed(s.ChatID) {
		GetMyLog().Warn("话题已关闭，拒绝发送")
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := s.client.SendText(ctx, s.ChatID, replyTo, text); err != nil {
		GetMyLog().Warn("即应发送失败：%v", err)
		return false
	}
	return true
}

// FinishAck 在该条可靠事件处理完成后确认，重复调用只生效一次。
func (s *JiyingSender) FinishAck() {
	if s == nil || s.ackCursor <= 0 || s.client == nil {
		return
	}
	s.ackOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := s.client.Ack(ctx, s.ackCursor); err != nil {
			GetMyLog().Warn("即应事件确认失败 cursor=%d: %v", s.ackCursor, err)
		}
	})
}

// finishSender 触发 sender 自带的确认回调（非即应消息为 no-op）。
func finishSender(s Sender) {
	if f, ok := s.(interface{ FinishAck() }); ok {
		f.FinishAck()
	}
}

const closedTopicTTL = 24 * time.Hour

var (
	closedTopicsMu sync.Mutex
	closedTopics   = map[string]time.Time{}
)

func markTopicClosed(id string) {
	if id == "" {
		return
	}
	now := time.Now()
	closedTopicsMu.Lock()
	defer closedTopicsMu.Unlock()
	closedTopics[id] = now
	if len(closedTopics) > 1024 {
		for k, t := range closedTopics {
			if now.Sub(t) > closedTopicTTL {
				delete(closedTopics, k)
			}
		}
	}
}

func topicClosed(id string) bool {
	if id == "" {
		return false
	}
	closedTopicsMu.Lock()
	defer closedTopicsMu.Unlock()
	t, ok := closedTopics[id]
	if !ok {
		return false
	}
	if time.Since(t) > closedTopicTTL {
		delete(closedTopics, id)
		return false
	}
	return true
}

var activeJiying *JiyingClient

func initJiying() {
	initTaskQueue()
	initFeatures()
	registerJiyingCommands()
	initAppChannelFromConfig()
	initDingTalkConnector()
}

func PostJSON(ctx context.Context, url string, body any) ([]byte, error) {
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := safeOutboundHTTPClient(60 * time.Second).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	out, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return out, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return out, nil
}

func jiyingState() string {
	c := currentJiying()
	if c == nil {
		return "即应未连接"
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil {
		return "即应未连接"
	}
	return "即应已连接"
}

func JiyingState() string         { return jiyingState() }
func JiyingAccountStatus() string { return UserSessionStatus() }

// AppChannelConnected 报告应用通道当前是否持有活动连接。
func AppChannelConnected() bool {
	c := currentJiying()
	if c == nil {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn != nil
}

// setLastError 记录最近一次连接失败原因，连接成功时清空。
func (c *JiyingClient) setLastError(msg string) {
	c.mu.Lock()
	c.lastErr = msg
	c.mu.Unlock()
}

// setRetryAt 记录下一次重连时间。
func (c *JiyingClient) setRetryAt(t time.Time) {
	c.mu.Lock()
	c.retryAt = t
	c.mu.Unlock()
}

// RetryAt 返回下一次重连时间；无计划时返回零值。
func (c *JiyingClient) RetryAt() time.Time {
	if c == nil {
		return time.Time{}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.retryAt
}

// LastError 返回最近一次连接失败原因；正常连接时为空串。
func (c *JiyingClient) LastError() string {
	if c == nil {
		return ""
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.lastErr
}

// AppChannelLastError 返回默认应用通道最近一次连接失败原因，供就绪面板与排障展示。
func AppChannelLastError() string {
	return currentJiying().LastError()
}

var jiyingStartedAt = time.Now()

func JiyingUptime() string {
	d := time.Since(jiyingStartedAt).Round(time.Second)
	return d.String()
}

// JiyingUptimeSeconds 返回进程已运行的整秒数，供前端本地递增展示。
func JiyingUptimeSeconds() int64 {
	return int64(time.Since(jiyingStartedAt).Seconds())
}
