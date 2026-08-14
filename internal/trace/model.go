package trace

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

const (
	MaxInputOutputBytes = 1 << 20
	MaxAttributesBytes  = 256 << 10
	MaxAttributesCount  = 128
	MaxSpanIDBytes      = 256
	MaxNameBytes        = 1024
)

var ErrPayloadTooLarge = errors.New("trace payload is too large")

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

func (s Span) Validate() error {
	if s.ProjectID == "" || s.TraceID == "" || s.SpanID == "" || s.Name == "" {
		return errors.New("project_id, trace_id, span_id and name are required")
	}
	if len(s.TraceID) > MaxSpanIDBytes || len(s.SpanID) > MaxSpanIDBytes || len(s.ParentSpanID) > MaxSpanIDBytes || len(s.Name) > MaxNameBytes {
		return ErrPayloadTooLarge
	}
	if s.StartTime.IsZero() || s.Duration < 0 || s.Duration > 24*time.Hour {
		return fmt.Errorf("invalid start_time or duration")
	}
	if len(s.Input) > MaxInputOutputBytes || len(s.Output) > MaxInputOutputBytes {
		return ErrPayloadTooLarge
	}
	if len(s.Attributes) > MaxAttributesCount {
		return ErrPayloadTooLarge
	}
	if len(s.Attributes) > 0 {
		encoded, err := json.Marshal(s.Attributes)
		if err != nil {
			return fmt.Errorf("invalid attributes: %w", err)
		}
		if len(encoded) > MaxAttributesBytes {
			return ErrPayloadTooLarge
		}
	}
	if s.InputTokens < 0 || s.OutputTokens < 0 {
		return fmt.Errorf("token counts cannot be negative")
	}
	return nil
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
