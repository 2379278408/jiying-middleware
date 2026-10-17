package core

import (
	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
)

type Faker struct {
	Message string
	Type    interface{}
	UserID  string
	ChatID  string
	Carry   bool
	BaseSender
}

type Sender interface {
	GetParentID() any
	GetMsg() string
	GetType() interface{}
	SetMsg(string)
	SetParentID(any)
	Send(interface{}) bool
	Reply(interface{}) bool
}

var Senders = make(chan Sender, 100)

var (
	Server       *gin.Engine
	PublicIPv4   string
	LocalIPv4    string
	SessionStore *sessions.CookieStore
	DataHome     string
)

func CheckCookie(c *gin.Context) bool {
	if SessionStore == nil {
		return false
	}
	s, err := SessionStore.Get(c.Request, "jiying_session")
	if err != nil {
		return false
	}
	id, _ := s.Values["user"].(string)
	if id == "" {
		return false
	}
	// 会话代次在改密、禁用、删除用户时递增，旧 Cookie 随即失效。
	var epoch int64
	switch n := s.Values["epoch"].(type) {
	case int64:
		epoch = n
	case int:
		epoch = int64(n)
	case float64:
		epoch = int64(n)
	}
	return epoch == SessionEpoch(id)
}

type BaseSender struct {
	RawMsg      string
	Mtype       int
	Msg         string
	FromUserID  string
	FromGroupID string
	FromChannel int
	IsAdmin     bool
	Toreply     interface{}
	Faker       *Faker
	Time        int64
	MsgID       string
	UserInfo    interface{}
}

func (f *Faker) GetMsg() string {
	return f.Message
}

func (f *Faker) GetType() interface{} {
	return f.Type
}

func (f *Faker) GetParentID() any {
	return f.FromGroupID
}

func (f *Faker) SetMsg(msg string) {
	f.Message = msg
}

func (f *Faker) SetParentID(id any) {
	_ = id
}

func (f *Faker) Send(msg interface{}) bool {
	f.Message = mgr2str(msg)
	return true
}

func (f *Faker) Reply(msg interface{}) bool {
	return f.Send(msg)
}

func mgr2str(msg interface{}) string {
	switch v := msg.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	default:
		return ""
	}
}
