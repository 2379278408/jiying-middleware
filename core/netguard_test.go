package core

import (
	"strings"
	"testing"
)

func TestValidateOutboundURLBlocksUnsafeTargets(t *testing.T) {
	bad := []string{
		"http://127.0.0.1:8080",
		"http://localhost:8080",
		"http://[::1]:8080",
		"http://10.0.0.8/api",
		"http://192.168.1.10/api",
		"http://172.16.0.9/api",
		"http://169.254.169.254/latest/meta-data",
		"http://100.100.100.200/latest/meta-data",
		"http://100.64.0.1/api",
		"http://0.0.0.0:8080",
		"https://user:pass@chat.example/api",
		"ftp://chat.example/api",
		"file:///etc/passwd",
		"not-a-url",
		"",
	}
	for _, raw := range bad {
		if err := validateOutboundURL(raw, false); err == nil {
			t.Fatalf("应拒绝地址 %q", raw)
		}
	}
	if err := validateOutboundURL("wss://8.8.8.8/api/app/ws", false); err == nil {
		t.Fatal("非 WebSocket 场景应拒绝 ws 协议")
	}
	if err := validateOutboundURL("wss://8.8.8.8/api/app/ws", true); err != nil {
		t.Fatalf("合法 WebSocket 地址不应被拒绝: %v", err)
	}
	if err := validateOutboundURL("https://8.8.8.8/api", false); err != nil {
		t.Fatalf("合法公网地址不应被拒绝: %v", err)
	}
}

func TestValidateOutboundURLPrivateOverride(t *testing.T) {
	if err := validateOutboundURL("http://10.1.2.3/api", false); err == nil {
		t.Fatal("默认应拒绝私网地址")
	}
	t.Setenv("JIYING_ALLOW_PRIVATE_HOSTS", "1")
	if err := validateOutboundURL("http://10.1.2.3/api", false); err != nil {
		t.Fatalf("开启内网放行后应允许私网地址: %v", err)
	}
	if err := validateOutboundURL("http://169.254.169.254/latest", false); err == nil || !strings.Contains(err.Error(), "元数据") {
		t.Fatalf("元数据地址必须始终拒绝: %v", err)
	}
	if err := validateOutboundURL("http://100.100.100.200/latest", false); err == nil {
		t.Fatal("元数据地址必须始终拒绝")
	}
}

func TestEndpointsEqual(t *testing.T) {
	same := [][2]string{
		{"wss://Relay.Example/api/app/ws", "wss://relay.example/api/app/ws"},
		{"https://chat.example/", "https://chat.example"},
		{"https://chat.example/api/", "https://chat.example/api"},
	}
	for _, pair := range same {
		if !endpointsEqual(pair[0], pair[1]) {
			t.Fatalf("%q 与 %q 应视为同一端点", pair[0], pair[1])
		}
	}
	diff := [][2]string{
		{"wss://relay.example/api/app/ws", "wss://attacker.example/api/app/ws"},
		{"wss://relay.example/api/app/ws", "ws://relay.example/api/app/ws"},
		{"https://chat.example/v1", "https://chat.example/v2"},
		{"https://chat.example/v1?a=1", "https://chat.example/v1?a=2"},
	}
	for _, pair := range diff {
		if endpointsEqual(pair[0], pair[1]) {
			t.Fatalf("%q 与 %q 不应视为同一端点", pair[0], pair[1])
		}
	}
}
