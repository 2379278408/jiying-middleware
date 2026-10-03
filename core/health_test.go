package core

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthzIsAnonymous(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	Server.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("healthz want 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"ok"`) {
		t.Fatalf("healthz body=%s", rec.Body.String())
	}
}

func TestMetricsIsAnonymousText(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	Server.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("metrics want 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{"jiying_uptime_seconds", "jiying_tasks{status=", "jiying_channel_up{channel="} {
		if !strings.Contains(body, want) {
			t.Fatalf("metrics 缺少 %s:\n%s", want, body)
		}
	}
}

func TestReadyzReportsNotReadyWithoutStorage(t *testing.T) {
	prev := BucketApp
	BucketApp = nil
	defer func() { BucketApp = prev }()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	Server.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("readyz without storage want 503, got %d (%s)", rec.Code, rec.Body.String())
	}
}
