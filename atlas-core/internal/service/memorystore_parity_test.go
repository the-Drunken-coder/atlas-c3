package service_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/model"
	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/service/servicetest"
	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/store"
)

func writeMemoryStoreStagedFile(t *testing.T, root, objectID string, payload []byte) string {
	t.Helper()
	stagedDir := filepath.Join(root, "staging", objectID)
	if err := os.MkdirAll(stagedDir, 0o755); err != nil {
		t.Fatalf("mkdir staged dir: %v", err)
	}
	staged := filepath.Join(stagedDir, "staged.bin")
	if err := os.WriteFile(staged, payload, 0o600); err != nil {
		t.Fatalf("write staged: %v", err)
	}
	return staged
}

// Issue #2: MemoryStore.ListObservations must include rows whose UpdatedAt is
// exactly equal to the UpdatedAfter cutoff, matching Postgres' `>=` semantics.
func TestMemoryStoreListObservationsUpdatedAfterIsInclusive(t *testing.T) {
	ctx := context.Background()
	mem := servicetest.NewMemoryStore()
	cutoff := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	obs := model.Observation{
		ObservationID: "obs-1",
		SourceAssetID: "asset-1",
		UpdatedAt:     cutoff,
		CreatedAt:     cutoff,
	}
	mem.Observations[obs.ObservationID] = obs

	_, total, err := mem.ListObservations(ctx, store.ObservationListFilter{
		UpdatedAfter: cutoff.Format(time.RFC3339Nano),
	}, model.Pagination{Limit: 10})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected boundary row to be returned (>=), got total=%d", total)
	}
}

// Issue #1: two CreateObjectFile calls that would produce the same object_files
// path must conflict, mirroring the UNIQUE constraint on object_files.path.
//
// Two distinct (object_id, file_id) pairs cannot collide via LogicalPath, so
// instead we plant a duplicate path in the map by hand and verify the second
// CreateObjectFile rejects it.
func TestMemoryStoreCreateObjectFileGlobalPathConflict(t *testing.T) {
	ctx := context.Background()
	mem := servicetest.NewMemoryStore()
	mem.Objects["obj-1"] = model.Object{ObjectID: "obj-1"}
	mem.Objects["obj-2"] = model.Object{ObjectID: "obj-2"}
	// Pre-existing entry: obj-1/file-collision.
	preKey := servicetest.ObjectFileKey{ObjectID: "obj-1", FileID: "file-collision"}
	mem.ObjectFiles[preKey] = model.ObjectFile{
		ObjectID: "obj-1", FileID: "file-collision",
		Path: filepath.ToSlash(filepath.Join("objects", "obj-1", "file-collision")),
	}
	// Now manually rewrite that entry's path to clash with the new upload below.
	clashing := mem.ObjectFiles[preKey]
	clashing.Path = filepath.ToSlash(filepath.Join("objects", "obj-2", "file-new"))
	mem.ObjectFiles[preKey] = clashing

	_, err := mem.CreateObjectFile(ctx, store.ObjectUploadInput{
		File:   model.ObjectFile{ObjectID: "obj-2", FileID: "file-new"},
		Reader: bytes.NewReader([]byte("payload")),
	})
	if err == nil {
		t.Fatal("expected conflict on duplicate path")
	}
	ce, ok := model.IsCoreError(err)
	if !ok || ce.ErrorCode != "conflict" {
		t.Fatalf("expected conflict error, got %v", err)
	}
	if reason, _ := ce.Details["reason"].(string); reason != "already_exists" {
		t.Fatalf("expected reason=already_exists, got reason=%q (details=%#v)", reason, ce.Details)
	}
}

// Issue #3: invalid path segments must be rejected (e.g., file_id containing
// "..") — Postgres rejects them via objectfiles.LogicalPath, MemoryStore must
// agree.
func TestMemoryStoreCreateObjectFileRejectsInvalidPathSegments(t *testing.T) {
	ctx := context.Background()
	mem := servicetest.NewMemoryStore()
	mem.Objects["obj-1"] = model.Object{ObjectID: "obj-1"}

	_, err := mem.CreateObjectFile(ctx, store.ObjectUploadInput{
		File:   model.ObjectFile{ObjectID: "obj-1", FileID: "../escape"},
		Reader: bytes.NewReader([]byte("payload")),
	})
	if err == nil {
		t.Fatal("expected validation error for ../escape file_id")
	}
	ce, ok := model.IsCoreError(err)
	if !ok || ce.ErrorCode != "validation_failed" {
		t.Fatalf("expected validation_failed ValidationError, got %v", err)
	}
}

// Issue #5: the ObjectUploadInput contract requires implementations to remove
// the pre-staged file on both success and error paths. Exercise the error path
// (size mismatch) and assert the file was deleted.
func TestMemoryStoreCreateObjectFileRemovesPreStagedOnError(t *testing.T) {
	ctx := context.Background()
	mem := servicetest.NewMemoryStore()
	mem.Objects["obj-1"] = model.Object{ObjectID: "obj-1"}

	dir := t.TempDir()
	staged := writeMemoryStoreStagedFile(t, dir, "obj-1", []byte("hello"))

	// Lie about the size — implementation must reject AND remove the path.
	_, err := mem.CreateObjectFile(ctx, store.ObjectUploadInput{
		File:               model.ObjectFile{ObjectID: "obj-1", FileID: "f1"},
		PreStagedPath:      staged,
		PreStagedSizeBytes: 999,
	})
	if err == nil {
		t.Fatal("expected validation error for size mismatch")
	}
	if _, statErr := os.Stat(staged); !os.IsNotExist(statErr) {
		t.Fatalf("expected staged file to be removed on error, stat err=%v", statErr)
	}
}

