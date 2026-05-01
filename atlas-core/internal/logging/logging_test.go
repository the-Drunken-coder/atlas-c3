package logging

import "testing"

func TestWithRequestHandlesNilContext(t *testing.T) {
	logger := New("info", "atlas-core", "run-1")
	ctx := logger.WithRequest(nil, "req-1")
	if got := RequestID(ctx); got != "req-1" {
		t.Fatalf("unexpected request id: %q", got)
	}
}

func TestRequestIDHandlesNilContext(t *testing.T) {
	if got := RequestID(nil); got != "" {
		t.Fatalf("expected empty request id, got %q", got)
	}
}
