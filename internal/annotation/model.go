package annotation

import "time"

type Annotation struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"project_id"`
	TraceID   string    `json:"trace_id"`
	SpanID    string    `json:"span_id,omitempty"`
	Key       string    `json:"key"`
	Score     *float64  `json:"score,omitempty"`
	Label     string    `json:"label,omitempty"`
	Comment   string    `json:"comment,omitempty"`
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
