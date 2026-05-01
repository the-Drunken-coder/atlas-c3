package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/catalog"
	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/events"
	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/httpapi"
	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/model"
	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/objectfiles"
	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/service"
	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/service/servicetest"
	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/sightingcatalog"
)

type routerFixture struct {
	stores *servicetest.MemoryStore
	router http.Handler
	files  *objectfiles.Store
}

func newRouterFixture(t *testing.T, maxUploadBytes int64) routerFixture {
	t.Helper()
	cat, err := servicetest.NewDefaultCommandCatalog()
	if err != nil {
		t.Fatal(err)
	}
	stores := servicetest.NewMemoryStore()
	stores.Entities["asset-1"] = model.Entity{EntityID: "asset-1", Type: "asset", JSON: model.JSONMap{"components": map[string]any{"supported_commands": map[string]any{"observed_at": time.Now().UTC().Format(time.RFC3339), "commands": []any{"move_to_location"}}}}}
	stores.Objects["obj-1"] = model.Object{ObjectID: "obj-1", Type: "f", OwnerType: "entity", OwnerID: "e", JSON: model.JSONMap{}}
	commands := &catalog.Active{}
	commands.Set(cat)
	sightings := &sightingcatalog.Active{}
	sightings.Set(sightingcatalog.Catalog{ByKind: map[string]sightingcatalog.Kind{"analysis": {Kind: "analysis", DataSchema: map[string]any{"type": "object", "additionalProperties": true}}}})
	hub := events.NewHub()
	svc := service.New(stores, commands, sightings, hub)
	files, err := objectfiles.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return routerFixture{stores: stores, files: files, router: httpapi.NewRouter(httpapi.Dependencies{
		AllowedOrigins: []string{"http://localhost:5173"},
		StartedAt:      time.Now().UTC(),
		MaxUploadBytes: maxUploadBytes,
		Services:       svc,
		Events:         hub,
		CommandCatalog: commands,
		ObjectStore:    stores,
		Files:          files,
		Readiness: func(_ context.Context) (model.ReadinessResponse, int) {
			return model.ReadinessResponse{Status: "ready", Timestamp: time.Now().UTC(), Dependencies: map[string]model.DependencyStatus{"postgres": {Status: "ready"}, "object_storage": {Status: "ready"}, "command_catalog": {Status: "ready"}}}, http.StatusOK
		},
		Descriptor: func() (model.ServiceDescriptor, error) {
			return model.ServiceDescriptor{Service: model.ServiceName, Status: "ok", Version: "test", StartedAt: time.Now().UTC(), ActiveCommandCatalogObjectID: cat.ObjectID, Links: map[string]string{"health": "/health"}}, nil
		},
	})}
}

func testRouter(t *testing.T) http.Handler {
	t.Helper()
	return newRouterFixture(t, 20*1024*1024).router
}

func testUploadRouter(t *testing.T) (*servicetest.MemoryStore, http.Handler, *objectfiles.Store) {
	t.Helper()
	fixture := newRouterFixture(t, 50)
	return fixture.stores, fixture.router, fixture.files
}

func assertCleanStagingDir(t *testing.T, files *objectfiles.Store) {
	t.Helper()
	root := files.StagingDir()
	var leftover []string
	ferr := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path != root {
			rel, relErr := filepath.Rel(root, path)
			if relErr != nil {
				return relErr
			}
			if d.IsDir() {
				rel += string(filepath.Separator)
			}
			leftover = append(leftover, rel)
		}
		return nil
	})
	if ferr != nil {
		t.Fatalf("failed to walk staging dir: %v", ferr)
	}
	if len(leftover) > 0 {
		t.Fatalf("staging directory not clean, found files: %v", leftover)
	}
}

func TestHealthEndpoint(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	testRouter(t).ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", rr.Code)
	}
}

