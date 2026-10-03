package core

import (
	"encoding/json"
	"sync"
)

type AppConn struct {
	WSURL     string `json:"ws_url"`
	AppID     string `json:"app_id"`
	AppSecret string `json:"app_secret"`
}

type AccountConn struct {
	BaseURL   string `json:"base_url"`
	Email     string `json:"email"`
	Secret    string `json:"secret"`
	CodeLogin bool   `json:"code_login"`
}

func (a AccountConn) configured() bool {
	return a.BaseURL != "" && a.Email != "" && a.Secret != ""
}

type RelayConn struct {
	App     AppConn     `json:"app"`
	Account AccountConn `json:"account"`
}

func loadRelayConn() RelayConn {
	var cfg RelayConn
	if BucketApp == nil {
		return cfg
	}
	_ = json.Unmarshal([]byte(BucketApp.GetString("relay_conn")), &cfg)
	// 密钥在库中以密文存放，读取后还原为明文供运行时使用。
	cfg.App.AppSecret = openSecret(cfg.App.AppSecret)
	cfg.Account.Secret = openSecret(cfg.Account.Secret)
	return cfg
}

func saveRelayConn(cfg RelayConn) {
	if BucketApp == nil {
		return
	}
	// 调用方传入的始终是明文（load 时已解密），落库前统一加密。
	cfg.App.AppSecret = sealSecret(cfg.App.AppSecret)
	cfg.Account.Secret = sealSecret(cfg.Account.Secret)
	raw, _ := json.Marshal(cfg)
	BucketApp.Set("relay_conn", string(raw))
}

func (c RelayConn) effectiveApp() JiyingConfig {
	if c.App.WSURL != "" && c.App.AppID != "" && c.App.AppSecret != "" {
		return JiyingConfig{WSURL: c.App.WSURL, AppID: c.App.AppID, Secret: c.App.AppSecret}
	}
	return JiyingConfigFromEnv()
}

func (c RelayConn) effectiveAccount() UserLogin {
	if c.Account.BaseURL != "" && c.Account.Email != "" && c.Account.Secret != "" {
		return UserLogin{BaseURL: c.Account.BaseURL, Email: c.Account.Email, Password: c.Account.Secret}
	}
	return UserLoginFromEnv()
}

func CurrentAppConfig() JiyingConfig {
	if a := DefaultAppConn(); a.AppID != "" {
		return JiyingConfig{WSURL: a.WSURL, AppID: a.AppID, Secret: a.AppSecret}
	}
	return loadRelayConn().effectiveApp()
}
func CurrentAccountConfig() UserLogin {
	account := CurrentAccountConn()
	if account.configured() {
		return UserLogin{BaseURL: account.BaseURL, Email: account.Email, Password: account.Secret}
	}
	return loadRelayConn().effectiveAccount()
}

func initAppChannelFromConfig() {
	applyAppConns(loadAppConns())
	conn := loadRelayConn()
	if conn.Account.BaseURL != "" && conn.Account.Email != "" && conn.Account.Secret != "" {
		SetUserSession(conn.Account.Email, "")
	}
	initUserChannel()
}

var connMu sync.Mutex

func SyncAppConn(cfg AppConn) error {
	connMu.Lock()
	defer connMu.Unlock()
	poolMu.Lock()
	defer poolMu.Unlock()
	// 兼容旧的单应用接入：写入连接池并作为首个应用。
	list := loadAppConns()
	for i := range list {
		if list[i].AppID == cfg.AppID {
			list[i] = cfg
			applyAppConns(list)
			return nil
		}
	}
	list = append([]AppConn{cfg}, list...)
	applyAppConns(list)
	return nil
}

func SyncAccountConn(cfg AccountConn) error {
	connMu.Lock()
	defer connMu.Unlock()
	conn := loadRelayConn()
	conn.Account = cfg
	saveRelayConn(conn)
	return UpsertAccountConn(cfg)
}
