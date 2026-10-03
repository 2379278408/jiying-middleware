package core

import "testing"

func TestLoginGuardLockout(t *testing.T) {
	key := "203.0.113.9"
	ResetLoginFailures(key)
	if LoginBlocked(key) {
		t.Fatal("初始状态不应锁定")
	}
	for i := 0; i < loginMaxFailures-1; i++ {
		NoteLoginFailure(key)
	}
	if LoginBlocked(key) {
		t.Fatal("未达阈值不应锁定")
	}
	NoteLoginFailure(key)
	if !LoginBlocked(key) {
		t.Fatal("达到阈值应锁定")
	}
	ResetLoginFailures(key)
	if LoginBlocked(key) {
		t.Fatal("重置后应解除锁定")
	}
}

func TestLoginGuardEmptyKey(t *testing.T) {
	NoteLoginFailure("")
	if LoginBlocked("") {
		t.Fatal("空来源不应被锁定")
	}
}
