package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/model"
	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/objectfiles"
	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/store"
)

type Store struct {
	pool      *pgxpool.Pool
	files     *objectfiles.Store
	maxUpload int64
	logger    *slog.Logger
}

// NewStore constructs a Store. logger is used for best-effort post-commit
// cleanup paths where errors cannot be returned to the caller. If nil,
// [slog.Default] is used.
func NewStore(pool *pgxpool.Pool, files *objectfiles.Store, maxUpload int64, logger *slog.Logger) *Store {
	if logger == nil {
		logger = slog.Default()
	}
	return &Store{pool: pool, files: files, maxUpload: maxUpload, logger: logger}
}

func (s *Store) StorageStatus(context.Context) model.DependencyStatus {
	return s.files.Status()
}

func (s *Store) CreateEntity(ctx context.Context, entity model.Entity) (model.Entity, error) {
	return entity, s.execUpsert(ctx, `INSERT INTO entities (entity_id, type, subtype, alias, json, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7)`, "entity", entity.EntityID, entity.EntityID, entity.Type, nullString(entity.Subtype), nullString(entity.Alias), mustJSON(entity.JSON), entity.CreatedAt, entity.UpdatedAt)
}

func (s *Store) GetEntity(ctx context.Context, id string) (model.Entity, error) {
	row := s.pool.QueryRow(ctx, `SELECT entity_id, type, COALESCE(subtype,''), COALESCE(alias,''), json, created_at, updated_at FROM entities WHERE entity_id=$1`, id)
	entity, err := scanEntity(row)
	if err != nil {
		return model.Entity{}, mapNotFound(err, "entity", id)
	}
	return entity, nil
}

