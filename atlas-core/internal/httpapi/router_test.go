package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
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
	cat, err := servicetest.NewDefaultCommandCatalog()
	if err != nil {
		panic(err)
	}
	stores := servicetest.NewMemoryStore()
	stores.Entities["asset-1"] = model.Entity{EntityID: "asset-1", Type: "asset", JSON: model.JSONMap{"components": map[string]any{"supported_commands": map[string]any{"observed_at": time.Now().UTC().Format(time.RFC3339), "commands": []any{"move_to_location"}}}}}
	commands := &catalog.Active{}
	commands.Set(cat)
	sightings := &sightingcatalog.Active{}
	sightings.Set(sightingcatalog.Catalog{ByKind: map[string]sightingcatalog.Kind{"analysis": {Kind: "analysis", DataSchema: map[string]any{"type": "object", "additionalProperties": true}}}})
	hub := events.NewHub()
	svc := service.New(stores, commands, sightings, hub)
	return httpapi.NewRouter(httpapi.Dependencies{
		AllowedOrigins: []string{"http://localhost:5173"},
		StartedAt:      time.Now().UTC(),
		MaxUploadBytes: 20 * 1024 * 1024,
		Services:       svc,
		Events:         hub,
		CommandCatalog: commands,
		ObjectStore:    stores,
		Readiness: func(_ context.Context) (model.ReadinessResponse, int) {
			return model.ReadinessResponse{Status: "ready", Timestamp: time.Now().UTC(), Dependencies: map[string]model.DependencyStatus{"postgres": {Status: "ready"}, "object_storage": {Status: "ready"}, "command_catalog": {Status: "ready"}}}, http.StatusOK
		},
		Descriptor: func() (model.ServiceDescriptor, error) {
			return model.ServiceDescriptor{Service: model.ServiceName, Status: "ok", Version: "test", StartedAt: time.Now().UTC(), ActiveCommandCatalogObjectID: cat.ObjectID, Links: map[string]string{"health": "/health"}}, nil
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

func testUploadRouter() http.Handler {
	cat, err := servicetest.NewDefaultCommandCatalog()
	if err != nil {
		panic(err)
	}
	stores := servicetest.NewMemoryStore()
	stores.Entities["asset-1"] = model.Entity{EntityID: "asset-1", Type: "asset", JSON: model.JSONMap{"components": map[string]any{"supported_commands": map[string]any{"observed_at": time.Now().UTC().Format(time.RFC3339), "commands": []any{"move_to_location"}}}}}
	commands := &catalog.Active{}
	commands.Set(cat)
	sightings := &sightingcatalog.Active{}
	sightings.Set(sightingcatalog.Catalog{ByKind: map[string]sightingcatalog.Kind{"analysis": {Kind: "analysis", DataSchema: map[string]any{"type": "object", "additionalProperties": true}}}})
	hub := events.NewHub()
	svc := service.New(stores, commands, sightings, hub)
	stores.Objects["obj-1"] = model.Object{ObjectID: "obj-1", Type: "f", OwnerType: "entity", OwnerID: "e", JSON: model.JSONMap{}}
	return httpapi.NewRouter(httpapi.Dependencies{
		AllowedOrigins: nil,
		StartedAt:      time.Now().UTC(),
		MaxUploadBytes: 50,
		Services:       svc,
		Events:         hub,
		CommandCatalog: commands,
		ObjectStore:    stores,
		Readiness:      func(_ context.Context) (model.ReadinessResponse, int) { return model.ReadinessResponse{Status: "ready", Timestamp: time.Now().UTC(), Dependencies: map[string]model.DependencyStatus{}}, 200 },
		Descriptor:     func() (model.ServiceDescriptor, error) { return model.ServiceDescriptor{}, nil },
	})
}

func TestObjectFileUploadOversizeReturns413(t *testing.T) {
	router := testUploadRouter()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	if err := mw.WriteField("file_id", "f1"); err != nil {
		t.Fatal(err)
	}
	pw, err := mw.CreateFormFile("file", "x.dat")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pw.Write([]byte(strings.Repeat("x", 200))); err != nil {
		t.Fatal(err)
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/objects/obj-1/files", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d: %s", rr.Code, rr.Body.String())
	}
}

