package gin

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"jiying/middleware/core"
)

func setupSysBucket(t *testing.T, name string) core.Bucket {
	t.Helper()
	t.Setenv("JIYING_DATA", t.TempDir())
	b, err := core.MakeBucket(name, 6)
	if err != nil {
		t.Fatal(err)
	}
	old := core.BucketApp
	core.BucketApp = b
	t.Cleanup(func() { core.BucketApp = old })
	return b
}

func TestSensitiveBucketKey(t *testing.T) {
	for _, k := range []string{"session_key", "SESSION_KEY", "plugincfg:demo", "api_key"} {
		if !sensitiveBucketKey(k) {
			t.Fatalf("%q 应判定为敏感键", k)
		}
	}
	for _, k := range []string{"appName", "icp", "masters", ""} {
		if sensitiveBucketKey(k) {
			t.Fatalf("%q 不应判定为敏感键", k)
		}
	}
}

func TestRequireConsoleMasterDeniesAnonymous(t *testing.T) {
	old := core.SessionStore
	core.SessionStore = sessions.NewCookieStore(securecookie.GenerateRandomKey(32))
	t.Cleanup(func() { core.SessionStore = old })

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/buckets", nil)

	requireConsoleMaster()(c)

	if !c.IsAborted() || w.Code != http.StatusForbidden {
		t.Fatalf("匿名访问应被拒绝, aborted=%v code=%d", c.IsAborted(), w.Code)
	}
}

func TestRequireConsoleRoleDeniesAnonymous(t *testing.T) {
	old := core.SessionStore
	core.SessionStore = sessions.NewCookieStore(securecookie.GenerateRandomKey(32))
	t.Cleanup(func() { core.SessionStore = old })

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/dependencies", nil)

	requireConsoleRole(core.RoleOps)(c)

	if !c.IsAborted() || w.Code != http.StatusForbidden {
		t.Fatalf("匿名访问应被拒绝, aborted=%v code=%d", c.IsAborted(), w.Code)
	}
}

func TestGetSystemKeyBlocksSessionKey(t *testing.T) {
	b := setupSysBucket(t, "sys-key-test")
	b.Set("session_key", "super-secret-signing-key")

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/system/session_key", nil)
	c.Params = gin.Params{{Key: "key", Value: "session_key"}}
	getSystemKey(c)

	if w.Code != http.StatusForbidden {
		t.Fatalf("读取 session_key 应返回 403, got %d body=%s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "super-secret-signing-key") {
		t.Fatal("响应不应包含会话签名密钥")
	}
}

func TestBucketSummaryHidesSensitive(t *testing.T) {
	b := setupSysBucket(t, "sys-summary-test")
	b.Set("session_key", "super-secret-signing-key")
	b.Set("appName", "demo")

	got := bucketSummary(b)
	if strings.Contains(got, "session_key") || strings.Contains(got, "super-secret-signing-key") {
		t.Fatalf("摘要不应包含敏感键: %q", got)
	}
	if !strings.Contains(got, "appName = demo") {
		t.Fatalf("摘要应保留普通键: %q", got)
	}
}

func TestSaveSystemParamsRejectsProtectedKeys(t *testing.T) {
	b := setupSysBucket(t, "sys-params-test")
	b.Set("masters", "boss")

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := "masters=hacker&jiying_roles=x&session_key=leak&appName=demo"
	c.Request = httptest.NewRequest(http.MethodPost, "/system/params", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	saveSystemParams(c)

	if got := b.GetString("masters"); got != "boss" {
		t.Fatalf("masters 不应被改写, got %q", got)
	}
	if got := b.GetString("session_key"); got != "" {
		t.Fatalf("session_key 不应被写入, got %q", got)
	}
	if got := b.GetString("appName"); got != "demo" {
		t.Fatalf("普通参数应写入, got %q", got)
	}
}
