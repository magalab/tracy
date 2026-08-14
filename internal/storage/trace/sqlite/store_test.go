package sqlite

import (
	"context"
	"testing"
	"time"

	sqlite "github.com/panda/tracy/internal/storage/sqlite"
	domain "github.com/panda/tracy/internal/trace"
)

func TestTraceSummaryAndFilters(t *testing.T) {
	ctx := context.Background()
	db, err := sqlite.Open(ctx, t.TempDir()+"/traces.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	store := NewStore(db)
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	spans := []domain.Span{{ProjectID: "p", TraceID: "trace-ok", SpanID: "root", Name: "model-call", Kind: "model", StartTime: base, ReceivedAt: base, Duration: 2 * time.Millisecond, Status: "ok", InputTokens: 2, OutputTokens: 3}, {ProjectID: "p", TraceID: "trace-ok", SpanID: "child", ParentSpanID: "root", Name: "tool-call", Kind: "tool", StartTime: base.Add(time.Millisecond), ReceivedAt: base, Duration: time.Millisecond, Status: "ok", InputTokens: 1}, {ProjectID: "p", TraceID: "trace-error", SpanID: "error", Name: "failed", Kind: "tool", StartTime: base.Add(time.Hour), ReceivedAt: base, Duration: 5 * time.Millisecond, Status: "error", InputTokens: 20}}
	if err := store.Append(ctx, spans); err != nil {
		t.Fatal(err)
	}
	page, err := store.ListTraces(ctx, domain.Query{ProjectID: "p", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 2 || page.Items[0].TraceID != "trace-error" || page.Items[0].SpanCount != 1 {
		t.Fatalf("page=%+v", page)
	}
	page, err = store.ListTraces(ctx, domain.Query{ProjectID: "p", Status: "error", MinTokens: 20, MinDuration: 4 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].TraceID != "trace-error" {
		t.Fatalf("filtered page=%+v", page)
	}
	page, err = store.ListTraces(ctx, domain.Query{ProjectID: "p", Kind: "model", StartTime: &base})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].TraceID != "trace-ok" {
		t.Fatalf("kind page=%+v", page)
	}
	spans[0].Status = "error"
	if err := store.Append(ctx, spans[:1]); err != nil {
		t.Fatal(err)
	}
	page, err = store.ListTraces(ctx, domain.Query{ProjectID: "p", TraceID: "trace-ok"})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].Status != "error" || page.Items[0].SpanCount != 2 {
		t.Fatalf("updated summary=%+v", page)
	}
}
