package bench

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/panda/tracy/internal/storage/sqlite"
	tracestore "github.com/panda/tracy/internal/storage/trace/sqlite"
	domain "github.com/panda/tracy/internal/trace"
)

func BenchmarkAppendBatch(b *testing.B) {
	dbPath := b.TempDir() + "/traces.db"
	db, err := sqlite.Open(context.Background(), dbPath)
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()
	store := tracestore.NewStore(db)
	if err := store.Migrate(context.Background()); err != nil {
		b.Fatal(err)
	}
	spans := makeSpans(256, 0)
	b.ReportMetric(float64(len(spans)), "spans/batch")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := range spans {
			spans[j].TraceID = fmt.Sprintf("trace-%d", i)
			spans[j].SpanID = fmt.Sprintf("span-%d", j)
		}
		if err := store.Append(context.Background(), spans); err != nil {
			b.Fatal(err)
		}
	}
	b.ReportMetric(float64(b.N*len(spans))/b.Elapsed().Seconds(), "spans/sec")
}

func BenchmarkGetTrace(b *testing.B) {
	dbPath := b.TempDir() + "/traces.db"
	db, err := sqlite.Open(context.Background(), dbPath)
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()
	store := tracestore.NewStore(db)
	if err := store.Migrate(context.Background()); err != nil {
		b.Fatal(err)
	}
	spans := makeSpans(100, 0)
	for i := range spans {
		spans[i].TraceID = "trace-0"
	}
	if err := store.Append(context.Background(), spans); err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := store.GetTrace(context.Background(), "bench", "trace-0")
		if err != nil {
			b.Fatal(err)
		}
		if len(result) != len(spans) {
			b.Fatalf("span count=%d", len(result))
		}
	}
}

func TestWorkloadSmoke(t *testing.T) {
	raw := os.Getenv("TRACY_BENCH_SPANS")
	if raw == "" {
		t.Skip("set TRACY_BENCH_SPANS to run the ingest workload")
	}
	count, err := strconv.Atoi(raw)
	if err != nil || count < 1 {
		t.Fatalf("invalid TRACY_BENCH_SPANS=%q", raw)
	}
	db, err := sqlite.Open(context.Background(), t.TempDir()+"/traces.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	store := tracestore.NewStore(db)
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	const batchSize = 256
	for offset := 0; offset < count; offset += batchSize {
		end := offset + batchSize
		if end > count {
			end = count
		}
		if err := store.Append(context.Background(), makeSpans(end-offset, offset)); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("appended %d spans in %s (%0.1f spans/sec)", count, time.Since(start), float64(count)/time.Since(start).Seconds())
}

func makeSpans(count, offset int) []domain.Span {
	now := time.Now().UTC()
	result := make([]domain.Span, count)
	for i := range result {
		result[i] = domain.Span{ProjectID: "bench", TraceID: fmt.Sprintf("trace-%d", (offset+i)/10), SpanID: fmt.Sprintf("span-%d", offset+i), Name: "benchmark", Kind: "custom", StartTime: now, ReceivedAt: now, Duration: time.Millisecond, Status: "ok", Input: "input", Output: "output", Attributes: map[string]any{"iteration": offset + i}}
	}
	return result
}
