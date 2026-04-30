package servicetest

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/mediatype"
	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/model"
	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/objectfiles"
	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/store"
)

// ObjectFileKey identifies an object file in MemoryStore. file_id is only
// unique per object, so the map key is composite. Postgres enforces the same
// identity via PRIMARY KEY (object_id, file_id) and global path uniqueness
// via UNIQUE on object_files.path; the in-memory store checks path conflicts
// separately so test and production semantics stay aligned.
type ObjectFileKey struct {
	ObjectID string
	FileID   string
}

type MemoryStore struct {
	mu           sync.RWMutex
	Entities     map[string]model.Entity
	Observations map[string]model.Observation
	Tasks        map[string]model.Task
	Objects      map[string]model.Object
	ObjectFiles  map[ObjectFileKey]model.ObjectFile
	FileBytes    map[ObjectFileKey][]byte
	Ready        model.DependencyStatus
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		Entities:     map[string]model.Entity{},
		Observations: map[string]model.Observation{},
		Tasks:        map[string]model.Task{},
		Objects:      map[string]model.Object{},
		ObjectFiles:  map[ObjectFileKey]model.ObjectFile{},
		FileBytes:    map[ObjectFileKey][]byte{},
		Ready:        model.DependencyStatus{Status: "ready"},
	}
}

