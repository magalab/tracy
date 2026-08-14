package contract

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
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
	body, _ := json.Marshal(map[string]any{"spans": []any{map[string]any{"trace_id": "contract-trace", "span_id": "contract-span", "name": "contract", "kind": "test", "start_time": "2026-01-01T00:00:00Z", "duration": 1000}}})
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
	cozeReq, _ := http.NewRequest(http.MethodPost, base+"/v1/loop/traces/ingest", bytes.NewReader(cozeBody))
	cozeReq.Header.Set("Authorization", "Bearer "+key)
	cozeReq.Header.Set("Content-Type", "application/json")
	res, err = http.DefaultClient.Do(cozeReq)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("cozeloop ingest status=%d", res.StatusCode)
	}
	_ = res.Body.Close()
	for attempt := 0; attempt < 20; attempt++ {
		getReq, _ := http.NewRequest(http.MethodGet, base+"/api/v1/traces/contract-trace", nil)
		getReq.Header.Set("Authorization", "Bearer "+key)
		res, err = http.DefaultClient.Do(getReq)
		if err != nil {
			t.Fatal(err)
		}
		if res.StatusCode == http.StatusOK {
			_ = res.Body.Close()
			break
		}
		_ = res.Body.Close()
		if res.StatusCode != http.StatusNotFound {
			t.Fatalf("authenticated get status=%d", res.StatusCode)
		}
		if attempt == 19 {
			t.Fatalf("trace was not visible after ingest")
		}
		time.Sleep(50 * time.Millisecond)
	}
	dashboardReq, _ := http.NewRequest(http.MethodGet, base+"/api/v1/dashboard?start_time=2025-12-31T00:00:00Z&end_time=2026-01-02T00:00:00Z", nil)
	dashboardReq.Header.Set("Authorization", "Bearer "+key)
	dashboardResp, err := http.DefaultClient.Do(dashboardReq)
	if err != nil {
		t.Fatal(err)
	}
	if dashboardResp.StatusCode != http.StatusOK {
		t.Fatalf("dashboard status=%d", dashboardResp.StatusCode)
	}
	var dashboard struct {
		RequestCount int64 `json:"request_count"`
	}
	if err := json.NewDecoder(dashboardResp.Body).Decode(&dashboard); err != nil {
		t.Fatal(err)
	}
	_ = dashboardResp.Body.Close()
	if dashboard.RequestCount < 1 {
		t.Fatalf("dashboard request count=%d", dashboard.RequestCount)
	}
	annotationBody, _ := json.Marshal(map[string]any{"key": "quality", "score": 0.9, "label": "good", "comment": "contract annotation"})
	annotationReq, _ := http.NewRequest(http.MethodPost, base+"/api/v1/traces/contract-trace/annotations", bytes.NewReader(annotationBody))
	annotationReq.Header.Set("Authorization", "Bearer "+key)
	annotationReq.Header.Set("Content-Type", "application/json")
	annotationResp, err := http.DefaultClient.Do(annotationReq)
	if err != nil {
		t.Fatal(err)
	}
	if annotationResp.StatusCode != http.StatusCreated {
		t.Fatalf("annotation create status=%d", annotationResp.StatusCode)
	}
	var created struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(annotationResp.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	_ = annotationResp.Body.Close()
	if created.ID == "" {
		t.Fatal("annotation id missing")
	}
	annotationListReq, _ := http.NewRequest(http.MethodGet, base+"/api/v1/traces/contract-trace/annotations", nil)
	annotationListReq.Header.Set("Authorization", "Bearer "+key)
	annotationListResp, err := http.DefaultClient.Do(annotationListReq)
	if err != nil {
		t.Fatal(err)
	}
	if annotationListResp.StatusCode != http.StatusOK {
		t.Fatalf("annotation list status=%d", annotationListResp.StatusCode)
	}
	_ = annotationListResp.Body.Close()
	annotationDeleteReq, _ := http.NewRequest(http.MethodDelete, base+"/api/v1/annotations/"+created.ID, nil)
	annotationDeleteReq.Header.Set("Authorization", "Bearer "+key)
	annotationDeleteResp, err := http.DefaultClient.Do(annotationDeleteReq)
	if err != nil {
		t.Fatal(err)
	}
	if annotationDeleteResp.StatusCode != http.StatusOK {
		t.Fatalf("annotation delete status=%d", annotationDeleteResp.StatusCode)
	}
	_ = annotationDeleteResp.Body.Close()
}

func TestAdminContract(t *testing.T) {
	base := strings.TrimRight(os.Getenv("BASE_URL"), "/")
	admin := os.Getenv("TRACY_ADMIN_API_KEY")
	if base == "" || admin == "" {
		t.Skip("set BASE_URL and TRACY_ADMIN_API_KEY to run admin contract tests")
	}
	body, _ := json.Marshal(map[string]string{"name": "contract-admin-project", "key_name": "contract-key"})
	req, _ := http.NewRequest(http.MethodPost, base+"/api/v1/projects", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+admin)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create project status=%d", resp.StatusCode)
	}
	var response struct {
		APIKey struct {
			ID        string `json:"id"`
			ProjectID string `json:"project_id"`
			Token     string `json:"token"`
		} `json:"api_key"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if response.APIKey.ID == "" || response.APIKey.ProjectID == "" || response.APIKey.Token == "" {
		t.Fatalf("incomplete key response: %+v", response)
	}
	getReq, _ := http.NewRequest(http.MethodGet, base+"/api/v1/projects/"+response.APIKey.ProjectID+"/keys", nil)
	getReq.Header.Set("Authorization", "Bearer "+admin)
	result, err := http.DefaultClient.Do(getReq)
	if err != nil {
		t.Fatal(err)
	}
	if result.StatusCode != http.StatusOK {
		t.Fatalf("list keys status=%d", result.StatusCode)
	}
	_ = result.Body.Close()
	revoke, _ := http.NewRequest(http.MethodPost, base+"/api/v1/keys/"+response.APIKey.ID+"/revoke", nil)
	revoke.Header.Set("Authorization", "Bearer "+admin)
	result, err = http.DefaultClient.Do(revoke)
	if err != nil {
		t.Fatal(err)
	}
	if result.StatusCode != http.StatusOK {
		t.Fatalf("revoke status=%d", result.StatusCode)
	}
	_ = result.Body.Close()
}
