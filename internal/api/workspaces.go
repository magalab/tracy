package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/panda/tracy/internal/storage/meta"
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
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "active_id": key.ProjectID})
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
	if body.Name == "" || len(body.Name) > 128 {
		errorJSON(w, http.StatusBadRequest, "invalid_workspace", "name is required and must be at most 128 bytes")
		return
	}
	id, err := randomID("ws_", 12)
	if err != nil {
		errorJSON(w, http.StatusInternalServerError, "id_generation_failed", "could not generate workspace id")
		return
	}
	now := time.Now().UTC()
	workspace := meta.Project{ID: id, Name: body.Name, CreatedAt: now, UpdatedAt: now}
	if err := s.meta.CreateWorkspaceForUser(r.Context(), workspace, meta.WorkspaceMember{WorkspaceID: id, UserID: key.UserID, Role: "owner", CreatedAt: now}); err != nil {
		errorJSON(w, http.StatusConflict, "workspace_exists", "could not create workspace")
		return
	}
	if err := s.meta.SwitchSessionWorkspace(r.Context(), meta.HashToken(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")), key.UserID, id); err != nil {
		errorJSON(w, http.StatusInternalServerError, "session_error", "could not activate workspace")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"workspace": workspace, "active_id": id})
}

func (s *Server) switchUserWorkspace(w http.ResponseWriter, r *http.Request) {
	key, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	workspaceID := r.PathValue("workspaceID")
	if err := s.meta.SwitchSessionWorkspace(r.Context(), meta.HashToken(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")), key.UserID, workspaceID); err != nil {
		if errors.Is(err, meta.ErrNotFound) {
			errorJSON(w, http.StatusForbidden, "workspace_forbidden", "user cannot access this workspace")
			return
		}
		errorJSON(w, http.StatusInternalServerError, "session_error", "could not switch workspace")
		return
	}
	workspace, err := s.meta.Project(r.Context(), workspaceID)
	if errors.Is(err, meta.ErrNotFound) {
		errorJSON(w, http.StatusNotFound, "workspace_not_found", "workspace was not found")
		return
	}
	if err != nil {
		errorJSON(w, http.StatusInternalServerError, "storage_error", "could not load workspace")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"workspace": workspace, "active_id": workspaceID})
}

func (s *Server) requireUser(w http.ResponseWriter, r *http.Request) (meta.APIKey, bool) {
	key, err := s.authenticate(r)
	if err != nil || key.UserID == "" {
		errorJSON(w, http.StatusUnauthorized, "login_required", "user login is required")
		return meta.APIKey{}, false
	}
	return key, true
}