// Issue #5 (success path companion): on a normal create, the pre-staged file
// is also removed.
func TestMemoryStoreCreateObjectFileRemovesPreStagedOnSuccess(t *testing.T) {
	ctx := context.Background()
	mem := servicetest.NewMemoryStore()
	mem.Objects["obj-1"] = model.Object{ObjectID: "obj-1"}

	dir := t.TempDir()
	payload := []byte("hello-world")
	staged := writeMemoryStoreStagedFile(t, dir, "obj-1", payload)

	_, err := mem.CreateObjectFile(ctx, store.ObjectUploadInput{
		File:               model.ObjectFile{ObjectID: "obj-1", FileID: "f1"},
		PreStagedPath:      staged,
		PreStagedSizeBytes: int64(len(payload)),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, statErr := os.Stat(staged); !os.IsNotExist(statErr) {
		t.Fatalf("expected staged file to be removed on success, stat err=%v", statErr)
	}
}

func TestMemoryStoreCreateObjectFileRejectsPreStagedOutsideStagingWithoutDeletingIt(t *testing.T) {
	ctx := context.Background()
	mem := servicetest.NewMemoryStore()
	mem.Objects["obj-1"] = model.Object{ObjectID: "obj-1"}

	dir := t.TempDir()
	staged := filepath.Join(dir, "staged.bin")
	payload := []byte("hello")
	if err := os.WriteFile(staged, payload, 0o600); err != nil {
		t.Fatalf("write staged: %v", err)
	}

	_, err := mem.CreateObjectFile(ctx, store.ObjectUploadInput{
		File:               model.ObjectFile{ObjectID: "obj-1", FileID: "f1"},
		PreStagedPath:      staged,
		PreStagedSizeBytes: int64(len(payload)),
	})
	if err == nil {
		t.Fatal("expected validation error for path outside staging")
	}
	ce, ok := model.IsCoreError(err)
	if !ok || ce.ErrorCode != "validation_failed" {
		t.Fatalf("expected validation_failed, got %v", err)
	}
	if _, statErr := os.Stat(staged); statErr != nil {
		t.Fatalf("expected rejected staged file to remain for caller cleanup, stat err=%v", statErr)
	}
}

func TestMemoryStoreGetFullQueryStateOrdersObjectFilesLikePostgres(t *testing.T) {
	mem := servicetest.NewMemoryStore()
	older := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	newer := older.Add(time.Minute)
	mem.ObjectFiles[servicetest.ObjectFileKey{ObjectID: "obj-b", FileID: "f2"}] = model.ObjectFile{ObjectID: "obj-b", FileID: "f2", UpdatedAt: older}
	mem.ObjectFiles[servicetest.ObjectFileKey{ObjectID: "obj-a", FileID: "f1"}] = model.ObjectFile{ObjectID: "obj-a", FileID: "f1", UpdatedAt: newer}
	mem.ObjectFiles[servicetest.ObjectFileKey{ObjectID: "obj-a", FileID: "f0"}] = model.ObjectFile{ObjectID: "obj-a", FileID: "f0", UpdatedAt: older}

	state, err := mem.GetFullQueryState(context.Background())
	if err != nil {
		t.Fatalf("full query state: %v", err)
	}
	got := make([]string, 0, len(state.ObjectFiles))
	for _, file := range state.ObjectFiles {
		got = append(got, file.ObjectID+"/"+file.FileID)
	}
	want := []string{"obj-a/f1", "obj-a/f0", "obj-b/f2"}
	if len(got) != len(want) {
		t.Fatalf("unexpected object file count: got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected object file order: got %v want %v", got, want)
		}
	}
}

func TestMemoryStoreGetFullQueryStateOrdersRecordsLikePostgres(t *testing.T) {
	mem := servicetest.NewMemoryStore()
	older := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	newer := older.Add(time.Minute)
	mem.Entities["b"] = model.Entity{EntityID: "b", UpdatedAt: older}
	mem.Entities["a"] = model.Entity{EntityID: "a", UpdatedAt: newer}
	mem.Observations["b"] = model.Observation{ObservationID: "b", UpdatedAt: older}
	mem.Observations["a"] = model.Observation{ObservationID: "a", UpdatedAt: newer}
	mem.Tasks["b"] = model.Task{TaskID: "b", UpdatedAt: older}
	mem.Tasks["a"] = model.Task{TaskID: "a", UpdatedAt: newer}
	mem.Objects["b"] = model.Object{ObjectID: "b", UpdatedAt: older}
	mem.Objects["a"] = model.Object{ObjectID: "a", UpdatedAt: newer}

	state, err := mem.GetFullQueryState(context.Background())
	if err != nil {
		t.Fatalf("full query state: %v", err)
	}
	if got := []string{state.Entities[0].EntityID, state.Entities[1].EntityID}; got[0] != "a" || got[1] != "b" {
		t.Fatalf("unexpected entity order: %v", got)
	}
	if got := []string{state.Observations[0].ObservationID, state.Observations[1].ObservationID}; got[0] != "a" || got[1] != "b" {
		t.Fatalf("unexpected observation order: %v", got)
	}
	if got := []string{state.Tasks[0].TaskID, state.Tasks[1].TaskID}; got[0] != "a" || got[1] != "b" {
		t.Fatalf("unexpected task order: %v", got)
	}
	if got := []string{state.Objects[0].ObjectID, state.Objects[1].ObjectID}; got[0] != "a" || got[1] != "b" {
		t.Fatalf("unexpected object order: %v", got)
	}
}
