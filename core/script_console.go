package core

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"strings"
	"sync"
)

// ScriptConsoleUser 是插件脚本访问本机控制台的专用账号。它只接受由
// scriptConsoleToken 生成的随机口令，避免把即应账号密码或控制台管理员密码
// 暴露给用户可编辑的脚本。
const ScriptConsoleUser = "script"

const scriptConsoleTokenKey = "script_console_token"

var scriptConsoleTokenCache = struct {
	sync.Mutex
	loaded bool
	token  string
}{}

// 脚本登录后只获得 ops 权限，不能读取系统参数或管理用户。

func randomScriptToken() string {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(buf)
}

// scriptConsoleToken 返回插件专用控制台口令。口令首次生成后加密落盘，
// 同一安装内保持稳定；静态加密不可用时退化为进程内口令并在日志中提示。
func scriptConsoleToken() string {
	scriptConsoleTokenCache.Lock()
	defer scriptConsoleTokenCache.Unlock()
	if scriptConsoleTokenCache.loaded {
		return scriptConsoleTokenCache.token
	}
	scriptConsoleTokenCache.loaded = true
	if BucketApp != nil {
		if raw := strings.TrimSpace(BucketApp.GetString(scriptConsoleTokenKey)); raw != "" {
			if opened := strings.TrimSpace(openSecret(raw)); opened != "" {
				scriptConsoleTokenCache.token = opened
				return opened
			}
		}
	}
	token := randomScriptToken()
	if token == "" {
		GetMyLog().Error("插件控制台口令生成失败")
		return ""
	}
	scriptConsoleTokenCache.token = token
	if BucketApp != nil {
		sealed, err := sealSecret(token)
		if err != nil {
			GetMyLog().Warn("插件控制台口令无法加密落盘，本次运行使用临时口令: %v", err)
		} else if !BucketApp.Set(scriptConsoleTokenKey, sealed) {
			GetMyLog().Warn("插件控制台口令落盘失败")
		}
	}
	return token
}

// ValidateScriptConsolePassword 校验插件专用口令，使用常量时间比较避免时序泄漏。
func ValidateScriptConsolePassword(password string) bool {
	expected := scriptConsoleToken()
	if expected == "" || password == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(expected), []byte(password)) == 1
}

// resetScriptConsoleTokenForTest 清理内存缓存，测试切换存储后重新读取。
func resetScriptConsoleTokenForTest() {
	scriptConsoleTokenCache.Lock()
	scriptConsoleTokenCache.loaded = false
	scriptConsoleTokenCache.token = ""
	scriptConsoleTokenCache.Unlock()
}
