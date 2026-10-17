package core

import (
	"context"
	"encoding/json"
	"testing"
)

func TestMediaMessageBuilders(t *testing.T) {
	img := ImageMessage("https://cdn.example.com/a.png", "hi", "")
	if img["type"] != "image" || img["content"] != "https://cdn.example.com/a.png" || img["caption"] != "hi" {
		t.Fatalf("image message mismatch: %+v", img)
	}
	if img["caption_type"] != "text" {
		t.Fatalf("caption_type should default to text, got %v", img["caption_type"])
	}
	if ImageMessage("https://cdn.example.com/a.png", "md", "markdown")["caption_type"] != "markdown" {
		t.Fatal("markdown caption_type must be preserved")
	}
	if _, ok := ImageMessage("https://cdn.example.com/a.png", "", "")["caption"]; ok {
		t.Fatal("empty caption must be omitted")
	}
	file := FileMessage("https://cdn.example.com/a.pdf", "a.pdf", 10)
	if file["type"] != "file" || file["url"] != "https://cdn.example.com/a.pdf" || file["size_bytes"] != int64(10) {
		t.Fatalf("file message mismatch: %+v", file)
	}
	if _, ok := file["content"]; ok {
		t.Fatal("file message must not put remote URL in content (server treats content as inline text)")
	}
	if got := MarkdownMessage("**hi**")["type"]; got != "markdown" {
		t.Fatalf("markdown type mismatch: %v", got)
	}
	if CardMessage("t", "d", "")["url"] != nil {
		t.Fatal("empty card url must be omitted")
	}
	if got := LinkMessage("u", "t")["type"]; got != "link" {
		t.Fatalf("link type mismatch: %v", got)
	}
	chart := ChartMessage("", "t", "d", map[string]any{"labels": []string{"a"}})
	if chart["chart_type"] != "bar" {
		t.Fatalf("empty chart type should default to bar, got %v", chart["chart_type"])
	}
}

func TestResolveMediaContentPassthroughURL(t *testing.T) {
	got, err := resolveMediaContent(context.Background(), "conv1", "", "https://cdn.example.com/x.png")
	if err != nil {
		t.Fatal(err)
	}
	if got != "https://cdn.example.com/x.png" {
		t.Fatalf("absolute URL must pass through, got %q", got)
	}
	if _, err := resolveMediaContent(context.Background(), "conv1", "", "   "); err == nil {
		t.Fatal("blank source must error")
	}
}

func TestNormalizeFeatureResponsePassthrough(t *testing.T) {
	raw := []byte(`{"ok":true,"code":0,"data":{"x":1}}`)
	out := normalizeFeatureResponse(raw, "trace-1", nil)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed["trace_id"] != "trace-1" {
		t.Fatalf("trace_id must be attached, got %v", parsed["trace_id"])
	}
	if parsed["data"] == nil {
		t.Fatal("data must be preserved")
	}
}

func TestNormalizeFeatureResponseWrapsPlain(t *testing.T) {
	out := normalizeFeatureResponse([]byte(`{"answer":42}`), "trace-2", nil)
	var parsed struct {
		OK      bool           `json:"ok"`
		TraceID string         `json:"trace_id"`
		Data    map[string]any `json:"data"`
	}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatal(err)
	}
	if !parsed.OK || parsed.TraceID != "trace-2" {
		t.Fatalf("envelope mismatch: %+v", parsed)
	}
	if parsed.Data["answer"] != float64(42) {
		t.Fatalf("plain payload must move into data, got %+v", parsed.Data)
	}
}

func TestNormalizeFeatureResponseError(t *testing.T) {
	out := normalizeFeatureResponse(nil, "trace-3", errTestBoom)
	var parsed struct {
		OK      bool   `json:"ok"`
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed.OK || parsed.Code != 1 || parsed.Message == "" {
		t.Fatalf("error envelope mismatch: %+v", parsed)
	}
}

type testBoom struct{}

func (testBoom) Error() string { return "boom" }

var errTestBoom = testBoom{}

func TestChoiceOptionsFromJS(t *testing.T) {
	opts := choiceOptionsFromJS([]any{
		map[string]any{"id": "yes", "label": "确认"},
		map[string]any{"id": "no"},
		map[string]any{"label": "缺少 ID"},
		"not-an-object",
	})
	if len(opts) != 2 {
		t.Fatalf("unexpected option count: %#v", opts)
	}
	if opts[0]["id"] != "yes" || opts[0]["label"] != "确认" {
		t.Fatalf("first option mismatch: %#v", opts[0])
	}
	if opts[1]["id"] != "no" || opts[1]["label"] != "no" {
		t.Fatalf("label should fall back to id: %#v", opts[1])
	}
}
