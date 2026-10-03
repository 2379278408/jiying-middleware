package core

import (
	"encoding/json"
	"testing"
)

func TestMediaMessageBuilders(t *testing.T) {
	img := ImageMessage("f1", "hi")
	if img["type"] != "image" || img["file_id"] != "f1" || img["caption"] != "hi" {
		t.Fatalf("image message mismatch: %+v", img)
	}
	if _, ok := ImageMessage("f1", "")["caption"]; ok {
		t.Fatal("empty caption must be omitted")
	}
	file := FileMessage("f2", "a.pdf", 10)
	if file["type"] != "file" || file["size_bytes"] != int64(10) {
		t.Fatalf("file message mismatch: %+v", file)
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
