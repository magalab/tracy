package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/panda/tracy/compat/cozeloop"
	"github.com/panda/tracy/internal/ingest"
	"github.com/panda/tracy/internal/storage/meta"
	tracestore "github.com/panda/tracy/internal/storage/trace"
	domain "github.com/panda/tracy/internal/trace"
	"github.com/panda/tracy/internal/web"
)

type Server struct {
	meta   *meta.Store
	writer *ingest.Writer
	traces interface {
		GetTracePage(ctx context.Context, projectID, traceID, cursor string, limit int) (tracestore.TracePage, error)
		ListTraces(ctx context.Context, query domain.Query) (domain.Page, error)
		Metrics(ctx context.Context, query domain.MetricsQuery) (domain.Metrics, error)
	}
	logger           *slog.Logger
	readyFlag        atomic.Bool
	loginMu          sync.Mutex
	loginAttempts    map[string]loginAttempt
	loginLastCleanup time.Time
	trustedProxies   []netip.Prefix
}

type loginAttempt struct {
	window time.Time
	count  int
}

func (s *Server) MarkReady() { s.readyFlag.Store(true) }

func NewServer(m *meta.Store, w *ingest.Writer, t interface {
	GetTracePage(context.Context, string, string, string, int) (tracestore.TracePage, error)
	ListTraces(context.Context, domain.Query) (domain.Page, error)
	Metrics(context.Context, domain.MetricsQuery) (domain.Metrics, error)
}, logger *slog.Logger, trustedProxyConfig ...string) *Server {
	s := &Server{meta: m, writer: w, traces: t, logger: logger, loginAttempts: make(map[string]loginAttempt)}
	if len(trustedProxyConfig) > 0 {
		for _, raw := range strings.Split(trustedProxyConfig[0], ",") {
			raw = strings.TrimSpace(raw)
			if raw == "" {
				continue
			}
			if prefix, err := netip.ParsePrefix(raw); err == nil {
				s.trustedProxies = append(s.trustedProxies, prefix)
			} else if addr, err := netip.ParseAddr(raw); err == nil {
				s.trustedProxies = append(s.trustedProxies, netip.PrefixFrom(addr, addr.BitLen()))
			}
		}
	}
	return s
}

func (s *Server) loginClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	remote, err := netip.ParseAddr(host)
	if err == nil {
		for _, proxy := range s.trustedProxies {
			if proxy.Contains(remote) {
				for _, forwarded := range strings.Split(r.Header.Get("X-Forwarded-For"), ",") {
					if candidate, parseErr := netip.ParseAddr(strings.TrimSpace(forwarded)); parseErr == nil {
						return candidate.String()
					}
				}
				break
			}
		}
	}
	return host
}

func (s *Server) allowLogin(clientIP string) bool {
	now := time.Now()
	s.loginMu.Lock()
	defer s.loginMu.Unlock()
	if s.loginLastCleanup.IsZero() || now.Sub(s.loginLastCleanup) >= time.Minute {
		for key, item := range s.loginAttempts {
			if now.Sub(item.window) >= time.Minute {
				delete(s.loginAttempts, key)
			}
		}
		s.loginLastCleanup = now
	}
	attempt := s.loginAttempts[clientIP]
	if attempt.window.IsZero() || now.Sub(attempt.window) >= time.Minute {
		attempt = loginAttempt{window: now}
	}
	attempt.count++
	s.loginAttempts[clientIP] = attempt
	return attempt.count <= 10
}

