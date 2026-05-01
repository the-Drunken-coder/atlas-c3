package objectfiles

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/mediatype"
	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/model"
)

type Store struct {
	root    string
	mu      sync.RWMutex
	healthy bool
	message string
}

func New(root string) (*Store, error) {
	if err := os.MkdirAll(filepath.Join(root, "staging"), 0o755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Join(root, "objects"), 0o755); err != nil {
		return nil, err
	}
	return &Store{root: root, healthy: true}, nil
}

func (s *Store) StagingDir() string {
	return filepath.Join(s.root, "staging")
}

func (s *Store) Status() model.DependencyStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.healthy {
		return model.DependencyStatus{Status: "ready"}
	}
	return model.DependencyStatus{Status: "error", Message: s.message}
}

func (s *Store) MarkMismatch(message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.healthy = false
	s.message = message
}

func (s *Store) Verify() error {
	testDir := filepath.Join(s.root, "objects")
	if err := os.MkdirAll(testDir, 0o755); err != nil {
		return err
	}
	file := filepath.Join(testDir, ".atlas-write-check")
	if err := os.WriteFile(file, []byte("ok"), 0o644); err != nil {
		return err
	}
	if rmErr := os.Remove(file); rmErr != nil {
		return fmt.Errorf("object storage write-check cleanup: %w", rmErr)
	}
	return nil
}

