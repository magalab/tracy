package ingest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	store "github.com/panda/tracy/internal/storage/trace"
	domain "github.com/panda/tracy/internal/trace"
)

var ErrFull = errors.New("trace ingest queue is full")
var ErrClosed = errors.New("trace writer is closed")

type Metrics struct {
	Accepted    uint64 `json:"accepted"`
	Rejected    uint64 `json:"rejected"`
	Dropped     uint64 `json:"dropped"`
	Written     uint64 `json:"written"`
	WriteErrors uint64 `json:"write_errors"`
	QueueDepth  int    `json:"queue_depth"`
	QueueBytes  int64  `json:"queue_bytes"`
	LastError   string `json:"last_error,omitempty"`
}

type errorState struct{ message string }

type Writer struct {
	store         store.Store
	queue         chan domain.Span
	batchSize     int
	interval      time.Duration
	done          chan struct{}
	wg            sync.WaitGroup
	closeOnce     sync.Once
	enqueueMu     sync.Mutex
	accepted      atomic.Uint64
	rejected      atomic.Uint64
	dropped       atomic.Uint64
	written       atomic.Uint64
	writeErrors   atomic.Uint64
	lastError     atomic.Pointer[errorState]
	inFlightBytes atomic.Int64
	maxQueueBytes int64
}

func NewWriter(s store.Store, batchSize int, interval time.Duration, queueSize int) *Writer {
	const defaultBytesPerQueueItem = 2 << 20
	return NewWriterWithBytes(s, batchSize, interval, queueSize, int64(queueSize)*defaultBytesPerQueueItem)
}

func NewWriterWithBytes(s store.Store, batchSize int, interval time.Duration, queueSize int, maxQueueBytes int64) *Writer {
	w := &Writer{store: s, queue: make(chan domain.Span, queueSize), batchSize: batchSize, interval: interval, done: make(chan struct{}), maxQueueBytes: maxQueueBytes}
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
	select {
	case <-w.done:
		w.rejected.Add(1)
		w.dropped.Add(uint64(len(spans)))
		return ErrClosed
	default:
	}
	bytes := int64(0)
	for _, span := range spans {
		bytes += spanSize(span)
	}
	if bytes > w.maxQueueBytes || w.inFlightBytes.Load()+bytes > w.maxQueueBytes || len(spans) > cap(w.queue)-len(w.queue) {
		w.rejected.Add(1)
		w.dropped.Add(uint64(len(spans)))
		return ErrFull
	}
	for _, sp := range spans {
		w.queue <- sp
	}
	w.inFlightBytes.Add(bytes)
	w.accepted.Add(uint64(len(spans)))
	return nil
}

func (w *Writer) Metrics() Metrics {
	result := Metrics{Accepted: w.accepted.Load(), Rejected: w.rejected.Load(), Dropped: w.dropped.Load(), Written: w.written.Load(), WriteErrors: w.writeErrors.Load(), QueueDepth: len(w.queue), QueueBytes: w.inFlightBytes.Load()}
	if err := w.lastError.Load(); err != nil {
		result.LastError = err.message
	}
	return result
}
func (w *Writer) Close(ctx context.Context) error {
	w.enqueueMu.Lock()
	w.closeOnce.Do(func() { close(w.done) })
	w.enqueueMu.Unlock()
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
				w.lastError.Store(nil)
				w.inFlightBytes.Add(-batchSize(batch))
				batch = batch[:0]
				return
			}
			time.Sleep(time.Duration(attempt+1) * 10 * time.Millisecond)
		}
		w.lastError.Store(&errorState{message: fmt.Sprintf("trace batch of %d spans: %v", len(batch), err)})
		w.writeErrors.Add(1)
		w.dropped.Add(uint64(len(batch)))
		w.inFlightBytes.Add(-batchSize(batch))
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

func spanSize(span domain.Span) int64 {
	attributes, _ := json.Marshal(span.Attributes)
	return int64(len(span.ProjectID) + len(span.TraceID) + len(span.SpanID) + len(span.ParentSpanID) + len(span.Name) + len(span.Kind) + len(span.Status) + len(span.StatusMessage) + len(span.Input) + len(span.Output) + len(attributes) + 256)
}

func batchSize(spans []domain.Span) int64 {
	var total int64
	for _, span := range spans {
		total += spanSize(span)
	}
	return total
}
