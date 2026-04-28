package objectfiles

import (
	"errors"
	"io"
	"os"
	"strings"
	"testing"
)

func TestAppendOversizeRollsBackToPreSize(t *testing.T) {
	dir := t.TempDir()
	s, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	logical, err := s.LogicalPath("o", "f")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Promote(writableTemp(t, dir, "f", "hello"), logical); err != nil {
		t.Fatalf("seed: %v", err)
	}
	f0, st0, _ := s.Open(logical)
	f0.Close()
	pre := st0.Size()
	if _, _, err := s.Append(logical, strings.NewReader("xxxxx"), 2); err == nil {
		t.Fatal("expected payload too large")
	}
	f1, st1, _ := s.Open(logical)
	f1.Close()
	if st1.Size() != pre {
		t.Fatalf("size after failed append: got %d want %d", st1.Size(), pre)
	}
}

func TestAppendEmptyBodyDoesNotChangeSize(t *testing.T) {
	dir := t.TempDir()
	s, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	path, err := s.LogicalPath("o", "f")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Promote(writableTemp(t, dir, "f", "zz"), path); err != nil {
		t.Fatal(err)
	}
	f0, s0, _ := s.Open(path)
	f0.Close()
	pre := s0.Size()
	if _, _, err := s.Append(path, strings.NewReader(""), 100); err == nil {
		t.Fatal("expected empty body error")
	}
	f1, s1, _ := s.Open(path)
	f1.Close()
	if s1.Size() != pre {
		t.Fatalf("size changed: %d vs %d", s1.Size(), pre)
	}
}

func TestTruncateBack(t *testing.T) {
	dir := t.TempDir()
	s, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	path, err := s.LogicalPath("a", "b")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Promote(writableTemp(t, dir, "b", "1234"), path); err != nil {
		t.Fatal(err)
	}
	if err := s.TruncateBack(path, 2); err != nil {
		t.Fatal(err)
	}
	f, st, _ := s.Open(path)
	f.Close()
	if st.Size() != 2 {
		t.Fatalf("size %d", st.Size())
	}
}

// writableTemp creates a temp file with content, returns its path, for [Store.Promote] in tests.
func writableTemp(t *testing.T, root, name, content string) string {
	t.Helper()
	stage := root + "/staging/seed"
	if err := os.MkdirAll(stage, 0o755); err != nil {
		t.Fatal(err)
	}
	f, err := os.CreateTemp(stage, "t-*")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(content); err != nil {
		_ = f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	return f.Name()
}

func TestLogicalPathRejectsUnsafeIDs(t *testing.T) {
	dir := t.TempDir()
	s, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.LogicalPath("a/b", "f"); err == nil {
		t.Fatal("expected error for object_id with separator")
	}
	if _, err := s.LogicalPath("a", "../x"); err == nil {
		t.Fatal("expected error for file_id with traversal")
	}
}

func TestAppendIOCopyErrorTruncates(t *testing.T) {
	dir := t.TempDir()
	s, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	p, err := s.LogicalPath("o", "f")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Promote(writableTemp(t, dir, "f", "abc"), p); err != nil {
		t.Fatal(err)
	}
	r, w := io.Pipe()
	_ = w.CloseWithError(errors.New("mock read error"))
	_, _, err = s.Append(p, r, 1000)
	if err == nil {
		t.Fatal("expected read error")
	}
	f, st, _ := s.Open(p)
	f.Close()
	if st.Size() != 3 {
		t.Fatalf("want size 3 after bad read, got %d", st.Size())
	}
}