// ValidateObjectFilePathSegments rejects empty IDs and characters that would
// escape a single directory segment under objects/.
func ValidateObjectFilePathSegments(objectID, fileID string) error {
	for _, pair := range []struct {
		value string
		field string
	}{{objectID, "object_id"}, {fileID, "file_id"}} {
		id := strings.TrimSpace(pair.value)
		if id == "" {
			return model.ValidationError(model.FieldError{Field: pair.field, Code: "required", Message: "value is required"})
		}
		if id == "." || id == ".." {
			return model.ValidationError(model.FieldError{Field: pair.field, Code: "invalid_value", Message: "invalid id"})
		}
		if strings.ContainsAny(id, `/\`) {
			return model.ValidationError(model.FieldError{Field: pair.field, Code: "invalid_value", Message: "id must not contain path separators"})
		}
	}
	return nil
}

func (s *Store) LogicalPath(objectID, fileID string) (string, error) {
	if err := ValidateObjectFilePathSegments(objectID, fileID); err != nil {
		return "", err
	}
	return filepath.ToSlash(filepath.Join("objects", objectID, fileID)), nil
}

func (s *Store) AbsolutePath(logicalPath string) (string, error) {
	if logicalPath == "" {
		return "", fmt.Errorf("logical path required")
	}
	clean := filepath.Clean(logicalPath)
	if strings.HasPrefix(clean, "../") || clean == ".." || strings.Contains(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("unsafe path")
	}
	absolute := filepath.Join(s.root, clean)
	rel, err := filepath.Rel(s.root, absolute)
	if err != nil {
		return "", err
	}
	if strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("unsafe path")
	}
	return absolute, nil
}

func (s *Store) Stage(ctx context.Context, objectID, fileID string, reader io.Reader, maxBytes int64) (stagePath string, size int64, detectedType string, err error) {
	if err := ValidateObjectFilePathSegments(objectID, fileID); err != nil {
		return "", 0, "", err
	}
	stageDir := filepath.Join(s.root, "staging", objectID)
	if err := os.MkdirAll(stageDir, 0o755); err != nil {
		return "", 0, "", err
	}
	temp, err := os.CreateTemp(stageDir, fileID+"-*")
	if err != nil {
		return "", 0, "", err
	}
	defer func() {
		_ = temp.Close()
		if err != nil {
			_ = os.Remove(temp.Name())
		}
	}()
	limiter := &io.LimitedReader{R: reader, N: maxBytes + 1}
	sniff := make([]byte, 512)
	n, readErr := io.ReadFull(limiter, sniff)
	if readErr != nil && !errors.Is(readErr, io.ErrUnexpectedEOF) && !errors.Is(readErr, io.EOF) {
		return "", 0, "", readErr
	}
	if n == 0 {
		return "", 0, "", model.ValidationError(model.FieldError{Field: "file", Code: "required", Message: "file bytes are required"})
	}
	detectedType = http.DetectContentType(sniff[:n])
	if _, err = temp.Write(sniff[:n]); err != nil {
		return "", 0, "", err
	}
	written, err := io.Copy(temp, limiter)
	if err != nil {
		return "", 0, "", err
	}
	size = int64(n) + written
	if size > maxBytes {
		return "", 0, "", model.PayloadTooLarge("upload exceeds configured limit")
	}
	if err := temp.Sync(); err != nil {
		return "", 0, "", err
	}
	normalizedType, _ := mediatype.NormalizeContentType(detectedType)
	return temp.Name(), size, normalizedType, nil
}

func (s *Store) Promote(stagePath, logicalPath string) error {
	target, err := s.AbsolutePath(logicalPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	if err := os.Rename(stagePath, target); err != nil {
		return err
	}
	pruneEmptyStagingParents(filepath.Dir(stagePath), s.StagingDir())
	return nil
}

func (s *Store) Delete(logicalPath string) error {
	target, err := s.AbsolutePath(logicalPath)
	if err != nil {
		return err
	}
	if err := os.Remove(target); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (s *Store) Open(logicalPath string) (io.ReadCloser, os.FileInfo, error) {
	target, err := s.AbsolutePath(logicalPath)
	if err != nil {
		return nil, nil, err
	}
	file, err := os.Open(target)
	if err != nil {
		return nil, nil, err
	}
	stat, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, nil, err
	}
	return file, stat, nil
}

func CleanupStagedPath(stagedPath, stagingDir string) error {
	if stagedPath == "" {
		return nil
	}
	if err := os.Remove(stagedPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	pruneEmptyStagingParents(filepath.Dir(stagedPath), stagingDir)
	return nil
}

func pruneEmptyStagingParents(path, stagingDir string) {
	if path == "" || stagingDir == "" {
		return
	}
	base, err := filepath.Abs(stagingDir)
	if err != nil {
		return
	}
	current, err := filepath.Abs(path)
	if err != nil {
		return
	}
	for current != base {
		within, relErr := pathWithinDir(base, current)
		if relErr != nil || !within {
			return
		}
		if err := os.Remove(current); err != nil {
			return
		}
		parent := filepath.Dir(current)
		if parent == current {
			return
		}
		current = parent
	}
}

func pathWithinDir(dir, path string) (bool, error) {
	rel, err := filepath.Rel(dir, path)
	if err != nil {
		return false, err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false, nil
	}
	return true, nil
}

func (s *Store) TruncateBack(logicalPath string, size int64) error {
	if size < 0 {
		return fmt.Errorf("invalid truncate size %d", size)
	}
	target, err := s.AbsolutePath(logicalPath)
	if err != nil {
		return err
	}
	return os.Truncate(target, size)
}

// Append writes reader to the on-disk file at logicalPath in O_APPEND mode and
// returns the actual pre-append disk size and the new disk size. preSize is
// taken from the open file after acquiring the append lock, so it matches the
// size other appenders observe for rollback. Callers MUST use preSize (not any
// database-recorded size) as the rollback truncate target if a downstream
// operation fails: database metadata can be stale (a prior commit may have
// failed and left the file ahead of metadata), so truncating to a stale size
// would discard committed bytes that existed before this call.
func (s *Store) Append(logicalPath string, reader io.Reader, maxBytes int64) (preSize, newSize int64, err error) {
	target, err := s.AbsolutePath(logicalPath)
	if err != nil {
		return 0, 0, err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return 0, 0, err
	}
	file, err := os.OpenFile(target, os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return 0, 0, err
	}
	if err := flockAppendLock(file); err != nil {
		if closeErr := file.Close(); closeErr != nil {
			return 0, 0, fmt.Errorf("%w; close: %v", err, closeErr)
		}
		return 0, 0, err
	}
	defer func() {
		_ = flockAppendUnlock(file)
		if closeErr := file.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()
	preStat, err := file.Stat()
	if err != nil {
		return 0, 0, err
	}
	preSize = preStat.Size()
	limiter := &io.LimitedReader{R: reader, N: maxBytes + 1}
	written, err := io.Copy(file, limiter)
	if err != nil {
		if tr := s.TruncateBack(logicalPath, preSize); tr != nil {
			return preSize, 0, fmt.Errorf("%w: rollback truncate failed: %v", err, tr)
		}
		return preSize, 0, err
	}
	if written == 0 {
		return preSize, 0, model.ValidationError(model.FieldError{Field: "body", Code: "required", Message: "append body must not be empty"})
	}
	if written > maxBytes {
		payloadErr := model.PayloadTooLarge("append exceeds configured limit")
		if tr := s.TruncateBack(logicalPath, preSize); tr != nil {
			return preSize, 0, fmt.Errorf("%w: rollback truncate failed: %v", payloadErr, tr)
		}
		return preSize, 0, payloadErr
	}
	if err := file.Sync(); err != nil {
		if tr := s.TruncateBack(logicalPath, preSize); tr != nil {
			return preSize, 0, fmt.Errorf("%w: rollback truncate failed: %v", err, tr)
		}
		return preSize, 0, err
	}
	newSize = preSize + written
	return preSize, newSize, nil
}

func SafeUsageHint(value string) string {
	return strings.TrimSpace(value)
}

func TouchTime() time.Time {
	return time.Now().UTC()
}
