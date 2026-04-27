package service_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/catalog"
	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/events"
	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/model"
	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/service"
	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/service/servicetest"
	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/sightingcatalog"
)

// oversizedCatalogByteCount is one byte over the 8 MiB pinned catalog limit.
const oversizedCatalogByteCount = (8 << 20) + 1

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

func setAssetSupportedCommands(stores *servicetest.MemoryStore, commands ...string) {
	stores.Entities["asset-1"] = model.Entity{EntityID: "asset-1", Type: "asset", JSON: supportedCommandsJSON(commands...)}
}

func anySlice(values []string) []any {
	out := make([]any, 0, len(values))
	for _, value := range values {
		out = append(out, value)
	}
	return out
}

func supportedCommandsJSON(commands ...string) model.JSONMap {
	return model.JSONMap{"components": map[string]any{"supported_commands": map[string]any{"observed_at": time.Now().UTC().Format(time.RFC3339), "commands": anySlice(commands)}}}
}

func assertConcreteImmutableFields(t *testing.T, err error) {
	t.Helper()
	ce, ok := model.IsCoreError(err)
	if !ok || ce.ErrorCode != "immutable_field" {
		t.Fatalf("expected immutable_field, got %v", err)
	}
	fields, ok := ce.Details["fields"].([]model.FieldError)
	if !ok || len(fields) == 0 {
		t.Fatalf("unexpected immutable fields detail: %#v", ce.Details["fields"])
	}
	allowed := map[string]bool{"json.components.command": true, "json.components.parameters": true}
	for _, field := range fields {
		if field.Field == "json.components.command|parameters" {
			t.Fatalf("unexpected pseudo field path: %#v", ce.Details["fields"])
		}
		if !allowed[field.Field] {
			t.Fatalf("unexpected immutable field path: %#v", ce.Details["fields"])
		}
	}
}

func TestCreateTaskValidatesSupportedCommands(t *testing.T) {
	svc, stores, _ := setupServices(t)
	setAssetSupportedCommands(stores, "hold_position")
	_, err := svc.CreateTask(context.Background(), service.TaskCreateInput{TaskID: "task-1", AssetID: "asset-1", JSON: model.JSONMap{"components": map[string]any{"command": map[string]any{"type": "move_to_location"}, "parameters": map[string]any{"latitude": 1.0}}}})
	if err == nil {
		t.Fatal("expected command validation error")
	}
}

func TestCreateTaskDistinguishesMissingAndWrongTypeCommandSections(t *testing.T) {
	t.Parallel()
	svc, stores, _ := setupServices(t)
	setAssetSupportedCommands(stores, "move_to_location")
	tests := []struct {
		name      string
		json      model.JSONMap
		wantField string
		wantCode  string
	}{
		{
			name:      "missing components",
			json:      model.JSONMap{},
			wantField: "json.components",
			wantCode:  "required",
		},
		{
			name:      "components wrong type",
			json:      model.JSONMap{"components": "bad"},
			wantField: "json.components",
			wantCode:  "invalid_type",
		},
		{
			name:      "missing command",
			json:      model.JSONMap{"components": map[string]any{}},
			wantField: "json.components.command",
			wantCode:  "required",
		},
		{
			name:      "command wrong type",
			json:      model.JSONMap{"components": map[string]any{"command": "bad"}},
			wantField: "json.components.command",
			wantCode:  "invalid_type",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := svc.CreateTask(context.Background(), service.TaskCreateInput{TaskID: "task-1", AssetID: "asset-1", JSON: test.json})
			if err == nil {
				t.Fatal("expected validation error")
			}
			ce, ok := model.IsCoreError(err)
			if !ok || ce.ErrorCode != "validation_failed" {
				t.Fatalf("expected validation_failed, got %v", err)
			}
			fields, ok := ce.Details["fields"].([]model.FieldError)
			if !ok || len(fields) == 0 {
				t.Fatalf("expected field errors, got %#v", ce.Details["fields"])
			}
			if fields[0].Field != test.wantField || fields[0].Code != test.wantCode {
				t.Fatalf("expected (%s, %s), got (%s, %s)", test.wantField, test.wantCode, fields[0].Field, fields[0].Code)
			}
		})
	}
}

func TestCreateObservationValidatesSightingHistoryObject(t *testing.T) {
	svc, stores, _ := setupServices(t)
	setAssetSupportedCommands(stores, "move_to_location")
	stores.Objects["obj-1"] = model.Object{ObjectID: "obj-1", Type: "observation_media", OwnerType: "observation", OwnerID: "obs-1"}
	_, err := svc.CreateObservation(context.Background(), service.ObservationCreateInput{ObservationID: "obs-1", SourceAssetID: "asset-1", JSON: model.JSONMap{"state": "active", "sightings_object_id": "obj-1"}})
	if err == nil {
		t.Fatal("expected sightings object validation failure")
	}
}

