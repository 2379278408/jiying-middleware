package core

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// API Key 让外部 AI / 脚本以 Bearer 令牌调用控制台 REST 接口，无需浏览器会话。
// 令牌只以 SHA-256 摘要落盘，创建时返回一次明文；权限通过 scope 映射到既有角色。
const (
	apiKeysKey   = "relay_api_keys"
	apiKeyPrefix = "jy_"
)

// APIKey 是对外展示的密钥元数据，不包含摘要或明文。
type APIKey struct {
	ID         string `json:"id"`
	Label      string `json:"label"`
	Prefix     string `json:"prefix"`
	Scope      string `json:"scope"`
	CreatedAt  string `json:"created_at"`
	LastUsedAt string `json:"last_used_at,omitempty"`
	ExpiresAt  string `json:"expires_at,omitempty"`
}

type savedAPIKey struct {
	APIKey
	Hash string `json:"hash"`
}

var apiKeysMu sync.Mutex

// apiKeyUsage 记录每把钥匙的最近使用时间，限制落盘频率。
var apiKeyUsage = struct {
	sync.Mutex
	lastSave map[string]time.Time
}{lastSave: map[string]time.Time{}}

// apiKeyWindows 是简单的固定窗口限流，避免单把钥匙打满控制台。
var apiKeyWindows = struct {
	sync.Mutex
	windows map[string]*apiKeyWindow
}{windows: map[string]*apiKeyWindow{}}

type apiKeyWindow struct {
	start time.Time
	count int
}

const (
	apiKeyRateLimit  = 180
	apiKeyRateWindow = time.Minute
)

func loadAPIKeys() map[string]savedAPIKey {
	out := map[string]savedAPIKey{}
	if BucketApp == nil {
		return out
	}
	raw := BucketApp.GetString(apiKeysKey)
	if raw == "" {
		return out
	}
	_ = json.Unmarshal([]byte(raw), &out)
	return out
}

func saveAPIKeys(items map[string]savedAPIKey) {
	if BucketApp == nil {
		return
	}
	raw, _ := json.Marshal(items)
	BucketApp.Set(apiKeysKey, string(raw))
}

func hashAPIKey(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func randomHex(n int) string {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}

// CreateAPIKey 生成新密钥并返回一次明文。expiresDays <= 0 表示长期有效。
func CreateAPIKey(label, scope string, expiresDays int) (string, APIKey, error) {
	label = strings.TrimSpace(label)
	if label == "" {
		return "", APIKey{}, fmt.Errorf("请填写密钥备注")
	}
	if len([]rune(label)) > 40 {
		label = string([]rune(label)[:40])
	}
	if _, ok := roleRank[scope]; !ok || scope == "" {
		return "", APIKey{}, fmt.Errorf("权限范围只能是 readonly / ops / master")
	}
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", APIKey{}, fmt.Errorf("生成密钥失败")
	}
	token := apiKeyPrefix + base64.RawURLEncoding.EncodeToString(buf)
	id := randomHex(8)
	now := time.Now()
	item := savedAPIKey{
		APIKey: APIKey{
			ID:        id,
			Label:     label,
			Prefix:    token[:11],
			Scope:     scope,
			CreatedAt: now.Format(time.RFC3339),
		},
		Hash: hashAPIKey(token),
	}
	if expiresDays > 0 {
		item.ExpiresAt = now.AddDate(0, 0, expiresDays).Format(time.RFC3339)
	}
	apiKeysMu.Lock()
	items := loadAPIKeys()
	items[id] = item
	saveAPIKeys(items)
	apiKeysMu.Unlock()
	return token, item.APIKey, nil
}

// ListAPIKeys 返回全部密钥元数据，按创建时间倒序。
func ListAPIKeys() []APIKey {
	apiKeysMu.Lock()
	defer apiKeysMu.Unlock()
	items := loadAPIKeys()
	out := make([]APIKey, 0, len(items))
	for _, item := range items {
		out = append(out, item.APIKey)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt > out[j].CreatedAt })
	return out
}

// DeleteAPIKey 立即撤销指定密钥。
func DeleteAPIKey(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("缺少密钥 ID")
	}
	apiKeysMu.Lock()
	items := loadAPIKeys()
	if _, ok := items[id]; !ok {
		apiKeysMu.Unlock()
		return fmt.Errorf("密钥不存在")
	}
	delete(items, id)
	saveAPIKeys(items)
	apiKeysMu.Unlock()
	apiKeyUsage.Lock()
	delete(apiKeyUsage.lastSave, id)
	apiKeyUsage.Unlock()
	apiKeyWindows.Lock()
	delete(apiKeyWindows.windows, id)
	apiKeyWindows.Unlock()
	return nil
}

