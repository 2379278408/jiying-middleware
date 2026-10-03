package core

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHttpMsgHookLocalOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	if Server == nil {
		Server = gin.New()
	}
	initHttpServer()

	Senders = make(chan Sender, 4)
	defer func() { Senders = make(chan Sender, 100) }()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/local/msg", strings.NewReader("msg=hello"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.RemoteAddr = "127.0.0.1:1234"
	Server.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("回环请求应放行, got %d", rec.Code)
	}
	select {
	case s := <-Senders:
		if s.GetMsg() != "hello" {
			t.Fatalf("消息内容错误: %q", s.GetMsg())
		}
	case <-never():
		t.Fatal("Senders 未收到消息")
	}

	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/local/msg", strings.NewReader("msg=evil"))
	req2.RemoteAddr = "8.8.8.8:9999"
	Server.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusForbidden {
		t.Fatalf("非回环请求应 403, got %d", rec2.Code)
	}
}

func TestDispatchLocalCommand(t *testing.T) {
	cmdRouter.Store("__local_test__", func() {})
	defer cmdRouter.Delete("__local_test__")

	if r := dispatchLocalCommand("__local_test__ args"); !strings.Contains(r, "已执行") {
		t.Fatalf("已注册命令应执行, got %q", r)
	}
	if r := dispatchLocalCommand("__no_such_cmd__"); !strings.Contains(r, "未注册") {
		t.Fatalf("未注册命令应提示未注册, got %q", r)
	}
	if r := dispatchLocalCommand("   "); r != "空命令" {
		t.Fatalf("空命令应提示, got %q", r)
	}
}

// 非回环来源即使伪造 X-Forwarded-For 为 127.0.0.1 也必须拒绝：
// 本机限制只能依据 TCP 对端地址判断。
func TestLocalOnlyRejectsSpoofedForwardedFor(t *testing.T) {
	gin.SetMode(gin.TestMode)
	if Server == nil {
		Server = gin.New()
	}
	initHttpServer()

	Senders = make(chan Sender, 4)
	defer func() { Senders = make(chan Sender, 100) }()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/local/msg", strings.NewReader("msg=evil"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Forwarded-For", "127.0.0.1")
	req.RemoteAddr = "8.8.8.8:9999"
	Server.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("伪造 X-Forwarded-For 不应放行, got %d", rec.Code)
	}
	select {
	case s := <-Senders:
		t.Fatalf("被拒绝的请求不应投递消息: %q", s.GetMsg())
	default:
	}
}

func never() <-chan struct{} {
	c := make(chan struct{})
	return c
}
