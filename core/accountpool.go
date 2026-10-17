package core

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
)

const (
	accountPoolKey        = "relay_accounts"
	activeAccountEmailKey = "relay_active_account"
	accountProfilesKey    = "relay_account_profiles"
)

// loadAccountConns 读取账号池；旧版单账号配置会自动作为首个账号迁移使用。
func loadAccountConns() []AccountConn {
	if BucketApp == nil {
		return nil
	}
	var list []AccountConn
	if raw := BucketApp.GetString(accountPoolKey); raw != "" {
		_ = json.Unmarshal([]byte(raw), &list)
	}
	for i := range list {
		list[i].Secret = openSecret(list[i].Secret)
		list[i].BaseURL = strings.TrimRight(strings.TrimSpace(list[i].BaseURL), "/")
		list[i].Email = strings.TrimSpace(list[i].Email)
	}
	if len(list) == 0 {
		legacy := loadRelayConn().Account
		if legacy.configured() {
			list = []AccountConn{legacy}
		}
	}
	return list
}

func saveAccountConns(list []AccountConn) {
	if BucketApp == nil {
		return
	}
	sealed := make([]AccountConn, 0, len(list))
	for _, account := range list {
		value, err := sealSecret(account.Secret)
		if err != nil {
			GetMyLog().Error("账号密钥加密失败，已取消保存: %v", err)
			return
		}
		account.Secret = value
		sealed = append(sealed, account)
	}
	raw, _ := json.Marshal(sealed)
	BucketApp.Set(accountPoolKey, string(raw))
}

func activeAccountEmail() string {
	if BucketApp == nil {
		return ""
	}
	return strings.TrimSpace(BucketApp.GetString(activeAccountEmailKey))
}

func setActiveAccountEmail(email string) {
	if BucketApp != nil {
		BucketApp.Set(activeAccountEmailKey, strings.TrimSpace(email))
	}
}

func CurrentAccountConn() AccountConn {
	list := loadAccountConns()
	active := activeAccountEmail()
	for _, account := range list {
		if account.Email == active {
			return account
		}
	}
	if len(list) > 0 {
		return list[0]
	}
	return AccountConn{}
}

type AccountPoolItem struct {
	BaseURL   string `json:"base_url"`
	Email     string `json:"email"`
	CodeLogin bool   `json:"code_login"`
	Active    bool   `json:"active"`
	LoggedIn  bool   `json:"logged_in"`
	Nickname  string `json:"nickname,omitempty"`
	AvatarURL string `json:"avatar_url,omitempty"`
}

// accountProfileStore 按邮箱缓存账号昵称与头像，供多账号列表稳定展示。
// 资料只在登录或状态轮询拿到 /api/client/me 时写入，不额外按账号发起请求。
var accountProfileStore = struct {
	sync.Mutex
	loaded bool
	items  map[string]AccountProfile
}{items: map[string]AccountProfile{}}

func loadAccountProfilesLocked() {
	if accountProfileStore.loaded {
		return
	}
	accountProfileStore.loaded = true
	if BucketApp == nil {
		return
	}
	raw := BucketApp.GetString(accountProfilesKey)
	if raw == "" {
		return
	}
	var items map[string]AccountProfile
	if json.Unmarshal([]byte(raw), &items) != nil {
		return
	}
	for email, p := range items {
		email = strings.TrimSpace(email)
		if email == "" {
			continue
		}
		p.Email = firstNonEmpty(p.Email, email)
		accountProfileStore.items[email] = p
	}
}

// SaveAccountProfile 持久化单账号资料；内容未变化时不重复写盘。
func SaveAccountProfile(profile AccountProfile) {
	email := strings.TrimSpace(firstNonEmpty(profile.Email, UserSessionEmail()))
	if email == "" {
		return
	}
	profile.Email = email
	accountProfileStore.Lock()
	defer accountProfileStore.Unlock()
	loadAccountProfilesLocked()
	if old, ok := accountProfileStore.items[email]; ok && old.Nickname == profile.Nickname && old.AvatarURL == profile.AvatarURL {
		return
	}
	accountProfileStore.items[email] = profile
	if BucketApp == nil {
		return
	}
	raw, err := json.Marshal(accountProfileStore.items)
	if err != nil {
		return
	}
	BucketApp.Set(accountProfilesKey, string(raw))
}

// SavedAccountProfile 读取已缓存的账号资料；未登录过的新账号返回零值。
func SavedAccountProfile(email string) AccountProfile {
	accountProfileStore.Lock()
	defer accountProfileStore.Unlock()
	loadAccountProfilesLocked()
	return accountProfileStore.items[strings.TrimSpace(email)]
}

func AccountPoolSnapshot() []AccountPoolItem {
	list := loadAccountConns()
	active := CurrentAccountConn().Email
	loggedEmail := ""
	if userLoggedIn() {
		loggedEmail = strings.TrimSpace(UserSessionEmail())
	}
	out := make([]AccountPoolItem, 0, len(list))
	for _, account := range list {
		profile := SavedAccountProfile(account.Email)
		out = append(out, AccountPoolItem{
			BaseURL: stripURLCredentials(account.BaseURL), Email: account.Email, CodeLogin: account.CodeLogin,
			Active: account.Email == active, LoggedIn: account.Email != "" && account.Email == loggedEmail,
			Nickname: profile.Nickname, AvatarURL: profile.AvatarURL,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Active != out[j].Active {
			return out[i].Active
		}
		return out[i].Email < out[j].Email
	})
	return out
}

func UpsertAccountConn(cfg AccountConn) error {
	cfg.BaseURL = strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	cfg.Email = strings.TrimSpace(cfg.Email)
	if !cfg.configured() {
		return fmt.Errorf("缺少账号地址 / 邮箱 / 密码")
	}
	poolMu.Lock()
	defer poolMu.Unlock()
	list := loadAccountConns()
	replaced := false
	for i := range list {
		if list[i].Email == cfg.Email {
			list[i] = cfg
			replaced = true
			break
		}
	}
	if !replaced {
		list = append(list, cfg)
	}
	saveAccountConns(list)
	setActiveAccountEmail(cfg.Email)
	return nil
}

func SelectAccountConn(email string) (AccountConn, error) {
	email = strings.TrimSpace(email)
	if email == "" {
		return AccountConn{}, fmt.Errorf("缺少账号")
	}
	for _, account := range loadAccountConns() {
		if account.Email == email {
			return account, nil
		}
	}
	return AccountConn{}, fmt.Errorf("账号不存在")
}

func RemoveAccountConn(email string) error {
	email = strings.TrimSpace(email)
	poolMu.Lock()
	defer poolMu.Unlock()
	list := loadAccountConns()
	kept := make([]AccountConn, 0, len(list))
	found := false
	for _, account := range list {
		if account.Email == email {
			found = true
			continue
		}
		kept = append(kept, account)
	}
	if !found {
		return fmt.Errorf("账号不存在")
	}
	saveAccountConns(kept)
	if activeAccountEmail() == email {
		if len(kept) > 0 {
			setActiveAccountEmail(kept[0].Email)
		} else {
			setActiveAccountEmail("")
		}
		SetUserSession("", "")
	}
	return nil
}
