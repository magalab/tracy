package api

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/magalab/tracy/internal/storage/meta"
)

func TestCreateWorkspaceRejectsDuplicateNameForUser(t *testing.T) {
	ctx, store := newTestMetaStore(t)
	now := time.Now().UTC()
	if err := store.CreateWorkspaceRecord(ctx, meta.Workspace{ID: "default", Name: "Default", CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	passwordHash, err := meta.HashPassword("password")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.CreateUser(ctx, meta.User{ID: "user", Email: "user@example.com", Name: "User", PasswordHash: passwordHash, CreatedAt: now}); err != nil {
		t.Fatal(err)
	}
	if err := store.AddWorkspaceMember(ctx, meta.WorkspaceMember{WorkspaceID: "default", UserID: "user", Role: "owner", CreatedAt: now}); err != nil {
		t.Fatal(err)
	}

	server := NewServer(store, nil, nil, slog.Default()).Routes()
	loginBody, _ := json.Marshal(map[string]string{"email": "user@example.com", "password": "password"})
	login := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(loginBody))
	login.Header.Set("Content-Type", "application/json")
	loginResponse := httptest.NewRecorder()
	server.ServeHTTP(loginResponse, login)
	if loginResponse.Code != http.StatusOK {
		t.Fatalf("login status=%d body=%s", loginResponse.Code, loginResponse.Body.String())
	}
	var session struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(loginResponse.Body).Decode(&session); err != nil {
		t.Fatal(err)
	}

	create := func(name string) *httptest.ResponseRecorder {
		body, _ := json.Marshal(map[string]string{"name": name})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces", bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+session.AccessToken)
		req.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		server.ServeHTTP(response, req)
		return response
	}

	first := create("Production")
	if first.Code != http.StatusCreated {
		t.Fatalf("first create status=%d body=%s", first.Code, first.Body.String())
	}
	second := create("production")
	if second.Code != http.StatusConflict {
		t.Fatalf("duplicate create status=%d body=%s", second.Code, second.Body.String())
	}
	var errorResponse struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.NewDecoder(second.Body).Decode(&errorResponse); err != nil {
		t.Fatal(err)
	}
	if errorResponse.Error.Code != "workspace_name_exists" {
		t.Fatalf("duplicate create error code=%q body=%s", errorResponse.Error.Code, second.Body.String())
	}
}

func TestCreateWorkspaceRejectsConcurrentDuplicateNames(t *testing.T) {
	ctx, store := newTestMetaStore(t)
	now := time.Now().UTC()
	if err := store.CreateUser(ctx, meta.User{ID: "user", Email: "user@example.com", Name: "User", PasswordHash: "hash", CreatedAt: now}); err != nil {
		t.Fatal(err)
	}

	const attempts = 8
	errs := make(chan error, attempts)
	var wg sync.WaitGroup
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			id := "ws_" + string(rune('a'+i))
			errs <- store.CreateWorkspaceForUser(ctx, meta.Workspace{ID: id, Name: "Concurrent", CreatedAt: now, UpdatedAt: now}, meta.WorkspaceMember{WorkspaceID: id, UserID: "user", Role: "owner", CreatedAt: now})
		}(i)
	}
	wg.Wait()
	close(errs)

	created := 0
	for err := range errs {
		if err == nil {
			created++
		} else if err != meta.ErrWorkspaceNameExists {
			t.Fatalf("unexpected concurrent create error: %v", err)
		}
	}
	if created != 1 {
		t.Fatalf("created=%d, want exactly one", created)
	}
}

func TestCreateWorkspaceAllowsSameNameAcrossUsers(t *testing.T) {
	ctx, store := newTestMetaStore(t)
	now := time.Now().UTC()
	for _, id := range []string{"user-1", "user-2"} {
		if err := store.CreateUser(ctx, meta.User{ID: id, Email: id + "@example.com", Name: id, PasswordHash: "hash", CreatedAt: now}); err != nil {
			t.Fatal(err)
		}
	}
	for i, userID := range []string{"user-1", "user-2"} {
		workspaceID := "workspace-" + string(rune('1'+i))
		if err := store.CreateWorkspaceForUser(ctx, meta.Workspace{ID: workspaceID, Name: "Shared name", CreatedAt: now, UpdatedAt: now}, meta.WorkspaceMember{WorkspaceID: workspaceID, UserID: userID, Role: "owner", CreatedAt: now}); err != nil {
			t.Fatalf("user %s: %v", userID, err)
		}
	}
}
