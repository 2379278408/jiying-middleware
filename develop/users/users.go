package users

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
	"jiying/middleware/core"
)

// BucketName 是保存中间件控制台用户的 bolt 桶名。
const BucketName = "jiyingUsers"

// initialPasswordFile 保存首次生成的管理员口令，权限 0600，管理员首次登录后删除。
const initialPasswordFile = "admin-initial-password.txt"

type User struct {
	ID        string
	Disable   bool
	NickName  string
	Password  string
	Email     string
	ImUsers   map[string]string
	RegDate   time.Time
	Alipay    string
	Code      string
	Point     int64
	SignDate  time.Time
	Charm     int64
	Prize     bool
	PrizeTime time.Time
	Url       string
	Date      time.Time
	Memo      string
	Other     string
}

var (
	Users       []*User
	usersBucket core.Bucket
	bucketOnce  sync.Once

	errEmpty  = errorsNew("用户名不能为空")
	errExists = errorsNew("用户名已存在")
)

func errorsNew(s string) error { return &pkgErr{s} }

type pkgErr struct{ s string }

func (e *pkgErr) Error() string { return e.s }

func ensureBucket() core.Bucket {
	bucketOnce.Do(func() {
		b, err := core.MakeBucket(BucketName, 0)
		if err != nil {
			core.GetMyLog().Error("创建用户桶失败: %v", err)
			return
		}
		usersBucket = b
	})
	return usersBucket
}

// DeleteUser 从持久化桶中删除用户，调用方负责同步内存列表。
func DeleteUser(username string) error {
	if username == "" {
		return errEmpty
	}
	bk := ensureBucket()
	if bk == nil {
		return errorsNew("bucket 不可用")
	}
	bk.Delete(username)
	return nil
}

func NewUser() *User {
	return &User{}
}

func init() {
	loadAll()
	ensureAdmin()
	core.SetSessionUserActive(func(id string) bool {
		u := GetUser(id)
		return u != nil && !u.Disable
	})
}

func loadAll() {
	bk := ensureBucket()
	if bk == nil {
		return
	}
	ks, ok := bk.(interface{ Keys() []string })
	if !ok {
		return
	}
	for _, k := range ks.Keys() {
		data := bk.GetBytes(k)
		var u User
		if json.Unmarshal(data, &u) != nil {
			continue
		}
		if u.ID == "" {
			u.ID = k
		}
		Users = append(Users, &u)
	}
}

func ensureAdmin() {
	if len(Users) > 0 {
		return
	}
	pw := strings.TrimSpace(os.Getenv("JIYING_ADMIN_PASSWORD"))
	generated := false
	if pw == "" {
		pw = randomPassword(16)
		generated = true
	}
	u := NewUser()
	u.ID = "admin"
	_ = u.SetPassword(pw)
	u.NickName = "管理员"
	u.RegDate = time.Now()
	u.SignDate = time.Now()
	Users = append(Users, u)
	_ = u.Save()
	if generated {
		if err := writeInitialPassword(pw); err != nil {
			// 无法落盘时退化为标准错误输出，避免把口令写入可被日志接口读取的日志文件。
			fmt.Fprintln(os.Stderr, "[jiying] 已创建默认管理员 admin，初始密码:", pw, "（请立即登录修改）")
			core.GetMyLog().Warn("已创建默认管理员 admin，初始口令落盘失败(%v)，已输出到标准错误，请立即登录修改", err)
		} else {
			core.GetMyLog().Warn("已创建默认管理员 admin，初始口令已写入 %s（权限 0600），登录后请立即修改并删除该文件", initialPasswordPath())
		}
	} else {
		core.GetMyLog().Warn("已创建默认管理员 admin，密码取自环境变量 JIYING_ADMIN_PASSWORD")
	}
}

func initialPasswordPath() string {
	return filepath.Join(core.DataHome, initialPasswordFile)
}

func writeInitialPassword(pw string) error {
	path := initialPasswordPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(pw+"\n"), 0o600)
}

// ClearInitialPasswordFile 删除一次性引导口令文件，在管理员首次成功登录后调用。
func ClearInitialPasswordFile() {
	_ = os.Remove(initialPasswordPath())
}

func randomPassword(n int) string {
	const alphabet = "abcdefghijkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("jy-%d", time.Now().UnixNano())
	}
	for i := range b {
		b[i] = alphabet[int(b[i])%len(alphabet)]
	}
	return string(b)
}

func GetUser(username string) *User {
	for _, u := range Users {
		if u.ID == username {
			return u
		}
	}
	return nil
}

func Count() int {
	return len(Users)
}

func GetAll() []*User {
	return Users
}

func SetUsers(list []*User) {
	Users = list
}

func GetDisabled() []*User {
	var out []*User
	for _, u := range Users {
		if u.Disable {
			out = append(out, u)
		}
	}
	return out
}

func VerifyPassword(username, password string) *User {
	u := GetUser(username)
	if u == nil {
		return nil
	}
	if u.Disable {
		return nil
	}
	if bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password)) != nil {
		return nil
	}
	return u
}

// ValidatePassword 校验控制台口令的基本强度：至少 8 个字符，且不超过 bcrypt 的 72 字节上限。
func ValidatePassword(plain string) error {
	if len(plain) < 8 {
		return errorsNew("密码至少需要 8 个字符")
	}
	if len(plain) > 72 {
		return errorsNew("密码不能超过 72 个字符")
	}
	return nil
}

// SetPassword 对明文口令做哈希后保存到 User。
func (u *User) SetPassword(plain string) error {
	h, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(h)
	return nil
}

func (u *User) Copy() *User {
	c := *u
	if u.ImUsers != nil {
		c.ImUsers = make(map[string]string, len(u.ImUsers))
		for k, v := range u.ImUsers {
			c.ImUsers[k] = v
		}
	}
	return &c
}

func (u *User) GetID() string {
	return u.ID
}

func (u *User) GetPassword() string {
	return u.Password
}

func (u *User) GetPoints() int64 {
	return u.Point
}

func (u *User) SetPoints(p int64) {
	u.Point = p
}

func (u *User) AddPoints(p int64) {
	u.Point += p
}

func (u *User) Save() error {
	bk := ensureBucket()
	if bk == nil {
		return errorsNew("bucket 不可用")
	}
	u.Date = time.Now()
	if !bk.Set(u.ID, u) {
		return errorsNew("保存失败")
	}
	return nil
}

func (u *User) Register(username, password, email string) error {
	username = strings.TrimSpace(username)
	if username == "" {
		return errEmpty
	}
	if GetUser(username) != nil {
		return errExists
	}
	if err := ValidatePassword(password); err != nil {
		return err
	}
	u.ID = username
	if err := u.SetPassword(password); err != nil {
		return err
	}
	u.Email = email
	u.RegDate = time.Now()
	u.SignDate = time.Now()
	u.Disable = false
	if u.NickName == "" {
		u.NickName = username
	}
	Users = append(Users, u)
	return u.Save()
}

func (u *User) toJson() string {
	b, err := json.Marshal(u)
	if err != nil {
		return "{}"
	}
	return string(b)
}

func (u *User) String() string {
	return u.toJson()
}

func (u *User) MyProfile() interface{} {
	return map[string]interface{}{
		"id":        u.ID,
		"nickname":  u.NickName,
		"email":     u.Email,
		"point":     u.Point,
		"charm":     u.Charm,
		"reg_date":  u.RegDate.Format("2006-01-02 15:04:05"),
		"sign_date": u.SignDate.Format("2006-01-02 15:04:05"),
	}
}
