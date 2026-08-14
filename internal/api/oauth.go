package api

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

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
	if err := verifyJWT(strings.TrimSpace(strings.TrimPrefix(authorization, "Bearer ")), app, r.Host, time.Now().UTC()); err != nil {
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
	accessToken, err := randomID("tr_oauth_", 32)
	if err != nil {
		oauthError(w, http.StatusInternalServerError, "server_error", "could not issue access token")
		return
	}
	now := time.Now().UTC()
	if err := s.meta.CreateOAuthAccessToken(r.Context(), meta.HashToken(accessToken), app.ProjectID, now.Add(time.Duration(duration)*time.Second), now); err != nil {
		oauthError(w, http.StatusInternalServerError, "server_error", "could not persist access token")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"access_token": accessToken, "token_type": "Bearer", "expires_in": duration})
}

func (s *Server) listOAuthApps(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	apps, err := s.meta.ListOAuthApps(r.Context())
	if err != nil {
		errorJSON(w, http.StatusInternalServerError, "storage_error", "could not list OAuth apps")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": apps})
}

func (s *Server) createOAuthApp(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	var body struct {
		ClientID    string `json:"client_id"`
		ProjectID   string `json:"project_id"`
		PublicKeyID string `json:"public_key_id"`
		PublicKey   string `json:"public_key"`
		Enabled     *bool  `json:"enabled"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 128<<10)).Decode(&body); err != nil {
		errorJSON(w, http.StatusBadRequest, "invalid_json", "request body is invalid")
		return
	}
	body.ClientID = strings.TrimSpace(body.ClientID)
	body.ProjectID = strings.TrimSpace(body.ProjectID)
	body.PublicKeyID = strings.TrimSpace(body.PublicKeyID)
	if body.ClientID == "" || len(body.ClientID) > 256 || body.ProjectID == "" || body.PublicKeyID == "" || len(body.PublicKeyID) > 256 || len(body.PublicKey) > 128<<10 {
		errorJSON(w, http.StatusBadRequest, "invalid_oauth_app", "client_id, project_id, public_key_id and public_key are required")
		return
	}
	if _, err := s.meta.Project(r.Context(), body.ProjectID); err != nil {
		errorJSON(w, http.StatusBadRequest, "invalid_project", "project was not found")
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
	app := meta.OAuthApp{ID: id, ClientID: body.ClientID, ProjectID: body.ProjectID, PublicKeyID: body.PublicKeyID, PublicKey: body.PublicKey, Enabled: enabled, CreatedAt: now, UpdatedAt: now}
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

func verifyJWT(raw string, app meta.OAuthApp, audience string, now time.Time) error {
	parts := strings.Split(raw, ".")
	if len(parts) != 3 {
		return errors.New("malformed JWT")
	}
	decode := func(value string, target any) error {
		data, err := base64.RawURLEncoding.DecodeString(value)
		if err != nil {
			return err
		}
		return json.Unmarshal(data, target)
	}
	var header struct {
		Algorithm string `json:"alg"`
		KeyID     string `json:"kid"`
	}
	if err := decode(parts[0], &header); err != nil || header.Algorithm != "RS256" || header.KeyID != app.PublicKeyID {
		return errors.New("JWT header is invalid")
	}
	var claims map[string]json.RawMessage
	if err := decode(parts[1], &claims); err != nil {
		return errors.New("JWT claims are invalid")
	}
	stringClaim := func(name string) (string, error) {
		var value string
		if err := json.Unmarshal(claims[name], &value); err != nil || value == "" {
			return "", fmt.Errorf("JWT claim %s is invalid", name)
		}
		return value, nil
	}
	issuer, err := stringClaim("iss")
	if err != nil || issuer != app.ClientID {
		return errors.New("JWT issuer is invalid")
	}
	issuedAt, err := numericClaim(claims, "iat")
	if err != nil || issuedAt > now.Add(60*time.Second).Unix() {
		return errors.New("JWT issued-at is invalid")
	}
	expiresAt, err := numericClaim(claims, "exp")
	if err != nil || expiresAt <= now.Unix() {
		return errors.New("JWT is expired")
	}
	if _, err := stringClaim("jti"); err != nil {
		return err
	}
	if !audienceClaimMatches(claims["aud"], audience) {
		return errors.New("JWT audience is invalid")
	}
	publicKey, err := parseRSAPublicKey(app.PublicKey)
	if err != nil {
		return errors.New("OAuth app public key is invalid")
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return errors.New("JWT signature is invalid")
	}
	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if err := rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, digest[:], signature); err != nil {
		return errors.New("JWT signature verification failed")
	}
	return nil
}

func numericClaim(claims map[string]json.RawMessage, name string) (int64, error) {
	var raw json.Number
	if err := json.Unmarshal(claims[name], &raw); err != nil {
		return 0, fmt.Errorf("JWT claim %s is invalid", name)
	}
	value, err := strconv.ParseInt(string(raw), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("JWT claim %s is invalid", name)
	}
	return value, nil
}

func audienceClaimMatches(raw json.RawMessage, audience string) bool {
	var single string
	if json.Unmarshal(raw, &single) == nil {
		return single == audience
	}
	var many []string
	if json.Unmarshal(raw, &many) != nil {
		return false
	}
	for _, item := range many {
		if item == audience {
			return true
		}
	}
	return false
}

func oauthError(w http.ResponseWriter, status int, code, description string) {
	writeJSON(w, status, map[string]string{"error": code, "error_description": description})
}
