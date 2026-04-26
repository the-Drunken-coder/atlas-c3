package httpapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/catalog"
	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/events"
	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/httpapi"
	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/model"
	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/service"
	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/service/servicetest"
	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/sightingcatalog"
)

func testRouter() http.Handler {
	stores := servicetest.NewMemoryStore()
	stores.Entities["asset-1"] = model.Entity{EntityID: "asset-1", Type: "asset", JSON: model.JSONMap{"components": map[string]any{"supported_commands": map[string]any{"observed_at": time.Now().UTC().Format(time.RFC3339), "commands": []any{"move_to_location"}}}}}
	commands := &catalog.Active{}
	commands.Set(catalog.Catalog{ObjectID: "command-catalog-test", ByType: map[string]catalog.Command{"move_to_location": {Type: "move_to_location", ParametersSchema: map[string]any{"type": "object", "additionalProperties": true}}}})
	sightings := &sightingcatalog.Active{}
	sightings.Set(sightingcatalog.Catalog{ByKind: map[string]sightingcatalog.Kind{"analysis": {Kind: "analysis", DataSchema: map[string]any{"type": "object", "additionalProperties": true}}}})
	svc := service.New(stores, commands, sightings, events.NewHub())
	return httpapi.NewRouter(httpapi.Dependencies{
		AllowedOrigins: []string{"http://localhost:5173"},
		StartedAt:      time.Now().UTC(),
		Services:       svc,
		Events:         events.NewHub(),
		CommandCatalog: commands,
		ObjectStore:    stores,
		Readiness: func(_ context.Context) (model.ReadinessResponse, int) {
			return model.ReadinessResponse{Status: "ready", Timestamp: time.Now().UTC(), Dependencies: map[string]model.DependencyStatus{"postgres": {Status: "ready"}, "object_storage": {Status: "ready"}, "command_catalog": {Status: "ready"}}}, http.StatusOK
		},
		Descriptor: func() (model.ServiceDescriptor, error) {
			return model.ServiceDescriptor{Service: model.ServiceName, Status: "ok", Version: "test", StartedAt: time.Now().UTC(), ActiveCommandCatalogObjectID: "command-catalog-test", Links: map[string]string{"health": "/health"}}, nil
		},
	})
}

func TestHealthEndpoint(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	testRouter().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", rr.Code)
	}
}

func TestCreateEntityRejectsUnknownFields(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/entities", strings.NewReader(`{"entity_id":"asset-2","type":"asset","json":{"components":{"supported_commands":{"observed_at":"2026-01-01T00:00:00Z","commands":[]}}},"extra":true}`))
	rr := httptest.NewRecorder()
	testRouter().ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestCreateTaskPublishesCreatedResource(t *testing.T) {
	router := testRouter()
	req := httptest.NewRequest(http.MethodPost, "/tasks", strings.NewReader(`{"task_id":"task-1","asset_id":"asset-1","json":{"components":{"command":{"type":"move_to_location"},"parameters":{"latitude":1}}}}`))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("unexpected status: %d body=%s", rr.Code, rr.Body.String())
	}
	var task model.Task
	if err := json.Unmarshal(rr.Body.Bytes(), &task); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if task.Status != "pending" {
		t.Fatalf("unexpected status: %s", task.Status)
	}
}
