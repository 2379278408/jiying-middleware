package core

import "testing"

func TestMaskedSentinel(t *testing.T) {
	cases := map[string]bool{
		"":                       true,
		"wss://********":         true,
		"https://********":       true,
		"wss://********/ws":      true,
		"wss://relay.example/ws": false,
		"not-a-url":              false,
	}
	for in, want := range cases {
		if got := maskedSentinel(in); got != want {
			t.Fatalf("maskedSentinel(%q)=%v want %v", in, got, want)
		}
	}
}

func TestResolveAppWSURLFallsBackOnMask(t *testing.T) {
	setupAccountPoolBucket(t, "app-wsurl-mask")
	saveAppConns([]AppConn{{WSURL: "wss://relay.example/api/app/ws", AppID: "app-1", AppSecret: "s1"}})

	if got := resolveAppWSURL("wss://********", "app-1"); got != "wss://relay.example/api/app/ws" {
		t.Fatalf("掩码应回退到已存地址, got %q", got)
	}
	if got := resolveAppWSURL("", "app-1"); got != "wss://relay.example/api/app/ws" {
		t.Fatalf("空值应回退到已存地址, got %q", got)
	}
	if got := resolveAppWSURL("wss://new.example/ws", "app-1"); got != "wss://new.example/ws" {
		t.Fatalf("真实地址应原样保留, got %q", got)
	}
	if got := resolveAppWSURL("wss://********", "unknown"); got != "" {
		t.Fatalf("未存应用不应编造地址, got %q", got)
	}
}

func TestResolveAccountBaseURLFallsBackOnMask(t *testing.T) {
	setupAccountPoolBucket(t, "acct-baseurl-mask")
	if err := SyncAccountConn(AccountConn{BaseURL: "https://chat.example", Email: "a@example.com", Secret: "s"}); err != nil {
		t.Fatal(err)
	}
	if got := resolveAccountBaseURL("https://********"); got != "https://chat.example" {
		t.Fatalf("掩码应回退到已存地址, got %q", got)
	}
	if got := resolveAccountBaseURL(""); got != "https://chat.example" {
		t.Fatalf("空值应回退到已存地址, got %q", got)
	}
	if got := resolveAccountBaseURL("https://new.example/"); got != "https://new.example" {
		t.Fatalf("真实地址应去掉尾斜杠, got %q", got)
	}
}

func TestStripURLCredentials(t *testing.T) {
	cases := map[string]string{
		"":                                 "",
		"https://chat.example":             "https://chat.example",
		"https://u:p@chat.example:8443/v1": "https://chat.example:8443/v1",
		"http://token@127.0.0.1:20080":     "http://127.0.0.1:20080",
		"not-a-url":                        "not-a-url",
	}
	for in, want := range cases {
		if got := stripURLCredentials(in); got != want {
			t.Fatalf("stripURLCredentials(%q)=%q want %q", in, got, want)
		}
	}
}