func (s *Store) ListEntities(ctx context.Context, filter store.EntityListFilter, pagination model.Pagination) ([]model.Entity, int, error) {
	where := ""
	args := []any{}
	if filter.Type != "" {
		where = " WHERE type=$1"
		args = append(args, filter.Type)
	}
	total, err := s.count(ctx, `SELECT count(*) FROM entities`+where, args...)
	if err != nil {
		return nil, 0, err
	}
	args = append(args, pagination.Limit, pagination.Offset)
	rows, err := s.pool.Query(ctx, `SELECT entity_id, type, COALESCE(subtype,''), COALESCE(alias,''), json, created_at, updated_at FROM entities`+where+` ORDER BY updated_at DESC, entity_id ASC LIMIT $`+itoa(len(args)-1)+` OFFSET $`+itoa(len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	items, _, err := collectEntities(rows)
	return items, total, err
}

func (s *Store) UpdateEntity(ctx context.Context, entity model.Entity) (model.Entity, error) {
	commandTag, err := s.pool.Exec(ctx, `UPDATE entities SET subtype=$2, alias=$3, json=$4, updated_at=$5 WHERE entity_id=$1`, entity.EntityID, nullString(entity.Subtype), nullString(entity.Alias), mustJSON(entity.JSON), entity.UpdatedAt)
	if err != nil {
		return model.Entity{}, err
	}
	if commandTag.RowsAffected() == 0 {
		return model.Entity{}, model.NotFound("entity", entity.EntityID)
	}
	return s.GetEntity(ctx, entity.EntityID)
}

func (s *Store) DeleteEntity(ctx context.Context, id string) error {
	result, err := s.pool.Exec(ctx, `DELETE FROM entities WHERE entity_id=$1`, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return model.NotFound("entity", id)
	}
	return nil
}

func (s *Store) CountEntityDependents(ctx context.Context, id string) (map[string]int, error) {
	counts := map[string]int{}
	if count, err := s.count(ctx, `SELECT count(*) FROM tasks WHERE asset_id=$1`, id); err != nil {
		return nil, err
	} else {
		counts["tasks"] = count
	}
	if count, err := s.count(ctx, `SELECT count(*) FROM observations WHERE source_asset_id=$1`, id); err != nil {
		return nil, err
	} else {
		counts["observations"] = count
	}
	if count, err := s.count(ctx, `SELECT count(*) FROM objects WHERE owner_type='entity' AND owner_id=$1`, id); err != nil {
		return nil, err
	} else {
		counts["objects"] = count
	}
	return counts, nil
}

func (s *Store) CreateObservation(ctx context.Context, observation model.Observation) (model.Observation, error) {
	return observation, s.execUpsert(ctx, `INSERT INTO observations (observation_id, source_asset_id, json, created_at, updated_at) VALUES ($1,$2,$3,$4,$5)`, "observation", observation.ObservationID, observation.ObservationID, observation.SourceAssetID, mustJSON(observation.JSON), observation.CreatedAt, observation.UpdatedAt)
}

func (s *Store) GetObservation(ctx context.Context, id string) (model.Observation, error) {
	row := s.pool.QueryRow(ctx, `SELECT observation_id, source_asset_id, json, created_at, updated_at FROM observations WHERE observation_id=$1`, id)
	observation, err := scanObservation(row)
	if err != nil {
		return model.Observation{}, mapNotFound(err, "observation", id)
	}
	return observation, nil
}

func (s *Store) ListObservations(ctx context.Context, filter store.ObservationListFilter, pagination model.Pagination) ([]model.Observation, int, error) {
	clauses := []string{}
	args := []any{}
	if filter.SourceAssetID != "" {
		args = append(args, filter.SourceAssetID)
		clauses = append(clauses, fmt.Sprintf("source_asset_id=$%d", len(args)))
	}
	if filter.UpdatedAfter != "" {
		args = append(args, filter.UpdatedAfter)
		clauses = append(clauses, fmt.Sprintf("updated_at >= $%d::timestamptz", len(args)))
	}
	where := joinWhere(clauses)
	total, err := s.count(ctx, `SELECT count(*) FROM observations`+where, args...)
	if err != nil {
		return nil, 0, err
	}
	args = append(args, pagination.Limit, pagination.Offset)
	query := `SELECT observation_id, source_asset_id, json, created_at, updated_at FROM observations` + where + fmt.Sprintf(` ORDER BY updated_at DESC, observation_id ASC LIMIT $%d OFFSET $%d`, len(args)-1, len(args))
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	return collectObservations(rows, total)
}

func (s *Store) UpdateObservation(ctx context.Context, observation model.Observation) (model.Observation, error) {
	result, err := s.pool.Exec(ctx, `UPDATE observations SET json=$2, updated_at=$3 WHERE observation_id=$1`, observation.ObservationID, mustJSON(observation.JSON), observation.UpdatedAt)
	if err != nil {
		return model.Observation{}, err
	}
	if result.RowsAffected() == 0 {
		return model.Observation{}, model.NotFound("observation", observation.ObservationID)
	}
	return s.GetObservation(ctx, observation.ObservationID)
}

func (s *Store) DeleteObservation(ctx context.Context, id string) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	paths, err := filePathsForOwner(ctx, tx, "observation", id)
	if err != nil {
		return err
	}
	result, err := tx.Exec(ctx, `DELETE FROM observations WHERE observation_id=$1`, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return model.NotFound("observation", id)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM objects WHERE owner_type='observation' AND owner_id=$1`, id); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	s.postCommitObjectFilePathsDelete("observation", id, paths)
	return nil
}

func (s *Store) CreateTask(ctx context.Context, task model.Task) (model.Task, error) {
	return task, s.execUpsert(ctx, `INSERT INTO tasks (task_id, status, asset_id, command_catalog_object_id, json, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7)`, "task", task.TaskID, task.TaskID, task.Status, task.AssetID, task.CommandCatalogObjectID, mustJSON(task.JSON), task.CreatedAt, task.UpdatedAt)
}

func (s *Store) GetTask(ctx context.Context, id string) (model.Task, error) {
	row := s.pool.QueryRow(ctx, `SELECT task_id, status, asset_id, command_catalog_object_id, json, created_at, updated_at FROM tasks WHERE task_id=$1`, id)
	task, err := scanTask(row)
	if err != nil {
		return model.Task{}, mapNotFound(err, "task", id)
	}
	return task, nil
}

func (s *Store) ListTasks(ctx context.Context, filter store.TaskListFilter, pagination model.Pagination) ([]model.Task, int, error) {
	clauses := []string{}
	args := []any{}
	if filter.AssetID != "" {
		args = append(args, filter.AssetID)
		clauses = append(clauses, fmt.Sprintf("asset_id=$%d", len(args)))
	}
	if filter.Status != "" {
		args = append(args, filter.Status)
		clauses = append(clauses, fmt.Sprintf("status=$%d", len(args)))
	}
	where := joinWhere(clauses)
	total, err := s.count(ctx, `SELECT count(*) FROM tasks`+where, args...)
	if err != nil {
		return nil, 0, err
	}
	args = append(args, pagination.Limit, pagination.Offset)
	query := `SELECT task_id, status, asset_id, command_catalog_object_id, json, created_at, updated_at FROM tasks` + where + fmt.Sprintf(` ORDER BY updated_at DESC, task_id ASC LIMIT $%d OFFSET $%d`, len(args)-1, len(args))
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	return collectTasks(rows, total)
}

func (s *Store) UpdateTask(ctx context.Context, task model.Task) (model.Task, error) {
	result, err := s.pool.Exec(ctx, `UPDATE tasks SET status=$2, json=$3, updated_at=$4 WHERE task_id=$1`, task.TaskID, task.Status, mustJSON(task.JSON), task.UpdatedAt)
	if err != nil {
		return model.Task{}, err
	}
	if result.RowsAffected() == 0 {
		return model.Task{}, model.NotFound("task", task.TaskID)
	}
	return s.GetTask(ctx, task.TaskID)
}

func (s *Store) DeleteTask(ctx context.Context, id string) error {
	result, err := s.pool.Exec(ctx, `DELETE FROM tasks WHERE task_id=$1`, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return model.NotFound("task", id)
	}
	return nil
}

func (s *Store) ListTasksForAsset(ctx context.Context, assetID string, pagination model.Pagination) ([]model.Task, int, error) {
	return s.ListTasks(ctx, store.TaskListFilter{AssetID: assetID}, pagination)
}

func (s *Store) CountTaskOwnedObjects(ctx context.Context, taskID string) (int, error) {
	return s.count(ctx, `SELECT count(*) FROM objects WHERE owner_type='task' AND owner_id=$1`, taskID)
}

func (s *Store) CreateObject(ctx context.Context, object model.Object) (model.Object, error) {
	return object, s.execUpsert(ctx, `INSERT INTO objects (object_id, type, owner_type, owner_id, json, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7)`, "object", object.ObjectID, object.ObjectID, object.Type, object.OwnerType, object.OwnerID, mustJSON(object.JSON), object.CreatedAt, object.UpdatedAt)
}

func (s *Store) GetObject(ctx context.Context, id string) (model.Object, error) {
	row := s.pool.QueryRow(ctx, `SELECT object_id, type, owner_type, owner_id, json, created_at, updated_at FROM objects WHERE object_id=$1`, id)
	object, err := scanObject(row)
	if err != nil {
		return model.Object{}, mapNotFound(err, "object", id)
	}
	return object, nil
}

func (s *Store) ListObjects(ctx context.Context, filter store.ObjectListFilter, pagination model.Pagination) ([]model.Object, int, error) {
	clauses := []string{}
	args := []any{}
	if filter.OwnerType != "" {
		args = append(args, filter.OwnerType)
		clauses = append(clauses, fmt.Sprintf("owner_type=$%d", len(args)))
		args = append(args, filter.OwnerID)
		clauses = append(clauses, fmt.Sprintf("owner_id=$%d", len(args)))
	}
	if filter.Type != "" {
		args = append(args, filter.Type)
		clauses = append(clauses, fmt.Sprintf("type=$%d", len(args)))
	}
	where := joinWhere(clauses)
	total, err := s.count(ctx, `SELECT count(*) FROM objects`+where, args...)
	if err != nil {
		return nil, 0, err
	}
	args = append(args, pagination.Limit, pagination.Offset)
	query := `SELECT object_id, type, owner_type, owner_id, json, created_at, updated_at FROM objects` + where + fmt.Sprintf(` ORDER BY updated_at DESC, object_id ASC LIMIT $%d OFFSET $%d`, len(args)-1, len(args))
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	return collectObjects(rows, total)
}

func (s *Store) UpdateObject(ctx context.Context, object model.Object) (model.Object, error) {
	result, err := s.pool.Exec(ctx, `UPDATE objects SET type=$2, json=$3, updated_at=$4 WHERE object_id=$1`, object.ObjectID, object.Type, mustJSON(object.JSON), object.UpdatedAt)
	if err != nil {
		return model.Object{}, err
	}
	if result.RowsAffected() == 0 {
		return model.Object{}, model.NotFound("object", object.ObjectID)
	}
	return s.GetObject(ctx, object.ObjectID)
}

func (s *Store) DeleteObject(ctx context.Context, id string) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	paths, err := filePathsForObject(ctx, tx, id)
	if err != nil {
		return err
	}
	result, err := tx.Exec(ctx, `DELETE FROM objects WHERE object_id=$1`, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return model.NotFound("object", id)
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	s.postCommitObjectFilePathsDelete("object", id, paths)
	return nil
}

// postCommitObjectFilePathsDelete runs after the database transaction has committed. Disk
// cleanup is best-effort: the delete already returned success to the caller; we log and
// mark a mismatch if bytes remain so readiness reflects an operational problem.
// This is not crash-proof cross-process atomicity: if the process dies after commit, files
// may need a separate sweeper.
//
// Failures are logged per-path but do not short-circuit the loop: a transient failure on
// one path must not strand the remaining paths as orphans. MarkMismatch is called once at
// the end if any path failed.
func (s *Store) postCommitObjectFilePathsDelete(contextLabel, resourceID string, paths []string) {
	var hadFailure bool
	for _, path := range paths {
		if err := s.files.Delete(path); err != nil {
			s.logger.Error("post-commit file delete failed after database delete; stored metadata no longer has this object",
				"context", contextLabel,
				"path", path,
				"resource_id", resourceID,
				"err", err,
			)
			hadFailure = true
		}
	}
	if hadFailure {
		s.files.MarkMismatch("post-delete file cleanup failed")
	}
}
func (s *Store) CreateObjectFile(ctx context.Context, input store.ObjectUploadInput) (model.ObjectFile, error) {
	var stagedPath string
	var size int64
	var detectedType string
	removeStaged := true
	defer func() {
		if removeStaged && stagedPath != "" {
			_ = os.Remove(stagedPath)
		}
	}()

	if input.PreStagedPath != "" {
		stagingDir := s.files.StagingDir()
		absPath, absErr := filepath.Abs(input.PreStagedPath)
		if absErr != nil {
			return model.ObjectFile{}, model.ValidationError(model.FieldError{Field: "pre_staged", Code: "invalid_value", Message: "pre-staged path could not be resolved"})
		}
		absStaging, stagingAbsErr := filepath.Abs(stagingDir)
		if stagingAbsErr != nil {
			return model.ObjectFile{}, fmt.Errorf("staging directory: %w", stagingAbsErr)
		}
		if !strings.HasPrefix(absPath, absStaging+string(filepath.Separator)) && absPath != absStaging {
			return model.ObjectFile{}, model.ValidationError(model.FieldError{Field: "pre_staged", Code: "invalid_value", Message: "pre-staged path must be within the staging directory"})
		}
		limit := chooseLimit(input.MaxBytes, s.maxUpload)
		if input.PreStagedSizeBytes < 0 {
			return model.ObjectFile{}, model.ValidationError(model.FieldError{Field: "pre_staged", Code: "invalid_value", Message: "pre-staged size must not be negative"})
		}
		stagedPath = input.PreStagedPath
		info, err := os.Lstat(stagedPath)
		if err != nil {
			return model.ObjectFile{}, err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return model.ObjectFile{}, model.ValidationError(model.FieldError{Field: "pre_staged", Code: "invalid_value", Message: "pre-staged path must not be a symbolic link"})
		}
		if !info.Mode().IsRegular() {
			return model.ObjectFile{}, model.ValidationError(model.FieldError{Field: "pre_staged", Code: "invalid_value", Message: "pre-staged path must be a regular file"})
		}
		if info.Size() != input.PreStagedSizeBytes {
			return model.ObjectFile{}, model.ValidationError(model.FieldError{Field: "pre_staged", Code: "invalid_value", Message: "pre-staged size does not match file"})
		}
		if info.Size() == 0 {
			return model.ObjectFile{}, model.ValidationError(model.FieldError{Field: "file", Code: "required", Message: "file bytes are required"})
		}
		if info.Size() > limit {
			return model.ObjectFile{}, model.PayloadTooLarge("upload exceeds configured limit")
		}
		size = info.Size()
		detectedType = input.PreStagedContentType
	} else {
		var err error
		stagedPath, size, detectedType, err = s.files.Stage(ctx, input.File.ObjectID, input.File.FileID, input.Reader, chooseLimit(input.MaxBytes, s.maxUpload))
		if err != nil {
			return model.ObjectFile{}, err
		}
	}
	// Default-clean-up the staged path. Two paths intentionally suppress this:
	//  (1) Promote success — the rename consumed the file, so a follow-up
	//      os.Remove would race against the destination; and
	//  (2) Promote failure after the metadata commit — we keep the staged
	//      bytes for forensics / manual recovery (see StorageUnavailable below).
	file := input.File
	logical, err := s.files.LogicalPath(file.ObjectID, file.FileID)
	if err != nil {
		return model.ObjectFile{}, err
	}
	file.Path = logical
	file.SizeBytes = size
	if file.ContentType == "" {
		file.ContentType = detectedType
		if file.ContentType == "" {
			file.ContentType = "application/octet-stream"
		}
	}
	now := time.Now().UTC()
	file.CreatedAt = now
	file.UpdatedAt = now
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return model.ObjectFile{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := ensureObjectExists(ctx, tx, file.ObjectID); err != nil {
		return model.ObjectFile{}, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO object_files (file_id, object_id, path, content_type, size_bytes, usage_hint, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`, file.FileID, file.ObjectID, file.Path, file.ContentType, file.SizeBytes, nullString(file.UsageHint), file.CreatedAt, file.UpdatedAt); err != nil {
		return model.ObjectFile{}, mapPGError(err, "object_file", file.FileID)
	}
	if _, err := tx.Exec(ctx, `UPDATE objects SET updated_at=$2 WHERE object_id=$1`, file.ObjectID, now); err != nil {
		return model.ObjectFile{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return model.ObjectFile{}, err
	}
	if err := s.files.Promote(stagedPath, file.Path); err != nil {
		removeStaged = false
		s.files.MarkMismatch("object file metadata committed but byte promotion failed")
		return model.ObjectFile{}, model.StorageUnavailable("object storage mismatch detected", map[string]any{
			"object_id":   file.ObjectID,
			"file_id":     file.FileID,
			"staged_path": stagedPath,
		})
	}
	removeStaged = false
	return file, nil
}

func (s *Store) GetObjectFile(ctx context.Context, objectID, fileID string) (model.ObjectFile, error) {
	row := s.pool.QueryRow(ctx, `SELECT file_id, object_id, path, content_type, size_bytes, COALESCE(usage_hint,''), created_at, updated_at FROM object_files WHERE file_id=$1 AND object_id=$2`, fileID, objectID)
	file, err := scanObjectFile(row)
	if err != nil {
		return model.ObjectFile{}, mapNotFound(err, "object_file", fileID)
	}
	return file, nil
}

// AppendObjectFile appends to the on-disk file after locking the row, then updates
// metadata. If the database update or commit fails after a successful write, the file
// is truncated to the pre-append size (best-effort; not full cross-media crash atomicity).
func (s *Store) AppendObjectFile(ctx context.Context, objectID, fileID string, reader io.Reader, maxBytes int64) (model.ObjectFile, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return model.ObjectFile{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	row := tx.QueryRow(ctx, `SELECT file_id, object_id, path, content_type, size_bytes, COALESCE(usage_hint,''), created_at, updated_at FROM object_files WHERE file_id=$1 AND object_id=$2 FOR UPDATE`, fileID, objectID)
	file, err := scanObjectFile(row)
	if err != nil {
		return model.ObjectFile{}, mapNotFound(err, "object_file", fileID)
	}
	// Roll back to the real pre-append disk size returned by Append, NOT to the
	// DB-recorded size (file.SizeBytes). The DB and disk can already be out of
	// sync (a prior commit failure raises MarkMismatch but leaves bytes on
	// disk), and truncating to a stale metadata size would discard data that
	// was on disk before this request.
	preDisk, newSize, err := s.files.Append(file.Path, reader, chooseLimit(maxBytes, s.maxUpload))
	if err != nil {
		return model.ObjectFile{}, err
	}
	now := time.Now().UTC()
	if _, err := tx.Exec(ctx, `UPDATE object_files SET size_bytes=$3, updated_at=$4 WHERE file_id=$1 AND object_id=$2`, fileID, objectID, newSize, now); err != nil {
		if tr := s.files.TruncateBack(file.Path, preDisk); tr != nil {
			s.files.MarkMismatch("object file append: database update failed; rollback truncate also failed: " + tr.Error())
		}
		return model.ObjectFile{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE objects SET updated_at=$2 WHERE object_id=$1`, objectID, now); err != nil {
		if tr := s.files.TruncateBack(file.Path, preDisk); tr != nil {
			s.files.MarkMismatch("object file append: object update failed; rollback truncate also failed: " + tr.Error())
		}
		return model.ObjectFile{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		if tr := s.files.TruncateBack(file.Path, preDisk); tr != nil {
			s.files.MarkMismatch("object file append: commit failed; rollback truncate also failed: " + tr.Error())
		} else {
			s.files.MarkMismatch("object file bytes appended but database commit failed")
		}
		return model.ObjectFile{}, model.StorageUnavailable("object storage mismatch detected", map[string]any{"object_id": objectID, "file_id": fileID})
	}
	return s.GetObjectFile(ctx, objectID, fileID)
}

func (s *Store) OpenObjectFile(ctx context.Context, objectID, fileID string) (model.ObjectFile, io.ReadCloser, error) {
	file, err := s.GetObjectFile(ctx, objectID, fileID)
	if err != nil {
		return model.ObjectFile{}, nil, err
	}
	rc, stat, err := s.files.Open(file.Path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			s.files.MarkMismatch("object metadata exists but bytes are missing")
			return model.ObjectFile{}, nil, model.StorageUnavailable("object storage mismatch detected", map[string]any{"object_id": objectID, "file_id": fileID})
		}
		return model.ObjectFile{}, nil, err
	}
	if stat.Size() != file.SizeBytes {
		s.files.MarkMismatch("object metadata size does not match bytes")
		_ = rc.Close()
		return model.ObjectFile{}, nil, model.StorageUnavailable("object storage mismatch detected", map[string]any{"object_id": objectID, "file_id": fileID})
	}
	return file, rc, nil
}

func (s *Store) DeleteObjectFile(ctx context.Context, objectID, fileID string) error {
	file, err := s.GetObjectFile(ctx, objectID, fileID)
	if err != nil {
		return err
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	result, err := tx.Exec(ctx, `DELETE FROM object_files WHERE file_id=$1 AND object_id=$2`, fileID, objectID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return model.NotFound("object_file", fileID)
	}
	if _, err := tx.Exec(ctx, `UPDATE objects SET updated_at=$2 WHERE object_id=$1`, objectID, time.Now().UTC()); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	if err := s.files.Delete(file.Path); err != nil {
		s.files.MarkMismatch("failed to delete object file from storage after database commit")
		return err
	}
	return nil
}

func (s *Store) ListObjectFilesForObject(ctx context.Context, objectID string) ([]model.ObjectFile, error) {
	rows, err := s.pool.Query(ctx, `SELECT file_id, object_id, path, content_type, size_bytes, COALESCE(usage_hint,''), created_at, updated_at FROM object_files WHERE object_id=$1 ORDER BY updated_at DESC, file_id ASC`, objectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	files := []model.ObjectFile{}
	for rows.Next() {
		file, err := scanObjectFile(rows)
		if err != nil {
			return nil, err
		}
		files = append(files, file)
	}
	return files, rows.Err()
}

func (s *Store) GetFullQueryState(ctx context.Context) (store.QueryState, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly, IsoLevel: pgx.RepeatableRead})
	if err != nil {
		return store.QueryState{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	state := store.QueryState{}
	if rows, err := tx.Query(ctx, `SELECT entity_id, type, COALESCE(subtype,''), COALESCE(alias,''), json, created_at, updated_at FROM entities ORDER BY updated_at DESC, entity_id ASC`); err != nil {
		return state, err
	} else {
		state.Entities, _, err = collectEntities(rows)
		if err != nil {
			return state, err
		}
	}
	if rows, err := tx.Query(ctx, `SELECT observation_id, source_asset_id, json, created_at, updated_at FROM observations ORDER BY updated_at DESC, observation_id ASC`); err != nil {
		return state, err
	} else {
		state.Observations, _, err = collectObservations(rows, 0)
		if err != nil {
			return state, err
		}
	}
	if rows, err := tx.Query(ctx, `SELECT task_id, status, asset_id, command_catalog_object_id, json, created_at, updated_at FROM tasks ORDER BY updated_at DESC, task_id ASC`); err != nil {
		return state, err
	} else {
		state.Tasks, _, err = collectTasks(rows, 0)
		if err != nil {
			return state, err
		}
	}
	if rows, err := tx.Query(ctx, `SELECT object_id, type, owner_type, owner_id, json, created_at, updated_at FROM objects ORDER BY updated_at DESC, object_id ASC`); err != nil {
		return state, err
	} else {
		state.Objects, _, err = collectObjects(rows, 0)
		if err != nil {
			return state, err
		}
	}
	if rows, err := tx.Query(ctx, `SELECT file_id, object_id, path, content_type, size_bytes, COALESCE(usage_hint,''), created_at, updated_at FROM object_files ORDER BY updated_at DESC, object_id ASC, file_id ASC`); err != nil {
		return state, err
	} else {
		state.ObjectFiles, err = collectObjectFiles(rows)
		if err != nil {
			return state, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return state, err
	}
	return state, nil
}

func (s *Store) execUpsert(ctx context.Context, sql string, resourceType, resourceID string, args ...any) error {
	_, err := s.pool.Exec(ctx, sql, args...)
	return mapPGError(err, resourceType, resourceID)
}

func (s *Store) count(ctx context.Context, query string, args ...any) (int, error) {
	var total int
	if err := s.pool.QueryRow(ctx, query, args...).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

func ensureObjectExists(ctx context.Context, tx pgx.Tx, objectID string) error {
	var exists int
	if err := tx.QueryRow(ctx, `SELECT 1 FROM objects WHERE object_id=$1`, objectID).Scan(&exists); err != nil {
		return mapNotFound(err, "object", objectID)
	}
	return nil
}

func filePathsForOwner(ctx context.Context, tx pgx.Tx, ownerType, ownerID string) ([]string, error) {
	rows, err := tx.Query(ctx, `SELECT of.path FROM object_files of JOIN objects o ON o.object_id = of.object_id WHERE o.owner_type=$1 AND o.owner_id=$2`, ownerType, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var paths []string
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return nil, err
		}
		paths = append(paths, path)
	}
	return paths, rows.Err()
}

func filePathsForObject(ctx context.Context, tx pgx.Tx, objectID string) ([]string, error) {
	rows, err := tx.Query(ctx, `SELECT path FROM object_files WHERE object_id=$1`, objectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var paths []string
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return nil, err
		}
		paths = append(paths, path)
	}
	return paths, rows.Err()
}

func scanEntity(scanner interface{ Scan(dest ...any) error }) (model.Entity, error) {
	var raw []byte
	entity := model.Entity{}
	if err := scanner.Scan(&entity.EntityID, &entity.Type, &entity.Subtype, &entity.Alias, &raw, &entity.CreatedAt, &entity.UpdatedAt); err != nil {
		return model.Entity{}, err
	}
	entity.JSON = unmarshalJSON(raw)
	return entity, nil
}

func scanObservation(scanner interface{ Scan(dest ...any) error }) (model.Observation, error) {
	var raw []byte
	item := model.Observation{}
	if err := scanner.Scan(&item.ObservationID, &item.SourceAssetID, &raw, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return model.Observation{}, err
	}
	item.JSON = unmarshalJSON(raw)
	return item, nil
}

func scanTask(scanner interface{ Scan(dest ...any) error }) (model.Task, error) {
	var raw []byte
	item := model.Task{}
	if err := scanner.Scan(&item.TaskID, &item.Status, &item.AssetID, &item.CommandCatalogObjectID, &raw, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return model.Task{}, err
	}
	item.JSON = unmarshalJSON(raw)
	return item, nil
}

func scanObject(scanner interface{ Scan(dest ...any) error }) (model.Object, error) {
	var raw []byte
	item := model.Object{}
	if err := scanner.Scan(&item.ObjectID, &item.Type, &item.OwnerType, &item.OwnerID, &raw, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return model.Object{}, err
	}
	item.JSON = unmarshalJSON(raw)
	return item, nil
}

func scanObjectFile(scanner interface{ Scan(dest ...any) error }) (model.ObjectFile, error) {
	item := model.ObjectFile{}
	if err := scanner.Scan(&item.FileID, &item.ObjectID, &item.Path, &item.ContentType, &item.SizeBytes, &item.UsageHint, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return model.ObjectFile{}, err
	}
	return item, nil
}

func collectEntities(rows pgx.Rows) ([]model.Entity, int, error) {
	defer rows.Close()
	items := []model.Entity{}
	for rows.Next() {
		item, err := scanEntity(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, 0, rows.Err()
}

func collectObservations(rows pgx.Rows, total int) ([]model.Observation, int, error) {
	defer rows.Close()
	items := []model.Observation{}
	for rows.Next() {
		item, err := scanObservation(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func collectTasks(rows pgx.Rows, total int) ([]model.Task, int, error) {
	defer rows.Close()
	items := []model.Task{}
	for rows.Next() {
		item, err := scanTask(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func collectObjects(rows pgx.Rows, total int) ([]model.Object, int, error) {
	defer rows.Close()
	items := []model.Object{}
	for rows.Next() {
		item, err := scanObject(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func collectObjectFiles(rows pgx.Rows) ([]model.ObjectFile, error) {
	defer rows.Close()
	items := []model.ObjectFile{}
	for rows.Next() {
		item, err := scanObjectFile(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func joinWhere(clauses []string) string {
	if len(clauses) == 0 {
		return ""
	}
	return " WHERE " + strings.Join(clauses, " AND ")
}

func nullString(value string) sql.NullString {
	if value == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: value, Valid: true}
}

func mustJSON(input model.JSONMap) []byte {
	raw, _ := json.Marshal(model.NormalizeJSONMap(input))
	return raw
}

func unmarshalJSON(raw []byte) model.JSONMap {
	if len(raw) == 0 {
		return model.JSONMap{}
	}
	var out model.JSONMap
	_ = json.Unmarshal(raw, &out)
	return model.NormalizeJSONMap(out)
}

func mapNotFound(err error, resourceType, resourceID string) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return model.NotFound(resourceType, resourceID)
	}
	return err
}

func mapPGError(err error, resourceType, resourceID string) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		if resourceType == "" {
			resourceType = "resource"
		}
		return model.Conflict(resourceType, resourceID, "already_exists", nil)
	}
	return err
}

// defaultUploadLimitBytes matches servicetest.memUploadLimit when no per-request cap is set.
const defaultUploadLimitBytes = 16 * 1024 * 1024

const minUploadLimitBytes = 1024

func chooseLimit(requested, fallback int64) int64 {
	if fallback < 1 {
		fallback = defaultUploadLimitBytes
	}
	if requested > 0 && requested < fallback {
		lim := requested
		if lim < minUploadLimitBytes {
			lim = minUploadLimitBytes
		}
		if lim > fallback {
			return fallback
		}
		return lim
	}
	return fallback
}

func itoa(v int) string { return fmt.Sprintf("%d", v) }
