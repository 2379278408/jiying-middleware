package core

import "testing"

func TestParseAccountProfileNestedUser(t *testing.T) {
	raw := []byte(`{"data":{"user":{"id":"u-1","email":"a@b.c","name":"Alice","nickname":"","avatar":"https://cdn.example.com/a.png"}}}`)
	got := parseAccountProfile(raw)
	if got.ID != "u-1" || got.Email != "a@b.c" || got.Nickname != "Alice" || got.AvatarURL != "https://cdn.example.com/a.png" {
		t.Fatalf("unexpected profile: %+v", got)
	}
}

func TestParseAccountProfilePrefersNickname(t *testing.T) {
	raw := []byte(`{"data":{"user":{"email":"a@b.c","name":"Alice","nickname":"Ali"}}}`)
	if got := parseAccountProfile(raw); got.Nickname != "Ali" {
		t.Fatalf("nickname should win over name: %+v", got)
	}
}

func TestParseAccountProfileIgnoresFlatShape(t *testing.T) {
	raw := []byte(`{"data":{"email":"flat@b.c","nickname":"x"}}`)
	if got := parseAccountProfile(raw); got.Email != "" {
		t.Fatalf("flat data.* must not be read as profile: %+v", got)
	}
}
