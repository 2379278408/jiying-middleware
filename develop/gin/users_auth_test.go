package gin

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"jiying/middleware/core"
)

// 用户列表会返回账号、昵称与邮箱，未登录不得访问。
func TestUsersListRequiresAuth(t *testing.T) {
	oldServer, oldStore := core.Server, core.SessionStore
	t.Cleanup(func() { core.Server, core.SessionStore = oldServer, oldStore })

	gin.SetMode(gin.TestMode)
	core.Server = gin.New()
	core.SessionStore = sessions.NewCookieStore(securecookie.GenerateRandomKey(32))
	initUsers()

	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	w := httptest.NewRecorder()
	core.Server.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("未登录访问 /users 应返回 401, got %d body=%s", w.Code, w.Body.String())
	}
}
