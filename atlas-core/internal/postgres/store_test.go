package postgres

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/model"
	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/objectfiles"
	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/store"
)

func symlinkOrSkip(t *testing.T, oldname, newname string) {
	t.Helper()
	if err := os.Symlink(oldname, newname); err != nil {
		if runtime.GOOS == "windows" || errors.Is(err, fs.ErrPermission) {
			t.Skipf("symlinks unavailable: %v", err)
		}
		t.Fatalf("symlink: %v", err)
	}
}

func TestChooseLimit(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		requested int64
		fallback  int64
		want      int64
	}{
		{name: "requested smaller than fallback", requested: 512, fallback: 2048, want: 512},
		{name: "requested larger than fallback is capped", requested: 4096, fallback: 2048, want: 2048},
		{name: "requested survives missing fallback", requested: 17, fallback: 0, want: 17},
		{name: "fallback used when request absent", requested: 0, fallback: 2048, want: 2048},
		{name: "default used when both absent", requested: 0, fallback: 0, want: defaultUploadLimitBytes},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := chooseLimit(test.requested, test.fallback); got != test.want {
				t.Fatalf("chooseLimit(%d, %d) = %d, want %d", test.requested, test.fallback, got, test.want)
			}
		})
	}
}

func TestChooseStoredContentType(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		explicit string
		detected string
		want     string
	}{
		{name: "valid explicit normalized", explicit: " text/plain ; charset=utf-8 ", detected: "application/json", want: "text/plain; charset=utf-8"},
		{name: "blank explicit falls back to detected", explicit: "  ", detected: "image/jpeg", want: "image/jpeg"},
		{name: "invalid explicit falls back to detected", explicit: "text/plain\r\nx:y", detected: "image/jpeg", want: "image/jpeg"},
		{name: "empty values default to octet stream", explicit: "", detected: "", want: "application/octet-stream"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := chooseStoredContentType(test.explicit, test.detected); got != test.want {
				t.Fatalf("chooseStoredContentType(%q, %q) = %q, want %q", test.explicit, test.detected, got, test.want)
			}
		})
	}
}

func TestCreateObjectFileRejectsPreStagedPathOutsideStagingViaSymlinkedParent(t *testing.T) {
	root := t.TempDir()
	files, err := objectfiles.New(root)
	if err != nil {
		t.Fatal(err)
	}
	outsideDir := t.TempDir()
	outsideFile := filepath.Join(outsideDir, "payload.bin")
	if err := os.WriteFile(outsideFile, []byte("abc"), 0o600); err != nil {
		t.Fatal(err)
	}
	linkPath := filepath.Join(files.StagingDir(), "linked")
	symlinkOrSkip(t, outsideDir, linkPath)
	st := NewStore(nil, files, 16*1024*1024, nil)
	_, err = st.CreateObjectFile(context.Background(), store.ObjectUploadInput{
		File:               model.ObjectFile{ObjectID: "obj-1", FileID: "f1"},
		PreStagedPath:      filepath.Join(linkPath, filepath.Base(outsideFile)),
		PreStagedSizeBytes: 3,
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
	coreErr, ok := model.IsCoreError(err)
	if !ok || coreErr.ErrorCode != "validation_failed" {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestEnsureSchemaDropsUpdatedAtIndexBeforeRecreatingIt(t *testing.T) {
	var dropIndex, createIndex int
	for i, stmt := range schemaStatements {
		switch stmt {
		case `DROP INDEX IF EXISTS object_files_updated_at_idx`:
			dropIndex = i + 1
		case `CREATE INDEX IF NOT EXISTS object_files_updated_at_idx ON object_files(updated_at DESC, object_id ASC, file_id ASC)`:
			createIndex = i + 1
		}
	}
	if dropIndex == 0 {
		t.Fatal("expected schema to drop object_files_updated_at_idx before recreating it")
	}
	if createIndex == 0 {
		t.Fatal("expected schema to recreate object_files_updated_at_idx")
	}
	if dropIndex >= createIndex {
		t.Fatalf("expected drop (%d) before create (%d)", dropIndex, createIndex)
	}
}
