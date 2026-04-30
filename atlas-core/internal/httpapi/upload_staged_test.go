package httpapi

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/model"
)

func TestUploadErrorRetainsPreStagedPath(t *testing.T) {
	staged := "/tmp/staged-xyz"
	t.Run("retained on promote failure", func(t *testing.T) {
		err := model.StorageUnavailable("object storage mismatch detected", map[string]any{
			"object_id":   "o1",
			"file_id":     "f1",
			"staged_path": staged,
		})
		if !uploadErrorRetainsPreStagedPath(err, staged) {
			t.Fatal("expected true")
		}
	})
	t.Run("cleanup on other storage errors", func(t *testing.T) {
		err := model.StorageUnavailable("other", map[string]any{"object_id": "o1"})
		if uploadErrorRetainsPreStagedPath(err, staged) {
			t.Fatal("expected false when staged_path missing")
		}
	})
	t.Run("cleanup on path mismatch", func(t *testing.T) {
		err := model.StorageUnavailable("x", map[string]any{"staged_path": "/other"})
		if uploadErrorRetainsPreStagedPath(err, staged) {
			t.Fatal("expected false when staged_path differs")
		}
	})
	t.Run("retained when canonical paths match", func(t *testing.T) {
		dir := t.TempDir()
		realDir := filepath.Join(dir, "real")
		if err := os.Mkdir(realDir, 0o755); err != nil {
			t.Fatalf("mkdir real: %v", err)
		}
		linkDir := filepath.Join(dir, "link")
		if err := os.Symlink(realDir, linkDir); err != nil {
			t.Fatalf("symlink: %v", err)
		}
		linkedPath := filepath.Join(linkDir, "staged.bin")
		resolvedPath := filepath.Join(realDir, "staged.bin")
		err := model.StorageUnavailable("x", map[string]any{"staged_path": resolvedPath})
		if !uploadErrorRetainsPreStagedPath(err, linkedPath) {
			t.Fatalf("expected true for %q vs %q", linkedPath, resolvedPath)
		}
	})
	t.Run("empty staged path never retains", func(t *testing.T) {
		err := model.StorageUnavailable("x", map[string]any{"staged_path": staged})
		if uploadErrorRetainsPreStagedPath(err, "") {
			t.Fatal("expected false")
		}
	})
}

func TestReadJSONMapFieldExplicitNullVsAbsent(t *testing.T) {
	t.Run("absent key returns nil", func(t *testing.T) {
		m, err := readJSONMapField(map[string]json.RawMessage{}, map[string]any{}, "json")
		if err != nil {
			t.Fatal(err)
		}
		if m != nil {
			t.Fatalf("want nil, got %v", m)
		}
	})
	t.Run("explicit null is validation error", func(t *testing.T) {
		raw := map[string]json.RawMessage{"json": json.RawMessage(`null`)}
		payload := map[string]any{"json": nil}
		_, err := readJSONMapField(raw, payload, "json")
		if err == nil {
			t.Fatal("expected validation error")
		}
		ce, ok := model.IsCoreError(err)
		if !ok || ce.ErrorCode != "validation_failed" {
			t.Fatalf("expected validation_failed, got %v", err)
		}
		fields, ok := ce.Details["fields"].([]model.FieldError)
		if !ok || len(fields) != 1 {
			t.Fatalf("expected one field error, got %#v", ce.Details["fields"])
		}
		want := model.FieldError{Field: "json", Code: "invalid_type", Message: "must be a JSON object"}
		if fields[0] != want {
			t.Fatalf("unexpected field error: got %#v want %#v", fields[0], want)
		}
	})
	t.Run("explicit empty object returns empty map", func(t *testing.T) {
		raw := map[string]json.RawMessage{"json": json.RawMessage(`{}`)}
		payload := map[string]any{"json": map[string]any{}}
		m, err := readJSONMapField(raw, payload, "json")
		if err != nil {
			t.Fatal(err)
		}
		if m == nil {
			t.Fatal("want non-nil empty map, got nil")
		}
		if len(m) != 0 {
			t.Fatalf("want empty map, got %v", m)
		}
	})
}
