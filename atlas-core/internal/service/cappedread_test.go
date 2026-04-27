package service

import (
	"io"
	"testing"

	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/model"
)

type repeatByteReader struct {
	byte   byte
	n      int
	offset int
}

func (r *repeatByteReader) Read(p []byte) (int, error) {
	if r.offset >= r.n {
		return 0, io.EOF
	}
	for i := range p {
		if r.offset >= r.n {
			return i, nil
		}
		p[i] = r.byte
		r.offset++
	}
	return len(p), nil
}

func TestReadAllCappedOversize(t *testing.T) {
	r := &repeatByteReader{byte: 'a', n: 8*1024*1024 + 1}
	_, err := readAllCapped(r, 8*1024*1024)
	if err == nil {
		t.Fatal("expected error for oversized read")
	}
	ce, ok := model.IsCoreError(err)
	if !ok || ce.ErrorCode != "payload_too_large" {
		t.Fatalf("expected payload_too_large, got %v", err)
	}
}
