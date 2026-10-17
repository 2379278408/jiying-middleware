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

func TestRankKeysRequired(t *testing.T) {
	t.Setenv("JIYING_RANK3_KEY", "")
	t.Setenv("JIYING_RANK4_KEY", "")
	if EncodeRank3("x") != "" || DecodeRank3("00") != "" || EncodeRank4("x") != "" || DecodeRank4("00") != "" {
		t.Fatal("未配置环境密钥时必须失败关闭")
	}
}

func TestSm4RejectsLegacyDeterministicIV(t *testing.T) {
	key := []byte("A7*&^%$#@!5aS2*&^%$#@!3s")
	if got := Sm4DecryptCbc(key, make([]byte, 32)); got != nil {
		t.Fatalf("缺少版本头的旧密文必须被拒绝: %q", got)
	}
}

func TestSm4RandomIV(t *testing.T) {
	key := []byte("0123456789abcdef")
	plain := []byte("same-plaintext")
	a := Sm4EncryptCbc(key, plain)
	b := Sm4EncryptCbc(key, plain)
	if len(a) == 0 || string(a) == string(b) {
		t.Fatal("相同明文两次加密结果不应相同")
	}
	if string(Sm4DecryptCbc(key, a)) != string(plain) || string(Sm4DecryptCbc(key, b)) != string(plain) {
		t.Fatal("随机 IV 密文无法还原")
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
	t.Setenv("JIYING_RANK3_KEY", "0123456789abcdef0123456789abcdef")
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
	t.Setenv("JIYING_RANK4_KEY", "fedcba9876543210fedcba9876543210")
	src := "9f86d081884c7d65"
	enc := EncodeRank4(src)
	if enc == "" {
		t.Fatal("EncodeRank4 empty")
	}
	if dec := DecodeRank4(enc); dec != src {
		t.Fatalf("Rank4 round-trip failed: %q", dec)
	}
}
