package gin

import "testing"

func TestSafeJoinUnderRejectsTraversal(t *testing.T) {
	root := "/srv/jiying/scripts"
	good := map[string]string{
		"":            "/srv/jiying/scripts",
		"/":           "/srv/jiying/scripts",
		"sub":         "/srv/jiying/scripts/sub",
		"sub/nested/": "/srv/jiying/scripts/sub/nested",
		"./sub":       "/srv/jiying/scripts/sub",
	}
	for in, want := range good {
		got, ok := safeJoinUnder(root, in)
		if !ok || got != want {
			t.Fatalf("safeJoinUnder(%q)=%q,%v want %q", in, got, ok, want)
		}
	}
	bad := []string{"../etc", "sub/../../etc", "..", "a/../../b", "dir\\..\\..\\x", "a\x00b"}
	for _, in := range bad {
		if got, ok := safeJoinUnder(root, in); ok {
			t.Fatalf("应拒绝路径 %q，得到 %q", in, got)
		}
	}
}

func TestSafeUploadName(t *testing.T) {
	if name, ok := safeUploadName("../../evil.js", ".js"); !ok || name != "evil.js" {
		t.Fatalf("应只保留文件名: %q %v", name, ok)
	}
	if name, ok := safeUploadName("a/b.js", ".js"); !ok || name != "b.js" {
		t.Fatalf("带目录的文件名应只保留 basename: %q %v", name, ok)
	}
	for _, in := range []string{"", ".", "..", "evil.sh", "evil", "evil.js.php"} {
		if _, ok := safeUploadName(in, ".js"); ok {
			t.Fatalf("应拒绝文件名 %q", in)
		}
	}
}
