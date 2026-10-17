package core

import (
	"encoding/base64"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"jiying/middleware/utils"
)

var OpenServer *gin.RouterGroup

func init() {
	gin.SetMode(gin.DebugMode)
	Server = gin.New()
	// multipart 超过该内存阈值后落临时文件，避免大文件上传直接占满进程内存。
	Server.MaxMultipartMemory = 8 << 20
	Server.Use(gin.RecoveryWithWriter(io.Discard))
	Server.Use(traceIDMiddleware())
	Server.Use(accessLog())

	SessionStore = newSessionStore(securecookie.GenerateRandomKey(32))
	OpenServer = Server.Group("/open")

	DataHome = utils.GetResHome()

	initWeb()
	initLogger()
	initGlob()
}

func accessLog() gin.HandlerFunc {
	// 心跳轮询与静态资源请求频率高、信息量低，跳过记录以保留有意义的接口调用。
	skip := map[string]bool{"/jiying/api/connect/status": true}
	// 静态资源可能经 NoRoute 兜底直接从资源目录返回（例如 /vendor/...），
	// 按扩展名跳过，避免逐条枚举目录前缀。
	staticExt := map[string]bool{
		".css": true, ".js": true, ".mjs": true, ".map": true,
		".svg": true, ".png": true, ".jpg": true, ".jpeg": true,
		".gif": true, ".webp": true, ".ico": true,
		".woff": true, ".woff2": true, ".ttf": true,
	}
	return func(c *gin.Context) {
		t := time.Now()
		defer func() {
			path := c.Request.URL.Path
			if skip[path] || strings.HasPrefix(path, "/assets/") || staticExt[strings.ToLower(filepath.Ext(path))] {
				return
			}
			trace, _ := c.Get(traceIDContextKey)
			GetMyLog().Debug("gin access %s %s %d %v trace=%v", c.Request.Method, path, c.Writer.Status(), time.Since(t), trace)
		}()
		c.Next()
	}
}

// traceIDMiddleware 为每个请求分配并回传 X-Trace-ID，便于把一次调用与运行日志按同一 ID 关联。
func traceIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Trace-ID", requestTraceID(c))
		c.Next()
	}
}

func initWeb() {
	wd := utils.GetResHome()
	_ = os.MkdirAll(filepath.Join(wd, "data"), 0o777)
	_ = os.MkdirAll(filepath.Join(wd, "logs"), 0o777)

	assetsRoot := filepath.Join(wd, "views2", "assets")
	if _, err := os.Stat(assetsRoot); err != nil {
		assetsRoot = filepath.Join(utils.GetExecPathDir(), "views2", "assets")
	}
	if st, err := os.Stat(assetsRoot); err == nil && st.IsDir() {
		Server.Static("/assets", assetsRoot)
	}

	Server.GET("/", pageIndex)
	Server.GET("/index.html", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/")
	})

	Server.NoRoute(func(c *gin.Context) {
		upath := filepath.Clean("/" + c.Request.URL.Path)
		if upath != "/" {
			full := filepath.Join(assetsRoot, upath)
			if st, err := os.Stat(full); err == nil && !st.IsDir() {
				c.File(full)
				return
			}
		}
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "路径不存在"})
	})
}

func newSessionStore(key []byte) *sessions.CookieStore {
	s := sessions.NewCookieStore(key)
	s.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   24 * 3600 * 7,
		HttpOnly: true,
		// 显式 SameSite=Lax：跨站请求不携带会话 Cookie，缓解 CSRF；
		// 同源表单登录与站内跳转不受影响，内网 HTTP 部署也不因 Secure 失效。
		SameSite: http.SameSiteLaxMode,
	}
	return s
}

func persistSessionKey() {
	const keyName = "session_key"
	if BucketApp != nil {
		if v := BucketApp.GetString(keyName); v != "" {
			if raw, err := base64.StdEncoding.DecodeString(v); err == nil && len(raw) > 0 {
				SessionStore = newSessionStore(raw)
				return
			}
		}
	}
	raw := securecookie.GenerateRandomKey(32)
	if raw == nil {
		return
	}
	if BucketApp != nil {
		BucketApp.Set(keyName, base64.StdEncoding.EncodeToString(raw))
	}
	SessionStore = newSessionStore(raw)
}