func (s *Server) resetLogin(clientIP string) {
	s.loginMu.Lock()
	delete(s.loginAttempts, clientIP)
	s.loginMu.Unlock()
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.health)
	mux.HandleFunc("GET /readyz", s.ready)
	mux.HandleFunc("POST /api/v1/auth/login", s.login)
	mux.HandleFunc("POST /api/v1/auth/logout", s.logout)
	mux.HandleFunc("GET /api/v1/auth/me", s.currentUser)
	mux.HandleFunc("GET /api/v1/workspaces", s.listUserWorkspaces)
	mux.HandleFunc("POST /api/v1/workspaces", s.createUserWorkspace)
	mux.HandleFunc("POST /api/v1/workspaces/{workspaceID}/switch", s.switchUserWorkspace)
	mux.HandleFunc("POST /api/v1/ingest", s.ingest)
	mux.HandleFunc("POST /v1/loop/traces/ingest", s.cozeLoopIngest)
	mux.HandleFunc("POST /api/permission/oauth2/token", s.oauthToken)
	mux.HandleFunc("GET /api/v1/traces/", s.getTrace)
	mux.HandleFunc("GET /api/v1/traces", s.listTraces)
	mux.HandleFunc("GET /api/v1/dashboard", s.dashboard)
	mux.HandleFunc("GET /api/v1/ingest/stats", s.ingestStats)
	mux.HandleFunc("GET /api/v1/projects", s.listProjects)
	mux.HandleFunc("POST /api/v1/projects", s.createProject)
	mux.HandleFunc("GET /api/v1/projects/{projectID}/keys", s.listProjectKeys)
	mux.HandleFunc("POST /api/v1/projects/{projectID}/keys", s.createProjectKey)
	mux.HandleFunc("POST /api/v1/keys/{keyID}/revoke", s.revokeKey)
	mux.HandleFunc("GET /api/v1/oauth/apps", s.listOAuthApps)
	mux.HandleFunc("POST /api/v1/oauth/apps", s.createOAuthApp)
	mux.Handle("/api/", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		errorJSON(w, http.StatusNotFound, "not_found", "API endpoint was not found")
	}))
	mux.Handle("/api", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		errorJSON(w, http.StatusNotFound, "not_found", "API endpoint was not found")
	}))
	mux.Handle("/v1/", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		errorJSON(w, http.StatusNotFound, "not_found", "API endpoint was not found")
	}))
	mux.Handle("/v1", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		errorJSON(w, http.StatusNotFound, "not_found", "API endpoint was not found")
	}))
	mux.Handle("/", web.Handler())
	return logging(mux, s.logger)
}
func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) ready(w http.ResponseWriter, _ *http.Request) {
	if !s.readyFlag.Load() {
		errorJSON(w, http.StatusServiceUnavailable, "not_ready", "server is still starting")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}
func (s *Server) ingest(w http.ResponseWriter, r *http.Request) {
	key, err := s.authenticate(r)
	if err != nil {
		errorJSON(w, http.StatusUnauthorized, "invalid_api_key", "invalid API key")
		return
	}
	var req struct {
		Spans []domain.Span `json:"spans"`
	}
	if err = json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<20)).Decode(&req); err != nil {
		errorJSON(w, http.StatusBadRequest, "invalid_json", "request body is invalid")
		return
	}
	if len(req.Spans) == 0 {
		errorJSON(w, http.StatusBadRequest, "empty_spans", "at least one span is required")
		return
	}
	now := time.Now().UTC()
	for i := range req.Spans {
		req.Spans[i].ProjectID = key.ProjectID
		if req.Spans[i].ReceivedAt.IsZero() {
			req.Spans[i].ReceivedAt = now
		}
		if err := req.Spans[i].Validate(); err != nil {
			writeValidationError(w, err)
			return
		}
	}
	if err = s.writer.Enqueue(req.Spans); err != nil {
		if errors.Is(err, ingest.ErrFull) {
			errorJSON(w, http.StatusTooManyRequests, "ingest_queue_full", "ingest queue is full")
			return
		}
		errorJSON(w, http.StatusServiceUnavailable, "ingest_unavailable", err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"accepted": len(req.Spans)})
}

