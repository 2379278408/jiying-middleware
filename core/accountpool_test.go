package core

import (
	"bytes"
	"testing"
)

func setupAccountPoolBucket(t *testing.T, name string) {
	t.Helper()
	t.Setenv("JIYING_SECRET_KEY", "00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff")
	t.Setenv("JIYING_DATA", t.TempDir())
	b, err := MakeBucket(name, 6)
	if err != nil {
		t.Fatal(err)
	}
	BucketApp = b
	BucketApp.Delete(accountPoolKey)
	BucketApp.Delete(activeAccountEmailKey)
	BucketApp.Delete("relay_conn")
	t.Cleanup(func() {
		BucketApp.Delete(accountPoolKey)
		BucketApp.Delete(activeAccountEmailKey)
		BucketApp.Delete("relay_conn")
	})
}

func TestAccountPoolRoundTripAndSelection(t *testing.T) {
	setupAccountPoolBucket(t, "accountpool-roundtrip")
	accounts := []AccountConn{
		{BaseURL: "https://chat.example", Email: "a@example.com", Secret: "secret-a"},
		{BaseURL: "https://chat.example", Email: "b@example.com", Secret: "secret-b"},
	}
	saveAccountConns(accounts)
	setActiveAccountEmail("b@example.com")

	raw := BucketApp.GetString(accountPoolKey)
	if bytes.Contains([]byte(raw), []byte("secret-a")) || bytes.Contains([]byte(raw), []byte("secret-b")) {
		t.Fatalf("account secrets must be sealed at rest: %s", raw)
	}
	if got := CurrentAccountConn().Email; got != "b@example.com" {
		t.Fatalf("active account = %q, want b@example.com", got)
	}
	if _, err := SelectAccountConn("missing@example.com"); err == nil {
		t.Fatal("missing account must be rejected")
	}
}

func TestAccountPoolFallsBackToLegacyAccount(t *testing.T) {
	setupAccountPoolBucket(t, "accountpool-legacy")
	BucketApp.Set("relay_conn", `{"account":{"base_url":"https://chat.example","email":"legacy@example.com","secret":"legacy-secret"}}`)
	list := loadAccountConns()
	if len(list) != 1 || list[0].Email != "legacy@example.com" || list[0].Secret != "legacy-secret" {
		t.Fatalf("legacy account fallback mismatch: %+v", list)
	}
}
