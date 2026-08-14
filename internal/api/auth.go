package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/panda/tracy/internal/storage/meta"
)

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&body); err != nil {
		errorJSON(w, http.StatusBadRequest, "invalid_json", "request body is invalid")
		return
	}
	user, err := s.meta.AuthenticateUser(r.Context(), strings.ToLower(strings.TrimSpace(body.Email)), body.Password)
	if err != nil {
		errorJSON(w, http.StatusUnauthorized, "invalid_credentials", "email or password is invalid")
		return
	}
	workspace, err := s.meta.FirstWorkspaceForUser(r.Context(), user.ID)
	if err != nil {
		errorJSON(w, http.StatusForbidden, "no_workspace", "user is not a member of a workspace")
		return
	}
	token, err := randomID("tr_session_", 32)
	if err != nil {
		errorJSON(w, http.StatusInternalServerError, "session_error", "could not create session")
		return
	}
	now := time.Now().UTC()
	if err := s.meta.CreateSession(r.Context(), meta.HashToken(token), user.ID, workspace.ID, now.Add(24*time.Hour), now); err != nil {
		errorJSON(w, http.StatusInternalServerError, "session_error", "could not persist session")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"access_token": token,
		"token_type":   "Bearer",
		"user":         map[string]string{"id": user.ID, "email": user.Email, "name": user.Name},
		"workspace":    workspace,
	})
}

func (s *Server) currentUser(w http.ResponseWriter, r *http.Request) {
	key, err := s.authenticate(r)
	if err != nil {
		errorJSON(w, http.StatusUnauthorized, "invalid_session", "login is required")
		return
	}
	if key.UserID == "" {
		writeJSON(w, http.StatusOK, map[string]any{"user": nil, "workspace_id": key.ProjectID, "auth_type": "api_key"})
		return
	}
	user, err := s.meta.UserByID(r.Context(), key.UserID)
	if err != nil {
		errorJSON(w, http.StatusUnauthorized, "invalid_session", "user session is invalid")
		return
	}
	workspace, err := s.meta.Project(r.Context(), key.ProjectID)
	if err != nil {
		errorJSON(w, http.StatusUnauthorized, "invalid_session", "workspace is unavailable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": user, "workspace": workspace, "auth_type": "session"})
}
