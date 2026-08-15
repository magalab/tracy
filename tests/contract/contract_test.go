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

func TestWorkspaceAPIKeysContract(t *testing.T) {
	base := strings.TrimRight(os.Getenv("BASE_URL"), "/")
	email, password := os.Getenv("TRACY_TEST_EMAIL"), os.Getenv("TRACY_TEST_PASSWORD")
	if base == "" || email == "" || password == "" {
		t.Skip("set BASE_URL, TRACY_TEST_EMAIL and TRACY_TEST_PASSWORD to run workspace key contract tests")
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
		Token     string `json:"access_token"`
		Workspace struct {
			ID string `json:"id"`
		} `json:"workspace"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&session); err != nil {
		t.Fatal(err)
	}
	if session.Token == "" || session.Workspace.ID == "" {
		t.Fatalf("incomplete session response: %+v", session)
	}
	workspaceID := session.Workspace.ID
	unauthenticatedKeys, _ := http.NewRequest(http.MethodGet, base+"/api/v1/workspaces/"+workspaceID+"/keys", nil)
	unauthenticatedKeysResp, err := http.DefaultClient.Do(unauthenticatedKeys)
	if err != nil {
		t.Fatal(err)
	}
	if unauthenticatedKeysResp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthenticated workspace keys status=%d", unauthenticatedKeysResp.StatusCode)
	}
	_ = unauthenticatedKeysResp.Body.Close()
	withoutWorkspace, _ := http.NewRequest(http.MethodGet, base+"/api/v1/traces", nil)
	withoutWorkspace.Header.Set("Authorization", "Bearer "+session.Token)
	withoutWorkspaceResp, err := http.DefaultClient.Do(withoutWorkspace)
	if err != nil {
		t.Fatal(err)
	}
	if withoutWorkspaceResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("trace request without workspace status=%d", withoutWorkspaceResp.StatusCode)
	}
	_ = withoutWorkspaceResp.Body.Close()
	withWorkspace, _ := http.NewRequest(http.MethodGet, base+"/api/v1/traces", nil)
	withWorkspace.Header.Set("Authorization", "Bearer "+session.Token)
	withWorkspace.Header.Set("X-Tracy-Workspace-ID", workspaceID)
	withWorkspaceResp, err := http.DefaultClient.Do(withWorkspace)
	if err != nil {
		t.Fatal(err)
	}
	if withWorkspaceResp.StatusCode != http.StatusOK {
		t.Fatalf("trace request with workspace status=%d", withWorkspaceResp.StatusCode)
	}
	_ = withWorkspaceResp.Body.Close()
	statsWithoutWorkspace, _ := http.NewRequest(http.MethodGet, base+"/api/v1/ingest/stats", nil)
	statsWithoutWorkspace.Header.Set("Authorization", "Bearer "+session.Token)
	statsWithoutWorkspaceResp, err := http.DefaultClient.Do(statsWithoutWorkspace)
	if err != nil {
		t.Fatal(err)
	}
	if statsWithoutWorkspaceResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("ingest stats without workspace status=%d", statsWithoutWorkspaceResp.StatusCode)
	}
	_ = statsWithoutWorkspaceResp.Body.Close()
	stats, _ := http.NewRequest(http.MethodGet, base+"/api/v1/ingest/stats", nil)
	stats.Header.Set("Authorization", "Bearer "+session.Token)
	stats.Header.Set("X-Tracy-Workspace-ID", workspaceID)
	statsResp, err := http.DefaultClient.Do(stats)
	if err != nil {
		t.Fatal(err)
	}
	if statsResp.StatusCode != http.StatusOK {
		t.Fatalf("ingest stats status=%d", statsResp.StatusCode)
	}
	_ = statsResp.Body.Close()
	oauthWithoutWorkspace, _ := http.NewRequest(http.MethodGet, base+"/api/v1/oauth/apps", nil)
	oauthWithoutWorkspace.Header.Set("Authorization", "Bearer "+session.Token)
	oauthWithoutWorkspaceResp, err := http.DefaultClient.Do(oauthWithoutWorkspace)
	if err != nil {
		t.Fatal(err)
	}
	if oauthWithoutWorkspaceResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("OAuth apps without workspace status=%d", oauthWithoutWorkspaceResp.StatusCode)
	}
	_ = oauthWithoutWorkspaceResp.Body.Close()
	oauthForbidden, _ := http.NewRequest(http.MethodGet, base+"/api/v1/oauth/apps", nil)
	oauthForbidden.Header.Set("Authorization", "Bearer "+session.Token)
	oauthForbidden.Header.Set("X-Tracy-Workspace-ID", "workspace-does-not-exist")
	oauthForbiddenResp, err := http.DefaultClient.Do(oauthForbidden)
	if err != nil {
		t.Fatal(err)
	}
	if oauthForbiddenResp.StatusCode != http.StatusForbidden {
		t.Fatalf("OAuth apps inaccessible workspace status=%d", oauthForbiddenResp.StatusCode)
	}
	_ = oauthForbiddenResp.Body.Close()
	oauthList, _ := http.NewRequest(http.MethodGet, base+"/api/v1/oauth/apps", nil)
	oauthList.Header.Set("Authorization", "Bearer "+session.Token)
	oauthList.Header.Set("X-Tracy-Workspace-ID", workspaceID)
	oauthListResp, err := http.DefaultClient.Do(oauthList)
	if err != nil {
		t.Fatal(err)
	}
	if oauthListResp.StatusCode != http.StatusOK {
		t.Fatalf("OAuth apps status=%d", oauthListResp.StatusCode)
	}
	_ = oauthListResp.Body.Close()
	keyBody, _ := json.Marshal(map[string]string{"name": "contract-session-key"})
	create, _ := http.NewRequest(http.MethodPost, base+"/api/v1/workspaces/"+workspaceID+"/keys", bytes.NewReader(keyBody))
	create.Header.Set("Authorization", "Bearer "+session.Token)
	create.Header.Set("Content-Type", "application/json")
	created, err := http.DefaultClient.Do(create)
	if err != nil {
		t.Fatal(err)
	}
	defer created.Body.Close()
	if created.StatusCode != http.StatusCreated {
		t.Fatalf("create workspace key status=%d", created.StatusCode)
	}
	var createdResponse struct {
		APIKey struct {
			ID    string `json:"id"`
			Token string `json:"token"`
		} `json:"api_key"`
	}
	if err := json.NewDecoder(created.Body).Decode(&createdResponse); err != nil {
		t.Fatal(err)
	}
	if createdResponse.APIKey.ID == "" || createdResponse.APIKey.Token == "" {
		t.Fatalf("plaintext key was not returned once: %+v", createdResponse)
	}
	ingestBody, _ := json.Marshal(map[string]any{"spans": []any{map[string]any{
		"trace_id": "contract-key-trace", "span_id": "contract-key-span", "name": "contract-key",
		"kind": "test", "start_time": "2026-01-01T00:00:00Z", "duration": 1000,
	}}})
	keyIngest, _ := http.NewRequest(http.MethodPost, base+"/api/v1/ingest", bytes.NewReader(ingestBody))
	keyIngest.Header.Set("Authorization", "Bearer "+createdResponse.APIKey.Token)
	keyIngest.Header.Set("Content-Type", "application/json")
	keyIngestResp, err := http.DefaultClient.Do(keyIngest)
	if err != nil {
		t.Fatal(err)
	}
	if keyIngestResp.StatusCode != http.StatusAccepted {
		t.Fatalf("created API key ingest status=%d", keyIngestResp.StatusCode)
	}
	_ = keyIngestResp.Body.Close()
	list, _ := http.NewRequest(http.MethodGet, base+"/api/v1/workspaces/"+workspaceID+"/keys", nil)
	list.Header.Set("Authorization", "Bearer "+session.Token)
	listed, err := http.DefaultClient.Do(list)
	if err != nil {
		t.Fatal(err)
	}
	if listed.StatusCode != http.StatusOK {
		t.Fatalf("list workspace keys status=%d", listed.StatusCode)
	}
	var listResponse struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.NewDecoder(listed.Body).Decode(&listResponse); err != nil {
		t.Fatal(err)
	}
	_ = listed.Body.Close()
	for _, item := range listResponse.Items {
		if _, exists := item["token"]; exists {
			t.Fatal("list endpoint exposed plaintext token")
		}
	}
	revoke, _ := http.NewRequest(http.MethodPost, base+"/api/v1/workspaces/"+workspaceID+"/keys/"+createdResponse.APIKey.ID+"/revoke", nil)
	revoke.Header.Set("Authorization", "Bearer "+session.Token)
	revoked, err := http.DefaultClient.Do(revoke)
	if err != nil {
		t.Fatal(err)
	}
	if revoked.StatusCode != http.StatusOK {
		t.Fatalf("revoke workspace key status=%d", revoked.StatusCode)
	}
	_ = revoked.Body.Close()
	revokedIngest, _ := http.NewRequest(http.MethodPost, base+"/api/v1/ingest", bytes.NewReader(ingestBody))
	revokedIngest.Header.Set("Authorization", "Bearer "+createdResponse.APIKey.Token)
	revokedIngest.Header.Set("Content-Type", "application/json")
	revokedIngestResp, err := http.DefaultClient.Do(revokedIngest)
	if err != nil {
		t.Fatal(err)
	}
	if revokedIngestResp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("revoked API key ingest status=%d", revokedIngestResp.StatusCode)
	}
	_ = revokedIngestResp.Body.Close()
	wrongWorkspaceRevoke, _ := http.NewRequest(http.MethodPost, base+"/api/v1/workspaces/workspace-does-not-exist/keys/"+createdResponse.APIKey.ID+"/revoke", nil)
	wrongWorkspaceRevoke.Header.Set("Authorization", "Bearer "+session.Token)
	wrongWorkspaceRevokeResp, err := http.DefaultClient.Do(wrongWorkspaceRevoke)
	if err != nil {
		t.Fatal(err)
	}
	if wrongWorkspaceRevokeResp.StatusCode != http.StatusForbidden {
		t.Fatalf("cross-workspace revoke status=%d", wrongWorkspaceRevokeResp.StatusCode)
	}
	_ = wrongWorkspaceRevokeResp.Body.Close()
}
