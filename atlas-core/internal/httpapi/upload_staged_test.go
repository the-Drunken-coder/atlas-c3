package httpapi

import (
	"encoding/json"
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
	t.Run("empty staged path never retains", func(t *testing.T) {
		err := model.StorageUnavailable("x", map[string]any{"staged_path": staged})
		if uploadErrorRetainsPreStagedPath(err, "") {
			t.Fatal("expected false")
		}
	})
}

func TestReadJSONMapFieldExplicitNullVsAbsent(t *testing.T) {
	t.Run("absent key is empty map", func(t *testing.T) {
		m, err := readJSONMapField(map[string]json.RawMessage{}, map[string]any{}, "json")
		if err != nil {
			t.Fatal(err)
		}
		if len(m) != 0 {
			t.Fatalf("want empty, got %v", m)
		}
	})
	t.Run("explicit null is validation error", func(t *testing.T) {
		raw := map[string]json.RawMessage{"json": json.RawMessage(`null`)}
		payload := map[string]any{"json": nil}
		_, err := readJSONMapField(raw, payload, "json")
		if err == nil {
			t.Fatal("expected validation error")
		}
	})
}