// verifyAPIKey 校验令牌并返回对应记录；失败返回 nil。
func verifyAPIKey(token string) *savedAPIKey {
	token = strings.TrimSpace(token)
	if !strings.HasPrefix(token, apiKeyPrefix) {
		return nil
	}
	hash := hashAPIKey(token)
	apiKeysMu.Lock()
	defer apiKeysMu.Unlock()
	for id, item := range loadAPIKeys() {
		if subtle.ConstantTimeCompare([]byte(item.Hash), []byte(hash)) != 1 {
			continue
		}
		if item.ExpiresAt != "" {
			if exp, err := time.Parse(time.RFC3339, item.ExpiresAt); err == nil && time.Now().After(exp) {
				return nil
			}
		}
		// 使用时间最多每分钟落盘一次，避免高频读取放大写入。
		apiKeyUsage.Lock()
		last := apiKeyUsage.lastSave[id]
		touch := time.Since(last) >= time.Minute
		if touch {
			apiKeyUsage.lastSave[id] = time.Now()
		}
		apiKeyUsage.Unlock()
		if touch {
			items := loadAPIKeys()
			if cur, ok := items[id]; ok {
				cur.LastUsedAt = time.Now().Format(time.RFC3339)
				items[id] = cur
				saveAPIKeys(items)
			}
		}
		item.LastUsedAt = time.Now().Format(time.RFC3339)
		return &item
	}
	return nil
}

// allowAPIKeyRequest 对单把钥匙做固定窗口限流。
func allowAPIKeyRequest(id string) bool {
	now := time.Now()
	apiKeyWindows.Lock()
	defer apiKeyWindows.Unlock()
	w := apiKeyWindows.windows[id]
	if w == nil || now.Sub(w.start) >= apiKeyRateWindow {
		apiKeyWindows.windows[id] = &apiKeyWindow{start: now, count: 1}
		return true
	}
	if w.count >= apiKeyRateLimit {
		return false
	}
	w.count++
	return true
}

// apiKeyContext 保存本次请求的密钥身份，供 consoleUser 与 requireRole 使用。
const (
	ctxAPIKeyID    = "jiying_api_key_id"
	ctxAPIKeyLabel = "jiying_api_key_label"
	ctxAPIKeyScope = "jiying_api_key_scope"
	ctxAPIKeyUser  = "jiying_api_key_user"
)

// authenticateAPIKey 解析 Authorization: Bearer，成功时写入请求上下文。
func authenticateAPIKey(c *gin.Context) bool {
	header := strings.TrimSpace(c.GetHeader("Authorization"))
	if header == "" || !strings.HasPrefix(strings.ToLower(header), "bearer ") {
		return false
	}
	token := strings.TrimSpace(header[len("bearer "):])
	item := verifyAPIKey(token)
	if item == nil {
		c.AbortWithStatusJSON(401, gin.H{"code": 401, "message": "API Key 无效或已过期"})
		return true
	}
	if !allowAPIKeyRequest(item.ID) {
		c.AbortWithStatusJSON(429, gin.H{"code": 429, "message": "请求过于频繁，请稍后重试"})
		return true
	}
	c.Set(ctxAPIKeyID, item.ID)
	c.Set(ctxAPIKeyLabel, item.Label)
	c.Set(ctxAPIKeyScope, item.Scope)
	c.Set(ctxAPIKeyUser, "key:"+item.Label+"("+item.Prefix+")")
	return true
}

// requireConsoleSession 限制敏感操作只能由浏览器会话发起，密钥不能自我复制或提权。
func requireConsoleSession(c *gin.Context) {
	if _, ok := c.Get(ctxAPIKeyID); ok {
		c.AbortWithStatusJSON(403, gin.H{"code": 403, "message": "该操作需要在控制台登录后执行"})
		return
	}
	c.Next()
}

func registerAPIKeysAPI() {
	g := Server.Group("/jiying/api", requireLogin)
	g.GET("/keys", requireRole(RoleMaster), requireConsoleSession, func(c *gin.Context) {
		jiyingJSON(c, gin.H{"keys": ListAPIKeys()})
	})
	g.POST("/keys", requireRole(RoleMaster), requireConsoleSession, func(c *gin.Context) {
		days := 30
		if v := strings.TrimSpace(c.PostForm("expires_days")); v != "" {
			n, err := strconv.Atoi(v)
			if err != nil || n < 0 || n > 3650 {
				jiyingErr(c, "有效期格式不正确")
				return
			}
			days = n
		}
		token, item, err := CreateAPIKey(c.PostForm("label"), c.PostForm("scope"), days)
		if err != nil {
			jiyingErr(c, err.Error())
			return
		}
		RecordAudit(consoleUser(c), "key.create", item.Label, item.Scope, "成功")
		jiyingJSON(c, gin.H{"token": token, "key": item})
	})
	g.POST("/keys/delete", requireRole(RoleMaster), requireConsoleSession, func(c *gin.Context) {
		id := strings.TrimSpace(c.PostForm("id"))
		if err := DeleteAPIKey(id); err != nil {
			jiyingErr(c, err.Error())
			return
		}
		RecordAudit(consoleUser(c), "key.delete", id, "", "成功")
		jiyingJSON(c, "ok")
	})
	g.GET("/ai/manifest", aiManifest)
}