func (m *MemoryStore) CreateEntity(_ context.Context, entity model.Entity) (model.Entity, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.Entities[entity.EntityID]; ok {
		return model.Entity{}, model.Conflict("entity", entity.EntityID, "already_exists", nil)
	}
	m.Entities[entity.EntityID] = entity
	return entity, nil
}
func (m *MemoryStore) GetEntity(_ context.Context, id string) (model.Entity, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	item, ok := m.Entities[id]
	if !ok {
		return model.Entity{}, model.NotFound("entity", id)
	}
	return item, nil
}
func (m *MemoryStore) ListEntities(_ context.Context, filter store.EntityListFilter, pagination model.Pagination) ([]model.Entity, int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	items := []model.Entity{}
	for _, item := range m.Entities {
		if filter.Type == "" || item.Type == filter.Type {
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].UpdatedAt.Equal(items[j].UpdatedAt) {
			return items[i].EntityID < items[j].EntityID
		}
		return items[i].UpdatedAt.After(items[j].UpdatedAt)
	})
	return paginate(items, pagination), len(items), nil
}
func (m *MemoryStore) UpdateEntity(_ context.Context, entity model.Entity) (model.Entity, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.Entities[entity.EntityID]; !ok {
		return model.Entity{}, model.NotFound("entity", entity.EntityID)
	}
	m.Entities[entity.EntityID] = entity
	return entity, nil
}
func (m *MemoryStore) DeleteEntity(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.Entities[id]; !ok {
		return model.NotFound("entity", id)
	}
	delete(m.Entities, id)
	return nil
}
func (m *MemoryStore) CountEntityDependents(_ context.Context, id string) (map[string]int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	counts := map[string]int{"tasks": 0, "observations": 0, "objects": 0}
	for _, t := range m.Tasks {
		if t.AssetID == id {
			counts["tasks"]++
		}
	}
	for _, o := range m.Observations {
		if o.SourceAssetID == id {
			counts["observations"]++
		}
	}
	for _, o := range m.Objects {
		if o.OwnerType == "entity" && o.OwnerID == id {
			counts["objects"]++
		}
	}
	return counts, nil
}
func (m *MemoryStore) CreateObservation(_ context.Context, item model.Observation) (model.Observation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.Observations[item.ObservationID]; ok {
		return model.Observation{}, model.Conflict("observation", item.ObservationID, "already_exists", nil)
	}
	m.Observations[item.ObservationID] = item
	return item, nil
}
func (m *MemoryStore) GetObservation(_ context.Context, id string) (model.Observation, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	item, ok := m.Observations[id]
	if !ok {
		return model.Observation{}, model.NotFound("observation", id)
	}
	return item, nil
}
func (m *MemoryStore) ListObservations(_ context.Context, filter store.ObservationListFilter, pagination model.Pagination) ([]model.Observation, int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	items := []model.Observation{}
	var updatedAfter time.Time
	if filter.UpdatedAfter != "" {
		var err error
		updatedAfter, err = time.Parse(time.RFC3339Nano, filter.UpdatedAfter)
		if err != nil {
			return nil, 0, model.ValidationError(model.FieldError{Field: "updated_after", Code: "invalid_format", Message: "must be RFC3339 format"})
		}
	}
	for _, item := range m.Observations {
		if filter.SourceAssetID != "" && item.SourceAssetID != filter.SourceAssetID {
			continue
		}
		// Match Postgres' `updated_at >= $N` (inclusive) — skip only rows
		// strictly before the cutoff, so equal-timestamp rows are returned.
		if filter.UpdatedAfter != "" && item.UpdatedAt.Before(updatedAfter) {
			continue
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].UpdatedAt.Equal(items[j].UpdatedAt) {
			return items[i].ObservationID < items[j].ObservationID
		}
		return items[i].UpdatedAt.After(items[j].UpdatedAt)
	})
	return paginate(items, pagination), len(items), nil
}
func (m *MemoryStore) UpdateObservation(_ context.Context, item model.Observation) (model.Observation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.Observations[item.ObservationID]; !ok {
		return model.Observation{}, model.NotFound("observation", item.ObservationID)
	}
	m.Observations[item.ObservationID] = item
	return item, nil
}
func (m *MemoryStore) DeleteObservation(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.Observations[id]; !ok {
		return model.NotFound("observation", id)
	}
	delete(m.Observations, id)
	for objectID, object := range m.Objects {
		if object.OwnerType == "observation" && object.OwnerID == id {
			delete(m.Objects, objectID)
			for key, file := range m.ObjectFiles {
				if file.ObjectID == objectID {
					delete(m.ObjectFiles, key)
					delete(m.FileBytes, key)
				}
			}
		}
	}
	return nil
}
func (m *MemoryStore) CreateTask(_ context.Context, item model.Task) (model.Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.Tasks[item.TaskID]; ok {
		return model.Task{}, model.Conflict("task", item.TaskID, "already_exists", nil)
	}
	m.Tasks[item.TaskID] = item
	return item, nil
}
func (m *MemoryStore) GetTask(_ context.Context, id string) (model.Task, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	item, ok := m.Tasks[id]
	if !ok {
		return model.Task{}, model.NotFound("task", id)
	}
	return item, nil
}
func (m *MemoryStore) ListTasks(_ context.Context, filter store.TaskListFilter, pagination model.Pagination) ([]model.Task, int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	items := []model.Task{}
	for _, item := range m.Tasks {
		if filter.AssetID != "" && item.AssetID != filter.AssetID {
			continue
		}
		if filter.Status != "" && item.Status != filter.Status {
			continue
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].UpdatedAt.Equal(items[j].UpdatedAt) {
			return items[i].TaskID < items[j].TaskID
		}
		return items[i].UpdatedAt.After(items[j].UpdatedAt)
	})
	return paginate(items, pagination), len(items), nil
}
func (m *MemoryStore) UpdateTask(_ context.Context, item model.Task) (model.Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.Tasks[item.TaskID]; !ok {
		return model.Task{}, model.NotFound("task", item.TaskID)
	}
	m.Tasks[item.TaskID] = item
	return item, nil
}
func (m *MemoryStore) DeleteTask(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.Tasks[id]; !ok {
		return model.NotFound("task", id)
	}
	delete(m.Tasks, id)
	return nil
}
func (m *MemoryStore) ListTasksForAsset(ctx context.Context, assetID string, pagination model.Pagination) ([]model.Task, int, error) {
	return m.ListTasks(ctx, store.TaskListFilter{AssetID: assetID}, pagination)
}
func (m *MemoryStore) CountTaskOwnedObjects(_ context.Context, taskID string) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	count := 0
	for _, object := range m.Objects {
		if object.OwnerType == "task" && object.OwnerID == taskID {
			count++
		}
	}
	return count, nil
}
func (m *MemoryStore) CreateObject(_ context.Context, item model.Object) (model.Object, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.Objects[item.ObjectID]; ok {
		return model.Object{}, model.Conflict("object", item.ObjectID, "already_exists", nil)
	}
	m.Objects[item.ObjectID] = item
	return item, nil
}
func (m *MemoryStore) GetObject(_ context.Context, id string) (model.Object, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	item, ok := m.Objects[id]
	if !ok {
		return model.Object{}, model.NotFound("object", id)
	}
	return item, nil
}
func (m *MemoryStore) ListObjects(_ context.Context, filter store.ObjectListFilter, pagination model.Pagination) ([]model.Object, int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	items := []model.Object{}
	for _, item := range m.Objects {
		if filter.OwnerType != "" && (item.OwnerType != filter.OwnerType || item.OwnerID != filter.OwnerID) {
			continue
		}
		if filter.Type != "" && item.Type != filter.Type {
			continue
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].UpdatedAt.Equal(items[j].UpdatedAt) {
			return items[i].ObjectID < items[j].ObjectID
		}
		return items[i].UpdatedAt.After(items[j].UpdatedAt)
	})
	return paginate(items, pagination), len(items), nil
}
func memUploadLimit(requested int64) int64 {
	if requested > 0 {
		return requested
	}
	return 16 * 1024 * 1024
}

