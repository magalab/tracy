package ingest

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	store "github.com/panda/tracy/internal/storage/trace"
	domain "github.com/panda/tracy/internal/trace"
)

var ErrFull = errors.New("trace ingest queue is full")

type Metrics struct {
	Accepted    uint64 `json:"accepted"`
	Rejected    uint64 `json:"rejected"`
	Dropped     uint64 `json:"dropped"`
	Written     uint64 `json:"written"`
	WriteErrors uint64 `json:"write_errors"`
	QueueDepth  int    `json:"queue_depth"`
	LastError   string `json:"last_error,omitempty"`
}

type errorState struct{ message string }

type Writer struct {
	store       store.Store
	queue       chan domain.Span
	batchSize   int
	interval    time.Duration
	done        chan struct{}
	wg          sync.WaitGroup
	closeOnce   sync.Once
	enqueueMu   sync.Mutex
	accepted    atomic.Uint64
	rejected    atomic.Uint64
	dropped     atomic.Uint64
	written     atomic.Uint64
	writeErrors atomic.Uint64
	lastError   atomic.Pointer[errorState]
}

func NewWriter(s store.Store, batchSize int, interval time.Duration, queueSize int) *Writer {
	w := &Writer{store: s, queue: make(chan domain.Span, queueSize), batchSize: batchSize, interval: interval, done: make(chan struct{})}
	w.wg.Add(1)
	go w.run()
	return w
}
func (w *Writer) Enqueue(spans []domain.Span) error {
	if len(spans) == 0 {
		return nil
	}
	w.enqueueMu.Lock()
	defer w.enqueueMu.Unlock()
	if len(spans) > cap(w.queue)-len(w.queue) {
		w.rejected.Add(1)
		w.dropped.Add(uint64(len(spans)))
		return ErrFull
	}
	for _, sp := range spans {
		select {
		case w.queue <- sp:
		case <-w.done:
			w.rejected.Add(1)
			w.dropped.Add(uint64(len(spans)))
			return errors.New("trace writer is closed")
		}
	}
	w.accepted.Add(uint64(len(spans)))
	return nil
}

func (w *Writer) Metrics() Metrics {
	result := Metrics{Accepted: w.accepted.Load(), Rejected: w.rejected.Load(), Dropped: w.dropped.Load(), Written: w.written.Load(), WriteErrors: w.writeErrors.Load(), QueueDepth: len(w.queue)}
	if err := w.lastError.Load(); err != nil {
		result.LastError = err.message
	}
	return result
}
func (w *Writer) Close(ctx context.Context) error {
	w.closeOnce.Do(func() { close(w.done) })
	finished := make(chan struct{})
	go func() { w.wg.Wait(); close(finished) }()
	select {
	case <-finished:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
func (w *Writer) run() {
	defer w.wg.Done()
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	batch := make([]domain.Span, 0, w.batchSize)
	flush := func() {
		if len(batch) == 0 {
			return
		}
		var err error
		for attempt := 0; attempt < 3; attempt++ {
			err = w.store.Append(context.Background(), batch)
			if err == nil {
				w.written.Add(uint64(len(batch)))
				batch = batch[:0]
				return
			}
			time.Sleep(time.Duration(attempt+1) * 10 * time.Millisecond)
		}
		w.lastError.Store(&errorState{message: fmt.Sprintf("trace batch of %d spans: %v", len(batch), err)})
		w.writeErrors.Add(1)
		w.dropped.Add(uint64(len(batch)))
		batch = batch[:0]
	}
	for {
		select {
		case sp := <-w.queue:
			batch = append(batch, sp)
			if len(batch) >= w.batchSize {
				flush()
			}
		case <-ticker.C:
			flush()
		case <-w.done:
			for {
				select {
				case sp := <-w.queue:
					batch = append(batch, sp)
					if len(batch) >= w.batchSize {
						flush()
					}
				default:
					flush()
					return
				}
			}
		}
	}
}
