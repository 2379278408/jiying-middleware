package core

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var errAccountNeedsCode = errors.New("账号使用邮箱验证码登录，请重新获取验证码")

type savedUserSession struct {
	Token   string `json:"token"`
	SavedAt string `json:"saved_at"`
	LastOK  string `json:"last_ok,omitempty"`
}

var savedSessions = struct {
	sync.Mutex
	items map[string]savedUserSession
}{items: map[string]savedUserSession{}}

var accountHealth = struct {
	sync.Mutex
	Status    string
	Since     string
	LastOK    string
	CheckedAt string
	Message   string
}{Status: "unknown"}

func userSessionsPath() string {
	return filepath.Join(DataHome, "user_sessions.json")
}

func loadSavedSessions() {
	raw, err := os.ReadFile(userSessionsPath())
	if err != nil {
		return
	}
	var items map[string]savedUserSession
	if json.Unmarshal(raw, &items) != nil || items == nil {
		return
	}
	for email, item := range items {
		item.Token = openSecret(item.Token)
		items[email] = item
	}
	savedSessions.Lock()
	savedSessions.items = items
	savedSessions.Unlock()
}

func persistSavedSessions() {
	savedSessions.Lock()
	sealed := make(map[string]savedUserSession, len(savedSessions.items))
	for email, item := range savedSessions.items {
		item.Token = sealSecret(item.Token)
		sealed[email] = item
	}
	savedSessions.Unlock()
	raw, err := json.Marshal(sealed)
	if err != nil {
		return
	}
	dir := filepath.Dir(userSessionsPath())
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return
	}
	tmp, err := os.CreateTemp(dir, ".user-sessions-*.tmp")
	if err != nil {
		return
	}
	name := tmp.Name()
	defer os.Remove(name)
	_ = tmp.Chmod(0o600)
	if _, err := tmp.Write(raw); err != nil {
		tmp.Close()
		return
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return
	}
	if err := tmp.Close(); err != nil {
		return
	}
	_ = os.Rename(name, userSessionsPath())
}

func saveUserToken(email, token string) {
	email = strings.TrimSpace(email)
	if email == "" || token == "" {
		return
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	savedSessions.Lock()
	item := savedSessions.items[email]
	item.Token = token
	item.SavedAt = now
	item.LastOK = now
	savedSessions.items[email] = item
	savedSessions.Unlock()
	persistSavedSessions()
	setAccountHealth("valid", "", now, "会话已保存")
}

func loadUserToken(email string) (string, savedUserSession, bool) {
	email = strings.TrimSpace(email)
	savedSessions.Lock()
	item, ok := savedSessions.items[email]
	savedSessions.Unlock()
	return item.Token, item, ok && item.Token != ""
}

func clearUserToken(email string) {
	email = strings.TrimSpace(email)
	savedSessions.Lock()
	delete(savedSessions.items, email)
	savedSessions.Unlock()
	persistSavedSessions()
	setAccountHealth("expired", "", "", "登录已失效")
}

func markUserSessionOK(email string) {
	email = strings.TrimSpace(email)
	if email == "" {
		return
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	savedSessions.Lock()
	item, ok := savedSessions.items[email]
	if ok {
		item.LastOK = now
		savedSessions.items[email] = item
	}
	savedSessions.Unlock()
	if ok {
		persistSavedSessions()
	}
	setAccountHealth("valid", "", now, "")
}

func setAccountHealth(status, since, lastOK, message string) {
	accountHealth.Lock()
	defer accountHealth.Unlock()
	if status != "" && status != accountHealth.Status {
		accountHealth.Since = time.Now().Format("2006-01-02 15:04:05")
	}
	accountHealth.Status = status
	accountHealth.CheckedAt = time.Now().Format("2006-01-02 15:04:05")
	if lastOK != "" {
		accountHealth.LastOK = lastOK
	}
	if message != "" || status == "valid" {
		accountHealth.Message = message
	}
}

// AccountHealth 是账号通道健康状态快照，token 本身不会出现在响应中。
type AccountHealth struct {
	Email     string `json:"email"`
	Status    string `json:"status"`
	Since     string `json:"since,omitempty"`
	LastOK    string `json:"last_ok,omitempty"`
	CheckedAt string `json:"checked_at,omitempty"`
	Message   string `json:"message,omitempty"`
}

func accountHealthSnapshot() AccountHealth {
	email := UserSessionEmail()
	if email == "" {
		email = CurrentAccountConn().Email
	}
	accountHealth.Lock()
	out := AccountHealth{
		Email:     email,
		Status:    accountHealth.Status,
		Since:     accountHealth.Since,
		LastOK:    accountHealth.LastOK,
		CheckedAt: accountHealth.CheckedAt,
		Message:   accountHealth.Message,
	}
	accountHealth.Unlock()
	if !userLoggedIn() {
		out.Status = "expired"
		if out.Message == "" {
			out.Message = "账号未登录"
		}
	}
	return out
}

func checkAccountHealth(ctx context.Context) AccountHealth {
	email := UserSessionEmail()
	if email == "" {
		email = CurrentAccountConn().Email
	}
	if email == "" {
		setAccountHealth("unconfigured", "", "", "账号通道未配置")
		return accountHealthSnapshot()
	}
	if !userLoggedIn() {
		setAccountHealth("expired", "", "", "账号未登录")
		return accountHealthSnapshot()
	}
	reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	raw, err := UserCall(reqCtx, http.MethodGet, "/api/client/me", nil)
	if err != nil {
		if strings.Contains(err.Error(), "账号会话失效") {
			clearUserToken(email)
		} else {
			setAccountHealth("unknown", "", "", "健康检查失败: "+err.Error())
		}
		return accountHealthSnapshot()
	}
	if parseAccountProfile(raw).Email == "" {
		setAccountHealth("unknown", "", "", "账号资料响应异常")
		return accountHealthSnapshot()
	}
	markUserSessionOK(email)
	return accountHealthSnapshot()
}

// initUserSessionHealth 启动时恢复落盘 token，并每小时做一次轻量健康检查。
func initUserSessionHealth() {
	loadSavedSessions()
	email := CurrentAccountConn().Email
	if email == "" {
		return
	}
	if token, item, ok := loadUserToken(email); ok {
		SetUserSession(email, token)
		accountHealth.Lock()
		accountHealth.Status = "unknown"
		accountHealth.LastOK = item.LastOK
		accountHealth.Unlock()
	}
	if !userLoggedIn() {
		setAccountHealth("expired", "", "", "账号未登录")
		return
	}
	go func() {
		checkAccountHealth(context.Background())
	}()
	go func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			if userLoggedIn() {
				checkAccountHealth(context.Background())
			}
		}
	}()
}
