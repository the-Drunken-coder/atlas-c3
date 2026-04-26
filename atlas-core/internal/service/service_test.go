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

func setupServices(t *testing.T) (*service.Services, *servicetest.MemoryStore, string) {
	t.Helper()
	cat, err := servicetest.NewDefaultCommandCatalog()
	if err != nil {
		t.Fatal(err)
	}
	stores := servicetest.NewMemoryStore()
	commands := &catalog.Active{}
	commands.Set(cat)
	sightings := &sightingcatalog.Active{}
	sightings.Set(sightingcatalog.Catalog{ByKind: map[string]sightingcatalog.Kind{"analysis": {Kind: "analysis", DataSchema: map[string]any{"type": "object", "additionalProperties": true}}}})
	return service.New(stores, commands, sightings, events.NewHub()), stores, cat.ObjectID
}

func TestCreateTaskValidatesSupportedCommands(t *testing.T) {
	svc, stores, _ := setupServices(t)
	stores.Entities["asset-1"] = model.Entity{EntityID: "asset-1", Type: "asset", JSON: model.JSONMap{"components": map[string]any{"supported_commands": map[string]any{"observed_at": time.Now().UTC().Format(time.RFC3339), "commands": []any{"hold_position"}}}}}
	_, err := svc.CreateTask(context.Background(), service.TaskCreateInput{TaskID: "task-1", AssetID: "asset-1", JSON: model.JSONMap{"components": map[string]any{"command": map[string]any{"type": "move_to_location"}, "parameters": map[string]any{"latitude": 1.0}}}})
	if err == nil {
		t.Fatal("expected command validation error")
	}
}

func TestCreateObservationValidatesSightingHistoryObject(t *testing.T) {
	svc, stores, _ := setupServices(t)
	stores.Entities["asset-1"] = model.Entity{EntityID: "asset-1", Type: "asset", JSON: model.JSONMap{"components": map[string]any{"supported_commands": map[string]any{"observed_at": time.Now().UTC().Format(time.RFC3339), "commands": []any{"move_to_location"}}}}}
	stores.Objects["obj-1"] = model.Object{ObjectID: "obj-1", Type: "observation_media", OwnerType: "observation", OwnerID: "obs-1"}
	_, err := svc.CreateObservation(context.Background(), service.ObservationCreateInput{ObservationID: "obs-1", SourceAssetID: "asset-1", JSON: model.JSONMap{"state": "active", "sightings_object_id": "obj-1"}})
	if err == nil {
		t.Fatal("expected sightings object validation failure")
	}
}

func TestDeleteEntityRejectsDependents(t *testing.T) {
	svc, stores, _ := setupServices(t)
	stores.Entities["asset-1"] = model.Entity{EntityID: "asset-1", Type: "asset", JSON: model.JSONMap{"components": map[string]any{"supported_commands": map[string]any{"observed_at": time.Now().UTC().Format(time.RFC3339), "commands": []any{"move_to_location"}}}}}
	stores.Tasks["task-1"] = model.Task{TaskID: "task-1", AssetID: "asset-1"}
	if err := svc.DeleteEntity(context.Background(), "asset-1"); err == nil {
		t.Fatal("expected dependent conflict")
	}
}

func TestTransitionTaskStatusIdempotent(t *testing.T) {
	svc, stores, catalogID := setupServices(t)
	stores.Tasks["task-1"] = model.Task{TaskID: "task-1", Status: "pending", AssetID: "asset-1", CommandCatalogObjectID: catalogID, JSON: model.JSONMap{"components": map[string]any{"command": map[string]any{"type": "move_to_location"}, "parameters": map[string]any{"latitude": 1.0}}}}
	task, err := svc.TransitionTaskStatus(context.Background(), "task-1", service.TaskStatusInput{Status: "pending"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if task.Status != "pending" {
		t.Fatalf("unexpected status: %s", task.Status)
	}
}

func TestPatchTaskAfterPendingRejectsCommandChange(t *testing.T) {
	svc, stores, catID := setupServices(t)
	stores.Entities["asset-1"] = model.Entity{EntityID: "asset-1", Type: "asset", JSON: model.JSONMap{"components": map[string]any{"supported_commands": map[string]any{"observed_at": time.Now().UTC().Format(time.RFC3339), "commands": []any{"move_to_location", "other"}}}}}
	stores.Tasks["task-1"] = model.Task{TaskID: "task-1", Status: "acknowledged", AssetID: "asset-1", CommandCatalogObjectID: catID, JSON: model.JSONMap{"components": map[string]any{
		"command":   map[string]any{"type": "move_to_location"},
		"parameters": map[string]any{"latitude": 1.0},
	}}}
	_, err := svc.PatchTask(context.Background(), "task-1", service.TaskPatchInput{JSON: model.JSONMap{
		"components": map[string]any{
			"command":   map[string]any{"type": "other"},
			"parameters": map[string]any{"latitude": 1.0},
		},
	}})
	if err == nil {
		t.Fatal("expected immutability error for command on non-pending task")
	}
}

func TestTransitionRejectsCommandEditAllowsProgress(t *testing.T) {
	svc, stores, catID := setupServices(t)
	obs := time.Now().UTC().Format(time.RFC3339)
	stores.Entities["asset-1"] = model.Entity{EntityID: "asset-1", Type: "asset", JSON: model.JSONMap{"components": map[string]any{"supported_commands": map[string]any{"observed_at": obs, "commands": []any{"move_to_location", "other"}}}}}
	stores.Tasks["task-1"] = model.Task{TaskID: "task-1", Status: "pending", AssetID: "asset-1", CommandCatalogObjectID: catID, JSON: model.JSONMap{"components": map[string]any{
		"command":   map[string]any{"type": "move_to_location"},
		"parameters": map[string]any{"latitude": 1.0},
	}}}
	_, err := svc.TransitionTaskStatus(context.Background(), "task-1", service.TaskStatusInput{
		Status: "acknowledged",
		JSON: model.JSONMap{
			"components": map[string]any{
				"command":  map[string]any{"type": "other"},
				"progress": 0.5,
			},
		},
	})
	if err == nil {
		t.Fatal("expected immutability error when changing command on status transition")
	}
	task, err := svc.TransitionTaskStatus(context.Background(), "task-1", service.TaskStatusInput{
		Status: "acknowledged",
		JSON: model.JSONMap{
			"components": map[string]any{"progress": 0.5},
		},
	})
	if err != nil {
		t.Fatalf("transition with progress: %v", err)
	}
	prog, _ := task.JSON["components"].(map[string]any)["progress"]
	if prog != 0.5 {
		t.Fatalf("expected progress, got %v", prog)
	}
}
