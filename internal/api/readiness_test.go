package api

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestReadinessRequiresStartupCompletion(t *testing.T) {
	server := NewServer(nil, nil, nil, slog.Default())
	res := httptest.NewRecorder()
	server.Routes().ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if res.Code != http.StatusServiceUnavailable {
		t.Fatalf("initial readiness status=%d", res.Code)
	}

	server.MarkReady()
	res = httptest.NewRecorder()
	server.Routes().ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("ready status=%d", res.Code)
	}
}
