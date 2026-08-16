package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/magalab/tracy/internal/storage/meta"
)

func (s *Server) listUserWorkspaces(w http.ResponseWriter, r *http.Request) {
	key, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	items, err := s.meta.ListWorkspacesForUser(r.Context(), key.UserID)
	if err != nil {
		errorJSON(w, http.StatusInternalServerError, "storage_error", "could not list workspaces")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) createUserWorkspace(w http.ResponseWriter, r *http.Request) {
	key, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&body); err != nil {
		errorJSON(w, http.StatusBadRequest, "invalid_json", "request body is invalid")
		return
	}
	body.Name = strings.TrimSpace(body.Name)
	if body.Name == "" || utf8.RuneCountInString(body.Name) > 128 {
		errorJSON(w, http.StatusBadRequest, "invalid_workspace", "name is required and must be at most 128 characters")
		return
	}
	id, err := randomID("ws_", 12)
	if err != nil {
		errorJSON(w, http.StatusInternalServerError, "id_generation_failed", "could not generate workspace id")
		return
	}
	now := time.Now().UTC()
	workspace := meta.Workspace{ID: id, Name: body.Name, CreatedAt: now, UpdatedAt: now}
	if err := s.meta.CreateWorkspaceForUser(r.Context(), workspace, meta.WorkspaceMember{WorkspaceID: id, UserID: key.UserID, Role: "owner", CreatedAt: now}); err != nil {
		if errors.Is(err, meta.ErrWorkspaceNameExists) {
			errorJSON(w, http.StatusConflict, "workspace_name_exists", "a workspace with this name already exists")
			return
		}
		errorJSON(w, http.StatusInternalServerError, "storage_error", "could not create workspace")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"workspace": workspace})
}

func (s *Server) switchUserWorkspace(w http.ResponseWriter, r *http.Request) {
	key, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	workspaceID := r.PathValue("workspaceID")
	if err := s.meta.UserCanAccessWorkspace(r.Context(), key.UserID, workspaceID); err != nil {
		if errors.Is(err, meta.ErrNotFound) {
			errorJSON(w, http.StatusForbidden, "workspace_forbidden", "user cannot access this workspace")
			return
		}
		errorJSON(w, http.StatusInternalServerError, "session_error", "could not switch workspace")
		return
	}
	workspace, err := s.meta.Workspace(r.Context(), workspaceID)
	if errors.Is(err, meta.ErrNotFound) {
		errorJSON(w, http.StatusNotFound, "workspace_not_found", "workspace was not found")
		return
	}
	if err != nil {
		errorJSON(w, http.StatusInternalServerError, "storage_error", "could not load workspace")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"workspace": workspace})
}

func (s *Server) requireUser(w http.ResponseWriter, r *http.Request) (meta.APIKey, bool) {
	key, err := s.authenticate(r)
	if err != nil || key.UserID == "" {
		errorJSON(w, http.StatusUnauthorized, "login_required", "user login is required")
		return meta.APIKey{}, false
	}
	return key, true
}

func (s *Server) requireWorkspaceOwner(w http.ResponseWriter, r *http.Request) (meta.APIKey, string, bool) {
	key, ok := s.requireUser(w, r)
	if !ok {
		return meta.APIKey{}, "", false
	}
	workspaceID := r.PathValue("workspaceID")
	return s.requireWorkspaceOwnerID(w, r, key, workspaceID)
}

func (s *Server) requireWorkspaceOwnerForHeaderValue(w http.ResponseWriter, r *http.Request) (meta.APIKey, string, bool) {
	key, ok := s.requireUser(w, r)
	if !ok {
		return meta.APIKey{}, "", false
	}
	workspaceID, ok := s.workspaceForRequest(w, r, key)
	if !ok {
		return meta.APIKey{}, "", false
	}
	return s.requireWorkspaceOwnerID(w, r, key, workspaceID)
}

func (s *Server) requireWorkspaceOwnerForHeader(w http.ResponseWriter, r *http.Request) bool {
	_, _, ok := s.requireWorkspaceOwnerForHeaderValue(w, r)
	return ok
}

