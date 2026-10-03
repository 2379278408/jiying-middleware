package core

import (
	"bytes"
	"strings"
	"testing"
)

func TestSealOpenSecretRoundTrip(t *testing.T) {
	key := bytes.Repeat([]byte{0x2a}, 32)
	plain := "example-app-connection-secret"
	sealed := sealWithKey(key, plain)
	if !strings.HasPrefix(sealed, secretEncPrefix) {
		t.Fatalf("sealed missing prefix: %s", sealed)
	}
	if strings.Contains(sealed, plain) {
		t.Fatal("ciphertext leaks plaintext")
	}
	if got := openWithKey(key, sealed); got != plain {
		t.Fatalf("round trip got %q", got)
	}
}

func TestSealSecretIdempotentAndLegacyPlaintext(t *testing.T) {
	key := bytes.Repeat([]byte{0x11}, 32)
	sealed := sealWithKey(key, "s3cr3t")
	if again := sealWithKey(key, sealed); again != sealed {
		t.Fatal("sealing an already-sealed value must be idempotent")
	}
	// 历史明文值：不带前缀，读取时原样返回，无需迁移。
	if got := openWithKey(key, "legacy-plain"); got != "legacy-plain" {
		t.Fatalf("legacy plaintext got %q", got)
	}
}

func TestOpenSecretRejectsTamperedAndWrongKey(t *testing.T) {
	key := bytes.Repeat([]byte{0x33}, 32)
	sealed := sealWithKey(key, "top-secret")
	if got := openWithKey(bytes.Repeat([]byte{0x44}, 32), sealed); got != "" {
		t.Fatalf("wrong key must fail closed, got %q", got)
	}
	if got := openWithKey(key, secretEncPrefix+"bm90LWEtZ2NtLXZhbHVl"); got != "" {
		t.Fatalf("garbage ciphertext must fail closed, got %q", got)
	}
}

func TestSealEmptyStaysEmpty(t *testing.T) {
	key := bytes.Repeat([]byte{0x55}, 32)
	if got := sealWithKey(key, ""); got != "" {
		t.Fatalf("empty got %q", got)
	}
	if got := openWithKey(key, ""); got != "" {
		t.Fatalf("empty got %q", got)
	}
}

func TestRelayConnSecretUpgradeOnSave(t *testing.T) {
	t.Setenv("JIYING_SECRET_KEY", "00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff")
	t.Setenv("JIYING_DATA", t.TempDir())
	BucketApp, _ = MakeBucket("relay-secret-test", 6)
	// 模拟历史版本留下的明文数据。
	BucketApp.Set("relay_conn", `{"app":{"ws_url":"ws://x","app_id":"a","app_secret":"plain-secret"},"account":{"base_url":"","email":"","secret":""}}`)

	if got := loadRelayConn().App.AppSecret; got != "plain-secret" {
		t.Fatalf("legacy plaintext must load as-is, got %q", got)
	}
	saveRelayConn(loadRelayConn())
	raw := BucketApp.GetString("relay_conn")
	if bytes.Contains([]byte(raw), []byte("plain-secret")) {
		t.Fatalf("plaintext must not remain after save: %s", raw)
	}
	if !bytes.Contains([]byte(raw), []byte(secretEncPrefix)) {
		t.Fatalf("secret must be sealed on save: %s", raw)
	}
	if got := loadRelayConn().App.AppSecret; got != "plain-secret" {
		t.Fatalf("decrypt after upgrade got %q", got)
	}
}
