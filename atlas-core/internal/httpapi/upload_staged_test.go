package httpapi

import (
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/model"
)

type retainedStagedPathCarrier struct {
	path string
}

func (e retainedStagedPathCarrier) Error() string { return e.path }
func (e retainedStagedPathCarrier) RetainedStagedPath() string {
	return e.path
}

func symlinkOrSkip(t *testing.T, oldname, newname string) {
	t.Helper()
	if err := os.Symlink(oldname, newname); err != nil {
		if runtime.GOOS == "windows" || errors.Is(err, fs.ErrPermission) {
			t.Skipf("symlinks unavailable: %v", err)
		}
		t.Fatalf("symlink: %v", err)
	}
}

func TestUploadErrorRetainsPreStagedPath(t *testing.T) {
	staged := "/tmp/staged-xyz"
	t.Run("retained on promote failure", func(t *testing.T) {
		err := model.StorageUnavailable("object storage mismatch detected", map[string]any{
			"object_id": "o1",
			"file_id":   "f1",
		}).WithCause(retainedStagedPathCarrier{path: staged})
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
		symlinkOrSkip(t, realDir, linkDir)
		linkedPath := filepath.Join(linkDir, "staged.bin")
		resolvedPath := filepath.Join(realDir, "staged.bin")
		err := model.StorageUnavailable("x", map[string]any{}).WithCause(retainedStagedPathCarrier{path: resolvedPath})
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

func TestWriteErrorDoesNotExposeRetainedStagedPath(t *testing.T) {
	router := &Router{}
	req := httptest.NewRequest(http.MethodPost, "/objects/obj-1/files/f1", nil)
	rr := httptest.NewRecorder()
	router.writeError(rr, req, model.StorageUnavailable("object storage mismatch detected", map[string]any{
		"object_id": "obj-1",
		"file_id":   "f1",
	}).WithCause(retainedStagedPathCarrier{path: "/tmp/secret/staged.bin"}))
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("unexpected status: %d", rr.Code)
	}
	var envelope model.ErrorEnvelope
	if err := json.Unmarshal(rr.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if envelope.Details != nil {
		if _, exists := envelope.Details["staged_path"]; exists {
			t.Fatalf("unexpected staged_path leak: %#v", envelope.Details)
		}
	}
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
		m, err := readJSONMapField(raw, payload, "json")
		if err == nil {
			t.Fatal("expected validation error")
		}
		if m != nil {
			t.Fatalf("expected nil map on error, got %#v", m)
		}
		ce, ok := model.IsCoreError(err)
		if !ok || ce.ErrorCode != "validation_failed" {
			t.Fatalf("expected validation_failed, got %v", err)
		}
		rawFields, exists := ce.Details["fields"]
		if !exists {
			t.Fatalf("expected fields detail, got %#v", ce.Details)
		}
		fields, ok := rawFields.([]model.FieldError)
		if !ok || len(fields) != 1 {
			t.Fatalf("expected one field error, got %#v", rawFields)
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
