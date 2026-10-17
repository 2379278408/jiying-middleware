package core

import (
	"strings"
	"testing"
	"time"
)

func setupAPIKeysBucket(t *testing.T, name string) {
	t.Helper()
	t.Setenv("JIYING_DATA", t.TempDir())
	b, err := MakeBucket(name, 6)
	if err != nil {
		t.Fatal(err)
	}
	BucketApp = b
	BucketApp.Delete(apiKeysKey)
	t.Cleanup(func() {
		BucketApp.Delete(apiKeysKey)
		apiKeyWindows.Lock()
		apiKeyWindows.windows = map[string]*apiKeyWindow{}
		apiKeyWindows.Unlock()
		apiKeyUsage.Lock()
		apiKeyUsage.lastSave = map[string]time.Time{}
		apiKeyUsage.Unlock()
	})
}

func TestAPIKeyLifecycle(t *testing.T) {
	setupAPIKeysBucket(t, "apikeys-life")
	token, item, err := CreateAPIKey("巡检脚本", RoleOps, 30)
	if err != nil {
		t.Fatal(err)
	}
	if token == "" || item.ID == "" {
		t.Fatal("create should return token and metadata")
	}
	if item.Scope != RoleOps || item.ExpiresAt == "" {
		t.Fatalf("unexpected item: %+v", item)
	}
	if raw := BucketApp.GetString(apiKeysKey); raw == "" || strings.Contains(raw, token) {
		t.Fatal("token must not be stored in plaintext")
	}
	got := verifyAPIKey(token)
	if got == nil || got.ID != item.ID {
		t.Fatal("token should verify")
	}
	if verifyAPIKey(token+"x") != nil {
		t.Fatal("modified token must not verify")
	}
	list := ListAPIKeys()
	if len(list) != 1 || list[0].Label != "巡检脚本" {
		t.Fatal("list should expose metadata only")
	}
	if err := DeleteAPIKey(item.ID); err != nil {
		t.Fatal(err)
	}
	if verifyAPIKey(token) != nil {
		t.Fatal("deleted token must not verify")
	}
	if err := DeleteAPIKey(item.ID); err == nil {
		t.Fatal("deleting twice should fail")
	}
}

func TestAPIKeyExpiry(t *testing.T) {
	setupAPIKeysBucket(t, "apikeys-expiry")
	token, item, err := CreateAPIKey("临时", RoleReadonly, 0)
	if err != nil {
		t.Fatal(err)
	}
	if item.ExpiresAt != "" {
		t.Fatal("zero days means no expiry")
	}
	saved := loadAPIKeys()
	rec := saved[item.ID]
	rec.ExpiresAt = time.Now().Add(-time.Minute).Format(time.RFC3339)
	saved[item.ID] = rec
	saveAPIKeys(saved)
	if verifyAPIKey(token) != nil {
		t.Fatal("expired token must not verify")
	}
}

func TestCreateAPIKeyRejectsBadInput(t *testing.T) {
	setupAPIKeysBucket(t, "apikeys-bad")
	if _, _, err := CreateAPIKey("", RoleOps, 30); err == nil {
		t.Fatal("empty label should fail")
	}
	if _, _, err := CreateAPIKey("x", "root", 30); err == nil {
		t.Fatal("unknown scope should fail")
	}
}

func TestAPIKeyRateLimit(t *testing.T) {
	setupAPIKeysBucket(t, "apikeys-rate")
	for i := 0; i < apiKeyRateLimit; i++ {
		if !allowAPIKeyRequest("k1") {
			t.Fatalf("request %d should pass", i)
		}
	}
	if allowAPIKeyRequest("k1") {
		t.Fatal("request over limit should fail")
	}
	if !allowAPIKeyRequest("k2") {
		t.Fatal("other keys have independent windows")
	}
}
