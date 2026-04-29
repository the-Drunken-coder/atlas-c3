package store

import (
	"context"
	"io"

	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/model"
)

type EntityListFilter struct {
	Type string
}

type ObservationListFilter struct {
	SourceAssetID string
	UpdatedAfter  string
}

type TaskListFilter struct {
	AssetID string
	Status  string
}

type ObjectListFilter struct {
	OwnerType string
	OwnerID   string
	Type      string
}

type QueryState struct {
	Entities     []model.Entity
	Observations []model.Observation
	Tasks        []model.Task
	Objects      []model.Object
	ObjectFiles  []model.ObjectFile
}

type Stores interface {
	EntityStore
	ObservationStore
	TaskStore
	ObjectStore
	QueryStore
}

type EntityStore interface {
	CreateEntity(context.Context, model.Entity) (model.Entity, error)
	GetEntity(context.Context, string) (model.Entity, error)
	ListEntities(context.Context, EntityListFilter, model.Pagination) ([]model.Entity, int, error)
	UpdateEntity(context.Context, model.Entity) (model.Entity, error)
	DeleteEntity(context.Context, string) error
	CountEntityDependents(context.Context, string) (map[string]int, error)
}

type ObservationStore interface {
	CreateObservation(context.Context, model.Observation) (model.Observation, error)
	GetObservation(context.Context, string) (model.Observation, error)
	ListObservations(context.Context, ObservationListFilter, model.Pagination) ([]model.Observation, int, error)
	UpdateObservation(context.Context, model.Observation) (model.Observation, error)
	DeleteObservation(context.Context, string) error
}

type TaskStore interface {
	CreateTask(context.Context, model.Task) (model.Task, error)
	GetTask(context.Context, string) (model.Task, error)
	ListTasks(context.Context, TaskListFilter, model.Pagination) ([]model.Task, int, error)
	UpdateTask(context.Context, model.Task) (model.Task, error)
	DeleteTask(context.Context, string) error
	ListTasksForAsset(context.Context, string, model.Pagination) ([]model.Task, int, error)
	CountTaskOwnedObjects(context.Context, string) (int, error)
}

type ObjectUploadInput struct {
	File     model.ObjectFile
	Reader   io.Reader
	MaxBytes int64
	// PreStagedPath, when non-empty, skips Stage: bytes are taken from this
	// path (Promote renames it into object storage on success). Implementations
	// attempt to remove the path after CreateObjectFile returns, on both success
	// and error paths. The only exception is a post-commit byte-promotion
	// failure (returned as StorageUnavailable with a "staged_path" key in
	// details), where the staged file is intentionally retained so an operator
	// can recover the bytes. Callers must not delete or reuse the path after
	// CreateObjectFile returns and must not touch it concurrently during the
	// call.
	PreStagedPath        string
	PreStagedSizeBytes   int64
	PreStagedContentType string
}

type ObjectStore interface {
	CreateObject(context.Context, model.Object) (model.Object, error)
	GetObject(context.Context, string) (model.Object, error)
	ListObjects(context.Context, ObjectListFilter, model.Pagination) ([]model.Object, int, error)
	UpdateObject(context.Context, model.Object) (model.Object, error)
	DeleteObject(context.Context, string) error
	CreateObjectFile(context.Context, ObjectUploadInput) (model.ObjectFile, error)
	GetObjectFile(context.Context, string, string) (model.ObjectFile, error)
	AppendObjectFile(context.Context, string, string, io.Reader, int64) (model.ObjectFile, error)
	OpenObjectFile(context.Context, string, string) (model.ObjectFile, io.ReadCloser, error)
	DeleteObjectFile(context.Context, string, string) error
	ListObjectFilesForObject(context.Context, string) ([]model.ObjectFile, error)
	StorageStatus(context.Context) model.DependencyStatus
}

type QueryStore interface {
	GetFullQueryState(context.Context) (QueryState, error)
}
