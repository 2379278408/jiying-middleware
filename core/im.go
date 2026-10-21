package core

import "sync"

// Channels 保存各 IM 类型的入站通道。消息循环会并发遍历广播，
// 而 RegisterImChannel 是可被运行期调用的公共入口，因此写操作必须串行化，
// 否则 map 并发读写会触发 data race。
var (
	channelsMu sync.RWMutex
	Channels   = map[string]chan Sender{}
)

func RegisterImChannel(name string) chan Sender {
	channelsMu.Lock()
	defer channelsMu.Unlock()
	if ch, ok := Channels[name]; ok {
		return ch
	}
	ch := make(chan Sender, 100)
	Channels[name] = ch
	return ch
}

// channelsSnapshot 复制当前通道列表，供广播时无锁遍历，避免长时间持锁。
func channelsSnapshot() []chan Sender {
	channelsMu.RLock()
	defer channelsMu.RUnlock()
	out := make([]chan Sender, 0, len(Channels))
	for _, ch := range Channels {
		out = append(out, ch)
	}
	return out
}

// resetImChannels 清空通道表，供测试隔离使用。
func resetImChannels() {
	channelsMu.Lock()
	Channels = map[string]chan Sender{}
	channelsMu.Unlock()
}

func initToHandleMessage() {
	Senders = make(chan Sender, 100)
	go handleMsgLoop()
}

func handleMsgLoop() {
	defer func() {
		if r := recover(); r != nil {
			GetMyLog().Errorf("消息循环崩溃: %v", r)
			go handleMsgLoop()
		}
	}()
	for s := range Senders {
		if s == nil {
			continue
		}
		handleSender(s)
	}
}

// handleSender 串行处理一条内部消息，并在业务完成后触发可靠事件的确认。
func handleSender(s Sender) {
	if handled, ok := dispatchMiddleware(s, s.GetMsg()); handled {
		if !ok {
			GetMyLog().Warn("中间件处理失败")
		}
		finishSender(s)
		return
	}
	if relayDingTalkTopicReply(s) {
		finishSender(s)
		return
	}
	if dispatchMessage(s) {
		// 命令在独立 goroutine 中执行，完成后由 HandleMessage 触发确认。
		return
	}
	// 先完成消息转换，再广播，避免下游通道读到未转换的文本。
	if msg := commandConvert(s.GetMsg()); msg != s.GetMsg() {
		s.SetMsg(msg)
		GetMyLog().Debug("消息转换回写: %q", msg)
	}
	for _, ch := range channelsSnapshot() {
		select {
		case ch <- s:
		default:
		}
	}
	finishSender(s)
}
