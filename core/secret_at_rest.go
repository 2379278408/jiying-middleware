package core

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// secretEncPrefix 标记已加密的连接密钥；无前缀的值按历史明文处理并在读取后原样可用。
const secretEncPrefix = "enc:v1:"

const relayKeyFile = ".relay_secret_key"

var (
	relayKeyOnce sync.Once
	relayKey     []byte
	relayKeyErr  error
)

// relaySecretKey 返回本地加密密钥：优先取环境变量 JIYING_SECRET_KEY，
// 否则使用数据目录下 0600 的密钥文件，首次使用时生成。
func relaySecretKey() ([]byte, error) {
	relayKeyOnce.Do(func() {
		if v := strings.TrimSpace(os.Getenv("JIYING_SECRET_KEY")); v != "" {
			if raw, err := decodeKey(v); err == nil {
				relayKey = raw
				return
			}
			relayKeyErr = errKeyFormat
			return
		}
		path := filepath.Join(DataHome, relayKeyFile)
		if raw, err := os.ReadFile(path); err == nil {
			if key, err := decodeKey(strings.TrimSpace(string(raw))); err == nil {
				relayKey = key
				return
			}
		}
		key := make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			relayKeyErr = err
			return
		}
		if err := os.MkdirAll(DataHome, 0o700); err != nil {
			relayKeyErr = err
			return
		}
		if err := os.WriteFile(path, []byte(hex.EncodeToString(key)+"\n"), 0o600); err != nil {
			relayKeyErr = err
			return
		}
		relayKey = key
	})
	return relayKey, relayKeyErr
}

func decodeKey(s string) ([]byte, error) {
	if raw, err := hex.DecodeString(s); err == nil && len(raw) == 32 {
		return raw, nil
	}
	if raw, err := base64.StdEncoding.DecodeString(s); err == nil && len(raw) == 32 {
		return raw, nil
	}
	return nil, errKeyFormat
}

var errKeyFormat = errors.New("密钥格式无效，需为 32 字节的 hex 或 base64")

// sealSecret 加密连接密钥；空值保持为空，已加密值幂等返回。
func sealSecret(s string) string {
	if s == "" {
		return s
	}
	key, err := relaySecretKey()
	if err != nil {
		// 无法加密时保留原值，避免丢失配置。
		GetMyLog().Warn("连接密钥加密不可用，暂以明文保存: %v", err)
		return s
	}
	return sealWithKey(key, s)
}

func sealWithKey(key []byte, s string) string {
	if s == "" || strings.HasPrefix(s, secretEncPrefix) {
		return s
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return s
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return s
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return s
	}
	out := gcm.Seal(nonce, nonce, []byte(s), nil)
	return secretEncPrefix + base64.StdEncoding.EncodeToString(out)
}

// openSecret 解密连接密钥；历史明文值原样返回。
func openSecret(s string) string {
	if s == "" || !strings.HasPrefix(s, secretEncPrefix) {
		return s
	}
	key, err := relaySecretKey()
	if err != nil {
		GetMyLog().Warn("连接密钥解密不可用: %v", err)
		return ""
	}
	return openWithKey(key, s)
}

func openWithKey(key []byte, s string) string {
	if s == "" || !strings.HasPrefix(s, secretEncPrefix) {
		return s
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(s, secretEncPrefix))
	if err != nil {
		return ""
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return ""
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return ""
	}
	if len(raw) < gcm.NonceSize() {
		return ""
	}
	plain, err := gcm.Open(nil, raw[:gcm.NonceSize()], raw[gcm.NonceSize():], nil)
	if err != nil {
		GetMyLog().Warn("连接密钥解密失败，请重新配置")
		return ""
	}
	return string(plain)
}