func TestDeleteEntityRejectsDependents(t *testing.T) {
	svc, stores, _ := setupServices(t)
	setAssetSupportedCommands(stores, "move_to_location")
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
	setAssetSupportedCommands(stores, "move_to_location", "other")
	stores.Tasks["task-1"] = model.Task{TaskID: "task-1", Status: "acknowledged", AssetID: "asset-1", CommandCatalogObjectID: catID, JSON: model.JSONMap{"components": map[string]any{
		"command":    map[string]any{"type": "move_to_location"},
		"parameters": map[string]any{"latitude": 1.0},
	}}}
	_, err := svc.PatchTask(context.Background(), "task-1", service.TaskPatchInput{JSON: model.JSONMap{
		"components": map[string]any{
			"command":    map[string]any{"type": "other"},
			"parameters": map[string]any{"latitude": 1.0},
		},
	}})
	if err == nil {
		t.Fatal("expected immutability error for command on non-pending task")
	}
	assertConcreteImmutableFields(t, err)
}

func TestTransitionRejectsCommandEditAllowsProgress(t *testing.T) {
	svc, stores, catID := setupServices(t)
	obs := time.Now().UTC().Format(time.RFC3339)
	stores.Entities["asset-1"] = model.Entity{EntityID: "asset-1", Type: "asset", JSON: model.JSONMap{"components": map[string]any{"supported_commands": map[string]any{"observed_at": obs, "commands": []any{"move_to_location", "other"}}}}}
	stores.Tasks["task-1"] = model.Task{TaskID: "task-1", Status: "pending", AssetID: "asset-1", CommandCatalogObjectID: catID, JSON: model.JSONMap{"components": map[string]any{
		"command":    map[string]any{"type": "move_to_location"},
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
	assertConcreteImmutableFields(t, err)
	task, err := svc.TransitionTaskStatus(context.Background(), "task-1", service.TaskStatusInput{
		Status: "acknowledged",
		JSON: model.JSONMap{
			"components": map[string]any{"progress": 0.5},
		},
	})
	if err != nil {
		t.Fatalf("transition with progress: %v", err)
	}
	components, ok := task.JSON["components"].(map[string]any)
	if !ok {
		t.Fatalf("expected task components map, got %#v", task.JSON["components"])
	}
	prog := components["progress"]
	if prog != 0.5 {
		t.Fatalf("expected progress, got %v", prog)
	}
}

func TestTransitionTaskStatusIgnoresSupportedCommandDrift(t *testing.T) {
	svc, stores, _ := setupServices(t)
	setAssetSupportedCommands(stores, "move_to_location")
	created, err := svc.CreateTask(context.Background(), service.TaskCreateInput{
		TaskID:  "task-1",
		AssetID: "asset-1",
		JSON: model.JSONMap{"components": map[string]any{
			"command":    map[string]any{"type": "move_to_location"},
			"parameters": map[string]any{"latitude": 1.0},
		}},
	})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	setAssetSupportedCommands(stores, "hold_position")
	acknowledged, err := svc.TransitionTaskStatus(context.Background(), created.TaskID, service.TaskStatusInput{
		Status: "acknowledged",
		JSON:   model.JSONMap{"components": map[string]any{"progress": 0.5}},
	})
	if err != nil {
		t.Fatalf("acknowledge task: %v", err)
	}
	completed, err := svc.TransitionTaskStatus(context.Background(), created.TaskID, service.TaskStatusInput{
		Status: "completed",
		JSON:   model.JSONMap{"components": map[string]any{"result": map[string]any{"ok": true}}},
	})
	if err != nil {
		t.Fatalf("complete task: %v", err)
	}
	if acknowledged.Status != "acknowledged" || completed.Status != "completed" {
		t.Fatalf("unexpected statuses: %s -> %s", acknowledged.Status, completed.Status)
	}
}

func TestTransitionTaskStatusMapsOversizedPinnedCatalogToCatalogUnavailable(t *testing.T) {
	svc, stores, _ := setupServices(t)
	setAssetSupportedCommands(stores, "move_to_location")
	stores.Objects["catalog-big"] = model.Object{ObjectID: "catalog-big", Type: "command_catalog", OwnerType: "system", OwnerID: "active_command_catalog"}
	raw := []byte(strings.Repeat("a", oversizedCatalogByteCount))
	stores.ObjectFiles["catalog-json"] = model.ObjectFile{FileID: "catalog-json", ObjectID: "catalog-big", ContentType: "application/json", SizeBytes: int64(len(raw))}
	stores.FileBytes["catalog-json"] = raw
	stores.Tasks["task-1"] = model.Task{TaskID: "task-1", Status: "pending", AssetID: "asset-1", CommandCatalogObjectID: "catalog-big", JSON: model.JSONMap{"components": map[string]any{
		"command":    map[string]any{"type": "move_to_location"},
		"parameters": map[string]any{"latitude": 1.0},
	}}}
	_, err := svc.TransitionTaskStatus(context.Background(), "task-1", service.TaskStatusInput{Status: "acknowledged"})
	if err == nil {
		t.Fatal("expected catalog unavailable error")
	}
	ce, ok := model.IsCoreError(err)
	if !ok || ce.ErrorCode != "catalog_unavailable" {
		t.Fatalf("expected catalog_unavailable, got %v", err)
	}
}
