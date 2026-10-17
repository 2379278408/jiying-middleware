package core

import "testing"

func TestReserveLoginAttemptBlocksAfterThreshold(t *testing.T) {
	key := "test-reserve|user"
	for i := 0; i < loginMaxFailures; i++ {
		if !ReserveLoginAttempt(key) {
			t.Fatalf("第 %d 次尝试应放行", i+1)
		}
	}
	if ReserveLoginAttempt(key) {
		t.Fatal("超过阈值后应拒绝")
	}
	ResetLoginFailures(key)
	if !ReserveLoginAttempt(key) {
		t.Fatal("重置后应重新放行")
	}
	ResetLoginFailures(key)
}
