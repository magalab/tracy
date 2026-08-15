package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/panda/tracy/internal/api"
	"github.com/panda/tracy/internal/config"
	"github.com/panda/tracy/internal/ingest"
	"github.com/panda/tracy/internal/storage/meta"
	sqlite "github.com/panda/tracy/internal/storage/sqlite"
	tracestore "github.com/panda/tracy/internal/storage/trace/sqlite"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	ctx := context.Background()
	cfg, err := config.Load()
	must(logger, err)
	metaDB, err := sqlite.Open(ctx, cfg.Metadata.Path)
	must(logger, err)
	defer metaDB.Close()
	traceDB, err := sqlite.Open(ctx, cfg.Trace.SQLite.Path)
	must(logger, err)
	defer traceDB.Close()
	metaStore := meta.NewStore(metaDB)
	must(logger, metaStore.Migrate(ctx))
	cleanupCtx, cleanupCancel := context.WithCancel(ctx)
	cleanupDone := make(chan struct{})
	go func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		defer close(cleanupDone)
		_ = metaStore.CleanupExpired(cleanupCtx, time.Now().UTC())
		for {
			select {
			case <-ticker.C:
				_ = metaStore.CleanupExpired(cleanupCtx, time.Now().UTC())
			case <-cleanupCtx.Done():
				return
			}
		}
	}()
	traceStore := tracestore.NewStore(traceDB)
	must(logger, traceStore.Migrate(ctx))
	ensureDefault(ctx, metaStore, logger)
	writer := ingest.NewWriterWithBytes(traceStore, cfg.Trace.Writer.BatchSize, cfg.Trace.Writer.FlushInterval, cfg.Trace.Writer.QueueSize, cfg.Trace.Writer.QueueBytes)
	apiServer := api.NewServer(metaStore, writer, traceStore, logger, cfg.Server.TrustedProxies)
	apiServer.MarkReady()
	server := &http.Server{Addr: cfg.Server.Addr, Handler: apiServer.Routes(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second}
	go func() {
		logger.Info("server started", "addr", cfg.Server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server stopped", "error", err)
			os.Exit(1)
		}
	}()
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdownCtx)
	_ = writer.Close(shutdownCtx)
	cleanupCancel()
	<-cleanupDone
}
func ensureDefault(ctx context.Context, s *meta.Store, logger *slog.Logger) {
	_, err := s.Project(ctx, "default")
	if err == nil {
		must(logger, s.EnsureAdmin(ctx, "default-key"))
		ensureDefaultUser(ctx, s, logger)
		return
	}
	if !errors.Is(err, meta.ErrNotFound) {
		must(logger, err)
	}
	now := time.Now().UTC()
	must(logger, s.CreateProject(ctx, meta.Project{ID: "default", Name: "Default Project", CreatedAt: now, UpdatedAt: now}))
	token := os.Getenv("TRACY_API_KEY")
	if token == "" {
		var b [24]byte
		must(logger, func() error { _, e := rand.Read(b[:]); return e }())
		token = "tr_" + hex.EncodeToString(b[:])
		logger.Info("created initial API key; store it securely", "api_key", token)
	}
	must(logger, s.CreateAPIKey(ctx, meta.APIKey{ID: "default-key", ProjectID: "default", Name: "Default API Key", Role: "admin", TokenHash: meta.HashToken(token)}))
	ensureDefaultUser(ctx, s, logger)
}

func ensureDefaultUser(ctx context.Context, s *meta.Store, logger *slog.Logger) {
	email := os.Getenv("TRACY_ADMIN_EMAIL")
	if email == "" {
		email = "admin@localhost"
	}
	if user, err := s.UserByEmail(ctx, email); err == nil {
		must(logger, s.AddWorkspaceMember(ctx, meta.WorkspaceMember{WorkspaceID: "default", UserID: user.ID, Role: "owner", CreatedAt: time.Now().UTC()}))
		return
	} else if !errors.Is(err, meta.ErrNotFound) {
		must(logger, err)
	}
	password := os.Getenv("TRACY_ADMIN_PASSWORD")
	if password == "" {
		var b [18]byte
		must(logger, func() error { _, e := rand.Read(b[:]); return e }())
		password = hex.EncodeToString(b[:])
		logger.Info("created initial user; store credentials securely", "email", email, "password", password)
	}
	hash, err := meta.HashPassword(password)
	must(logger, err)
	now := time.Now().UTC()
	must(logger, s.CreateUser(ctx, meta.User{ID: "default-user", Email: email, Name: "Admin", PasswordHash: hash, CreatedAt: now}))
	must(logger, s.AddWorkspaceMember(ctx, meta.WorkspaceMember{WorkspaceID: "default", UserID: "default-user", Role: "owner", CreatedAt: now}))
}
func must(logger *slog.Logger, err error) {
	if err != nil {
		logger.Error("fatal", "error", err)
		os.Exit(1)
	}
}
