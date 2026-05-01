package logging

import (
	"context"
	"testing"
)

func nilContext() context.Context { return nil }

func TestWithRequestHandlesNilContext(t *testing.T) {
	logger := New("info", "atlas-core", "run-1")
	ctx := logger.WithRequest(nilContext(), "req-1")
	if got := RequestID(ctx); got != "req-1" {
		t.Fatalf("unexpected request id: %q", got)
	}
}

func TestRequestIDHandlesNilContext(t *testing.T) {
	if got := RequestID(nilContext()); got != "" {
		t.Fatalf("expected empty request id, got %q", got)
	}
}
