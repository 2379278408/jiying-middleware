package utils

import "testing"

func TestAesRoundTrip(t *testing.T) {
	key := "testKey123"
	plain := "http://127.0.0.1:8080/login"
	enc := EncryptByAes(key, plain)
	dec := DecryptByAes(key, enc)
	if dec != plain {
		t.Fatalf("AES round-trip failed: got %q", dec)
	}
}

func TestAesKeyPad(t *testing.T) {
	for _, k := range []string{"a", "1234567890", "123456789012345", "12345678901234567890123456", "123456789012345678901234567890123456"} {
		plain := "abc中文"
		enc := EncryptByAes(k, plain)
		if dec := DecryptByAes(k, enc); dec != plain {
			t.Fatalf("AES pad round-trip failed for key %q: %q", k, dec)
		}
	}
}

func TestSm4RoundTrip(t *testing.T) {
	key := []byte("A7*&^%$#@!5aS2*&^%$#@!3s")
	plain := "https://example.com/demo"
	enc := Sm4EncryptCbc(key, []byte(plain))
	dec := Sm4DecryptCbc(key, enc)
	if string(dec) != plain {
		t.Fatalf("SM4 round-trip failed: %q", string(dec))
	}
}

func TestRank3RoundTrip(t *testing.T) {
	src := `{"machineId":"abcdef","dueTime":"2099-01-01","beforeSecret":"x"}`
	enc := EncodeRank3(src)
	if enc == "" {
		t.Fatal("EncodeRank3 empty")
	}
	if dec := DecodeRank3(enc); dec != src {
		t.Fatalf("Rank3 round-trip failed: %q", dec)
	}
}

func TestRank4RoundTrip(t *testing.T) {
	src := "9f86d081884c7d65"
	enc := EncodeRank4(src)
	if enc == "" {
		t.Fatal("EncodeRank4 empty")
	}
	if dec := DecodeRank4(enc); dec != src {
		t.Fatalf("Rank4 round-trip failed: %q", dec)
	}
}
