package api

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/panda/tracy/internal/storage/meta"
)

const jwtBearerGrantType = "urn:ietf:params:oauth:grant-type:jwt-bearer"

func (s *Server) oauthToken(w http.ResponseWriter, r *http.Request) {
	var request struct {
		ClientID        string `json:"client_id"`
		GrantType       string `json:"grant_type"`
		DurationSeconds int64  `json:"duration_seconds"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&request); err != nil {
		oauthError(w, http.StatusBadRequest, "invalid_request", "request body is invalid")
		return
	}
	if request.ClientID == "" || request.GrantType != jwtBearerGrantType {
		oauthError(w, http.StatusBadRequest, "invalid_grant", "client_id and jwt bearer grant_type are required")
		return
	}
	app, err := s.meta.OAuthAppByClientID(r.Context(), request.ClientID)
	if err != nil || !app.Enabled {
		oauthError(w, http.StatusUnauthorized, "invalid_client", "OAuth app is not available")
		return
	}
	authorization := r.Header.Get("Authorization")
	if !strings.HasPrefix(authorization, "Bearer ") {
		oauthError(w, http.StatusUnauthorized, "invalid_grant", "JWT bearer token is required")
		return
	}
	now := time.Now().UTC()
	claims, err := verifyJWT(strings.TrimSpace(strings.TrimPrefix(authorization, "Bearer ")), app, r.Host, now)
	if err != nil {
		oauthError(w, http.StatusUnauthorized, "invalid_grant", err.Error())
		return
	}
	duration := request.DurationSeconds
	if duration == 0 {
		duration = 900
	}
	if duration < 60 || duration > 24*60*60 {
		oauthError(w, http.StatusBadRequest, "invalid_request", "duration_seconds must be between 60 and 86400")
		return
	}
	if err := s.meta.ConsumeOAuthJTI(r.Context(), app.ClientID, claims.ID, claims.ExpiresAt.Time, now); err != nil {
		if errors.Is(err, meta.ErrAlreadyUsed) {
			oauthError(w, http.StatusUnauthorized, "invalid_grant", "JWT has already been used")
			return
		}
		oauthError(w, http.StatusInternalServerError, "server_error", "could not record JWT usage")
		return
	}
	accessToken, err := randomID("tr_oauth_", 32)
	if err != nil {
		oauthError(w, http.StatusInternalServerError, "server_error", "could not issue access token")
		return
	}
	if err := s.meta.CreateOAuthAccessToken(r.Context(), meta.HashToken(accessToken), app.WorkspaceID, now.Add(time.Duration(duration)*time.Second), now); err != nil {
		oauthError(w, http.StatusInternalServerError, "server_error", "could not persist access token")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"access_token": accessToken, "token_type": "Bearer", "expires_in": duration})
}

func (s *Server) listOAuthApps(w http.ResponseWriter, r *http.Request) {
	_, workspaceID, ok := s.requireWorkspaceOwnerForHeaderValue(w, r)
	if !ok {
		return
	}
	apps, err := s.meta.ListOAuthAppsByWorkspace(r.Context(), workspaceID)
	if err != nil {
		errorJSON(w, http.StatusInternalServerError, "storage_error", "could not list OAuth apps")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": apps})
}

func (s *Server) createOAuthApp(w http.ResponseWriter, r *http.Request) {
	_, workspaceID, ok := s.requireWorkspaceOwnerForHeaderValue(w, r)
	if !ok {
		return
	}
	var body struct {
		ClientID    string `json:"client_id"`
		WorkspaceID string `json:"workspace_id"`
		PublicKeyID string `json:"public_key_id"`
		PublicKey   string `json:"public_key"`
		Enabled     *bool  `json:"enabled"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 128<<10)).Decode(&body); err != nil {
		errorJSON(w, http.StatusBadRequest, "invalid_json", "request body is invalid")
		return
	}
	body.ClientID = strings.TrimSpace(body.ClientID)
	body.WorkspaceID = strings.TrimSpace(body.WorkspaceID)
	body.PublicKeyID = strings.TrimSpace(body.PublicKeyID)
	if body.ClientID == "" || len(body.ClientID) > 256 || body.WorkspaceID == "" || body.PublicKeyID == "" || len(body.PublicKeyID) > 256 || len(body.PublicKey) > 128<<10 {
		errorJSON(w, http.StatusBadRequest, "invalid_oauth_app", "client_id, workspace_id, public_key_id and public_key are required")
		return
	}
	if body.WorkspaceID != workspaceID {
		errorJSON(w, http.StatusForbidden, "workspace_forbidden", "OAuth app must belong to the selected workspace")
		return
	}
	if _, err := s.meta.Workspace(r.Context(), body.WorkspaceID); err != nil {
		errorJSON(w, http.StatusBadRequest, "invalid_workspace", "workspace was not found")
		return
	}
	if _, err := parseRSAPublicKey(body.PublicKey); err != nil {
		errorJSON(w, http.StatusBadRequest, "invalid_public_key", "public_key must be an RSA PEM public key")
		return
	}
	enabled := true
	if body.Enabled != nil {
		enabled = *body.Enabled
	}
	now := time.Now().UTC()
	id, err := randomID("oauth_app_", 12)
	if err != nil {
		errorJSON(w, http.StatusInternalServerError, "id_generation_failed", "could not generate OAuth app id")
		return
	}
	app := meta.OAuthApp{ID: id, ClientID: body.ClientID, WorkspaceID: body.WorkspaceID, PublicKeyID: body.PublicKeyID, PublicKey: body.PublicKey, Enabled: enabled, CreatedAt: now, UpdatedAt: now}
	if err := s.meta.CreateOAuthApp(r.Context(), app); err != nil {
		errorJSON(w, http.StatusConflict, "oauth_app_exists", "client_id is already registered")
		return
	}
	writeJSON(w, http.StatusCreated, app)
}

func parseRSAPublicKey(raw string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(strings.TrimSpace(raw)))
	if block == nil {
		return nil, errors.New("PEM block is required")
	}
	if key, err := x509.ParsePKIXPublicKey(block.Bytes); err == nil {
		if rsaKey, ok := key.(*rsa.PublicKey); ok {
			return rsaKey, nil
		}
	}
	if key, err := x509.ParsePKCS1PublicKey(block.Bytes); err == nil {
		return key, nil
	}
	return nil, errors.New("unsupported RSA public key")
}

func verifyJWT(raw string, app meta.OAuthApp, audience string, now time.Time) (*jwt.RegisteredClaims, error) {
	publicKey, err := parseRSAPublicKey(app.PublicKey)
	if err != nil {
		return nil, errors.New("OAuth app public key is invalid")
	}
	claims := &jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(raw, claims, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodRS256 {
			return nil, errors.New("JWT algorithm is invalid")
		}
		if keyID, ok := token.Header["kid"].(string); !ok || keyID != app.PublicKeyID {
			return nil, errors.New("JWT key id is invalid")
		}
		return publicKey, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}), jwt.WithIssuer(app.ClientID), jwt.WithAudience(audience), jwt.WithLeeway(60*time.Second), jwt.WithTimeFunc(func() time.Time { return now }))
	if err != nil || !token.Valid {
		return nil, errors.New("JWT validation failed")
	}
	if claims.ID == "" || claims.ExpiresAt == nil {
		return nil, errors.New("JWT claim jti is invalid")
	}
	return claims, nil
}

func oauthError(w http.ResponseWriter, status int, code, description string) {
	writeJSON(w, status, map[string]string{"error": code, "error_description": description})
}