func (s *Server) requireWorkspaceOwnerID(w http.ResponseWriter, r *http.Request, key meta.APIKey, workspaceID string) (meta.APIKey, string, bool) {
	role, err := s.meta.WorkspaceMemberRole(r.Context(), key.UserID, workspaceID)
	if errors.Is(err, meta.ErrNotFound) {
		errorJSON(w, http.StatusForbidden, "workspace_forbidden", "user cannot manage this workspace")
		return meta.APIKey{}, "", false
	}
	if err != nil {
		errorJSON(w, http.StatusInternalServerError, "storage_error", "could not load workspace membership")
		return meta.APIKey{}, "", false
	}
	if role != "owner" {
		errorJSON(w, http.StatusForbidden, "owner_required", "workspace owner is required")
		return meta.APIKey{}, "", false
	}
	return key, workspaceID, true
}

func (s *Server) listUserWorkspaceKeys(w http.ResponseWriter, r *http.Request) {
	_, workspaceID, ok := s.requireWorkspaceOwner(w, r)
	if !ok {
		return
	}
	if _, err := s.meta.Workspace(r.Context(), workspaceID); errors.Is(err, meta.ErrNotFound) {
		errorJSON(w, http.StatusNotFound, "workspace_not_found", "workspace was not found")
		return
	} else if err != nil {
		errorJSON(w, http.StatusInternalServerError, "storage_error", "could not load workspace")
		return
	}
	keys, err := s.meta.ListAPIKeys(r.Context(), workspaceID)
	if err != nil {
		errorJSON(w, http.StatusInternalServerError, "storage_error", "could not list API keys")
		return
	}
	result := make([]keyView, 0, len(keys))
	for _, key := range keys {
		result = append(result, keyView{ID: key.ID, WorkspaceID: key.WorkspaceID, Name: key.Name, Role: key.Role, ExpiresAt: key.ExpiresAt, Revoked: key.Revoked, LastUsedAt: key.LastUsedAt})
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": result})
}

func (s *Server) createUserWorkspaceKey(w http.ResponseWriter, r *http.Request) {
	_, workspaceID, ok := s.requireWorkspaceOwner(w, r)
	if !ok {
		return
	}
	var body struct {
		Name      string `json:"name"`
		ExpiresAt string `json:"expires_at"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&body); err != nil {
		errorJSON(w, http.StatusBadRequest, "invalid_json", "request body is invalid")
		return
	}
	key, token, err := s.newKey(workspaceID, body.Name, "workspace")
	if err != nil {
		errorJSON(w, http.StatusBadRequest, "invalid_key", err.Error())
		return
	}
	if body.ExpiresAt != "" {
		expires, parseErr := time.Parse(time.RFC3339, body.ExpiresAt)
		if parseErr != nil || !expires.After(time.Now().UTC()) {
			errorJSON(w, http.StatusBadRequest, "invalid_expires_at", "expires_at must be a future RFC3339 timestamp")
			return
		}
		key.ExpiresAt = &expires
	}
	if err := s.meta.CreateAPIKey(r.Context(), key); err != nil {
		errorJSON(w, http.StatusInternalServerError, "storage_error", "could not save API key")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"api_key": keyView{ID: key.ID, WorkspaceID: key.WorkspaceID, Name: key.Name, Role: key.Role, Token: token, ExpiresAt: key.ExpiresAt}})
}

func (s *Server) revokeUserWorkspaceKey(w http.ResponseWriter, r *http.Request) {
	_, workspaceID, ok := s.requireWorkspaceOwner(w, r)
	if !ok {
		return
	}
	target, err := s.meta.APIKeyByID(r.Context(), r.PathValue("keyID"))
	if errors.Is(err, meta.ErrNotFound) {
		errorJSON(w, http.StatusNotFound, "key_not_found", "API key was not found")
		return
	}
	if err != nil {
		errorJSON(w, http.StatusInternalServerError, "storage_error", "could not inspect API key")
		return
	}
	if target.WorkspaceID != workspaceID {
		errorJSON(w, http.StatusNotFound, "key_not_found", "API key was not found")
		return
	}
	if err := s.meta.RevokeAPIKey(r.Context(), target.ID); err != nil {
		errorJSON(w, http.StatusInternalServerError, "storage_error", "could not revoke API key")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"revoked": true})
}
