package core

import "sync"

var senderStack = struct {
	sync.Mutex
	s []Sender
}{}

func pushSender(s Sender) {
	senderStack.Lock()
	senderStack.s = append(senderStack.s, s)
	senderStack.Unlock()
}

func popSender() {
	senderStack.Lock()
	if n := len(senderStack.s); n > 0 {
		senderStack.s = senderStack.s[:n-1]
	}
	senderStack.Unlock()
}

func CurrentSender() Sender {
	senderStack.Lock()
	defer senderStack.Unlock()
	if n := len(senderStack.s); n > 0 {
		return senderStack.s[n-1]
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
