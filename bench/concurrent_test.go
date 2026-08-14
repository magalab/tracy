package bench

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/panda/tracy/internal/storage/sqlite"
	tracestore "github.com/panda/tracy/internal/storage/trace/sqlite"
)

func TestConcurrentReadWrite(t *testing.T) {
	seconds := 0
	if raw := os.Getenv("TRACY_BENCH_CONCURRENT"); raw != "" {
		seconds, _ = strconv.Atoi(raw)
	}
	if seconds < 1 {
		t.Skip("set TRACY_BENCH_CONCURRENT to a duration in seconds")
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(seconds)*time.Second)
	defer cancel()
	var next atomic.Int64
	var writes, reads atomic.Uint64
	var latenciesMu sync.Mutex
	latencies := make([]int64, 0, 10000)
	writerDone := make(chan struct{})
	go func() {
		defer close(writerDone)
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			offset := int(next.Add(256) - 256)
			if err := store.Append(ctx, makeSpans(256, offset)); err != nil {
				if ctx.Err() != nil {
					return
				}
				t.Errorf("append: %v", err)
				return
			}
			writes.Add(256)
		}
	}()
	var readers sync.WaitGroup
	for i := 0; i < 4; i++ {
		readers.Add(1)
		go func() {
			defer readers.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}
				start := time.Now()
				_, err := store.GetTrace(ctx, "bench", fmt.Sprintf("trace-%d", next.Load()/10%256))
				if err != nil {
					if ctx.Err() != nil {
						return
					}
					t.Errorf("get trace: %v", err)
					return
				}
				latency := time.Since(start).Nanoseconds()
				latenciesMu.Lock()
				latencies = append(latencies, latency)
				latenciesMu.Unlock()
				reads.Add(1)
			}
		}()
	}
	<-writerDone
	readers.Wait()
	if len(latencies) == 0 {
		t.Fatal("no read samples")
	}
	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
	t.Logf("duration=%ds writes=%d reads=%d read_latency_p50=%s read_latency_p95=%s read_latency_p99=%s", seconds, writes.Load(), reads.Load(), time.Duration(percentile(latencies, .50)), time.Duration(percentile(latencies, .95)), time.Duration(percentile(latencies, .99)))
}

func percentile(values []int64, p float64) int64 {
	index := int(float64(len(values)-1) * p)
	if index < 0 {
		index = 0
	}
	if index >= len(values) {
		index = len(values) - 1
	}
	return values[index]
}
