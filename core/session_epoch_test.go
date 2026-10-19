package core

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
)

func TestSessionEpochInvalidatesOldCookie(t *testing.T) {
	t.Setenv("JIYING_DATA", t.TempDir())
	b, err := MakeBucket("session-epoch", 6)
	if err != nil {
		t.Fatal(err)
	}
	prevBucket, prevStore := BucketApp, SessionStore
	BucketApp, SessionStore = b, sessions.NewCookieStore([]byte("0123456789abcdef0123456789abcdef"))
	t.Cleanup(func() { BucketApp, SessionStore = prevBucket, prevStore })

	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest("GET", "/", nil)
	s, err := SessionStore.Get(c.Request, "jiying_session")
	if err != nil {
		t.Fatal(err)
	}
	s.Values["user"] = "u1"
	s.Values["epoch"] = SessionEpoch("u1")
	if err := s.Save(c.Request, rec); err != nil {
		t.Fatal(err)
	}
	cookies := rec.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("会话 Cookie 未签发")
	}

	check := func() bool {
		r := httptest.NewRequest("GET", "/", nil)
		r.AddCookie(cookies[0])
		ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
		ctx.Request = r
		return CheckCookie(ctx)
	}
	if !check() {
		t.Fatal("当前代次的会话应有效")
	}
	BumpSessionEpoch("u1")
	if check() {
		t.Fatal("改密/禁用后旧会话应立即失效")
	}
	if SessionEpoch("u1") != 1 {
		t.Fatalf("代次应递增到 1，实际 %d", SessionEpoch("u1"))
	}
}
