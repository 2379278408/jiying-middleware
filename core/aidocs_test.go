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

func TestAIDocsCoverMultiAppContract(t *testing.T) {
	guide := buildAIGuide("http://127.0.0.1:8080")
	for _, want := range []string{"多应用与会话归属", "relay_app_id", "source=app", "apps_locked", "reset_apps", "app_systems", "apps: app-id-1"} {
		if !strings.Contains(guide, want) {
			t.Errorf("ai.txt 缺少多应用说明: %q", want)
		}
	}
	docs := map[string]string{}
	for _, ep := range aiEndpoints {
		docs[ep.Method+" "+ep.Path] = ep.Summary + " " + ep.Notes
	}
	checks := map[string][]string{
		"POST /jiying/api/feature/update":           {"apps_locked", "reset_apps"},
		"GET /jiying/api/conversations":             {"relay_app_"},
		"GET /jiying/api/messages":                  {"source=app"},
		"GET /jiying/api/conversation/capabilities": {"app_online", "app_known"},
		"POST /jiying/api/send/unified":             {"app_id", "dry_run"},
		"GET /jiying/api/tasks":                     {"app_id"},
		"POST /jiying/api/agent/app-system":         {"app_id", "20000"},
		"POST /jiying/api/agent/run":                {"会话归属应用"},
	}
	for key, wants := range checks {
		text, ok := docs[key]
		if !ok {
			t.Errorf("aiEndpoints 缺少 %s", key)
			continue
		}
		for _, want := range wants {
			if !strings.Contains(text, want) {
				t.Errorf("%s 文档缺少 %q", key, want)
			}
		}
	}
}
