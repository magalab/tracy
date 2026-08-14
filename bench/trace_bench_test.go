package bench

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"sort"
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
	dbPath := t.TempDir() + "/traces.db"
	db, err := sqlite.Open(context.Background(), dbPath)
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
	ingestDuration := time.Since(start)
	querySamples := make([]int64, 0, 100)
	for i := 0; i < 100; i++ {
		sampleStart := time.Now()
		if _, err := store.ListTraces(context.Background(), domain.Query{ProjectID: "bench", Limit: 50}); err != nil {
			t.Fatal(err)
		}
		querySamples = append(querySamples, time.Since(sampleStart).Nanoseconds())
	}
	sort.Slice(querySamples, func(i, j int) bool { return querySamples[i] < querySamples[j] })
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	dbSize, walSize := fileSize(dbPath), fileSize(dbPath+"-wal")
	t.Logf("spans=%d ingest=%s ingest_rate=%0.1f spans/sec list_p50=%s list_p95=%s list_p99=%s db=%s wal=%s alloc=%d sys=%d", count, ingestDuration, float64(count)/ingestDuration.Seconds(), time.Duration(percentile(querySamples, .50)), time.Duration(percentile(querySamples, .95)), time.Duration(percentile(querySamples, .99)), humanBytes(dbSize), humanBytes(walSize), mem.Alloc, mem.Sys)
}

func fileSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}
func humanBytes(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%dB", size)
	}
	return fmt.Sprintf("%.2fMiB", float64(size)/(1024*1024))
}

func makeSpans(count, offset int) []domain.Span {
	now := time.Now().UTC()
	result := make([]domain.Span, count)
	for i := range result {
		result[i] = domain.Span{ProjectID: "bench", TraceID: fmt.Sprintf("trace-%d", (offset+i)/10), SpanID: fmt.Sprintf("span-%d", offset+i), Name: "benchmark", Kind: "custom", StartTime: now, ReceivedAt: now, Duration: time.Millisecond, Status: "ok", Input: "input", Output: "output", Attributes: map[string]any{"iteration": offset + i}}
	}
	return result
}
