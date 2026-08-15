package trace

import (
	"context"
	"errors"
	domain "github.com/panda/tracy/internal/trace"
)

var ErrTooLarge = errors.New("trace exceeds response limits")

type TracePage struct {
	Spans      []domain.Span
	NextCursor string
}

type Store interface {
	Append(ctx context.Context, spans []domain.Span) error
	GetTrace(ctx context.Context, workspaceID, traceID string) ([]domain.Span, error)
	GetTracePage(ctx context.Context, workspaceID, traceID, cursor string, limit int) (TracePage, error)
	ListTraces(ctx context.Context, query domain.Query) (domain.Page, error)
	Metrics(ctx context.Context, query domain.MetricsQuery) (domain.Metrics, error)
}
