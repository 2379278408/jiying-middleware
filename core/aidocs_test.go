package core

import (
	"sort"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAIEndpointsMatchRegisteredRoutes(t *testing.T) {
	actual := map[string]bool{}
	for _, r := range Server.Routes() {
		if strings.HasPrefix(r.Path, "/jiying/api/") {
			actual[r.Method+" "+r.Path] = true
		}
	}
	listed := map[string]bool{}
	for _, ep := range aiEndpoints {
		listed[ep.Method+" "+ep.Path] = true
	}
	var missing, extra []string
	for key := range listed {
		if !actual[key] {
			missing = append(missing, key)
		}
	}
	for key := range actual {
		if !listed[key] {
			extra = append(extra, key)
		}
	}
	sort.Strings(missing)
	sort.Strings(extra)
	if len(missing) > 0 {
		t.Fatalf("aiEndpoints 中登记了未注册的路由: %v", missing)
	}
	if len(extra) > 0 {
		t.Fatalf("已注册但未写入 aiEndpoints 的路由: %v", extra)
	}
	if len(listed) != len(aiEndpoints) {
		t.Fatalf("aiEndpoints 存在重复条目: %d/%d", len(listed), len(aiEndpoints))
	}
}

func TestOpenAPIHasSecuritySchemesAndPaths(t *testing.T) {
	spec := buildOpenAPI("http://127.0.0.1:8080")
	paths, ok := spec["paths"].(gin.H)
	if !ok {
		t.Fatal("OpenAPI 缺少 paths")
	}
	ops := 0
	for _, item := range paths {
		if methods, ok := item.(gin.H); ok {
			ops += len(methods)
		}
	}
	if ops != len(aiEndpoints) {
		t.Fatalf("OpenAPI 操作数应为 %d，实际 %d", len(aiEndpoints), ops)
	}
	components, ok := spec["components"].(gin.H)
	if !ok {
		t.Fatal("OpenAPI 缺少 components")
	}
	schemes, ok := components["securitySchemes"].(gin.H)
	if !ok || schemes["bearerAuth"] == nil || schemes["cookieAuth"] == nil {
		t.Fatal("OpenAPI 缺少 bearerAuth 或 cookieAuth")
	}
}
