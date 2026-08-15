package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUnknownAPIPathDoesNotFallBackToSPA(t *testing.T) {
	server := NewServer(nil, nil, nil, slog.Default())
	res := httptest.NewRecorder()
	server.Routes().ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/v1/bogus", nil))
	if res.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	var body map[string]map[string]string
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil || body["error"]["code"] != "not_found" {
		t.Fatalf("body=%s", res.Body.String())
	}
}

func TestWrongMethodDoesNotFallBackToSPA(t *testing.T) {
	server := NewServer(nil, nil, nil, slog.Default())
	res := httptest.NewRecorder()
	server.Routes().ServeHTTP(res, httptest.NewRequest(http.MethodDelete, "/api/v1/ingest", nil))
	if res.Code != http.StatusMethodNotAllowed && res.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	if got := res.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("content-type=%q body=%s", got, res.Body.String())
	}
}
