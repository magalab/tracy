package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Server   ServerConfig
	Metadata SQLiteConfig
	Trace    TraceConfig
}

type ServerConfig struct{ Addr string }
type SQLiteConfig struct{ Path string }
type TraceConfig struct {
	SQLite SQLiteConfig
	Writer WriterConfig
}
type WriterConfig struct {
	BatchSize     int
	FlushInterval time.Duration
	QueueSize     int
}

func Load() (Config, error) {
	c := Config{
		Server:   ServerConfig{Addr: env("TRACY_ADDR", ":8080")},
		Metadata: SQLiteConfig{Path: env("TRACY_META_DB", "./data/meta.db")},
		Trace: TraceConfig{
			SQLite: SQLiteConfig{Path: env("TRACY_TRACE_DB", "./data/traces.db")},
			Writer: WriterConfig{BatchSize: envInt("TRACY_BATCH_SIZE", 128), FlushInterval: envDuration("TRACY_FLUSH_INTERVAL", 50*time.Millisecond), QueueSize: envInt("TRACY_QUEUE_SIZE", 4096)},
		},
	}
	if c.Server.Addr == "" || c.Metadata.Path == "" || c.Trace.SQLite.Path == "" {
		return Config{}, fmt.Errorf("server address and database paths are required")
	}
	if c.Trace.Writer.BatchSize < 1 || c.Trace.Writer.QueueSize < 1 || c.Trace.Writer.FlushInterval <= 0 {
		return Config{}, fmt.Errorf("invalid trace writer configuration")
	}
	return c, nil
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
func envInt(key string, fallback int) int {
	v, err := strconv.Atoi(os.Getenv(key))
	if err != nil || v < 1 {
		return fallback
	}
	return v
}
func envDuration(key string, fallback time.Duration) time.Duration {
	v, err := time.ParseDuration(os.Getenv(key))
	if err != nil {
		return fallback
	}
	return v
}
