package contract

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
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
	res, err = http.Get(base + "/readyz")
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("ready status=%d", res.StatusCode)
	}
	_ = res.Body.Close()
	for _, path := range []string{"/api/v1/does-not-exist", "/api"} {
		unknown, err := http.Get(base + path)
		if err != nil {
			t.Fatal(err)
		}
		if unknown.StatusCode != http.StatusNotFound || !strings.Contains(unknown.Header.Get("Content-Type"), "application/json") {
			t.Fatalf("unknown API path %s status=%d content-type=%q", path, unknown.StatusCode, unknown.Header.Get("Content-Type"))
		}
		_ = unknown.Body.Close()
	}
	wrongMethod, err := http.NewRequest(http.MethodPut, base+"/api/v1/ingest", nil)
	if err != nil {
		t.Fatal(err)
	}
	wrongResp, err := http.DefaultClient.Do(wrongMethod)
	if err != nil {
		t.Fatal(err)
	}
	if (wrongResp.StatusCode != http.StatusNotFound && wrongResp.StatusCode != http.StatusMethodNotAllowed) || !strings.Contains(wrongResp.Header.Get("Content-Type"), "application/json") {
		t.Fatalf("wrong method status=%d content-type=%q", wrongResp.StatusCode, wrongResp.Header.Get("Content-Type"))
	}
	_ = wrongResp.Body.Close()
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
	largeSpans := make([]map[string]any, 101)
	for index := range largeSpans {
		largeSpans[index] = map[string]any{"trace_id": "contract-large-trace", "span_id": fmt.Sprintf("contract-span-%03d", index), "name": "contract", "kind": "test", "start_time": "2026-01-01T00:00:00Z", "duration": 1000}
	}
	largeBody, _ := json.Marshal(map[string]any{"spans": largeSpans})
	largeReq, _ := http.NewRequest(http.MethodPost, base+"/api/v1/ingest", bytes.NewReader(largeBody))
	largeReq.Header.Set("Authorization", "Bearer "+key)
	largeReq.Header.Set("Content-Type", "application/json")
	largeResp, err := http.DefaultClient.Do(largeReq)
	if err != nil {
		t.Fatal(err)
	}
	if largeResp.StatusCode != http.StatusAccepted {
		t.Fatalf("large ingest status=%d", largeResp.StatusCode)
	}
	_ = largeResp.Body.Close()
	var firstPage struct {
		Spans      []any  `json:"spans"`
		NextCursor string `json:"next_cursor"`
	}
	for attempt := 0; attempt < 20; attempt++ {
		pageReq, _ := http.NewRequest(http.MethodGet, base+"/api/v1/traces/contract-large-trace?limit=100", nil)
		pageReq.Header.Set("Authorization", "Bearer "+key)
		pageResp, requestErr := http.DefaultClient.Do(pageReq)
		if requestErr != nil {
			t.Fatal(requestErr)
		}
		if pageResp.StatusCode == http.StatusOK {
			if err := json.NewDecoder(pageResp.Body).Decode(&firstPage); err != nil {
				t.Fatal(err)
			}
			_ = pageResp.Body.Close()
			if len(firstPage.Spans) == 100 && firstPage.NextCursor != "" {
				break
			}
		} else {
			_ = pageResp.Body.Close()
		}
		if attempt == 19 {
			t.Fatalf("trace page did not return a cursor: spans=%d cursor=%q", len(firstPage.Spans), firstPage.NextCursor)
		}
		time.Sleep(50 * time.Millisecond)
	}
	secondReq, _ := http.NewRequest(http.MethodGet, base+"/api/v1/traces/contract-large-trace?limit=100&cursor="+url.QueryEscape(firstPage.NextCursor), nil)
	secondReq.Header.Set("Authorization", "Bearer "+key)
	secondResp, err := http.DefaultClient.Do(secondReq)
	if err != nil {
		t.Fatal(err)
	}
	if secondResp.StatusCode != http.StatusOK {
		t.Fatalf("second trace page status=%d", secondResp.StatusCode)
	}
	_ = secondResp.Body.Close()
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
}

func TestLogoutContract(t *testing.T) {
	base := strings.TrimRight(os.Getenv("BASE_URL"), "/")
	email, password := os.Getenv("TRACY_TEST_EMAIL"), os.Getenv("TRACY_TEST_PASSWORD")
	if base == "" || email == "" || password == "" {
		t.Skip("set BASE_URL, TRACY_TEST_EMAIL and TRACY_TEST_PASSWORD to run session contract tests")
	}
	body, _ := json.Marshal(map[string]string{"email": email, "password": password})
	login, _ := http.NewRequest(http.MethodPost, base+"/api/v1/auth/login", bytes.NewReader(body))
	login.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(login)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login status=%d", resp.StatusCode)
	}
	var session struct {
		Token string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&session); err != nil {
		t.Fatal(err)
	}
	logout, _ := http.NewRequest(http.MethodPost, base+"/api/v1/auth/logout", nil)
	logout.Header.Set("Authorization", "Bearer "+session.Token)
	logoutResp, err := http.DefaultClient.Do(logout)
	if err != nil {
		t.Fatal(err)
	}
	defer logoutResp.Body.Close()
	if logoutResp.StatusCode != http.StatusOK {
		t.Fatalf("logout status=%d", logoutResp.StatusCode)
	}
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