func (s *Server) cozeLoopIngest(w http.ResponseWriter, r *http.Request) {
	key, err := s.authenticate(r)
	if err != nil {
		errorJSON(w, http.StatusUnauthorized, "invalid_api_key", "invalid API key")
		return
	}
	var req cozeloop.UploadSpanData
	if err = json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<20)).Decode(&req); err != nil {
		errorJSON(w, http.StatusBadRequest, "invalid_json", "request body is invalid")
		return
	}
	spans, err := req.Map(key.ProjectID, time.Now().UTC())
	if err != nil {
		errorJSON(w, http.StatusBadRequest, "invalid_span", err.Error())
		return
	}
	if len(spans) == 0 {
		errorJSON(w, http.StatusBadRequest, "empty_spans", "at least one span is required")
		return
	}
	for _, span := range spans {
		if err := span.Validate(); err != nil {
			writeValidationError(w, err)
			return
		}
	}
	if err = s.writer.Enqueue(spans); err != nil {
		if errors.Is(err, ingest.ErrFull) {
			errorJSON(w, http.StatusTooManyRequests, "ingest_queue_full", "ingest queue is full")
			return
		}
		errorJSON(w, http.StatusServiceUnavailable, "ingest_unavailable", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"code": 0, "msg": ""})
}

func (s *Server) ingestStats(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	writeJSON(w, http.StatusOK, s.writer.Metrics())
}

