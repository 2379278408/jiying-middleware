package core

import (
	"runtime"
	"strconv"
	"strings"
	"sync"
)

// senderStack 按 goroutine 隔离发送者。旧实现使用全局 LIFO 栈，
// 并发消息会在不同用户之间串栈，导致回复和权限判定落到别人身上。
var senderStack = struct {
	sync.Mutex
	m map[uint64][]Sender
}{m: map[uint64][]Sender{}}

func currentGoroutineID() uint64 {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	line := strings.TrimPrefix(string(buf[:n]), "goroutine ")
	end := strings.IndexByte(line, ' ')
	if end <= 0 {
		return 0
	}
	id, err := strconv.ParseUint(line[:end], 10, 64)
	if err != nil {
		return 0
	}
	return id
}

func pushSender(s Sender) {
	gid := currentGoroutineID()
	senderStack.Lock()
	senderStack.m[gid] = append(senderStack.m[gid], s)
	senderStack.Unlock()
}

func popSender() {
	gid := currentGoroutineID()
	senderStack.Lock()
	stack := senderStack.m[gid]
	if len(stack) <= 1 {
		delete(senderStack.m, gid)
	} else {
		senderStack.m[gid] = stack[:len(stack)-1]
	}
	senderStack.Unlock()
}

func CurrentSender() Sender {
	gid := currentGoroutineID()
	senderStack.Lock()
	defer senderStack.Unlock()
	if stack := senderStack.m[gid]; len(stack) > 0 {
		return stack[len(stack)-1]
	}
	return nil
}

func ReplyCurrent(msg string) {
	s := CurrentSender()
	if s == nil {
		GetMyLog().Info("%s", msg)
		return
	}
	if !s.Reply(msg) {
		s.Send(msg)
	}
}

func withSender(s Sender, fn func()) {
	pushSender(s)
	defer popSender()
	fn()
}
