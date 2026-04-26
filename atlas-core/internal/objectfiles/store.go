package objectfiles

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

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
	_ = os.Remove(file)
	return nil
}

func (s *Store) LogicalPath(objectID, fileID string) string {
	return filepath.ToSlash(filepath.Join("objects", objectID, fileID))
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
	return temp.Name(), size, normalizeContentType(detectedType), nil
}

func (s *Store) Promote(stagePath, logicalPath string) error {
	target, err := s.AbsolutePath(logicalPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	return os.Rename(stagePath, target)
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

func (s *Store) Append(logicalPath string, reader io.Reader, maxBytes int64) (int64, error) {
	target, err := s.AbsolutePath(logicalPath)
	if err != nil {
		return 0, err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return 0, err
	}
	preStat, err := os.Stat(target)
	if err != nil {
		return 0, err
	}
	preSize := preStat.Size()
	file, err := os.OpenFile(target, os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return 0, err
	}
	limiter := &io.LimitedReader{R: reader, N: maxBytes + 1}
	written, err := io.Copy(file, limiter)
	if err != nil {
		_ = file.Close()
		if tr := s.TruncateBack(logicalPath, preSize); tr != nil {
			return 0, fmt.Errorf("%w: rollback truncate failed: %v", err, tr)
		}
		return 0, err
	}
	if written == 0 {
		_ = file.Close()
		return 0, model.ValidationError(model.FieldError{Field: "body", Code: "required", Message: "append body must not be empty"})
	}
	if written > maxBytes {
		_ = file.Close()
		if tr := s.TruncateBack(logicalPath, preSize); tr != nil {
			return 0, fmt.Errorf("append payload too large: rollback truncate failed: %v", tr)
		}
		return 0, model.PayloadTooLarge("append exceeds configured limit")
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		if tr := s.TruncateBack(logicalPath, preSize); tr != nil {
			return 0, fmt.Errorf("%w: rollback truncate failed: %v", err, tr)
		}
		return 0, err
	}
	if err := file.Close(); err != nil {
		if tr := s.TruncateBack(logicalPath, preSize); tr != nil {
			return 0, fmt.Errorf("%w: rollback truncate failed: %v", err, tr)
		}
		return 0, err
	}
	st, err := os.Stat(target)
	if err != nil {
		return 0, err
	}
	return st.Size(), nil
}

func normalizeContentType(value string) string {
	if value == "" {
		return "application/octet-stream"
	}
	mediaType, _, err := mime.ParseMediaType(value)
	if err != nil {
		return value
	}
	return mediaType
}

func SafeUsageHint(value string) string {
	return strings.TrimSpace(value)
}

func TouchTime() time.Time {
	return time.Now().UTC()
}
