package core

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

var iconCache = struct {
	sync.Mutex
	m     map[string][]byte
	order []string
	bytes int
}{m: map[string][]byte{}}

const (
	iconCacheMaxBytes = 8 << 20
	iconCacheMaxItem  = 1 << 20
)

// cacheIcon 以 FIFO 加总字节预算的方式限制缓存规模：关键词由用户提供，
// 不设上限的 map 会随不同关键词持续增长，单条超限的响应直接跳过缓存。
func cacheIcon(key string, data []byte) {
	if len(data) == 0 || len(data) > iconCacheMaxItem {
		return
	}
	iconCache.Lock()
	defer iconCache.Unlock()
	if _, ok := iconCache.m[key]; !ok {
		iconCache.order = append(iconCache.order, key)
	} else {
		iconCache.bytes -= len(iconCache.m[key])
	}
	iconCache.m[key] = data
	iconCache.bytes += len(data)
	for len(iconCache.order) > 0 && iconCache.bytes > iconCacheMaxBytes {
		oldest := iconCache.order[0]
		iconCache.order = iconCache.order[1:]
		if d, ok := iconCache.m[oldest]; ok {
			iconCache.bytes -= len(d)
			delete(iconCache.m, oldest)
		}
	}
}

func registerIconAPI() {
	g := Server.Group("/jiying/api", requireLogin)
	g.GET("/icon", iconProxy)
}

func iconProxy(c *gin.Context) {
	kw := strings.TrimSpace(c.Query("msg"))
	n := c.Query("n")
	if kw == "" {
		jiyingErr(c, "缺少图标关键词 msg")
		return
	}
	cacheKey := kw + "|" + n
	iconCache.Lock()
	if v, ok := iconCache.m[cacheKey]; ok {
		iconCache.Unlock()
		ct := "application/json; charset=utf-8"
		if n != "" {
			ct = "image/png"
		}
		c.Data(http.StatusOK, ct, v)
		return
	}
	iconCache.Unlock()

	u := "http://xiaoapi.cn/v1/ico.php?msg=" + url.QueryEscape(kw)
	if n != "" {
		u += "&n=" + url.QueryEscape(n)
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 12*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	req.Header.Set("User-Agent", "Relay/1.0")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		jiyingErr(c, "图标服务不可达")
		return
	}
	defer resp.Body.Close()
	out, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	cacheIcon(cacheKey, out)
	ct := "application/json; charset=utf-8"
	if n != "" {
		ct = "image/png"
	}
	c.Data(http.StatusOK, ct, out)
}
