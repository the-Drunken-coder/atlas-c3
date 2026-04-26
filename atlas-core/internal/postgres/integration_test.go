package postgres_test

import (
	"context"
	"os"
	"testing"

	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/objectfiles"
	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/postgres"
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
