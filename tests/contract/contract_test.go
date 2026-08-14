package contract

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestHTTPContract(t *testing.T) {
	base := strings.TrimRight(os.Getenv("BASE_URL"), "/")
	key := os.Getenv("TRACY_TEST_API_KEY")
	if base == "" || key == "" {
		t.Skip("set BASE_URL and TRACY_TEST_API_KEY to run black-box contract tests")
	}
	res, err := http.Get(base + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("health status=%d", res.StatusCode)
	}
	_ = res.Body.Close()
	body, _ := json.Marshal(map[string]any{"spans": []any{map[string]any{"trace_id": "contract-trace", "span_id": "contract-span", "name": "contract", "kind": "test"}}})
	req, _ := http.NewRequest(http.MethodPost, base+"/api/v1/ingest", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")
	res, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusAccepted {
		t.Fatalf("ingest status=%d", res.StatusCode)
	}
	_ = res.Body.Close()
	res, err = http.DefaultClient.Get(base + "/api/v1/traces/contract-trace")
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status=%d", res.StatusCode)
	}
	_ = res.Body.Close()
	cozeBody, _ := json.Marshal(map[string]any{"spans": []any{map[string]any{"started_at_micros": 1767225600000000, "duration_micros": 1, "trace_id": "contract-coze", "span_id": "contract-coze-span", "span_name": "compat", "span_type": "custom"}}})
	cozeReq, _ := http.NewRequest(http.MethodPost, base+"/v1/loop/traces/ingest", bytes.NewReader(cozeBody)); cozeReq.Header.Set("Authorization", "Bearer "+key); cozeReq.Header.Set("Content-Type", "application/json")
	res, err = http.DefaultClient.Do(cozeReq); if err != nil { t.Fatal(err) }; if res.StatusCode != http.StatusOK { t.Fatalf("cozeloop ingest status=%d", res.StatusCode) }; _ = res.Body.Close()
	getReq, _ := http.NewRequest(http.MethodGet, base+"/api/v1/traces/contract-trace", nil); getReq.Header.Set("Authorization", "Bearer "+key); res, err = http.DefaultClient.Do(getReq); if err != nil { t.Fatal(err) }; if res.StatusCode != http.StatusOK { t.Fatalf("authenticated get status=%d", res.StatusCode) }; _ = res.Body.Close()
}
