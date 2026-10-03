package core

import (
	"strconv"
	"testing"
	"time"
)

func setupLimitBucket(t *testing.T, name string) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("JIYING_DATA", dir)
	b, err := MakeBucket(name, 6)
	if err != nil {
		t.Fatal(err)
	}
	BucketApp = b
}

func TestGlobalQueueCapRejects(t *testing.T) {
	setupLimitBucket(t, "limit-global")
	active := ActiveTaskCount()
	t.Setenv("JIYING_MAX_QUEUE", strconv.Itoa(active+2))
	t.Setenv("JIYING_USER_RATE_PER_MIN", "100")
	user := "ulim-" + t.Name()
	for i := 0; i < 2; i++ {
		if _, err := EnqueueTask(Task{Feature: "帮助", UserID: user, Confirm: true}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := EnqueueTask(Task{Feature: "帮助", UserID: user, Confirm: true}); err == nil {
		t.Fatal("超过全局队列上限应被拒绝")
	}
}

func TestPerUserQueueCapRejects(t *testing.T) {
	setupLimitBucket(t, "limit-user")
	active := ActiveTaskCount()
	t.Setenv("JIYING_MAX_QUEUE", strconv.Itoa(active+10))
	t.Setenv("JIYING_MAX_USER_QUEUE", "2")
	t.Setenv("JIYING_USER_RATE_PER_MIN", "100")
	user := "uqueue-" + t.Name()
	for i := 0; i < 2; i++ {
		if _, err := EnqueueTask(Task{Feature: "帮助", UserID: user, Confirm: true}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := EnqueueTask(Task{Feature: "帮助", UserID: user, Confirm: true}); err == nil {
		t.Fatal("超过用户并发排队上限应被拒绝")
	}
}

func TestUserRateLimitRejects(t *testing.T) {
	setupLimitBucket(t, "limit-rate")
	active := ActiveTaskCount()
	t.Setenv("JIYING_MAX_QUEUE", strconv.Itoa(active+10))
	t.Setenv("JIYING_MAX_USER_QUEUE", "100")
	t.Setenv("JIYING_USER_RATE_PER_MIN", "1")
	user := "urate-" + t.Name()
	ResetUserRateLimit(user)
	if _, err := EnqueueTask(Task{Feature: "帮助", UserID: user, Confirm: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := EnqueueTask(Task{Feature: "帮助", UserID: user, Confirm: true}); err == nil {
		t.Fatal("超过用户速率上限应被拒绝")
	}
}

// 大量不同用户访问后，空闲令牌桶应被回收，避免 map 无界增长。
func TestUserBucketReclaim(t *testing.T) {
	userBuckets.mu.Lock()
	userBuckets.m = map[string]*tokenBucket{}
	userBuckets.lastSweep = time.Time{}
	userBuckets.mu.Unlock()
	t.Cleanup(func() {
		userBuckets.mu.Lock()
		userBuckets.m = map[string]*tokenBucket{}
		userBuckets.lastSweep = time.Time{}
		userBuckets.mu.Unlock()
	})

	total := userBucketSweepThreshold*3 + 10
	for i := 0; i < total; i++ {
		allowUser("ureclaim-" + strconv.Itoa(i))
	}

	// 把所有桶标记为空闲，并把上次清理时间往前拨，模拟下一轮清理周期，再触发一次访问。
	userBuckets.mu.Lock()
	for _, b := range userBuckets.m {
		b.last = time.Now().Add(-2 * userBucketTTL)
	}
	userBuckets.lastSweep = time.Now().Add(-2 * userBucketTTL)
	userBuckets.mu.Unlock()
	allowUser("ureclaim-trigger")

	userBuckets.mu.Lock()
	n := len(userBuckets.m)
	userBuckets.mu.Unlock()
	if n > userBucketSweepThreshold {
		t.Fatalf("空闲令牌桶未被回收，仍有 %d 个（阈值 %d）", n, userBucketSweepThreshold)
	}
}
