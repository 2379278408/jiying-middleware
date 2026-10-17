package core

import (
	"strconv"
	"strings"
	"sync"
)

// 会话代次用于在改密后让旧会话立即失效。
// 代次保存在应用桶中，值为十进制整数；缺少记录时视为 0。
const sessionEpochPrefix = "session_epoch:"

var sessionEpochMu sync.Mutex

// SessionEpoch 返回用户当前会话代次。
func SessionEpoch(userID string) int64 {
	userID = strings.TrimSpace(userID)
	if userID == "" || BucketApp == nil {
		return 0
	}
	sessionEpochMu.Lock()
	defer sessionEpochMu.Unlock()
	raw := BucketApp.GetString(sessionEpochPrefix + userID)
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || n < 0 {
		return 0
	}
	return n
}

// BumpSessionEpoch 递增用户会话代次，使已签发的旧会话全部失效。
func BumpSessionEpoch(userID string) {
	userID = strings.TrimSpace(userID)
	if userID == "" || BucketApp == nil {
		return
	}
	sessionEpochMu.Lock()
	defer sessionEpochMu.Unlock()
	var next int64 = 1
	if n, err := strconv.ParseInt(BucketApp.GetString(sessionEpochPrefix+userID), 10, 64); err == nil && n >= 0 {
		next = n + 1
	}
	BucketApp.Set(sessionEpochPrefix+userID, strconv.FormatInt(next, 10))
}
