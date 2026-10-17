package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUserSessionTokenRoundTrip(t *testing.T) {
	oldHome := DataHome
	DataHome = t.TempDir()
	defer func() { DataHome = oldHome }()
	savedSessions.Lock()
	savedSessions.items = map[string]savedUserSession{}
	savedSessions.Unlock()

	saveUserToken("a@example.com", "tok-123")
	token, meta, ok := loadUserToken("a@example.com")
	if !ok || token != "tok-123" || meta.LastOK == "" {
		t.Fatalf("token 保存失败: %q %+v %v", token, meta, ok)
	}
	info, err := os.Stat(filepath.Join(DataHome, "user_sessions.json"))
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("会话文件权限错误: %v %v", info, err)
	}
	savedSessions.Lock()
	delete(savedSessions.items, "a@example.com")
	savedSessions.Unlock()
	loadSavedSessions()
	if token, _, ok := loadUserToken("a@example.com"); !ok || token != "tok-123" {
		t.Fatalf("重启恢复失败: %q %v", token, ok)
	}
	clearUserToken("a@example.com")
	if _, _, ok := loadUserToken("a@example.com"); ok {
		t.Fatal("退出后 token 未清除")
	}
}
