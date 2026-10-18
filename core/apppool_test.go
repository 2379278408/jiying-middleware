package core

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
)

func resetAppPoolState() {
	appConnPool.mu.Lock()
	appConnPool.items = map[string]*appRuntime{}
	appConnPool.convApp = map[string]string{}
	appConnPool.convOrder = nil
	appConnPool.convBucket = ""
	appConnPool.convLoaded = false
	appConnPool.defaultID = ""
	activeJiying = nil
	appConnPool.mu.Unlock()
}

func setupAppPoolBucket(t *testing.T, name string) {
	t.Helper()
	t.Setenv("JIYING_SECRET_KEY", "00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff")
	t.Setenv("JIYING_DATA", t.TempDir())
	b, err := MakeBucket(name, 6)
	if err != nil {
		t.Fatal(err)
	}
	BucketApp = b
	BucketApp.Delete(apppoolKey)
	BucketApp.Delete(convAppsKey)
	BucketApp.Delete("relay_conn")
	resetAppPoolState()
	t.Cleanup(func() {
		BucketApp.Delete(apppoolKey)
		BucketApp.Delete(convAppsKey)
		BucketApp.Delete("relay_conn")
		resetAppPoolState()
	})
}

func TestAppConnsRoundTripSealsSecrets(t *testing.T) {
	setupAppPoolBucket(t, "apppool-roundtrip")
	saveAppConns([]AppConn{{WSURL: "ws://x/api/app/ws", AppID: "app-a", AppSecret: "plain-secret"}})

	raw := BucketApp.GetString(apppoolKey)
	if bytes.Contains([]byte(raw), []byte("plain-secret")) {
		t.Fatalf("secret must be sealed at rest: %s", raw)
	}
	list := loadAppConns()
	if len(list) != 1 || list[0].AppID != "app-a" || list[0].AppSecret != "plain-secret" {
		t.Fatalf("round trip mismatch: %+v", list)
	}
}

func TestLoadAppConnsFallsBackToLegacy(t *testing.T) {
	setupAppPoolBucket(t, "apppool-legacy")
	BucketApp.Set("relay_conn", `{"app":{"ws_url":"ws://legacy","app_id":"legacy-id","app_secret":"legacy-secret"},"account":{}}`)
	list := loadAppConns()
	if len(list) != 1 || list[0].AppID != "legacy-id" || list[0].AppSecret != "legacy-secret" {
		t.Fatalf("legacy fallback mismatch: %+v", list)
	}
}

func TestClientForConvRoutesByRememberedApp(t *testing.T) {
	setupAppPoolBucket(t, "apppool-routing")

	a := NewJiyingClient(JiyingConfig{AppID: "app-a"}, nil)
	b := NewJiyingClient(JiyingConfig{AppID: "app-b"}, nil)
	appConnPool.mu.Lock()
	appConnPool.items["app-a"] = &appRuntime{cfg: AppConn{AppID: "app-a"}, client: a}
	appConnPool.items["app-b"] = &appRuntime{cfg: AppConn{AppID: "app-b"}, client: b}
	appConnPool.defaultID = "app-a"
	appConnPool.syncDefaultLocked()
	appConnPool.mu.Unlock()

	if activeJiying != a {
		t.Fatal("default client must mirror into activeJiying")
	}
	if got := currentJiying(); got != a {
		t.Fatal("currentJiying must return the default client")
	}
	rememberConvApp("conv-b", "app-b")
	if got := clientForConv("conv-b"); got != b {
		t.Fatal("conversation must route to its own app client")
	}
	if got := clientForConv("conv-unknown"); got != a {
		t.Fatal("unknown conversation must fall back to default app")
	}
}

func TestSetDefaultAppConnReordersPool(t *testing.T) {
	setupAppPoolBucket(t, "apppool-default")
	t.Cleanup(stopAppPool)
	saveAppConns([]AppConn{
		{WSURL: "ws://a", AppID: "app-a", AppSecret: "secret-a"},
		{WSURL: "ws://b", AppID: "app-b", AppSecret: "secret-b"},
	})
	if err := SetDefaultAppConn("app-b"); err != nil {
		t.Fatal(err)
	}
	list := loadAppConns()
	if len(list) != 2 || list[0].AppID != "app-b" {
		t.Fatalf("expected app-b first, got %#v", list)
	}
}

func TestSendTextToConvWithoutClientFails(t *testing.T) {
	appConnPool.mu.Lock()
	appConnPool.items = map[string]*appRuntime{}
	appConnPool.convApp = map[string]string{}
	appConnPool.defaultID = ""
	appConnPool.mu.Unlock()
	activeJiying = nil
	if err := SendTextToConv(context.Background(), "conv", "", "hi"); err == nil {
		t.Fatal("send without any channel must fail")
	}
}

// 并发 upsert 不同应用时必须全部落库：连接池配置是整份列表覆写，
// 缺少串行化会发生丢失更新。
func TestUpsertAppConnConcurrentNoLostUpdate(t *testing.T) {
	setupAppPoolBucket(t, "apppool-concurrent")
	t.Cleanup(stopAppPool)

	const n = 8
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			cfg := AppConn{
				WSURL:     "ws://127.0.0.1:1/api/app/ws",
				AppID:     fmt.Sprintf("app-%d", i),
				AppSecret: fmt.Sprintf("secret-%d", i),
			}
			if err := UpsertAppConn(cfg); err != nil {
				t.Errorf("upsert app-%d: %v", i, err)
			}
		}(i)
	}
	wg.Wait()

	list := loadAppConns()
	if len(list) != n {
		t.Fatalf("lost update: want %d apps, got %d (%+v)", n, len(list), list)
	}
	seen := map[string]bool{}
	for _, a := range list {
		seen[a.AppID] = true
	}
	for i := 0; i < n; i++ {
		if !seen[fmt.Sprintf("app-%d", i)] {
			t.Fatalf("app-%d missing after concurrent upserts: %+v", i, list)
		}
	}
}

