package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/panda/tracy/compat/cozeloop"
	"github.com/panda/tracy/internal/ingest"
	"github.com/panda/tracy/internal/storage/meta"
	domain "github.com/panda/tracy/internal/trace"
	"github.com/panda/tracy/internal/web"
)

type Server struct {
	meta   *meta.Store
	writer *ingest.Writer
	traces interface {
		GetTrace(ctx context.Context, projectID, traceID string) ([]domain.Span, error)
		ListTraces(ctx context.Context, query domain.Query) (domain.Page, error)
	}
	logger *slog.Logger
}

func NewServer(m *meta.Store, w *ingest.Writer, t interface {
	GetTrace(context.Context, string, string) ([]domain.Span, error)
	ListTraces(context.Context, domain.Query) (domain.Page, error)
}, logger *slog.Logger) *Server {
	return &Server{meta: m, writer: w, traces: t, logger: logger}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.health)
	mux.HandleFunc("GET /readyz", s.ready)
	mux.HandleFunc("POST /api/v1/ingest", s.ingest)
	mux.HandleFunc("POST /v1/loop/traces/ingest", s.cozeLoopIngest)
	mux.HandleFunc("GET /api/v1/traces/", s.getTrace)
	mux.HandleFunc("GET /api/v1/traces", s.listTraces)
	mux.Handle("/", web.Handler())
	return logging(mux, s.logger)
}
func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
func (s *Server) ready(w http.ResponseWriter, _ *http.Request) {
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
		if req.Spans[i].TraceID == "" || req.Spans[i].SpanID == "" || req.Spans[i].Name == "" {
			errorJSON(w, http.StatusBadRequest, "invalid_span", "trace_id, span_id and name are required")
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
	spans, err := s.traces.GetTrace(r.Context(), key.ProjectID, traceID)
	if err != nil {
		s.logger.Error("get trace", "error", err)
		errorJSON(w, http.StatusInternalServerError, "storage_error", "could not read trace")
		return
	}
	if len(spans) == 0 {
		errorJSON(w, http.StatusNotFound, "trace_not_found", "trace was not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"trace_id": traceID, "spans": spans})
}

func (s *Server) listTraces(w http.ResponseWriter, r *http.Request) {
	key, err := s.authenticate(r)
	if err != nil {
		errorJSON(w, http.StatusUnauthorized, "invalid_api_key", "invalid API key")
		return
	}
	q := domain.Query{ProjectID: key.ProjectID, Status: r.URL.Query().Get("status"), Kind: r.URL.Query().Get("kind"), Name: r.URL.Query().Get("name"), TraceID: r.URL.Query().Get("trace_id"), Cursor: r.URL.Query().Get("cursor")}
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
func logging(next http.Handler, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		logger.Info("http request", "method", r.Method, "path", r.URL.Path, "duration", time.Since(start))
	})
}
