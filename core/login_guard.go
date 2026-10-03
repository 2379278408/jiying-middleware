package core

import (
	"sync"
	"time"
)

// 控制台登录失败锁定：同一来源在窗口内连续失败达到阈值后短时锁定，
// 用于抵御针对管理员账号的密码暴力破解。阈值可按需调整。
const (
	loginMaxFailures = 5
	loginFailWindow  = 10 * time.Minute
	loginLockWindow  = 5 * time.Minute
	loginGuardMaxKey = 4096
)

type loginAttempt struct {
	count int
	first time.Time
	until time.Time
}

var loginGuard = struct {
	mu sync.Mutex
	m  map[string]*loginAttempt
}{m: map[string]*loginAttempt{}}

// LoginBlocked 报告来源是否处于锁定中。
func LoginBlocked(key string) bool {
	if key == "" {
		return false
	}
	loginGuard.mu.Lock()
	defer loginGuard.mu.Unlock()
	a := loginGuard.m[key]
	if a == nil {
		return false
	}
	if a.until.After(time.Now()) {
		return true
	}
	// 锁定期已过：窗口也过期则清理，避免长期占用内存。
	if time.Since(a.first) > loginFailWindow {
		delete(loginGuard.m, key)
	}
	return false
}

// NoteLoginFailure 记录一次失败，达到阈值后开启锁定窗口。
func NoteLoginFailure(key string) {
	if key == "" {
		return
	}
	loginGuard.mu.Lock()
	defer loginGuard.mu.Unlock()
	now := time.Now()
	a := loginGuard.m[key]
	if a == nil || now.Sub(a.first) > loginFailWindow {
		a = &loginAttempt{first: now}
		loginGuard.m[key] = a
	}
	a.count++
	if a.count >= loginMaxFailures {
		a.until = now.Add(loginLockWindow)
	}
	pruneLoginGuardLocked(now)
}

// ResetLoginFailures 在登录成功后清除来源的失败记录。
func ResetLoginFailures(key string) {
	if key == "" {
		return
	}
	loginGuard.mu.Lock()
	defer loginGuard.mu.Unlock()
	delete(loginGuard.m, key)
}

// pruneLoginGuardLocked 在条目过多时清理已过期的记录，防止无界增长。
func pruneLoginGuardLocked(now time.Time) {
	if len(loginGuard.m) <= loginGuardMaxKey {
		return
	}
	for k, a := range loginGuard.m {
		if a == nil || (now.After(a.until) && now.Sub(a.first) > loginFailWindow) {
			delete(loginGuard.m, k)
		}
	}
}