func TestConvAppOwnershipPersistsAndFailsClosed(t *testing.T) {
	setupAppPoolBucket(t, "apppool-owner")
	a := NewJiyingClient(JiyingConfig{AppID: "app-a"}, nil)
	b := NewJiyingClient(JiyingConfig{AppID: "app-b"}, nil)
	appConnPool.mu.Lock()
	appConnPool.items["app-a"] = &appRuntime{cfg: AppConn{AppID: "app-a"}, client: a}
	appConnPool.items["app-b"] = &appRuntime{cfg: AppConn{AppID: "app-b"}, client: b}
	appConnPool.defaultID = "app-a"
	appConnPool.syncDefaultLocked()
	appConnPool.mu.Unlock()

	rememberConvApp("conv-b", "app-b")
	if got := clientForConv("conv-b"); got != b {
		t.Fatal("conversation must route to owning app")
	}
	if appID, _, _, known := ConvAppOwner("conv-b"); !known || appID != "app-b" {
		t.Fatalf("owner mismatch: %s %v", appID, known)
	}

	// 模拟进程重启：内存清空后应从存储恢复归属，而不是落回默认应用。
	appConnPool.mu.Lock()
	appConnPool.convApp = map[string]string{}
	appConnPool.convOrder = nil
	appConnPool.convBucket = ""
	appConnPool.convLoaded = false
	appConnPool.items["app-a"] = &appRuntime{cfg: AppConn{AppID: "app-a"}, client: a}
	appConnPool.items["app-b"] = &appRuntime{cfg: AppConn{AppID: "app-b"}, client: b}
	appConnPool.mu.Unlock()
	if got := clientForConv("conv-b"); got != b {
		t.Fatal("ownership must survive reload")
	}

	// 归属应用被移除时失败关闭，不能把消息误发到默认应用。
	appConnPool.mu.Lock()
	delete(appConnPool.items, "app-b")
	appConnPool.mu.Unlock()
	if got := clientForConv("conv-b"); got != nil {
		t.Fatal("known conversation without app must fail closed")
	}
	if got := clientForConv("conv-unknown"); got != a {
		t.Fatal("unknown conversation may fall back to default")
	}
}

func TestRemovedAppClearsConversationOwnership(t *testing.T) {
	setupAppPoolBucket(t, "apppool-cleanup")
	t.Cleanup(stopAppPool)
	saveAppConns([]AppConn{
		{WSURL: "ws://a", AppID: "app-a", AppSecret: "secret-a"},
		{WSURL: "ws://b", AppID: "app-b", AppSecret: "secret-b"},
	})
	applyAppConns(loadAppConns())
	rememberConvApp("conv-b", "app-b")
	if err := RemoveAppConn("app-b"); err != nil {
		t.Fatal(err)
	}
	if _, _, _, known := ConvAppOwner("conv-b"); known {
		t.Fatal("removed app ownership must be cleared")
	}
}

func TestAnnotateConversationsAddsAppOwnership(t *testing.T) {
	setupAppPoolBucket(t, "apppool-annotate")
	c := NewJiyingClient(JiyingConfig{AppID: "app-a"}, nil)
	appConnPool.mu.Lock()
	appConnPool.items["app-a"] = &appRuntime{cfg: AppConn{AppID: "app-a"}, client: c}
	appConnPool.defaultID = "app-a"
	appConnPool.syncDefaultLocked()
	appConnPool.mu.Unlock()
	rememberConvApp("conv-a", "app-a")

	in := []byte(`{"conversations":[{"id":"conv-a","name":"A"},{"id":"conv-x","name":"X"}],"total":2}`)
	out := annotateConversations(in)
	var root struct {
		Conversations []map[string]any `json:"conversations"`
		Total         int              `json:"total"`
	}
	if err := json.Unmarshal(out, &root); err != nil {
		t.Fatal(err)
	}
	if root.Total != 2 || len(root.Conversations) != 2 {
		t.Fatalf("upstream fields lost: %s", out)
	}
	a := root.Conversations[0]
	if a["relay_app_id"] != "app-a" || a["relay_app_known"] != true || a["name"] != "A" {
		t.Fatalf("owned conversation mismatch: %+v", a)
	}
	x := root.Conversations[1]
	if x["relay_app_known"] != false || x["relay_app_id"] != nil {
		t.Fatalf("unknown conversation must stay unassigned: %+v", x)
	}
}

func TestEnqueueTaskCapturesConversationApp(t *testing.T) {
	setupAppPoolBucket(t, "apppool-task")
	c := NewJiyingClient(JiyingConfig{AppID: "app-a"}, nil)
	appConnPool.mu.Lock()
	appConnPool.items["app-a"] = &appRuntime{cfg: AppConn{AppID: "app-a"}, client: c}
	appConnPool.defaultID = "app-a"
	appConnPool.syncDefaultLocked()
	appConnPool.mu.Unlock()
	rememberConvApp("conv-task", "app-a")

	task, err := EnqueueTask(Task{Feature: "帮助", UserID: "user-app-task", ConversationID: "conv-task", Confirm: true})
	if err != nil {
		t.Fatal(err)
	}
	defer DeleteTask(task.ID)
	if task.AppID != "app-a" {
		t.Fatalf("task must capture conversation app, got %q", task.AppID)
	}
}
