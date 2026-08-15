package trace

import (
	"strings"
	"testing"
	"time"
)

func validSpan() Span {
	return Span{WorkspaceID: "workspace", TraceID: "trace", SpanID: "span", Name: "test", StartTime: time.Now().UTC(), Duration: time.Millisecond, Status: "ok"}
}

func TestValidatePayloadLimits(t *testing.T) {
	span := validSpan()
	span.Input = strings.Repeat("x", MaxInputOutputBytes+1)
	if err := span.Validate(); err != ErrPayloadTooLarge {
		t.Fatalf("input error=%v", err)
	}
	span = validSpan()
	span.Attributes = map[string]any{"value": strings.Repeat("x", MaxAttributesBytes)}
	if err := span.Validate(); err != ErrPayloadTooLarge {
		t.Fatalf("attributes error=%v", err)
	}
	span = validSpan()
	span.StartTime = time.Time{}
	if err := span.Validate(); err == nil {
		t.Fatal("expected missing start time error")
	}
}
