package postgres_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/model"
	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/objectfiles"
	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/postgres"
	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/store"
)

func TestEnsureSchemaAndStoreRoundTrip(t *testing.T) {
	databaseURL := os.Getenv("ATLAS_CORE_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("ATLAS_CORE_TEST_DATABASE_URL not set")
	}
	pool, err := postgres.Open(context.Background(), databaseURL)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer pool.Close()
	if err := postgres.EnsureSchema(context.Background(), pool); err != nil {
		t.Fatalf("ensure schema: %v", err)
	}
	dir := t.TempDir()
	files, err := objectfiles.New(dir)
	if err != nil {
		t.Fatalf("objectfiles.New: %v", err)
	}
	if files.Status().Status != "ready" {
		t.Fatalf("unexpected storage status: %+v", files.Status())
	}
}

// TestPostCommitDeleteObjectMarksMismatchOnNonRemovableFile verifies that when
// a database delete commits but on-disk delete fails, object storage is marked
// not ready (B23 / post-commit cleanup best-effort).
func TestPostCommitDeleteObjectMarksMismatchOnNonRemovableFile(t *testing.T) {
	databaseURL := os.Getenv("ATLAS_CORE_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("ATLAS_CORE_TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	pool, err := postgres.Open(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer pool.Close()
	if err := postgres.EnsureSchema(ctx, pool); err != nil {
		t.Fatalf("schema: %v", err)
	}
	base := t.TempDir()
	files, err := objectfiles.New(base)
	if err != nil {
		t.Fatalf("objectfiles: %v", err)
	}
	if err := files.Verify(); err != nil {
		t.Fatalf("verify: %v", err)
	}
	st := postgres.NewStore(pool, files, 16*1024*1024)
	objID := "obj-b23"
	if _, err := st.CreateObject(ctx, model.Object{ObjectID: objID, Type: "t", OwnerType: "system", OwnerID: "active_command_catalog", JSON: model.JSONMap{}}); err != nil {
		t.Fatalf("create object: %v", err)
	}
	if _, err := st.CreateObjectFile(ctx, store.ObjectUploadInput{File: model.ObjectFile{FileID: "f1", ObjectID: objID, ContentType: "application/octet-stream"}, Reader: strings.NewReader("data"), MaxBytes: 16}); err != nil {
		t.Fatalf("file: %v", err)
	}
	meta, err := st.GetObjectFile(ctx, objID, "f1")
	if err != nil {
		t.Fatalf("get file: %v", err)
	}
	abs, err := files.AbsolutePath(meta.Path)
	if err != nil {
		t.Fatalf("abspath: %v", err)
	}
	if err := os.Remove(abs); err != nil {
		t.Fatalf("remove file: %v", err)
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(abs+"/h", []byte("x"), 0o644); err != nil {
		t.Fatalf("subfile: %v", err)
	}
	if err := st.DeleteObject(ctx, objID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if files.Status().Status == "ready" {
		t.Fatal("expected storage to be not ready after failed post-commit delete")
	}
}
