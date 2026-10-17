package core

import (
	"testing"
	"time"
)

func setupMsgStatsBucket(t *testing.T, name string) {
	t.Helper()
	t.Setenv("JIYING_DATA", t.TempDir())
	b, err := MakeBucket(name, 6)
	if err != nil {
		t.Fatal(err)
	}
	BucketApp = b
	BucketApp.Delete(jiyingMsgStatsKey)
	resetMsgStatsForTest()
	t.Cleanup(func() {
		BucketApp.Delete(jiyingMsgStatsKey)
		resetMsgStatsForTest()
	})
}

func resetMsgStatsForTest() {
	jiyingMsgStats.Lock()
	jiyingMsgStats.day = ""
	jiyingMsgStats.inbound = 0
	jiyingMsgStats.out = 0
	jiyingMsgStats.Unlock()
}

func TestMsgStatsPersistAcrossReload(t *testing.T) {
	setupMsgStatsBucket(t, "msgstats-persist")

	jiyingCountInbound()
	jiyingCountOutbound()
	jiyingCountOutbound()
	if in, out := jiyingMsgCounts(); in != 1 || out != 2 {
		t.Fatalf("unexpected live counts: in=%d out=%d", in, out)
	}

	// 模拟进程重启：内存清零，再从桶中懒加载。
	resetMsgStatsForTest()
	if in, out := jiyingMsgCounts(); in != 1 || out != 2 {
		t.Fatalf("counts not restored after reload: in=%d out=%d", in, out)
	}
}

func TestMsgStatsResetOnNewDay(t *testing.T) {
	setupMsgStatsBucket(t, "msgstats-day")

	jiyingMsgStats.Lock()
	jiyingMsgStats.day = time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	jiyingMsgStats.inbound = 9
	jiyingMsgStats.out = 7
	saveJiyingMsgStatsLocked()
	jiyingMsgStats.Unlock()
	resetMsgStatsForTest()

	if in, out := jiyingMsgCounts(); in != 0 || out != 0 {
		t.Fatalf("stale day should read as zero: in=%d out=%d", in, out)
	}
	jiyingCountInbound()
	if in, out := jiyingMsgCounts(); in != 1 || out != 0 {
		t.Fatalf("new day should reset counters: in=%d out=%d", in, out)
	}
}
