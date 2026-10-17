package core

import (
	"strings"
	"testing"
)

func TestSanitizeFilenameStripsHeaderInjection(t *testing.T) {
	got := sanitizeFilename("a\"\r\nX-Evil: 1\\b.js")
	if strings.ContainsAny(got, "\r\n\"") {
		t.Fatalf("文件名仍含可注入字符: %q", got)
	}
	if strings.ContainsAny(got, `/\\`) {
		t.Fatalf("文件名仍含路径分隔符: %q", got)
	}
	if got == "" || got == "file" {
		t.Fatalf("正常字符应保留: %q", got)
	}
	if sanitizeFilename("...") != "file" {
		t.Fatalf("空文件名应回退: %q", sanitizeFilename("..."))
	}
}

func TestSafeContentType(t *testing.T) {
	cases := map[string]string{
		"image/png":                    "image/png",
		"text/plain; charset=utf-8":    "text/plain",
		"text/plain\r\nX-Inject: 1":    "application/octet-stream",
		"":                             "application/octet-stream",
		"not a type":                   "application/octet-stream",
		"text/html; charset=\"utf-8\"": "text/html",
	}
	for in, want := range cases {
		if got := safeContentType(in); got != want {
			t.Fatalf("safeContentType(%q)=%q want %q", in, got, want)
		}
	}
}

func TestSanitizeLogText(t *testing.T) {
	got := sanitizeLogText("line1\n[[ fake ] 192.0.2.1]\r\nnext")
	if strings.Contains(got, "\n") || strings.Contains(got, "\r") {
		t.Fatalf("日志文本仍含换行: %q", got)
	}
	if strings.Contains(got, "[[") && !strings.Contains(got, "line1 [[") {
		t.Fatalf("换行应被替换为空格: %q", got)
	}
}