func (s *Server) listProjects(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	projects, err := s.meta.ListProjects(r.Context())
	if err != nil {
		errorJSON(w, http.StatusInternalServerError, "storage_error", "could not list projects")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": projects})
}

func (s *Server) createProject(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	var body struct {
		Name    string `json:"name"`
		KeyName string `json:"key_name"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&body); err != nil {
		errorJSON(w, http.StatusBadRequest, "invalid_json", "request body is invalid")
		return
	}
	if strings.TrimSpace(body.Name) == "" || len(body.Name) > 128 {
		errorJSON(w, http.StatusBadRequest, "invalid_project", "name is required and must be at most 128 bytes")
		return
	}
	projectID, err := randomID("proj_", 12)
	if err != nil {
		errorJSON(w, http.StatusInternalServerError, "id_generation_failed", "could not generate project id")
		return
	}
	now := time.Now().UTC()
	project := meta.Project{ID: projectID, Name: strings.TrimSpace(body.Name), CreatedAt: now, UpdatedAt: now}
	key, token, err := s.newKey(projectID, body.KeyName, "project")
	if err != nil {
		errorJSON(w, http.StatusInternalServerError, "key_generation_failed", "could not create API key")
		return
	}
	if err := s.meta.CreateProjectWithAPIKey(r.Context(), project, key); err != nil {
		errorJSON(w, http.StatusConflict, "project_exists", "could not create project")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"project": project, "api_key": keyView{ID: key.ID, ProjectID: key.ProjectID, Name: key.Name, Role: key.Role, Token: token}})
}

func (s *Server) listProjectKeys(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	projectID := r.PathValue("projectID")
	if _, err := s.meta.Project(r.Context(), projectID); err != nil {
		errorJSON(w, http.StatusNotFound, "project_not_found", "project was not found")
		return
	}
	keys, err := s.meta.ListAPIKeys(r.Context(), projectID)
	if err != nil {
		errorJSON(w, http.StatusInternalServerError, "storage_error", "could not list API keys")
		return
	}
	result := make([]keyView, 0, len(keys))
	for _, key := range keys {
		result = append(result, keyView{ID: key.ID, ProjectID: key.ProjectID, Name: key.Name, Role: key.Role, ExpiresAt: key.ExpiresAt, Revoked: key.Revoked, LastUsedAt: key.LastUsedAt})
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": result})
}

func (s *Server) createProjectKey(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	projectID := r.PathValue("projectID")
	if _, err := s.meta.Project(r.Context(), projectID); err != nil {
		errorJSON(w, http.StatusNotFound, "project_not_found", "project was not found")
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
	key, token, err := s.newKey(projectID, body.Name, "project")
	if err != nil {
		errorJSON(w, http.StatusBadRequest, "invalid_key", err.Error())
		return
	}
	if body.ExpiresAt != "" {
		expires, parseErr := time.Parse(time.RFC3339, body.ExpiresAt)
		if parseErr != nil {
			errorJSON(w, http.StatusBadRequest, "invalid_expires_at", "expires_at must be RFC3339")
			return
		}
		if !expires.After(time.Now().UTC()) {
			errorJSON(w, http.StatusBadRequest, "invalid_expires_at", "expires_at must be in the future")
			return
		}
		key.ExpiresAt = &expires
	}
	if err := s.meta.CreateAPIKey(r.Context(), key); err != nil {
		errorJSON(w, http.StatusInternalServerError, "storage_error", "could not save API key")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"api_key": keyView{ID: key.ID, ProjectID: key.ProjectID, Name: key.Name, Role: key.Role, Token: token, ExpiresAt: key.ExpiresAt}})
}

func (s *Server) revokeKey(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	if err := s.meta.RevokeAPIKey(r.Context(), r.PathValue("keyID")); err != nil {
		if errors.Is(err, meta.ErrNotFound) {
			errorJSON(w, http.StatusNotFound, "key_not_found", "API key was not found")
			return
		}
		errorJSON(w, http.StatusInternalServerError, "storage_error", "could not revoke API key")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"revoked": true})
}

type keyView struct {
	ID         string     `json:"id"`
	ProjectID  string     `json:"project_id"`
	Name       string     `json:"name"`
	Role       string     `json:"role"`
	Token      string     `json:"token,omitempty"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	Revoked    bool       `json:"revoked,omitempty"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
}

func (s *Server) requireAdmin(w http.ResponseWriter, r *http.Request) bool {
	key, err := s.authenticate(r)
	if err != nil {
		errorJSON(w, http.StatusUnauthorized, "invalid_api_key", "invalid API key")
		return false
	}
	if key.Role != "admin" {
		errorJSON(w, http.StatusForbidden, "admin_required", "admin API key required")
		return false
	}
	return true
}
func (s *Server) newKey(projectID, name, role string) (meta.APIKey, string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "API Key"
	}
	if len(name) > 128 {
		return meta.APIKey{}, "", errors.New("key name must be at most 128 bytes")
	}
	id, err := randomID("key_", 12)
	if err != nil {
		return meta.APIKey{}, "", err
	}
	var bytes [24]byte
	if _, err = rand.Read(bytes[:]); err != nil {
		return meta.APIKey{}, "", err
	}
	token := "tr_" + hex.EncodeToString(bytes[:])
	return meta.APIKey{ID: id, ProjectID: projectID, Name: strings.TrimSpace(name), Role: role, TokenHash: meta.HashToken(token)}, token, nil
}
func randomID(prefix string, size int) (string, error) {
	bytes := make([]byte, size)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return prefix + hex.EncodeToString(bytes), nil
}
func (s *Server) getTrace(w http.ResponseWriter, r *http.Request) {
	key, err := s.authenticate(r)
	if err != nil {
		errorJSON(w, http.StatusUnauthorized, "invalid_api_key", "invalid API key")
		return
	}
	traceID := strings.TrimPrefix(r.URL.Path, "/api/v1/traces/")
	if traceID == "" || strings.Contains(traceID, "/") {
		errorJSON(w, http.StatusBadRequest, "invalid_trace_id", "trace id is required")
		return
	}
	limit := 100
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, parseErr := strconv.Atoi(raw)
		if parseErr != nil || parsed < 1 || parsed > 1000 {
			errorJSON(w, http.StatusBadRequest, "invalid_limit", "limit must be between 1 and 1000")
			return
		}
		limit = parsed
	}
	page, err := s.traces.GetTracePage(r.Context(), key.ProjectID, traceID, r.URL.Query().Get("cursor"), limit)
	if err != nil {
		s.logger.Error("get trace", "error", err)
		errorJSON(w, http.StatusInternalServerError, "storage_error", "could not read trace")
		return
	}
	if len(page.Spans) == 0 {
		errorJSON(w, http.StatusNotFound, "trace_not_found", "trace was not found")
		return
	}
	response := struct {
		TraceID    string        `json:"trace_id"`
		Spans      []domain.Span `json:"spans"`
		NextCursor string        `json:"next_cursor,omitempty"`
	}{TraceID: traceID, Spans: page.Spans, NextCursor: page.NextCursor}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) listTraces(w http.ResponseWriter, r *http.Request) {
	key, err := s.authenticate(r)
	if err != nil {
		errorJSON(w, http.StatusUnauthorized, "invalid_api_key", "invalid API key")
		return
	}
	q := domain.Query{ProjectID: key.ProjectID, Status: r.URL.Query().Get("status"), Kind: r.URL.Query().Get("kind"), Name: r.URL.Query().Get("name"), TraceID: r.URL.Query().Get("trace_id"), Cursor: r.URL.Query().Get("cursor")}
	for name, target := range map[string]**time.Time{"start_time": &q.StartTime, "end_time": &q.EndTime} {
		if raw := r.URL.Query().Get(name); raw != "" {
			parsed, parseErr := time.Parse(time.RFC3339, raw)
			if parseErr != nil {
				errorJSON(w, http.StatusBadRequest, "invalid_"+name, name+" must be RFC3339")
				return
			}
			*target = &parsed
		}
	}
	for name, target := range map[string]*time.Duration{"min_duration_ms": &q.MinDuration, "max_duration_ms": &q.MaxDuration} {
		if raw := r.URL.Query().Get(name); raw != "" {
			value, parseErr := strconv.ParseInt(raw, 10, 64)
			if parseErr != nil || value < 0 {
				errorJSON(w, http.StatusBadRequest, "invalid_"+name, name+" must be a non-negative integer")
				return
			}
			*target = time.Duration(value) * time.Millisecond
		}
	}
	for name, target := range map[string]*int64{"min_tokens": &q.MinTokens, "max_tokens": &q.MaxTokens} {
		if raw := r.URL.Query().Get(name); raw != "" {
			value, parseErr := strconv.ParseInt(raw, 10, 64)
			if parseErr != nil || value < 0 {
				errorJSON(w, http.StatusBadRequest, "invalid_"+name, name+" must be a non-negative integer")
				return
			}
			*target = value
		}
	}
	if raw := r.URL.Query().Get("limit"); raw != "" {
		q.Limit, err = strconv.Atoi(raw)
		if err != nil || q.Limit < 1 || q.Limit > 100 {
			errorJSON(w, http.StatusBadRequest, "invalid_limit", "limit must be between 1 and 100")
			return
		}
	}
	page, err := s.traces.ListTraces(r.Context(), q)
	if err != nil {
		errorJSON(w, http.StatusBadRequest, "invalid_query", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, page)
}

func (s *Server) dashboard(w http.ResponseWriter, r *http.Request) {
	key, err := s.authenticate(r)
	if err != nil {
		errorJSON(w, http.StatusUnauthorized, "invalid_api_key", "invalid API key")
		return
	}
	q := domain.MetricsQuery{ProjectID: key.ProjectID}
	for name, target := range map[string]*time.Time{"start_time": &q.StartTime, "end_time": &q.EndTime} {
		if raw := r.URL.Query().Get(name); raw != "" {
			parsed, parseErr := time.Parse(time.RFC3339, raw)
			if parseErr != nil {
				errorJSON(w, http.StatusBadRequest, "invalid_"+name, name+" must be RFC3339")
				return
			}
			*target = parsed
		}
	}
	metrics, err := s.traces.Metrics(r.Context(), q)
	if err != nil {
		errorJSON(w, http.StatusBadRequest, "invalid_query", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, metrics)
}
func (s *Server) authenticate(r *http.Request) (meta.APIKey, error) {
	h := r.Header.Get("Authorization")
	if !strings.HasPrefix(h, "Bearer ") {
		return meta.APIKey{}, meta.ErrNotFound
	}
	return s.meta.Authenticate(r.Context(), strings.TrimSpace(strings.TrimPrefix(h, "Bearer ")))
}
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func errorJSON(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}
func writeValidationError(w http.ResponseWriter, err error) {
	if errors.Is(err, domain.ErrPayloadTooLarge) {
		errorJSON(w, http.StatusRequestEntityTooLarge, "payload_too_large", "span payload exceeds configured limits")
		return
	}
	errorJSON(w, http.StatusBadRequest, "invalid_span", err.Error())
}
func logging(next http.Handler, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(recorder, r)
		logger.Info("http request", "method", r.Method, "path", r.URL.Path, "status", recorder.status, "remote_addr", r.RemoteAddr, "duration", time.Since(start))
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(body []byte) (int, error) {
	return r.ResponseWriter.Write(body)
}
