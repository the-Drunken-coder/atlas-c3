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

func TestReadAllCappedOneOver(t *testing.T) {
	limit := int64(100)
	r := &repeatByteReader{byte: 'a', n: int(limit + 1)}
	_, err := readAllCapped(r, limit)
	if err == nil {
		t.Fatal("expected error for one byte over read")
	}
	ce, ok := model.IsCoreError(err)
	if !ok || ce.ErrorCode != "payload_too_large" {
		t.Fatalf("expected payload_too_large, got %v", err)
	}
}

func TestReadAllCappedExact(t *testing.T) {
	limit := int64(100)
	r := &repeatByteReader{byte: 'a', n: int(limit)}
	data, err := readAllCapped(r, limit)
	if err != nil {
		t.Fatalf("unexpected error for exact limit read: %v", err)
	}
	if int64(len(data)) != limit {
		t.Fatalf("expected length %d, got %d", limit, len(data))
	}
}

func TestReadAllCappedEmpty(t *testing.T) {
	limit := int64(100)
	r := &repeatByteReader{byte: 'a', n: 0}
	data, err := readAllCapped(r, limit)
	if err != nil {
		t.Fatalf("unexpected error for empty read: %v", err)
	}
	if len(data) != 0 {
		t.Fatalf("expected length 0, got %d", len(data))
	}
}
