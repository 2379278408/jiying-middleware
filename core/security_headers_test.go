package core

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func newHeaderTestServer() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(securityHeaders())
	r.GET("/ping", func(c *gin.Context) { c.String(http.StatusOK, "ok") })
	return r
}

func TestSecurityHeaders(t *testing.T) {
	r := newHeaderTestServer()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ping", nil))
	if got := w.Header().Get("X-Frame-Options"); got != "DENY" {
		t.Fatalf("X-Frame-Options=%q", got)
	}
	if got := w.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("X-Content-Type-Options=%q", got)
	}
	if got := w.Header().Get("Referrer-Policy"); got != "no-referrer" {
		t.Fatalf("Referrer-Policy=%q", got)
	}
	if !strings.Contains(w.Header().Get("Content-Security-Policy"), "frame-ancestors 'none'") {
		t.Fatalf("CSP=%q", w.Header().Get("Content-Security-Policy"))
	}
	if w.Header().Get("Strict-Transport-Security") != "" {
		t.Fatal("明文请求不应下发 HSTS")
	}
}

func TestSecurityHeadersHSTSBehindProxy(t *testing.T) {
	r := newHeaderTestServer()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Header().Get("Strict-Transport-Security") == "" {
		t.Fatal("HTTPS 请求应下发 HSTS")
	}
}

func TestLocalMsgRejectsBrowserOrigin(t *testing.T) {
	r := gin.New()
	r.POST("/local/msg", httpMsgHook)
	req := httptest.NewRequest(http.MethodPost, "/local/msg", strings.NewReader("msg=restart"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "https://evil.example")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("浏览器来源应被拒绝, got %d", w.Code)
	}
}
