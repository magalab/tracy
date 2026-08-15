// Package cozeloop contains the protocol boundary for the CozeLoop SDK.
// CozeLoop DTOs are intentionally kept here and are mapped to Tracy's domain model.
package cozeloop

import (
	"fmt"
	"time"

	domain "github.com/panda/tracy/internal/trace"
)

type UploadSpanData struct {
	Spans []UploadSpan `json:"spans"`
}

// UploadSpan mirrors the public payload emitted by cozeloop-go's trace exporter.
// See https://github.com/coze-dev/cozeloop-go/blob/main/entity/export.go.
type UploadSpan struct {
	StartedATMicros  int64              `json:"started_at_micros"`
	LogID            string             `json:"log_id"`
	SpanID           string             `json:"span_id"`
	ParentID         string             `json:"parent_id"`
	TraceID          string             `json:"trace_id"`
	DurationMicros   int64              `json:"duration_micros"`
	ServiceName      string             `json:"service_name"`
	WorkspaceID      string             `json:"workspace_id"`
	SpanName         string             `json:"span_name"`
	SpanType         string             `json:"span_type"`
	StatusCode       int32              `json:"status_code"`
	Input            string             `json:"input"`
	Output           string             `json:"output"`
	ObjectStorage    string             `json:"object_storage"`
	SystemTagsString map[string]string  `json:"system_tags_string"`
	SystemTagsLong   map[string]int64   `json:"system_tags_long"`
	SystemTagsDouble map[string]float64 `json:"system_tags_double"`
	TagsString       map[string]string  `json:"tags_string"`
	TagsLong         map[string]int64   `json:"tags_long"`
	TagsDouble       map[string]float64 `json:"tags_double"`
	TagsBool         map[string]bool    `json:"tags_bool"`
}

func (d UploadSpanData) Map(workspaceID string, receivedAt time.Time) ([]domain.Span, error) {
	result := make([]domain.Span, 0, len(d.Spans))
	for _, item := range d.Spans {
		if item.TraceID == "" || item.SpanID == "" || item.SpanName == "" {
			return nil, fmt.Errorf("trace_id, span_id and span_name are required")
		}
		attrs := make(map[string]any, len(item.TagsString)+len(item.TagsLong)+len(item.TagsDouble)+len(item.TagsBool)+len(item.SystemTagsString)+len(item.SystemTagsLong)+len(item.SystemTagsDouble))
		for k, v := range item.TagsString {
			attrs[k] = v
		}
		for k, v := range item.TagsLong {
			attrs[k] = v
		}
		for k, v := range item.TagsDouble {
			attrs[k] = v
		}
		for k, v := range item.TagsBool {
			attrs[k] = v
		}
		for k, v := range item.SystemTagsString {
			attrs["system."+k] = v
		}
		for k, v := range item.SystemTagsLong {
			attrs["system."+k] = v
		}
		for k, v := range item.SystemTagsDouble {
			attrs["system."+k] = v
		}
		if item.LogID != "" {
			attrs["cozeloop.log_id"] = item.LogID
		}
		if item.ObjectStorage != "" {
			attrs["cozeloop.object_storage"] = item.ObjectStorage
		}
		if item.ServiceName != "" {
			attrs["service.name"] = item.ServiceName
		}
		status := "ok"
		if item.StatusCode != 0 {
			status = "error"
			attrs["cozeloop.status_code"] = item.StatusCode
		}
		result = append(result, domain.Span{WorkspaceID: workspaceID, TraceID: item.TraceID, SpanID: item.SpanID, ParentSpanID: item.ParentID, Name: item.SpanName, Kind: item.SpanType, StartTime: time.UnixMicro(item.StartedATMicros).UTC(), ReceivedAt: receivedAt.UTC(), Duration: time.Duration(item.DurationMicros) * time.Microsecond, Status: status, Input: item.Input, Output: item.Output, Attributes: attrs})
	}
	return result, nil
}
