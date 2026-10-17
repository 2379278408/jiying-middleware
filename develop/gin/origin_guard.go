package gin

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

// sameOriginOnly 拒绝浏览器跨站表单提交，缓解登录 CSRF。
// 无 Origin/Referer 的请求（脚本、API 客户端）不携带浏览器 Cookie，放行即可。
func sameOriginOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		if origin := strings.TrimSpace(c.GetHeader("Origin")); origin != "" {
			if !sameRequestHost(origin, c.Request.Host) {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"code": 403, "message": "跨站请求被拒绝"})
				return
			}
		} else if ref := strings.TrimSpace(c.GetHeader("Referer")); ref != "" {
			if !sameRequestHost(ref, c.Request.Host) {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"code": 403, "message": "跨站请求被拒绝"})
				return
			}
		}
		c.Next()
	}
}

func sameRequestHost(raw, host string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.Host == "null" {
		return false
	}
	if strings.EqualFold(u.Host, host) {
		return true
	}
	return strings.EqualFold(u.Hostname(), hostnameOnly(host))
}

func hostnameOnly(host string) string {
	if i := strings.LastIndex(host, ":"); i > 0 && !strings.Contains(host[i:], "]") {
		return host[:i]
	}
	return host
}
