package ingest

import (
	"context"
	"errors"
	"testing"
	"time"

	domain "github.com/panda/tracy/internal/trace"
)

type fakeStore struct {
	appends int
	fail    bool
}

func (f *fakeStore) Append(context.Context, []domain.Span) error {
	f.appends++
	if f.fail {
		return errors.New("write failed")
	}
	return nil
}
func (f *fakeStore) GetTrace(context.Context, string, string) ([]domain.Span, error) { return nil, nil }
func (f *fakeStore) ListTraces(context.Context, domain.Query) (domain.Page, error) {
	return domain.Page{}, nil
}
func (f *fakeStore) Metrics(context.Context, domain.MetricsQuery) (domain.Metrics, error) {
	return domain.Metrics{}, nil
}

func TestEnqueueRejectsWholeBatchWhenFull(t *testing.T) {
	store := &fakeStore{}
	writer := NewWriter(store, 1, time.Hour, 1)
	defer writer.Close(context.Background())
	span := domain.Span{ProjectID: "p", TraceID: "t", SpanID: "s", Name: "n"}
	if err := writer.Enqueue([]domain.Span{span}); err != nil {
		t.Fatal(err)
	}
	if err := writer.Enqueue([]domain.Span{span}); !errors.Is(err, ErrFull) {
		t.Fatalf("error=%v", err)
	}
	stats := writer.Metrics()
	if stats.Accepted != 1 || stats.Rejected != 1 || stats.Dropped != 1 {
		t.Fatalf("stats=%+v", stats)
	}
}

func TestWriterReportsWriteFailure(t *testing.T) {
	store := &fakeStore{fail: true}
	writer := NewWriter(store, 1, time.Hour, 2)
	defer writer.Close(context.Background())
	span := domain.Span{ProjectID: "p", TraceID: "t", SpanID: "s", Name: "n"}
	if err := writer.Enqueue([]domain.Span{span}); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(time.Second)
	for writer.Metrics().WriteErrors == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	stats := writer.Metrics()
	if stats.WriteErrors != 1 || stats.Dropped != 1 || stats.LastError == "" {
		t.Fatalf("stats=%+v", stats)
	}
}
