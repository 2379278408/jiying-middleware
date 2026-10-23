package core

import "testing"

func TestApplyYamlConf(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("JIYING_DATA", dir)
	BucketApp, _ = MakeBucket("config-test", 6)
	BucketApp.Set("port", 9999)
	applyYamlConf("http:\n  port: 8081\n  certFile: /tmp/a.pem\n  keyFile: /tmp/a.key\n  local_host: 127.0.0.1\nsystem:\n  AutoCheck: false\n")
	if got := BucketApp.GetInt("port"); got != 8081 {
		t.Fatalf("port=%d", got)
	}
	if got := BucketApp.GetString("certFile"); got != "/tmp/a.pem" {
		t.Fatalf("certFile=%q", got)
	}
	if BucketApp.GetBool("AutoCheck") {
		t.Fatal("AutoCheck 应被配置为 false")
	}
	applyYamlConf("::not yaml::")
	if got := BucketApp.GetInt("port"); got != 8081 {
		t.Fatal("解析失败时不应覆盖已有配置")
	}
}
