package core

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestTraceIDHeaderEchoesClientValue(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Header.Set("X-Trace-ID", "trace-client-123")
	Server.ServeHTTP(rec, req)
	if got := rec.Header().Get("X-Trace-ID"); got != "trace-client-123" {
		t.Fatalf("X-Trace-ID=%q want trace-client-123", got)
	}
}

func TestTraceIDHeaderGeneratedWhenAbsent(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	Server.ServeHTTP(rec, req)
	if got := rec.Header().Get("X-Trace-ID"); got == "" {
		t.Fatal("缺少 X-Trace-ID 响应头")
	}
}

func TestRequestTraceIDStablePerContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Request.Header.Set("X-Trace-ID", "abc123")
	if got := requestTraceID(c); got != "abc123" {
		t.Fatalf("首次应取客户端值, got %q", got)
	}
	if got := requestTraceID(c); got != "abc123" {
		t.Fatalf("同请求应复用缓存值, got %q", got)
	}

	rec2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(rec2)
	c2.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	first := requestTraceID(c2)
	if first == "" || requestTraceID(c2) != first {
		t.Fatal("无客户端 ID 时应生成且保持稳定")
	}
}

// 含控制字符或非法字符的客户端追踪 ID 必须被拒绝并改为服务端生成，
// 避免污染访问日志与响应头。
func TestRequestTraceIDRejectsUnsafeValue(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, raw := range []string{
		"bad id\nwith-newline",
		"trace\r\ninject",
		"has space",
		"emoji-\U0001F600",
	} {
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		c.Request.Header.Set("X-Trace-ID", raw)
		got := requestTraceID(c)
		if got == raw {
			t.Fatalf("非法追踪 ID 不应被采纳: %q", raw)
		}
		if !validTraceID(got) {
			t.Fatalf("生成的追踪 ID 应合法: %q", got)
		}
	}
}
