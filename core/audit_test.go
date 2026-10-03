package core

import (
	"testing"
)

func setupAuditBucket(t *testing.T, name string) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("JIYING_DATA", dir)
	b, err := MakeBucket(name, 6)
	if err != nil {
		t.Fatal(err)
	}
	BucketApp = b
	BucketApp.Delete("jiying_audit")
	t.Cleanup(func() {
		BucketApp.Delete("jiying_audit")
	})
}

func TestAuditRecordAndList(t *testing.T) {
	setupAuditBucket(t, "audit-basic")
	RecordAudit("admin", "feature.toggle", "帮助", "enabled=true", "成功")
	RecordAudit("admin", "connect.app", "app-1", "", "失败: 缺少参数")

	list := ListAudit(10)
	if len(list) != 2 {
		t.Fatalf("want 2 entries, got %d", len(list))
	}
	if list[0].Actor != "admin" || list[0].Action != "feature.toggle" {
		t.Fatalf("unexpected first entry: %+v", list[0])
	}
	if list[1].Result != "失败: 缺少参数" {
		t.Fatalf("unexpected second result: %q", list[1].Result)
	}
	if list[0].Time == "" {
		t.Fatal("time should be filled")
	}
}

func TestAuditUnknownActorAndTruncate(t *testing.T) {
	setupAuditBucket(t, "audit-edge")
	long := ""
	for i := 0; i < 400; i++ {
		long += "x"
	}
	RecordAudit("", "feature.run", "帮助", long, "成功")
	list := ListAudit(10)
	if len(list) != 1 {
		t.Fatalf("want 1 entry, got %d", len(list))
	}
	if list[0].Actor != "unknown" {
		t.Fatalf("empty actor should become unknown, got %q", list[0].Actor)
	}
	if len([]rune(list[0].Detail)) > 203 {
		t.Fatalf("detail should be truncated, got %d runes", len([]rune(list[0].Detail)))
	}
}

func TestAuditKeepsRecentEntries(t *testing.T) {
	setupAuditBucket(t, "audit-cap")
	for i := 0; i < auditKeep+20; i++ {
		RecordAudit("admin", "cron.add", "job", "n", "成功")
	}
	list := ListAudit(auditKeep + 100)
	if len(list) != auditKeep {
		t.Fatalf("want capped at %d, got %d", auditKeep, len(list))
	}
}

func TestAuditSummary(t *testing.T) {
	setupAuditBucket(t, "audit-summary")
	if got := AuditLogSummary(10); got != "没有审计记录" {
		t.Fatalf("empty summary mismatch: %q", got)
	}
	RecordAudit("admin", "script.install", "a.js", "", "成功")
	if got := AuditLogSummary(10); got == "" || got == "没有审计记录" {
		t.Fatalf("summary should contain the entry, got %q", got)
	}
}

func TestAuditURLStripsSecrets(t *testing.T) {
	cases := []struct{ in, want string }{
		{"https://user:pass@api.example.com/hook?token=secret#frag", "https://api.example.com/hook"},
		{"http://127.0.0.1:9000/bucket/key?X-Amz-Signature=abc", "http://127.0.0.1:9000/bucket/key"},
		{"https://api.example.com/plain", "https://api.example.com/plain"},
		{"not-a-url", "not-a-url"},
		{"  ", ""},
	}
	for _, c := range cases {
		if got := auditURL(c.in); got != c.want {
			t.Errorf("auditURL(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
