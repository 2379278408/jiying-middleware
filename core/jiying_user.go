package core

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"os"
	"strings"
	"sync"
	"time"
)

type UserLogin struct {
	BaseURL  string
	Email    string
	Password string
}

func UserLoginFromEnv() UserLogin {
	return UserLogin{
		BaseURL:  strings.TrimRight(strings.TrimSpace(os.Getenv("JIYING_HTTP_URL")), "/"),
		Email:    strings.TrimSpace(os.Getenv("JIYING_USER_EMAIL")),
		Password: os.Getenv("JIYING_USER_PASSWORD"),
	}
}

func (u UserLogin) configured() bool {
	return u.BaseURL != "" && u.Email != "" && u.Password != ""
}

var userSession = struct {
	sync.Mutex
	email  string
	token  string
	client *http.Client
}{}

func SetUserSession(email, token string) {
	userSession.Lock()
	userSession.email = email
	userSession.token = token
	userSession.Unlock()
}

func UserSessionStatus() string {
	userSession.Lock()
	defer userSession.Unlock()
	if userSession.token == "" {
		return "账号未登录"
	}
	return "账号已登录 " + userSession.email
}

func UserSessionEmail() string {
	userSession.Lock()
	defer userSession.Unlock()
	return userSession.email
}

func userLoggedIn() bool {
	userSession.Lock()
	defer userSession.Unlock()
	return userSession.token != ""
}

func loginUser(ctx context.Context, cfg UserLogin) error {
	return loginAccount(ctx, cfg, false)
}

func loginAccount(ctx context.Context, cfg UserLogin, code bool) error {
	if !cfg.configured() {
		return fmt.Errorf("账号通道未配置")
	}
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Timeout: 20 * time.Second, Jar: jar}
	loginBody := map[string]string{"email": cfg.Email}
	path := cfg.BaseURL + "/api/client/auth/login"
	if code {
		loginBody["code"] = cfg.Password
		path = cfg.BaseURL + "/api/client/auth/email-code/login"
	} else {
		loginBody["password"] = cfg.Password
	}
	raw, cookies, err := postJSONClient(ctx, client, path, loginBody)
	if err != nil {
		return err
	}
	var parsed struct {
		Data struct {
			MobileSession string `json:"mobile_session"`
		} `json:"data"`
	}
	_ = json.Unmarshal(raw, &parsed)
	token := parsed.Data.MobileSession
	if token == "" {
		for _, c := range cookies {
			if c.Name == "user_session" && c.Value != "" {
				token = c.Value
			}
		}
	}
	if token == "" {
		return fmt.Errorf("登录未返回会话")
	}
	userSession.Lock()
	userSession.email = cfg.Email
	userSession.token = token
	userSession.client = client
	userSession.Unlock()
	saveUserToken(cfg.Email, token)
	return nil
}

func postJSONClient(ctx context.Context, client *http.Client, url string, body map[string]string) ([]byte, []*http.Cookie, error) {
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Dianbao-Mobile-Session", "1")
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	out, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, nil, err
	}
	if resp.StatusCode >= 300 {
		var env struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		_ = json.Unmarshal(out, &env)
		if env.Error.Message != "" {
			return nil, nil, fmt.Errorf("%s", env.Error.Message)
		}
		return nil, nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return out, resp.Cookies(), nil
}

func RequestLoginCode(ctx context.Context, base, email string) (int, int, error) {
	client := &http.Client{Timeout: 20 * time.Second}
	raw, _, err := postJSONClient(ctx, client, strings.TrimRight(base, "/")+"/api/client/auth/email-code/request", map[string]string{"email": email})
	if err != nil {
		return 0, 0, err
	}
	var parsed struct {
		Data struct {
			ExpiresInSeconds  int `json:"expires_in_seconds"`
			RetryAfterSeconds int `json:"retry_after_seconds"`
		} `json:"data"`
	}
	if json.Unmarshal(raw, &parsed) != nil {
		return 0, 0, fmt.Errorf("验证码请求响应解析失败")
	}
	return parsed.Data.ExpiresInSeconds, parsed.Data.RetryAfterSeconds, nil
}

func UserCall(ctx context.Context, method, path string, body any) ([]byte, error) {
	if !userLoggedIn() {
		account := CurrentAccountConn()
		if account.CodeLogin {
			return nil, errAccountNeedsCode
		}
		cfg := CurrentAccountConfig()
		if err := loginUser(ctx, cfg); err != nil {
			return nil, err
		}
	}
	userSession.Lock()
	token := userSession.token
	base := CurrentAccountConfig().BaseURL
	client := userSession.client
	userSession.Unlock()
	if client == nil {
		client = http.DefaultClient
	}
	var rdr io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		rdr = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, base+path, rdr)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	out, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusUnauthorized {
		userSession.Lock()
		email := userSession.email
		userSession.token = ""
		userSession.Unlock()
		clearUserToken(email)
		return out, fmt.Errorf("账号会话失效")
	}
	if resp.StatusCode >= 300 {
		return out, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return out, nil
}

func initUserChannel() {
	account := CurrentAccountConn()
	if !account.configured() {
		return
	}
	if userLoggedIn() {
		return
	}
	if account.CodeLogin {
		// 验证码账号不能在启动时自动登录，等待面板重新获取验证码。
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		if err := loginUser(ctx, CurrentAccountConfig()); err != nil {
			GetMyLog().Warn("即应账号登录失败")
			return
		}
		GetMyLog().Info("即应账号通道已登录")
	}()
}