func (m *MemoryStore) UpdateObject(_ context.Context, item model.Object) (model.Object, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.Objects[item.ObjectID]; !ok {
		return model.Object{}, model.NotFound("object", item.ObjectID)
	}
	m.Objects[item.ObjectID] = item
	return item, nil
}
func (m *MemoryStore) DeleteObject(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.Objects[id]; !ok {
		return model.NotFound("object", id)
	}
	delete(m.Objects, id)
	for key, file := range m.ObjectFiles {
		if file.ObjectID == id {
			delete(m.ObjectFiles, key)
			delete(m.FileBytes, key)
		}
	}
	return nil
}
func (m *MemoryStore) CreateObjectFile(_ context.Context, input store.ObjectUploadInput) (model.ObjectFile, error) {
	// Honour the ObjectUploadInput contract: when a PreStagedPath is provided,
	// implementations must remove it on both success and error paths (the only
	// exception being a post-commit byte-promotion failure, which the in-memory
	// store does not have). A defer is the cleanest way to guarantee that.
	if input.PreStagedPath != "" {
		defer func() { _ = objectfiles.CleanupStagedPath(input.PreStagedPath, stagingDirForPath(input.PreStagedPath)) }()
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.Objects[input.File.ObjectID]; !ok {
		return model.ObjectFile{}, model.NotFound("object", input.File.ObjectID)
	}
	// Validate the path segments before any I/O. Postgres performs this via
	// objectfiles.Store.LogicalPath; the in-memory store must agree so that
	// invalid IDs are rejected the same way in both backends.
	if err := objectfiles.ValidateObjectFilePathSegments(input.File.ObjectID, input.File.FileID); err != nil {
		return model.ObjectFile{}, err
	}
	logicalPath := filepath.ToSlash(filepath.Join("objects", input.File.ObjectID, input.File.FileID))
	key := ObjectFileKey{ObjectID: input.File.ObjectID, FileID: input.File.FileID}
	if _, ok := m.ObjectFiles[key]; ok {
		return model.ObjectFile{}, model.Conflict("object_file", input.File.FileID, "already_exists", nil)
	}
	// Global path uniqueness — mirrors `path text NOT NULL UNIQUE` in the
	// Postgres schema. Two different (object_id, file_id) pairs can produce
	// the same path only if a caller bypasses LogicalPath, but checking
	// defensively here keeps the test store honest.
	for _, existing := range m.ObjectFiles {
		if existing.Path == logicalPath {
			return model.ObjectFile{}, model.Conflict("object_file", input.File.FileID, "already_exists", nil)
		}
	}
	var bytesValue []byte
	var err error
	if input.PreStagedPath != "" {
		limit := memUploadLimit(input.MaxBytes)
		if input.PreStagedSizeBytes < 0 {
			return model.ObjectFile{}, model.ValidationError(model.FieldError{Field: "pre_staged", Code: "invalid_value", Message: "pre-staged size must not be negative"})
		}
		info, lerr := os.Lstat(input.PreStagedPath)
		if lerr != nil {
			return model.ObjectFile{}, lerr
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return model.ObjectFile{}, model.ValidationError(model.FieldError{Field: "pre_staged", Code: "invalid_value", Message: "pre-staged path must not be a symbolic link"})
		}
		if !info.Mode().IsRegular() {
			return model.ObjectFile{}, model.ValidationError(model.FieldError{Field: "pre_staged", Code: "invalid_value", Message: "pre-staged path must be a regular file"})
		}
		f, openErr := os.Open(input.PreStagedPath)
		if openErr != nil {
			return model.ObjectFile{}, openErr
		}
		defer func() { _ = f.Close() }()
		if info.Size() != input.PreStagedSizeBytes {
			return model.ObjectFile{}, model.ValidationError(model.FieldError{Field: "pre_staged", Code: "invalid_value", Message: "pre-staged size does not match file"})
		}
		if info.Size() == 0 {
			return model.ObjectFile{}, model.ValidationError(model.FieldError{Field: "file", Code: "required", Message: "file bytes are required"})
		}
		if info.Size() > limit {
			return model.ObjectFile{}, model.PayloadTooLarge("upload exceeds configured limit")
		}
		lr := &io.LimitedReader{R: f, N: info.Size() + 1}
		bytesValue, err = io.ReadAll(lr)
		if err != nil {
			return model.ObjectFile{}, err
		}
		if int64(len(bytesValue)) != info.Size() {
			return model.ObjectFile{}, model.ValidationError(model.FieldError{Field: "pre_staged", Code: "invalid_value", Message: "pre-staged file size changed during read"})
		}
		// Cleanup of the pre-staged file is handled by the deferred removal at
		// the top of CreateObjectFile, which fires on every return path.
	} else {
		limit := memUploadLimit(input.MaxBytes)
		lr := &io.LimitedReader{R: input.Reader, N: limit + 1}
		bytesValue, err = io.ReadAll(lr)
		if err != nil {
			return model.ObjectFile{}, err
		}
	}
	limit := memUploadLimit(input.MaxBytes)
	if int64(len(bytesValue)) == 0 {
		return model.ObjectFile{}, model.ValidationError(model.FieldError{Field: "file", Code: "required", Message: "file bytes are required"})
	}
	if int64(len(bytesValue)) > limit {
		return model.ObjectFile{}, model.PayloadTooLarge("upload exceeds configured limit")
	}
	file := input.File
	file.Path = logicalPath
	file.SizeBytes = int64(len(bytesValue))
	if normalized, valid := mediatype.NormalizeContentType(file.ContentType); valid && strings.TrimSpace(file.ContentType) != "" {
		file.ContentType = normalized
	} else if input.PreStagedPath != "" {
		file.ContentType, _ = mediatype.NormalizeContentType(input.PreStagedContentType)
	} else {
		file.ContentType = "application/octet-stream"
	}
	now := time.Now().UTC()
	if file.CreatedAt.IsZero() {
		file.CreatedAt = now
	}
	if file.UpdatedAt.IsZero() {
		file.UpdatedAt = now
	}
	m.ObjectFiles[key] = file
	m.FileBytes[key] = bytesValue
	return file, nil
}
func (m *MemoryStore) GetObjectFile(_ context.Context, objectID, fileID string) (model.ObjectFile, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	key := ObjectFileKey{ObjectID: objectID, FileID: fileID}
	file, ok := m.ObjectFiles[key]
	if !ok || file.ObjectID != objectID {
		return model.ObjectFile{}, model.NotFound("object_file", fileID)
	}
	return file, nil
}
func (m *MemoryStore) AppendObjectFile(_ context.Context, objectID, fileID string, reader io.Reader, maxBytes int64) (model.ObjectFile, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := ObjectFileKey{ObjectID: objectID, FileID: fileID}
	file, ok := m.ObjectFiles[key]
	if !ok || file.ObjectID != objectID {
		return model.ObjectFile{}, model.NotFound("object_file", fileID)
	}
	limit := memUploadLimit(maxBytes)
	lr := &io.LimitedReader{R: reader, N: limit + 1}
	bytesValue, err := io.ReadAll(lr)
	if err != nil {
		return model.ObjectFile{}, err
	}
	if len(bytesValue) == 0 {
		return model.ObjectFile{}, model.ValidationError(model.FieldError{Field: "body", Code: "required", Message: "append body must not be empty"})
	}
	if int64(len(bytesValue)) > limit {
		return model.ObjectFile{}, model.PayloadTooLarge("append exceeds configured limit")
	}
	m.FileBytes[key] = append(m.FileBytes[key], bytesValue...)
	file.SizeBytes = int64(len(m.FileBytes[key]))
	file.UpdatedAt = time.Now().UTC()
	m.ObjectFiles[key] = file
	return file, nil
}
func (m *MemoryStore) OpenObjectFile(_ context.Context, objectID, fileID string) (model.ObjectFile, io.ReadCloser, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	key := ObjectFileKey{ObjectID: objectID, FileID: fileID}
	file, ok := m.ObjectFiles[key]
	if !ok || file.ObjectID != objectID {
		return model.ObjectFile{}, nil, model.NotFound("object_file", fileID)
	}
	return file, io.NopCloser(bytes.NewReader(m.FileBytes[key])), nil
}
func (m *MemoryStore) DeleteObjectFile(_ context.Context, objectID, fileID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := ObjectFileKey{ObjectID: objectID, FileID: fileID}
	file, ok := m.ObjectFiles[key]
	if !ok || file.ObjectID != objectID {
		return model.NotFound("object_file", fileID)
	}
	delete(m.ObjectFiles, key)
	delete(m.FileBytes, key)
	return nil
}
func (m *MemoryStore) ListObjectFilesForObject(_ context.Context, objectID string) ([]model.ObjectFile, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	items := []model.ObjectFile{}
	for _, file := range m.ObjectFiles {
		if file.ObjectID == objectID {
			items = append(items, file)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].FileID < items[j].FileID })
	return items, nil
}
func (m *MemoryStore) StorageStatus(context.Context) model.DependencyStatus { return m.Ready }
func (m *MemoryStore) GetFullQueryState(_ context.Context) (store.QueryState, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return store.QueryState{Entities: values(m.Entities), Observations: values(m.Observations), Tasks: values(m.Tasks), Objects: values(m.Objects), ObjectFiles: objectFileValues(m.ObjectFiles)}, nil
}

func paginate[T any](items []T, pagination model.Pagination) []T {
	if pagination.Offset >= len(items) {
		return []T{}
	}
	end := pagination.Offset + pagination.Limit
	if end > len(items) {
		end = len(items)
	}
	return append([]T(nil), items[pagination.Offset:end]...)
}

func objectFileValues(input map[ObjectFileKey]model.ObjectFile) []model.ObjectFile {
	out := make([]model.ObjectFile, 0, len(input))
	for _, value := range input {
		out = append(out, value)
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].UpdatedAt.Equal(out[j].UpdatedAt) {
			return out[i].UpdatedAt.After(out[j].UpdatedAt)
		}
		if out[i].ObjectID != out[j].ObjectID {
			return out[i].ObjectID < out[j].ObjectID
		}
		return out[i].FileID < out[j].FileID
	})
	return out
}

func stagingDirForPath(path string) string {
	for dir := filepath.Dir(path); dir != "." && dir != string(filepath.Separator); dir = filepath.Dir(dir) {
		if filepath.Base(dir) == "staging" {
			return dir
		}
	}
	return ""
}

func values[T any](input map[string]T) []T {
	out := make([]T, 0, len(input))
	for _, value := range input {
		out = append(out, value)
	}
	return out
}