func TestCreateEntityRejectsUnknownFields(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/entities", strings.NewReader(`{"entity_id":"asset-2","type":"asset","json":{"components":{"supported_commands":{"observed_at":"2026-01-01T00:00:00Z","commands":[]}}},"extra":true}`))
	rr := httptest.NewRecorder()
	testRouter(t).ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestCreateTaskPublishesCreatedResource(t *testing.T) {
	router := testRouter(t)
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

func TestObjectFileUploadOversizeReturns413(t *testing.T) {
	_, router, _ := testUploadRouter(t)
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
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
	req := httptest.NewRequest(http.MethodPost, "/objects/obj-1/files/f1", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestObjectFileUploadMalformedMultipartReturns400(t *testing.T) {
	_, router, _ := testUploadRouter(t)
	req := httptest.NewRequest(http.MethodPost, "/objects/obj-1/files/f1", strings.NewReader(""))
	req.Header.Set("Content-Type", "multipart/form-data")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestObjectFileUploadTruncatedMultipartReturns400(t *testing.T) {
	_, router, _ := testUploadRouter(t)
	body := "--testboundary\r\n" +
		"Content-Disposition: form-data; name=\"file\"; filename=\"x.dat\"\r\n" +
		"Content-Type: application/octet-stream\r\n\r\nabc"
	req := httptest.NewRequest(http.MethodPost, "/objects/obj-1/files/f1", strings.NewReader(body))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=testboundary")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestObjectFileUploadRejectsInvalidContentTypeOverride(t *testing.T) {
	_, router, _ := testUploadRouter(t)
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	if err := mw.WriteField("content_type", "text/plain\r\nX-Test: injected"); err != nil {
		t.Fatal(err)
	}
	pw, err := mw.CreateFormFile("file", "x.dat")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pw.Write([]byte("abc")); err != nil {
		t.Fatal(err)
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/objects/obj-1/files/f1", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestObjectFileUploadRejectsLegacyFileIDFormField(t *testing.T) {
	_, router, _ := testUploadRouter(t)
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	if err := mw.WriteField("file_id", "f1"); err != nil {
		t.Fatal(err)
	}
	pw, err := mw.CreateFormFile("file", "x.dat")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pw.Write([]byte("abc")); err != nil {
		t.Fatal(err)
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/objects/obj-1/files/f1", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestObjectFileUploadRejectsLateFileIDFormField(t *testing.T) {
	stores, router, _ := testUploadRouter(t)
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	pw, err := mw.CreateFormFile("file", "x.dat")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pw.Write([]byte("abc")); err != nil {
		t.Fatal(err)
	}
	// Add file_id AFTER the file part to bypass early rejection
	if err := mw.WriteField("file_id", "f1"); err != nil {
		t.Fatal(err)
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/objects/obj-1/files/f1", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
	if _, ok := stores.ObjectFiles[servicetest.ObjectFileKey{ObjectID: "obj-1", FileID: "f1"}]; ok {
		t.Fatal("expected file to be deleted from store after late file_id rejection")
	}
	if _, ok := stores.FileBytes[servicetest.ObjectFileKey{ObjectID: "obj-1", FileID: "f1"}]; ok {
		t.Fatal("expected file bytes to be deleted from store after late file_id rejection")
	}
}

func TestObjectFileUploadRejectsOverlongFileIDBeforeStaging(t *testing.T) {
	_, router, files := testUploadRouter(t)
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	pw, err := mw.CreateFormFile("file", "x.dat")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pw.Write([]byte("abc")); err != nil {
		t.Fatal(err)
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/objects/obj-1/files/"+strings.Repeat("x", 51), &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
	assertCleanStagingDir(t, files)
}

func TestObjectFileUploadAcceptsValidContentTypeOverride(t *testing.T) {
	_, router, _ := testUploadRouter(t)
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	if err := mw.WriteField("content_type", "text/plain; charset=utf-8"); err != nil {
		t.Fatal(err)
	}
	pw, err := mw.CreateFormFile("file", "x.dat")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pw.Write([]byte("abc")); err != nil {
		t.Fatal(err)
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/objects/obj-1/files/f1", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	var file model.ObjectFile
	if err := json.Unmarshal(rr.Body.Bytes(), &file); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if file.ContentType != "text/plain; charset=utf-8" {
		t.Fatalf("unexpected content type: %s", file.ContentType)
	}
}

func TestObjectFileUploadFallsBackToDetectedContentType(t *testing.T) {
	stores, router, _ := testUploadRouter(t)
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	header := textproto.MIMEHeader{}
	header.Set("Content-Disposition", `form-data; name="file"; filename="x.dat"`)
	pw, err := mw.CreatePart(header)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pw.Write([]byte("abc")); err != nil {
		t.Fatal(err)
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/objects/obj-1/files/f1", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	got := stores.ObjectFiles[servicetest.ObjectFileKey{ObjectID: "obj-1", FileID: "f1"}]
	if got.ContentType != "text/plain; charset=utf-8" {
		t.Fatalf("unexpected content type: %q", got.ContentType)
	}
}

func TestObjectFileUploadAcceptsMetadataAfterFilePart(t *testing.T) {
	stores, router, files := testUploadRouter(t)
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	pw, err := mw.CreateFormFile("file", "x.dat")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pw.Write([]byte("abc")); err != nil {
		t.Fatal(err)
	}
	if err := mw.WriteField("usage_hint", "thumbnail"); err != nil {
		t.Fatal(err)
	}
	if err := mw.WriteField("content_type", "text/plain; charset=utf-8"); err != nil {
		t.Fatal(err)
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/objects/obj-1/files/f1", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	got, ok := stores.ObjectFiles[servicetest.ObjectFileKey{ObjectID: "obj-1", FileID: "f1"}]
	if !ok {
		t.Fatal("expected object file to be persisted")
	}
	if got.UsageHint != "thumbnail" {
		t.Fatalf("unexpected usage hint: %q", got.UsageHint)
	}
	if got.ContentType != "text/plain; charset=utf-8" {
		t.Fatalf("unexpected content type: %q", got.ContentType)
	}
	assertCleanStagingDir(t, files)
}

func TestGetObjectFileContentFallsBackForInvalidStoredContentType(t *testing.T) {
	stores, router, _ := testUploadRouter(t)
	stores.ObjectFiles[servicetest.ObjectFileKey{ObjectID: "obj-1", FileID: "f1"}] = model.ObjectFile{FileID: "f1", ObjectID: "obj-1", ContentType: "bad\r\nvalue", SizeBytes: 3}
	stores.FileBytes[servicetest.ObjectFileKey{ObjectID: "obj-1", FileID: "f1"}] = []byte("abc")
	req := httptest.NewRequest(http.MethodGet, "/objects/obj-1/files/f1/content", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if got := rr.Header().Get("Content-Type"); got != "application/octet-stream" {
		t.Fatalf("unexpected content type header: %q", got)
	}
}

// Issue #8: a multipart payload with TWO parts after the file part is rejected
// and no object file ends up persisted. The previous loop-based implementation
// happened to behave the same way for this case (it returned after the first
// extra part) — this test pins the behaviour so the simplified implementation
// can't regress.
func TestObjectFileUploadRejectsMultipleTrailingParts(t *testing.T) {
	stores, router, files := testUploadRouter(t)
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	pw, err := mw.CreateFormFile("file", "x.dat")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pw.Write([]byte("abc")); err != nil {
		t.Fatal(err)
	}
	if err := mw.WriteField("trailing_one", "v1"); err != nil {
		t.Fatal(err)
	}
	if err := mw.WriteField("trailing_two", "v2"); err != nil {
		t.Fatal(err)
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/objects/obj-1/files/f1", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
	if _, ok := stores.ObjectFiles[servicetest.ObjectFileKey{ObjectID: "obj-1", FileID: "f1"}]; ok {
		t.Fatal("expected staged file to be removed when extra parts are present")
	}
	assertCleanStagingDir(t, files)
}

// Issue #9: io.Copy errors while draining an extra trailing part are now
// surfaced via mapMultipartReadError — a truncated trailing part still yields
// a 400 with no persisted file.
func TestObjectFileUploadTruncatedTrailingPartReturns400(t *testing.T) {
	stores, router, files := testUploadRouter(t)
	// Construct a body where the trailing extra part is truncated mid-data.
	body := "--testboundary\r\n" +
		"Content-Disposition: form-data; name=\"file\"; filename=\"x.dat\"\r\n" +
		"Content-Type: application/octet-stream\r\n\r\nabc\r\n" +
		"--testboundary\r\n" +
		"Content-Disposition: form-data; name=\"extra\"\r\n\r\npartial-without-terminator"
	req := httptest.NewRequest(http.MethodPost, "/objects/obj-1/files/f1", strings.NewReader(body))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=testboundary")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
	if _, ok := stores.ObjectFiles[servicetest.ObjectFileKey{ObjectID: "obj-1", FileID: "f1"}]; ok {
		t.Fatal("expected no object file to be persisted when trailing part is truncated")
	}
	assertCleanStagingDir(t, files)
}
