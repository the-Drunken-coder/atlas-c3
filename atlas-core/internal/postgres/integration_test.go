package postgres_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
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
	st := postgres.NewStore(pool, files, 16*1024*1024, nil)
	objID := fmt.Sprintf("obj-rt-%d", time.Now().UnixNano())
	if _, err := st.CreateObject(context.Background(), model.Object{ObjectID: objID, Type: "t", OwnerType: "system", OwnerID: "active_command_catalog", JSON: model.JSONMap{}}); err != nil {
		t.Fatalf("create object: %v", err)
	}
	obj, err := st.GetObject(context.Background(), objID)
	if err != nil {
		t.Fatalf("get object: %v", err)
	}
	if obj.ObjectID != objID {
		t.Fatalf("expected object ID %s, got %s", objID, obj.ObjectID)
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
	st := postgres.NewStore(pool, files, 16*1024*1024, nil)
	objID := fmt.Sprintf("obj-b23-%d", time.Now().UnixNano())
	fileID := fmt.Sprintf("f1-%d", time.Now().UnixNano())
	if _, err := st.CreateObject(ctx, model.Object{ObjectID: objID, Type: "t", OwnerType: "system", OwnerID: "active_command_catalog", JSON: model.JSONMap{}}); err != nil {
		t.Fatalf("create object: %v", err)
	}
	if _, err := st.CreateObjectFile(ctx, store.ObjectUploadInput{File: model.ObjectFile{FileID: fileID, ObjectID: objID, ContentType: "application/octet-stream"}, Reader: strings.NewReader("data"), MaxBytes: 16}); err != nil {
		t.Fatalf("file: %v", err)
	}
	meta, err := st.GetObjectFile(ctx, objID, fileID)
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

func TestEnsureSchemaMigratesObjectFilesPrimaryKeyToComposite(t *testing.T) {
	databaseURL := os.Getenv("ATLAS_CORE_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("ATLAS_CORE_TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	adminPool, err := postgres.Open(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open admin pool: %v", err)
	}
	defer adminPool.Close()

	schemaName := fmt.Sprintf("atlas_schema_%d", time.Now().UnixNano())
	if _, err := adminPool.Exec(ctx, "CREATE SCHEMA "+schemaName); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	defer func() {
		_, _ = adminPool.Exec(ctx, "DROP SCHEMA "+schemaName+" CASCADE")
	}()

	pool := openSchemaScopedPool(t, ctx, databaseURL, schemaName)
	defer pool.Close()
	if err := postgres.EnsureSchema(ctx, pool); err != nil {
		t.Fatalf("initial ensure schema: %v", err)
	}
	if _, err := pool.Exec(ctx, `ALTER TABLE object_files DROP CONSTRAINT object_files_pkey`); err != nil {
		t.Fatalf("drop composite primary key: %v", err)
	}
	if _, err := pool.Exec(ctx, `ALTER TABLE object_files ADD CONSTRAINT object_files_pkey PRIMARY KEY (file_id)`); err != nil {
		t.Fatalf("add legacy primary key: %v", err)
	}
	if _, err := pool.Exec(ctx, `CREATE INDEX object_files_object_idx ON object_files(object_id)`); err != nil {
		t.Fatalf("add legacy object_id index: %v", err)
	}

	if err := postgres.EnsureSchema(ctx, pool); err != nil {
		t.Fatalf("migrated ensure schema: %v", err)
	}

	var pkColumns []string
	row := pool.QueryRow(ctx, `SELECT ARRAY(
		SELECT att.attname
		FROM pg_constraint AS con
		JOIN unnest(con.conkey) WITH ORDINALITY AS cols(attnum, ord) ON true
		JOIN pg_attribute AS att
		  ON att.attrelid = con.conrelid
		 AND att.attnum = cols.attnum
		WHERE con.conrelid = 'object_files'::regclass
		  AND con.contype = 'p'
		ORDER BY cols.ord
	)`)
	if err := row.Scan(&pkColumns); err != nil {
		t.Fatalf("scan primary key columns: %v", err)
	}
	if len(pkColumns) != 2 || pkColumns[0] != "object_id" || pkColumns[1] != "file_id" {
		t.Fatalf("unexpected primary key columns: %v", pkColumns)
	}
	var legacyIndexCount int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM pg_indexes WHERE schemaname = current_schema() AND indexname = 'object_files_object_idx'`).Scan(&legacyIndexCount); err != nil {
		t.Fatalf("count legacy index: %v", err)
	}
	if legacyIndexCount != 0 {
		t.Fatalf("expected redundant object_files_object_idx to be dropped, found %d entries", legacyIndexCount)
	}

	for _, objectID := range []string{"obj-a", "obj-b"} {
		if _, err := pool.Exec(ctx, `INSERT INTO objects (object_id, type, owner_type, owner_id, json, created_at, updated_at) VALUES ($1,'t','system','active_command_catalog','{}'::jsonb, now(), now())`, objectID); err != nil {
			t.Fatalf("insert object %s: %v", objectID, err)
		}
	}
	for _, objectID := range []string{"obj-a", "obj-b"} {
		if _, err := pool.Exec(ctx, `INSERT INTO object_files (file_id, object_id, path, content_type, size_bytes, usage_hint, created_at, updated_at) VALUES ('shared-file',$1,$2,'application/octet-stream',1,NULL,now(),now())`, objectID, fmt.Sprintf("objects/%s/shared-file", objectID)); err != nil {
			t.Fatalf("insert object file for %s: %v", objectID, err)
		}
	}
}

func openSchemaScopedPool(t *testing.T, ctx context.Context, databaseURL, schemaName string) *pgxpool.Pool {
	t.Helper()
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if cfg.ConnConfig.RuntimeParams == nil {
		cfg.ConnConfig.RuntimeParams = map[string]string{}
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schemaName
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("open scoped pool: %v", err)
	}
	return pool
}
