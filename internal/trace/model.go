package trace

import "time"

type Span struct {
	ProjectID     string         `json:"project_id"`
	TraceID       string         `json:"trace_id"`
	SpanID        string         `json:"span_id"`
	ParentSpanID  string         `json:"parent_span_id,omitempty"`
	Name          string         `json:"name"`
	Kind          string         `json:"kind"`
	StartTime     time.Time      `json:"start_time"`
	ReceivedAt    time.Time      `json:"received_at"`
	Duration      time.Duration  `json:"duration"`
	Status        string         `json:"status"`
	StatusMessage string         `json:"status_message,omitempty"`
	Input         string         `json:"input,omitempty"`
	Output        string         `json:"output,omitempty"`
	InputTokens   int64          `json:"input_tokens,omitempty"`
	OutputTokens  int64          `json:"output_tokens,omitempty"`
	Attributes    map[string]any `json:"attributes,omitempty"`
}

type Query struct {
	ProjectID, TraceID, Status, Kind, Name string
	Limit                                  int
	Cursor                                 string
}

type Summary struct {
	ProjectID    string    `json:"project_id"`
	TraceID      string    `json:"trace_id"`
	StartTime    time.Time `json:"start_time"`
	EndTime      time.Time `json:"end_time"`
	SpanCount    int       `json:"span_count"`
	Status       string    `json:"status"`
	InputTokens  int64     `json:"input_tokens"`
	OutputTokens int64     `json:"output_tokens"`
}

type Page struct {
	Items      []Summary `json:"items"`
	NextCursor string    `json:"next_cursor,omitempty"`
}
