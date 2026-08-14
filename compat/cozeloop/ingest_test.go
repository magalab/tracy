package cozeloop

import (
	"testing"
	"time"
)

func TestUploadSpanMapsToDomain(t *testing.T) {
	received := time.Date(2026, 8, 14, 0, 0, 0, 0, time.UTC)
	spans, err := (UploadSpanData{Spans: []UploadSpan{{StartedATMicros: 1_000_000, DurationMicros: 250, TraceID: "trace", SpanID: "span", ParentID: "parent", SpanName: "model", SpanType: "model", StatusCode: 500, Input: "in", Output: "out", TagsString: map[string]string{"model": "x"}, TagsLong: map[string]int64{"tokens": 3}}}}).Map("project", received)
	if err != nil {
		t.Fatal(err)
	}
	if len(spans) != 1 {
		t.Fatalf("span count=%d", len(spans))
	}
	span := spans[0]
	if span.ProjectID != "project" || span.TraceID != "trace" || span.ParentSpanID != "parent" || span.Name != "model" || span.Status != "error" {
		t.Fatalf("unexpected span: %#v", span)
	}
	if span.StartTime.UnixMicro() != 1_000_000 || span.Duration != 250*time.Microsecond || span.ReceivedAt != received {
		t.Fatalf("unexpected timing: %#v", span)
	}
	if span.Attributes["model"] != "x" || span.Attributes["tokens"] != int64(3) || span.Attributes["cozeloop.status_code"] != int32(500) {
		t.Fatalf("unexpected attributes: %#v", span.Attributes)
	}
}
