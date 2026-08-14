package trace

import (
	"context"
	"errors"

	domain "github.com/panda/tracy/internal/trace"
)

var ErrNotFound = errors.New("trace not found")

type Store interface {
	Append(ctx context.Context, spans []domain.Span) error
	GetTrace(ctx context.Context, projectID, traceID string) ([]domain.Span, error)
	ListTraces(ctx context.Context, query domain.Query) (domain.Page, error)
}
