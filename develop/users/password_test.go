package users

import "testing"

func TestValidatePassword(t *testing.T) {
	if err := ValidatePassword("1234567"); err == nil {
		t.Fatal("7 位口令应当被拒绝")
	}
	if err := ValidatePassword("12345678"); err != nil {
		t.Fatalf("8 位口令应当通过: %v", err)
	}
	if err := ValidatePassword(string(make([]byte, 73))); err == nil {
		t.Fatal("超过 72 字节应当被拒绝")
	}
}
