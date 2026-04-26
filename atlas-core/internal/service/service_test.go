package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/catalog"
	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/events"
	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/model"
	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/service"
	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/service/servicetest"
	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/sightingcatalog"
)

func setupServices(t *testing.T) (*service.Services, *servicetest.MemoryStore) {
	t.Helper()
	stores := servicetest.NewMemoryStore()
	commands := &catalog.Active{}
	commands.Set(catalog.Catalog{ObjectID: "command-catalog-test", ByType: map[string]catalog.Command{"move_to_location": {Type: "move_to_location", ParametersSchema: map[string]any{"type": "object", "required": []any{"latitude"}, "additionalProperties": false, "properties": map[string]any{"latitude": map[string]any{"type": "number"}}}}}})
	sightings := &sightingcatalog.Active{}
	sightings.Set(sightingcatalog.Catalog{ByKind: map[string]sightingcatalog.Kind{"analysis": {Kind: "analysis", DataSchema: map[string]any{"type": "object", "additionalProperties": true}}}})
	return service.New(stores, commands, sightings, events.NewHub()), stores
}

func TestCreateTaskValidatesSupportedCommands(t *testing.T) {
	svc, stores := setupServices(t)
	stores.Entities["asset-1"] = model.Entity{EntityID: "asset-1", Type: "asset", JSON: model.JSONMap{"components": map[string]any{"supported_commands": map[string]any{"observed_at": time.Now().UTC().Format(time.RFC3339), "commands": []any{"hold_position"}}}}}
	_, err := svc.CreateTask(context.Background(), service.TaskCreateInput{TaskID: "task-1", AssetID: "asset-1", JSON: model.JSONMap{"components": map[string]any{"command": map[string]any{"type": "move_to_location"}, "parameters": map[string]any{"latitude": 1.0}}}})
	if err == nil {
		t.Fatal("expected command validation error")
	}
}

func TestCreateObservationValidatesSightingHistoryObject(t *testing.T) {
	svc, stores := setupServices(t)
	stores.Entities["asset-1"] = model.Entity{EntityID: "asset-1", Type: "asset", JSON: model.JSONMap{"components": map[string]any{"supported_commands": map[string]any{"observed_at": time.Now().UTC().Format(time.RFC3339), "commands": []any{"move_to_location"}}}}}
	stores.Objects["obj-1"] = model.Object{ObjectID: "obj-1", Type: "observation_media", OwnerType: "observation", OwnerID: "obs-1"}
	_, err := svc.CreateObservation(context.Background(), service.ObservationCreateInput{ObservationID: "obs-1", SourceAssetID: "asset-1", JSON: model.JSONMap{"state": "active", "sightings_object_id": "obj-1"}})
	if err == nil {
		t.Fatal("expected sightings object validation failure")
	}
}

func TestDeleteEntityRejectsDependents(t *testing.T) {
	svc, stores := setupServices(t)
	stores.Entities["asset-1"] = model.Entity{EntityID: "asset-1", Type: "asset", JSON: model.JSONMap{"components": map[string]any{"supported_commands": map[string]any{"observed_at": time.Now().UTC().Format(time.RFC3339), "commands": []any{"move_to_location"}}}}}
	stores.Tasks["task-1"] = model.Task{TaskID: "task-1", AssetID: "asset-1"}
	if err := svc.DeleteEntity(context.Background(), "asset-1"); err == nil {
		t.Fatal("expected dependent conflict")
	}
}

func TestTransitionTaskStatusIdempotent(t *testing.T) {
	svc, stores := setupServices(t)
	stores.Tasks["task-1"] = model.Task{TaskID: "task-1", Status: "pending", AssetID: "asset-1", CommandCatalogObjectID: "command-catalog-test", JSON: model.JSONMap{"components": map[string]any{"command": map[string]any{"type": "move_to_location"}, "parameters": map[string]any{"latitude": 1.0}}}}
	task, err := svc.TransitionTaskStatus(context.Background(), "task-1", service.TaskStatusInput{Status: "pending"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if task.Status != "pending" {
		t.Fatalf("unexpected status: %s", task.Status)
	}
}
