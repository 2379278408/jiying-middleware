package core

import "testing"

func TestMsgExecSemaphoreBound(t *testing.T) {
	// 清空可能残留的令牌，保证测试起点确定。
	for {
		select {
		case <-msgExecSem:
			continue
		default:
		}
		break
	}

	got := 0
	for tryAcquireMsgExec() {
		got++
		if got > cap(msgExecSem)+1 {
			t.Fatalf("信号量超出容量：cap=%d", cap(msgExecSem))
		}
	}
	if got != cap(msgExecSem) {
		t.Fatalf("应恰好取到 %d 个槽位，实际 %d", cap(msgExecSem), got)
	}

	releaseMsgExec()
	if !tryAcquireMsgExec() {
		t.Fatal("释放一个槽位后应能再次获取")
	}
	// 复位，避免影响其他用例。
	for {
		select {
		case <-msgExecSem:
			continue
		default:
		}
		break
	}
}
