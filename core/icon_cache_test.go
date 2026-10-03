package core

import (
	"strconv"
	"testing"
)

func resetIconCache() {
	iconCache.Lock()
	iconCache.m = map[string][]byte{}
	iconCache.order = nil
	iconCache.bytes = 0
	iconCache.Unlock()
}

func TestIconCacheEvictsOldest(t *testing.T) {
	resetIconCache()
	t.Cleanup(resetIconCache)

	// 每条 64 KiB，总预算 8 MiB 约等于 128 条；写入 200 条后应只保留尾部。
	payload := make([]byte, 64<<10)
	for i := 0; i < 200; i++ {
		cacheIcon("kw-"+strconv.Itoa(i), payload)
	}

	iconCache.Lock()
	entries := len(iconCache.m)
	order := len(iconCache.order)
	total := iconCache.bytes
	_, firstKept := iconCache.m["kw-0"]
	lastKept := iconCache.m["kw-199"]
	iconCache.Unlock()

	if entries != order {
		t.Fatalf("条目与顺序表不一致：entries=%d order=%d", entries, order)
	}
	if total > iconCacheMaxBytes || total != entries*len(payload) {
		t.Fatalf("总字节预算未生效：bytes=%d 期望<=%d", total, iconCacheMaxBytes)
	}
	if firstKept {
		t.Fatal("最旧的缓存项应被淘汰")
	}
	if lastKept == nil {
		t.Fatal("最新的缓存项应被保留")
	}
}

func TestIconCacheReplacesSameKey(t *testing.T) {
	resetIconCache()
	t.Cleanup(resetIconCache)

	cacheIcon("same", make([]byte, 1024))
	cacheIcon("same", make([]byte, 2048))

	iconCache.Lock()
	entries := len(iconCache.m)
	order := len(iconCache.order)
	total := iconCache.bytes
	iconCache.Unlock()
	if entries != 1 || order != 1 || total != 2048 {
		t.Fatalf("同一键应替换而非追加：entries=%d order=%d bytes=%d", entries, order, total)
	}
}

func TestIconCacheSkipsOversized(t *testing.T) {
	resetIconCache()
	t.Cleanup(resetIconCache)

	cacheIcon("big", make([]byte, iconCacheMaxItem+1))
	cacheIcon("empty", nil)
	cacheIcon("ok", []byte("x"))

	iconCache.Lock()
	_, hasBig := iconCache.m["big"]
	_, hasEmpty := iconCache.m["empty"]
	_, hasOK := iconCache.m["ok"]
	iconCache.Unlock()
	if hasBig || hasEmpty {
		t.Fatal("超大或空的图标不应入缓存")
	}
	if !hasOK {
		t.Fatal("正常大小的图标应入缓存")
	}
}
