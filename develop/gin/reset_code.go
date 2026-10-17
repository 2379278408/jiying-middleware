package gin

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"strings"
	"sync"
	"time"
)

const (
	resetTicketTTL      = 10 * time.Minute
	resetTicketAttempts = 5
	resetCookieName     = "jiying_reset"
)

// resetTicket 保存服务端的一次性重置凭据；验证码只以哈希形式驻留内存，
// 会话 Cookie 仅携带不可猜测的票据 ID。
type resetTicket struct {
	UserID   string
	CodeHash [sha256.Size]byte
	Expires  time.Time
	Attempts int
}

var resetTickets = struct {
	sync.Mutex
	m map[string]*resetTicket
}{m: map[string]*resetTicket{}}

func hashResetCode(code string) [sha256.Size]byte {
	return sha256.Sum256([]byte(strings.TrimSpace(code)))
}

func randomTicketID() string {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return ""
	}
	return hex.EncodeToString(buf)
}

// issueResetTicket 为用户创建重置票据并返回票据 ID 与明文验证码。
func issueResetTicket(userID string) (string, string) {
	token := randomTicketID()
	if token == "" {
		return "", ""
	}
	code := genVerifyCode()
	now := time.Now()
	resetTickets.Lock()
	defer resetTickets.Unlock()
	pruneResetTicketsLocked(now)
	resetTickets.m[token] = &resetTicket{
		UserID:   userID,
		CodeHash: hashResetCode(code),
		Expires:  now.Add(resetTicketTTL),
	}
	return token, code
}

// verifyResetTicket 校验票据与验证码；失败累计次数，达到上限即销毁票据。
func verifyResetTicket(token, code string) (string, bool) {
	token = strings.TrimSpace(token)
	if token == "" || strings.TrimSpace(code) == "" {
		return "", false
	}
	resetTickets.Lock()
	defer resetTickets.Unlock()
	t := resetTickets.m[token]
	if t == nil || time.Now().After(t.Expires) {
		delete(resetTickets.m, token)
		return "", false
	}
	want := t.CodeHash
	got := hashResetCode(code)
	if subtle.ConstantTimeCompare(got[:], want[:]) != 1 {
		t.Attempts++
		if t.Attempts >= resetTicketAttempts {
			delete(resetTickets.m, token)
		}
		return "", false
	}
	delete(resetTickets.m, token)
	return t.UserID, true
}

func pruneResetTicketsLocked(now time.Time) {
	for k, t := range resetTickets.m {
		if t == nil || now.After(t.Expires) {
			delete(resetTickets.m, k)
		}
	}
}
