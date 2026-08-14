package ingest

import (
	"context"
	"errors"
	"sync"
	"time"

	store "github.com/panda/tracy/internal/storage/trace"
	domain "github.com/panda/tracy/internal/trace"
)

var ErrFull = errors.New("trace ingest queue is full")

type Writer struct {
	store     store.Store
	queue     chan domain.Span
	batchSize int
	interval  time.Duration
	done      chan struct{}
	wg        sync.WaitGroup
	closeOnce sync.Once
}

func NewWriter(s store.Store, batchSize int, interval time.Duration, queueSize int) *Writer {
	w := &Writer{store: s, queue: make(chan domain.Span, queueSize), batchSize: batchSize, interval: interval, done: make(chan struct{})}
	w.wg.Add(1)
	go w.run()
	return w
}
func (w *Writer) Enqueue(spans []domain.Span) error {
	for _, sp := range spans {
		select {
		case w.queue <- sp:
		case <-w.done:
			return errors.New("trace writer is closed")
		default:
			return ErrFull
		}
	}
	return nil
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
		if len(batch) > 0 {
			_ = w.store.Append(context.Background(), batch)
			batch = batch[:0]
		}
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
