package core

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
	spec := buildOpenAPI("http://127.0.0.1:8080", RoleMaster, false)
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
	guide := buildAIGuide("http://127.0.0.1:8080", RoleMaster, false)
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

func TestAIDocsAreFilteredByScope(t *testing.T) {
	publicGuide := buildAIGuide("http://example.test", RoleReadonly, true)
	if !strings.Contains(publicGuide, "只读公开文档") {
		t.Fatal("公开指南缺少范围说明")
	}
	for _, forbidden := range []string{"script/install", "script/run", "cron/add", "/jiying/api/roles", "agent/config", "系统命令", "插件安装", "新建后默认启用", "批量发送"} {
		if strings.Contains(publicGuide, forbidden) {
			t.Fatalf("公开指南泄露了 %q", forbidden)
		}
	}
	readonlyGuide := buildAIGuide("http://example.test", RoleReadonly, false)
	if strings.Contains(readonlyGuide, "script/install") {
		t.Fatal("只读密钥指南不应包含写接口")
	}
	if !strings.Contains(buildAIGuide("http://example.test", RoleOps, false), "script/install") {
		t.Fatal("ops 指南应包含插件安装入口")
	}
	if !strings.Contains(buildAIGuide("http://example.test", RoleMaster, false), "/jiying/api/keys") {
		t.Fatal("master 指南应包含密钥管理入口")
	}

	cases := map[string]int{
		RoleReadonly: len(filterEndpointsByScope(aiEndpoints, RoleReadonly)),
		RoleOps:      len(filterEndpointsByScope(aiEndpoints, RoleOps)),
		RoleMaster:   len(filterEndpointsByScope(aiEndpoints, RoleMaster)),
	}
	if cases[RoleMaster] != len(aiEndpoints) {
		t.Fatalf("master 应看到全部 %d 个端点，实际 %d", len(aiEndpoints), cases[RoleMaster])
	}
	if !(cases[RoleReadonly] < cases[RoleOps] && cases[RoleOps] < cases[RoleMaster]) {
		t.Fatalf("端点应随权限递增，实际 readonly=%d ops=%d master=%d", cases[RoleReadonly], cases[RoleOps], cases[RoleMaster])
	}

	spec := buildOpenAPI("http://example.test", RoleReadonly, true)
	paths := spec["paths"].(gin.H)
	for _, forbidden := range []string{"/jiying/api/roles", "/jiying/api/keys", "/jiying/api/agent/config", "/jiying/api/script/install"} {
		if paths[forbidden] != nil {
			t.Fatalf("公开 OpenAPI 泄露了 %s", forbidden)
		}
	}
}

func TestPublicOpenAPIContainsOnlyGet(t *testing.T) {
	for _, ep := range visibleEndpoints(aiEndpoints, RoleReadonly, true) {
		if ep.Method != "GET" {
			t.Fatalf("匿名 OpenAPI 不应包含 %s %s", ep.Method, ep.Path)
		}
	}
	readonlyPaths := buildOpenAPI("http://example.test", RoleReadonly, false)["paths"].(gin.H)
	entry, ok := readonlyPaths["/jiying/api/connections/account/profile"].(gin.H)
	if !ok || entry["post"] == nil {
		t.Fatal("readonly 密钥应能查到幂等刷新端点")
	}
	publicPaths := buildOpenAPI("http://example.test", RoleReadonly, true)["paths"].(gin.H)
	if publicPaths["/jiying/api/connections/account/profile"] != nil {
		t.Fatal("匿名 OpenAPI 不应包含 POST 刷新端点")
	}
}

func TestAIDocsRejectInvalidBearer(t *testing.T) {
	setupAPIKeysBucket(t, "aidocs-invalid-key")
	for _, path := range []string{"/jiying/ai.txt", "/jiying/openapi.json", "/jiying/api/ai/manifest"} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer jy_invalid")
		Server.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s 使用无效密钥应返回 401，实际 %d", path, rec.Code)
		}
	}
}

func TestAIDocsTreatInvalidCookieAsPublic(t *testing.T) {
	setupAPIKeysBucket(t, "aidocs-public")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/jiying/openapi.json", nil)
	req.AddCookie(&http.Cookie{Name: "jiying_session", Value: "forged"})
	Server.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("无效 Cookie 应按匿名处理，实际 %d", rec.Code)
	}
	if cc := rec.Header().Get("Cache-Control"); cc != "no-store" {
		t.Fatalf("公开文档应禁止共享缓存，实际 %q", cc)
	}
	spec := map[string]any{}
	if err := json.Unmarshal(rec.Body.Bytes(), &spec); err != nil {
		t.Fatal(err)
	}
	paths, _ := spec["paths"].(map[string]any)
	for path := range paths {
		if path == "/jiying/api/script/install" || path == "/jiying/api/keys" {
			t.Fatalf("匿名 OpenAPI 泄露了 %s", path)
		}
	}
}
