package core

import (
	"testing"
	"time"
)

func TestMessageBroadcast(t *testing.T) {
	Senders = make(chan Sender, 4)
	resetImChannels()
	go handleMsgLoop()

	ch := RegisterImChannel("test-im")
	msg := &Faker{Message: "hello-im", Type: 2}

	Senders <- msg

	select {
	case got := <-ch:
		if got.GetMsg() != "hello-im" {
			t.Fatalf("消息内容不符: %q", got.GetMsg())
		}
	case <-time.After(time.Second):
		t.Fatal("广播超时：im 通道未收到消息")
	}
}

func TestRegisterImChannelDedup(t *testing.T) {
	a := RegisterImChannel("dup")
	b := RegisterImChannel("dup")
	if a != b {
		t.Fatal("重复注册应返回同一通道")
	}
}

var _ Sender = (*Faker)(nil)

func TestConvertApplyInLoop(t *testing.T) {
	Senders = make(chan Sender, 4)
	resetImChannels()
	ch := RegisterImChannel("test-conv")
	go handleMsgLoop()

	cv := &Convert{ID: 1, On: true, ConvertFrom: "(旧口令)", ConvertTo: "新口令"}
	cvtC.RLock()
	backup := cvtC.Pts
	cvtC.Pts = append([]*Convert{}, backup...)
	cvtC.Pts = append(cvtC.Pts, cv)
	cvtC.RUnlock()
	defer func() {
		cvtC.Lock()
		cvtC.Pts = backup
		cvtC.Unlock()
	}()

	msg := &Faker{Message: "请使用旧口令登录", Type: 2}
	Senders <- msg

	select {
	case got := <-ch:
		if got.GetMsg() != "请使用新口令登录" {
			t.Fatalf("转换未生效: %q", got.GetMsg())
		}
	case <-time.After(time.Second):
		t.Fatal("广播超时：转换消息未达 im 通道")
	}
}

func TestConvertMultiRuleOrder(t *testing.T) {
	cvtC.RLock()
	backup := cvtC.Pts
	cvtC.Pts = []*Convert{
		{ID: 1, On: true, ConvertFrom: "ab", ConvertTo: "AB"},
		{ID: 2, On: true, ConvertFrom: "a(.)b", ConvertTo: "一二"},
	}
	cvtC.RUnlock()
	defer func() {
		cvtC.Lock()
		cvtC.Pts = backup
		cvtC.Unlock()
	}()

	if got := commandConvert("xabx"); got != "xABx" {
		t.Fatalf("首条精确规则应命中: %q", got)
	}
	if got := commandConvert("xaxbx"); got != "x一二x" {
		t.Fatalf("第二条正则规则应命中: %q", got)
	}
}

func TestConvertDisabledRule(t *testing.T) {
	cvtC.RLock()
	backup := cvtC.Pts
	cvtC.Pts = []*Convert{
		{ID: 3, On: false, ConvertFrom: "停用的", ConvertTo: "不应命中"},
		{ID: 4, On: true, ConvertFrom: "(启用的)", ConvertTo: "命中"},
	}
	cvtC.RUnlock()
	defer func() {
		cvtC.Lock()
		cvtC.Pts = backup
		cvtC.Unlock()
	}()

	if got := commandConvert("启用的规则"); got != "命中规则" {
		t.Fatalf("应命中启用规则: %q", got)
	}
	if got := commandConvert("停用的规则"); got != "停用的规则" {
		t.Fatalf("停用规则不应生效: %q", got)
	}
}
