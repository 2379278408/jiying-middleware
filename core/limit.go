package core

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// 限流与配额保护队列不被单个用户、单个功能或脚本打满。
// 阈值可用环境变量覆盖，便于按部署规模调整。
func maxQueueDepth() int        { return envInt("JIYING_MAX_QUEUE", 200) }
func maxUserQueueDepth() int    { return envInt("JIYING_MAX_USER_QUEUE", 20) }
func maxFeatureQueueDepth() int { return envInt("JIYING_MAX_FEATURE_QUEUE", 30) }
func userRatePerMinute() int    { return envInt("JIYING_USER_RATE_PER_MIN", 30) }

func envInt(name string, def int) int {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return def
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v <= 0 {
		return def
	}
	return v
}

var rateLimitRejected atomic.Int64

// RateLimitRejected 返回被限流拒绝的任务总数，供指标输出。
func RateLimitRejected() int64 { return rateLimitRejected.Load() }

func checkQuota(t Task) error {
	if depth := ActiveTaskCount(); depth >= maxQueueDepth() {
		return fmt.Errorf("队列繁忙（%d/%d），请稍后再试", depth, maxQueueDepth())
	}
	if t.Feature != "" {
		if n := ActiveCountByFeature(t.Feature); n >= maxFeatureQueueDepth() {
			return fmt.Errorf("功能 %s 排队过多，请稍后再试", t.Feature)
		}
	}
	// system 代表定时与内部任务，跳过按用户限流，避免定时任务被个人配额挤掉。
	if t.UserID == "" || t.UserID == "system" {
		return nil
	}
	if n := ActiveCountByUser(t.UserID); n >= maxUserQueueDepth() {
		return fmt.Errorf("你排队中的任务过多（%d/%d），请稍后再试", n, maxUserQueueDepth())
	}
	if !allowUser(t.UserID) {
		return fmt.Errorf("操作过于频繁，请稍后再试")
	}
	return nil
}

type tokenBucket struct {
	mu     sync.Mutex
	tokens float64
	last   time.Time
}

func (b *tokenBucket) allow(ratePerMin, burst float64) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	now := time.Now()
	if b.last.IsZero() {
		b.tokens = burst
	} else {
		b.tokens += now.Sub(b.last).Minutes() * ratePerMin
	}
	if b.tokens > burst {
		b.tokens = burst
	}
	b.last = now
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

var userBuckets = struct {
	mu        sync.Mutex
	m         map[string]*tokenBucket
	lastSweep time.Time
}{m: map[string]*tokenBucket{}}

// 令牌桶满载周期：桶容量等于每分钟补充的令牌数，空闲一分钟后与新建桶等价。
const userBucketTTL = time.Minute

// 超过该数量后回收空闲桶，避免用户数增长导致 map 无界膨胀。
const userBucketSweepThreshold = 1024

// sweepIdleUserBucketsLocked 需在持有 userBuckets.mu 时调用。
func sweepIdleUserBucketsLocked(now time.Time) {
	for id, b := range userBuckets.m {
		b.mu.Lock()
		// 从未使用过的桶（last 为零值）保留，避免误清掉刚创建、即将用于限流的桶。
		idle := !b.last.IsZero() && now.Sub(b.last) >= userBucketTTL
		b.mu.Unlock()
		if idle {
			delete(userBuckets.m, id)
		}
	}
}

func allowUser(userID string) bool {
	rate := float64(userRatePerMinute())
	now := time.Now()
	userBuckets.mu.Lock()
	b, ok := userBuckets.m[userID]
	if !ok {
		b = &tokenBucket{}
		userBuckets.m[userID] = b
	}
	// 最多每分钟清理一次，避免桶数量增长后每次调用都遍历整张表。
	if len(userBuckets.m) > userBucketSweepThreshold && now.Sub(userBuckets.lastSweep) >= userBucketTTL {
		sweepIdleUserBucketsLocked(now)
		userBuckets.lastSweep = now
	}
	userBuckets.mu.Unlock()
	return b.allow(rate, rate)
}

// ResetUserRateLimit 清除指定用户的令牌桶，供测试与解封使用。
func ResetUserRateLimit(userID string) {
	userBuckets.mu.Lock()
	delete(userBuckets.m, userID)
	userBuckets.mu.Unlock()
}
