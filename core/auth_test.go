package core

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestJiyingAPIRequiresLogin(t *testing.T) {
	cases := []string{
		"/jiying/api/connect/status",
		"/jiying/api/overview",
		"/jiying/api/logs",
		"/jiying/status",
		"/jiying/ping",
	}
	for _, path := range cases {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		Server.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s: want 401, got %d", path, rec.Code)
		}
	}
}

func TestRequireLoginAllowsSession(t *testing.T) {
	// 先颁发一个合法会话 cookie。
	storeRec := httptest.NewRecorder()
	storeReq := httptest.NewRequest(http.MethodGet, "/", nil)
	sess, err := SessionStore.Get(storeReq, "jiying_session")
	if err != nil {
		t.Fatal(err)
	}
	sess.Values["user"] = "admin"
	if err := sess.Save(storeReq, storeRec); err != nil {
		t.Fatal(err)
	}
	cookies := storeRec.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("no session cookie issued")
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/jiying/api/connect/status", nil)
	for _, c := range cookies {
		req.AddCookie(c)
	}
	Server.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("authenticated request got %d", rec.Code)
	}
}
