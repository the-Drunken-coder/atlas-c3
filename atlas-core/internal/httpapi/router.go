package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/catalog"
	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/events"
	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/logging"
	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/model"
	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/objectfiles"
	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/service"
	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/store"
)

type Dependencies struct {
	AllowedOrigins []string
	StartedAt      time.Time
	MaxUploadBytes int64
	Services       *service.Services
	Events         *events.Hub
	CommandCatalog *catalog.Active
	ObjectStore    interface {
		StorageStatus(context.Context) model.DependencyStatus
	}
	Logger     *logging.Logger
	Readiness  func(context.Context) (model.ReadinessResponse, int)
	Descriptor func() (model.ServiceDescriptor, error)
}

type Router struct {
	deps Dependencies
}

func NewRouter(deps Dependencies) http.Handler {
	router := &Router{deps: deps}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", router.handleDescriptor)
	mux.HandleFunc("GET /health", router.handleHealth)
	mux.HandleFunc("GET /readiness", router.handleReadiness)
	mux.HandleFunc("GET /entities", router.handleListEntities)
	mux.HandleFunc("POST /entities", router.handleCreateEntity)
	mux.HandleFunc("GET /entities/{entity_id}", router.handleGetEntity)
	mux.HandleFunc("PATCH /entities/{entity_id}", router.handlePatchEntity)
	mux.HandleFunc("DELETE /entities/{entity_id}", router.handleDeleteEntity)
	mux.HandleFunc("GET /entities/{entity_id}/tasks", router.handleListEntityTasks)
	mux.HandleFunc("GET /observations", router.handleListObservations)
	mux.HandleFunc("POST /observations", router.handleCreateObservation)
	mux.HandleFunc("GET /observations/{observation_id}", router.handleGetObservation)
	mux.HandleFunc("PATCH /observations/{observation_id}", router.handlePatchObservation)
	mux.HandleFunc("DELETE /observations/{observation_id}", router.handleDeleteObservation)
	mux.HandleFunc("GET /tasks", router.handleListTasks)
	mux.HandleFunc("POST /tasks", router.handleCreateTask)
	mux.HandleFunc("GET /tasks/{task_id}", router.handleGetTask)
	mux.HandleFunc("PATCH /tasks/{task_id}", router.handlePatchTask)
	mux.HandleFunc("DELETE /tasks/{task_id}", router.handleDeleteTask)
	mux.HandleFunc("POST /tasks/{task_id}/status", router.handleTaskStatus)
	mux.HandleFunc("GET /objects", router.handleListObjects)
	mux.HandleFunc("POST /objects", router.handleCreateObject)
	mux.HandleFunc("GET /objects/{object_id}", router.handleGetObject)
	mux.HandleFunc("PATCH /objects/{object_id}", router.handlePatchObject)
	mux.HandleFunc("DELETE /objects/{object_id}", router.handleDeleteObject)
	mux.HandleFunc("POST /objects/{object_id}/files", router.handleUploadObjectFile)
	mux.HandleFunc("GET /objects/{object_id}/files/{file_id}", router.handleGetObjectFile)
	mux.HandleFunc("POST /objects/{object_id}/files/{file_id}/append", router.handleAppendObjectFile)
	mux.HandleFunc("GET /objects/{object_id}/files/{file_id}/content", router.handleGetObjectFileContent)
	mux.HandleFunc("DELETE /objects/{object_id}/files/{file_id}", router.handleDeleteObjectFile)
	mux.HandleFunc("GET /queries/full", router.handleFullQuery)
	mux.HandleFunc("GET /stream/changes", router.handleStream)
	return router.wrap(mux)
}

func (r *Router) wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		started := time.Now()
		requestID := newID()
		ctx := req.Context()
		if r.deps.Logger != nil {
			ctx = r.deps.Logger.WithRequest(ctx, requestID)
		}
		req = req.WithContext(ctx)
		origin := req.Header.Get("Origin")
		if origin != "" {
			w.Header().Set("Vary", "Origin")
			if allowedOrigin(r.deps, origin) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
				w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PATCH,DELETE,OPTIONS")
			}
		}
		if req.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		ww := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		defer func() {
			if recovered := recover(); recovered != nil {
				r.writeError(ww, req, model.InternalError("internal server error", fmt.Errorf("panic: %v", recovered)))
			}
			if r.deps.Logger != nil {
				r.deps.Logger.Component("api").InfoContext(req.Context(), "request complete",
					slog.String("event", "api.request"),
					slog.String("request_id", requestID),
					slog.String("method", req.Method),
					slog.String("path", req.URL.Path),
					slog.Int("status", ww.status),
					slog.Int64("duration_ms", time.Since(started).Milliseconds()),
				)
			}
		}()
		next.ServeHTTP(ww, req)
	})
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func allowedOrigin(deps Dependencies, origin string) bool {
	for _, allowed := range deps.AllowedOrigins {
		if allowed == origin {
			return true
		}
	}
	return false
}

func (r *Router) handleDescriptor(w http.ResponseWriter, req *http.Request) {
	descriptor, err := r.deps.Descriptor()
	if err != nil {
		r.writeError(w, req, err)
		return
	}
	writeJSON(w, http.StatusOK, descriptor)
}

func (r *Router) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, model.HealthResponse{Status: "ok", Service: model.ServiceName, Timestamp: time.Now().UTC()})
}

func (r *Router) handleReadiness(w http.ResponseWriter, req *http.Request) {
	response, status := r.deps.Readiness(req.Context())
	writeJSON(w, status, response)
}

func (r *Router) handleListEntities(w http.ResponseWriter, req *http.Request) {
	if err := rejectUnknownQuery(req, "type", "limit", "offset"); err != nil {
		r.writeError(w, req, err)
		return
	}
	pagination, pageErr := model.ParsePagination(req.URL.Query())
	if pageErr != nil {
		r.writeError(w, req, pageErr)
		return
	}
	items, total, err := r.deps.Services.ListEntities(req.Context(), store.EntityListFilter{Type: req.URL.Query().Get("type")}, pagination)
	if err != nil {
		r.writeError(w, req, err)
		return
	}
	writePaginatedJSON(w, http.StatusOK, items, pagination, total)
}

func (r *Router) handleCreateEntity(w http.ResponseWriter, req *http.Request) {
	payload, raw, err := decodeJSON(req.Body)
	if err != nil {
		r.writeError(w, req, err)
		return
	}
	if unknown := model.TopLevelUnknownFields(raw, "entity_id", "type", "subtype", "alias", "json"); len(unknown) > 0 {
		r.writeError(w, req, validationUnknownFields(unknown...))
		return
	}
	input := service.EntityCreateInput{EntityID: readString(payload, "entity_id"), Type: readString(payload, "type"), Subtype: readString(payload, "subtype"), Alias: readString(payload, "alias"), JSON: readJSONMap(payload["json"])}
	entity, err := r.deps.Services.CreateEntity(req.Context(), input)
	if err != nil {
		r.writeError(w, req, err)
		return
	}
	writeJSON(w, http.StatusCreated, entity)
}

func (r *Router) handleGetEntity(w http.ResponseWriter, req *http.Request) {
	item, err := r.deps.Services.GetEntity(req.Context(), req.PathValue("entity_id"))
	if err != nil {
		r.writeError(w, req, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (r *Router) handlePatchEntity(w http.ResponseWriter, req *http.Request) {
	payload, raw, err := decodeJSON(req.Body)
	if err != nil {
		r.writeError(w, req, err)
		return
	}
	if unknown := model.TopLevelUnknownFields(raw, "subtype", "alias", "json"); len(unknown) > 0 {
		r.writeError(w, req, validationUnknownFields(unknown...))
		return
	}
	var subtype *string
	if _, ok := payload["subtype"]; ok {
		value := readString(payload, "subtype")
		subtype = &value
	}
	var alias *string
	if _, ok := payload["alias"]; ok {
		value := readString(payload, "alias")
		alias = &value
	}
	entity, err := r.deps.Services.PatchEntity(req.Context(), req.PathValue("entity_id"), service.EntityPatchInput{Subtype: subtype, Alias: alias, JSON: readJSONMap(payload["json"])})
	if err != nil {
		r.writeError(w, req, err)
		return
	}
	writeJSON(w, http.StatusOK, entity)
}

func (r *Router) handleDeleteEntity(w http.ResponseWriter, req *http.Request) {
	if err := r.deps.Services.DeleteEntity(req.Context(), req.PathValue("entity_id")); err != nil {
		r.writeError(w, req, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (r *Router) handleListEntityTasks(w http.ResponseWriter, req *http.Request) {
	if err := rejectUnknownQuery(req, "limit", "offset"); err != nil {
		r.writeError(w, req, err)
		return
	}
	pagination, pageErr := model.ParsePagination(req.URL.Query())
	if pageErr != nil {
		r.writeError(w, req, pageErr)
		return
	}
	items, total, err := r.deps.Services.ListEntityTasks(req.Context(), req.PathValue("entity_id"), pagination)
	if err != nil {
		r.writeError(w, req, err)
		return
	}
	writePaginatedJSON(w, http.StatusOK, items, pagination, total)
}

func (r *Router) handleListObservations(w http.ResponseWriter, req *http.Request) {
	if err := rejectUnknownQuery(req, "source_asset_id", "updated_after", "limit", "offset"); err != nil {
		r.writeError(w, req, err)
		return
	}
	pagination, pageErr := model.ParsePagination(req.URL.Query())
	if pageErr != nil {
		r.writeError(w, req, pageErr)
		return
	}
	updatedAfter := req.URL.Query().Get("updated_after")
	if updatedAfter != "" {
		if _, err := time.Parse(time.RFC3339, updatedAfter); err != nil {
			r.writeError(w, req, model.ValidationError(model.FieldError{Field: "updated_after", Code: "invalid_value", Message: "updated_after must be a valid RFC3339 timestamp"}))
			return
		}
	}
	items, total, err := r.deps.Services.ListObservations(req.Context(), store.ObservationListFilter{SourceAssetID: req.URL.Query().Get("source_asset_id"), UpdatedAfter: updatedAfter}, pagination)
	if err != nil {
		r.writeError(w, req, err)
		return
	}
	writePaginatedJSON(w, http.StatusOK, items, pagination, total)
}

func (r *Router) handleCreateObservation(w http.ResponseWriter, req *http.Request) {
	payload, raw, err := decodeJSON(req.Body)
	if err != nil {
		r.writeError(w, req, err)
		return
	}
	if unknown := model.TopLevelUnknownFields(raw, "observation_id", "source_asset_id", "json"); len(unknown) > 0 {
		r.writeError(w, req, validationUnknownFields(unknown...))
		return
	}
	item, err := r.deps.Services.CreateObservation(req.Context(), service.ObservationCreateInput{ObservationID: readString(payload, "observation_id"), SourceAssetID: readString(payload, "source_asset_id"), JSON: readJSONMap(payload["json"])})
	if err != nil {
		r.writeError(w, req, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (r *Router) handleGetObservation(w http.ResponseWriter, req *http.Request) {
	item, err := r.deps.Services.GetObservation(req.Context(), req.PathValue("observation_id"))
	if err != nil {
		r.writeError(w, req, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (r *Router) handlePatchObservation(w http.ResponseWriter, req *http.Request) {
	payload, raw, err := decodeJSON(req.Body)
	if err != nil {
		r.writeError(w, req, err)
		return
	}
	if unknown := model.TopLevelUnknownFields(raw, "json"); len(unknown) > 0 {
		r.writeError(w, req, validationUnknownFields(unknown...))
		return
	}
	item, err := r.deps.Services.PatchObservation(req.Context(), req.PathValue("observation_id"), service.ObservationPatchInput{JSON: readJSONMap(payload["json"])})
	if err != nil {
		r.writeError(w, req, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (r *Router) handleDeleteObservation(w http.ResponseWriter, req *http.Request) {
	if err := r.deps.Services.DeleteObservation(req.Context(), req.PathValue("observation_id")); err != nil {
		r.writeError(w, req, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (r *Router) handleListTasks(w http.ResponseWriter, req *http.Request) {
	if err := rejectUnknownQuery(req, "asset_id", "status", "limit", "offset"); err != nil {
		r.writeError(w, req, err)
		return
	}
	pagination, pageErr := model.ParsePagination(req.URL.Query())
	if pageErr != nil {
		r.writeError(w, req, pageErr)
		return
	}
	items, total, err := r.deps.Services.ListTasks(req.Context(), store.TaskListFilter{AssetID: req.URL.Query().Get("asset_id"), Status: req.URL.Query().Get("status")}, pagination)
	if err != nil {
		r.writeError(w, req, err)
		return
	}
	writePaginatedJSON(w, http.StatusOK, items, pagination, total)
}

func (r *Router) handleCreateTask(w http.ResponseWriter, req *http.Request) {
	payload, raw, err := decodeJSON(req.Body)
	if err != nil {
		r.writeError(w, req, err)
		return
	}
	if unknown := model.TopLevelUnknownFields(raw, "task_id", "asset_id", "json"); len(unknown) > 0 {
		r.writeError(w, req, validationUnknownFields(unknown...))
		return
	}
	item, err := r.deps.Services.CreateTask(req.Context(), service.TaskCreateInput{TaskID: readString(payload, "task_id"), AssetID: readString(payload, "asset_id"), JSON: readJSONMap(payload["json"])})
	if err != nil {
		r.writeError(w, req, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (r *Router) handleGetTask(w http.ResponseWriter, req *http.Request) {
	item, err := r.deps.Services.GetTask(req.Context(), req.PathValue("task_id"))
	if err != nil {
		r.writeError(w, req, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (r *Router) handlePatchTask(w http.ResponseWriter, req *http.Request) {
	payload, raw, err := decodeJSON(req.Body)
	if err != nil {
		r.writeError(w, req, err)
		return
	}
	if unknown := model.TopLevelUnknownFields(raw, "json"); len(unknown) > 0 {
		r.writeError(w, req, validationUnknownFields(unknown...))
		return
	}
	item, err := r.deps.Services.PatchTask(req.Context(), req.PathValue("task_id"), service.TaskPatchInput{JSON: readJSONMap(payload["json"])})
	if err != nil {
		r.writeError(w, req, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (r *Router) handleDeleteTask(w http.ResponseWriter, req *http.Request) {
	if err := r.deps.Services.DeleteTask(req.Context(), req.PathValue("task_id")); err != nil {
		r.writeError(w, req, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (r *Router) handleTaskStatus(w http.ResponseWriter, req *http.Request) {
	payload, raw, err := decodeJSON(req.Body)
	if err != nil {
		r.writeError(w, req, err)
		return
	}
	if unknown := model.TopLevelUnknownFields(raw, "status", "json"); len(unknown) > 0 {
		r.writeError(w, req, validationUnknownFields(unknown...))
		return
	}
	item, err := r.deps.Services.TransitionTaskStatus(req.Context(), req.PathValue("task_id"), service.TaskStatusInput{Status: readString(payload, "status"), JSON: readJSONMap(payload["json"])})
	if err != nil {
		r.writeError(w, req, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (r *Router) handleListObjects(w http.ResponseWriter, req *http.Request) {
	if err := rejectUnknownQuery(req, "owner_type", "owner_id", "type", "limit", "offset"); err != nil {
		r.writeError(w, req, err)
		return
	}
	if (req.URL.Query().Get("owner_type") == "") != (req.URL.Query().Get("owner_id") == "") {
		r.writeError(w, req, model.ValidationError(model.FieldError{Field: "owner", Code: "required", Message: "owner_type and owner_id must be provided together"}))
		return
	}
	pagination, pageErr := model.ParsePagination(req.URL.Query())
	if pageErr != nil {
		r.writeError(w, req, pageErr)
		return
	}
	items, total, err := r.deps.Services.ListObjects(req.Context(), store.ObjectListFilter{OwnerType: req.URL.Query().Get("owner_type"), OwnerID: req.URL.Query().Get("owner_id"), Type: req.URL.Query().Get("type")}, pagination)
	if err != nil {
		r.writeError(w, req, err)
		return
	}
	writePaginatedJSON(w, http.StatusOK, items, pagination, total)
}

func (r *Router) handleCreateObject(w http.ResponseWriter, req *http.Request) {
	payload, raw, err := decodeJSON(req.Body)
	if err != nil {
		r.writeError(w, req, err)
		return
	}
	if unknown := model.TopLevelUnknownFields(raw, "object_id", "type", "owner_type", "owner_id", "json"); len(unknown) > 0 {
		r.writeError(w, req, validationUnknownFields(unknown...))
		return
	}
	item, err := r.deps.Services.CreateObject(req.Context(), service.ObjectCreateInput{ObjectID: readString(payload, "object_id"), Type: readString(payload, "type"), OwnerType: readString(payload, "owner_type"), OwnerID: readString(payload, "owner_id"), JSON: readJSONMap(payload["json"])})
	if err != nil {
		r.writeError(w, req, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (r *Router) handleGetObject(w http.ResponseWriter, req *http.Request) {
	item, err := r.deps.Services.GetObject(req.Context(), req.PathValue("object_id"))
	if err != nil {
		r.writeError(w, req, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (r *Router) handlePatchObject(w http.ResponseWriter, req *http.Request) {
	payload, raw, err := decodeJSON(req.Body)
	if err != nil {
		r.writeError(w, req, err)
		return
	}
	if unknown := model.TopLevelUnknownFields(raw, "type", "json"); len(unknown) > 0 {
		r.writeError(w, req, validationUnknownFields(unknown...))
		return
	}
	var objectType *string
	if _, ok := payload["type"]; ok {
		value := readString(payload, "type")
		objectType = &value
	}
	item, err := r.deps.Services.PatchObject(req.Context(), req.PathValue("object_id"), service.ObjectPatchInput{Type: objectType, JSON: readJSONMap(payload["json"])})
	if err != nil {
		r.writeError(w, req, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (r *Router) handleDeleteObject(w http.ResponseWriter, req *http.Request) {
	if err := r.deps.Services.DeleteObject(req.Context(), req.PathValue("object_id")); err != nil {
		r.writeError(w, req, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

const (
	multipartRequestOverhead = 256 << 10
	smallFormFieldMax        = 64 * 1024
)

func readMultipartFormFieldString(part *multipart.Part) (string, error) {
	b, err := io.ReadAll(io.LimitReader(part, smallFormFieldMax+1))
	if err != nil {
		return "", err
	}
	if int64(len(b)) > smallFormFieldMax {
		return "", model.PayloadTooLarge("form field is too large")
	}
	return strings.TrimSpace(string(b)), nil
}

func (r *Router) handleUploadObjectFile(w http.ResponseWriter, req *http.Request) {
	maxFile := r.deps.MaxUploadBytes
	if maxFile <= 0 {
		maxFile = 16 * 1024 * 1024
	}
	req.Body = http.MaxBytesReader(w, req.Body, maxFile+multipartRequestOverhead)
	mr, err := req.MultipartReader()
	if err != nil {
		r.writeError(w, req, r.mapMultipartReaderError(err))
		return
	}
	objectID := req.PathValue("object_id")
	var fileID, usageHint, contentTypeOverride string
	var filePart *multipart.Part
parts:
	for {
		part, err := mr.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			r.writeError(w, req, r.mapMultipartReadError(err))
			return
		}
		name := part.FormName()
		switch name {
		case "file_id":
			v, perr := readMultipartFormFieldString(part)
			_ = part.Close()
			if perr != nil {
				r.writeError(w, req, r.mapMultipartReadError(perr))
				return
			}
			fileID = v
		case "usage_hint":
			v, perr := readMultipartFormFieldString(part)
			_ = part.Close()
			if perr != nil {
				r.writeError(w, req, r.mapMultipartReadError(perr))
				return
			}
			usageHint = v
		case "content_type":
			v, perr := readMultipartFormFieldString(part)
			_ = part.Close()
			if perr != nil {
				r.writeError(w, req, r.mapMultipartReadError(perr))
				return
			}
			contentTypeOverride = v
		case "file":
			if fileID == "" {
				_ = part.Close()
				r.writeError(w, req, model.ValidationError(model.FieldError{Field: "multipart", Code: "invalid_value", Message: "file_id part must be sent before file part"}))
				return
			}
			if filePart != nil {
				_ = part.Close()
				r.writeError(w, req, model.ValidationError(model.FieldError{Field: "file", Code: "invalid_value", Message: "only one file part is allowed"}))
				return
			}
			// The file part must be the last part we read: another NextPart() would
			// discard the unread file body (see mime/multipart.Reader docs).
			filePart = part
			break parts
		default:
			_, _ = io.Copy(io.Discard, part)
			_ = part.Close()
		}
	}
	if fileID == "" {
		r.writeError(w, req, model.ValidationError(model.FieldError{Field: "file_id", Code: "required", Message: "file_id is required"}))
		return
	}
	if filePart == nil {
		r.writeError(w, req, model.ValidationError(model.FieldError{Field: "file", Code: "required", Message: "file upload is required"}))
		return
	}
	defer filePart.Close()
	contentType, err := filePartContentType(filePart, contentTypeOverride)
	if err != nil {
		r.writeError(w, req, err)
		return
	}
	item, err := r.deps.Services.UploadObjectFile(req.Context(), store.ObjectUploadInput{File: model.ObjectFile{FileID: fileID, ObjectID: objectID, UsageHint: objectfiles.SafeUsageHint(usageHint), ContentType: contentType}, Reader: filePart, MaxBytes: maxFile})
	if err != nil {
		r.writeError(w, req, r.mapChunkedReadError(err))
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (r *Router) mapMultipartReaderError(err error) error {
	if errors.Is(err, http.ErrNotMultipart) {
		return model.ValidationError(model.FieldError{Field: "content_type", Code: "invalid_value", Message: "multipart form required"})
	}
	if errors.Is(err, http.ErrMissingBoundary) {
		return model.ValidationError(model.FieldError{Field: "content_type", Code: "invalid_value", Message: "multipart form boundary is required"})
	}
	return r.mapChunkedReadError(err)
}

func (r *Router) mapMultipartReadError(err error) error {
	if errors.Is(err, io.EOF) {
		return model.ValidationError(model.FieldError{Field: "multipart", Code: "invalid_value", Message: "truncated multipart body"})
	}
	return r.mapChunkedReadError(err)
}

func (r *Router) mapChunkedReadError(err error) error {
	var mbe *http.MaxBytesError
	if errors.As(err, &mbe) {
		return model.PayloadTooLarge("request body or upload exceeds size limit")
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return model.ValidationError(model.FieldError{Field: "body", Code: "invalid_value", Message: "request body is incomplete or truncated"})
	}
	return err
}

func normalizeContentType(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "application/octet-stream", true
	}
	mediaType, params, err := mime.ParseMediaType(value)
	if err != nil || mediaType == "" {
		return "application/octet-stream", false
	}
	if len(params) == 0 {
		return mediaType, true
	}
	formatted := mime.FormatMediaType(mediaType, params)
	if formatted == "" {
		return mediaType, true
	}
	return formatted, true
}

func filePartContentType(p *multipart.Part, formOverride string) (string, error) {
	if formOverride != "" {
		normalized, ok := normalizeContentType(formOverride)
		if !ok {
			return "", model.ValidationError(model.FieldError{Field: "content_type", Code: "invalid_value", Message: "content_type must be a valid media type"})
		}
		return normalized, nil
	}
	ct := p.Header.Get("Content-Type")
	normalized, _ := normalizeContentType(ct)
	return normalized, nil
}

func (r *Router) handleGetObjectFile(w http.ResponseWriter, req *http.Request) {
	item, err := r.deps.Services.GetObjectFile(req.Context(), req.PathValue("object_id"), req.PathValue("file_id"))
	if err != nil {
		r.writeError(w, req, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (r *Router) handleAppendObjectFile(w http.ResponseWriter, req *http.Request) {
	maxB := r.deps.MaxUploadBytes
	if maxB <= 0 {
		maxB = 16 * 1024 * 1024
	}
	const appendBodyOverhead = 4096
	req.Body = http.MaxBytesReader(w, req.Body, maxB+appendBodyOverhead)
	item, err := r.deps.Services.AppendObjectFile(req.Context(), req.PathValue("object_id"), req.PathValue("file_id"), req.Body, maxB)
	if err != nil {
		r.writeError(w, req, r.mapChunkedReadError(err))
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (r *Router) handleGetObjectFileContent(w http.ResponseWriter, req *http.Request) {
	meta, rc, err := r.deps.Services.OpenObjectFile(req.Context(), req.PathValue("object_id"), req.PathValue("file_id"))
	if err != nil {
		r.writeError(w, req, err)
		return
	}
	defer rc.Close()
	contentType, _ := normalizeContentType(meta.ContentType)
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", meta.SizeBytes))
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, rc)
}

func (r *Router) handleDeleteObjectFile(w http.ResponseWriter, req *http.Request) {
	if err := r.deps.Services.DeleteObjectFile(req.Context(), req.PathValue("object_id"), req.PathValue("file_id")); err != nil {
		r.writeError(w, req, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (r *Router) handleFullQuery(w http.ResponseWriter, req *http.Request) {
	snapshot, err := r.deps.Services.FullQuery(req.Context())
	if err != nil {
		r.writeError(w, req, err)
		return
	}
	writeJSON(w, http.StatusOK, snapshot)
}

func (r *Router) handleStream(w http.ResponseWriter, req *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		r.writeError(w, req, model.InternalError("streaming unsupported", nil))
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	ch, cancel := r.deps.Events.Subscribe()
	defer cancel()
	keepAlive := time.NewTicker(15 * time.Second)
	defer keepAlive.Stop()
	for {
		select {
		case <-req.Context().Done():
			return
		case <-keepAlive.C:
			_, _ = w.Write([]byte(": keepalive\n\n"))
			flusher.Flush()
		case event, ok := <-ch:
			if !ok {
				// The hub closed our channel because we couldn't keep up
				// (slow-subscriber eviction). End the stream so the client
				// reconnects and re-snapshots state from /queries/full.
				return
			}
			payload, err := events.EncodeSSE(event)
			if err != nil {
				return
			}
			_, _ = w.Write(payload)
			flusher.Flush()
		}
	}
}

const maxJSONBodyBytes = 4 << 20

func decodeJSON(body io.Reader) (map[string]any, map[string]json.RawMessage, error) {
	rawBody, err := io.ReadAll(io.LimitReader(body, maxJSONBodyBytes+1))
	if err != nil {
		return nil, nil, model.ValidationError(model.FieldError{Field: "body", Code: "invalid_value", Message: "unable to read request body"})
	}
	if int64(len(rawBody)) > maxJSONBodyBytes {
		return nil, nil, model.PayloadTooLarge("request body exceeds maximum allowed size")
	}
	if len(strings.TrimSpace(string(rawBody))) == 0 {
		return map[string]any{}, map[string]json.RawMessage{}, nil
	}
	var payload map[string]any
	if err := json.Unmarshal(rawBody, &payload); err != nil {
		return nil, nil, model.ValidationError(model.FieldError{Field: "body", Code: "invalid_value", Message: "body must be valid JSON"})
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(rawBody, &raw); err != nil {
		return nil, nil, err
	}
	return payload, raw, nil
}

func readString(payload map[string]any, key string) string {
	value, _ := payload[key].(string)
	return value
}
func readJSONMap(value any) model.JSONMap {
	if value == nil {
		return model.JSONMap{}
	}
	if out, ok := value.(map[string]any); ok {
		return out
	}
	return model.JSONMap{}
}

func rejectUnknownQuery(req *http.Request, allowed ...string) error {
	allowedSet := map[string]struct{}{}
	for _, key := range allowed {
		allowedSet[key] = struct{}{}
	}
	fields := []model.FieldError{}
	for key := range req.URL.Query() {
		if _, ok := allowedSet[key]; !ok {
			fields = append(fields, model.FieldError{Field: key, Code: "unknown_field", Message: "unknown query parameter"})
		}
	}
	if len(fields) > 0 {
		return model.ValidationError(fields...)
	}
	return nil
}

func validationUnknownFields(fields ...string) error {
	list := make([]model.FieldError, 0, len(fields))
	for _, field := range fields {
		list = append(list, model.FieldError{Field: field, Code: "unknown_field", Message: "unknown field"})
	}
	return model.ValidationError(list...)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writePaginatedJSON(w http.ResponseWriter, status int, value any, pagination model.Pagination, total int) {
	w.Header().Set("X-Total-Count", fmt.Sprintf("%d", total))
	w.Header().Set("X-Limit", fmt.Sprintf("%d", pagination.Limit))
	w.Header().Set("X-Offset", fmt.Sprintf("%d", pagination.Offset))
	switch items := value.(type) {
	case []model.Entity:
		w.Header().Set("X-Returned-Count", fmt.Sprintf("%d", len(items)))
	case []model.Observation:
		w.Header().Set("X-Returned-Count", fmt.Sprintf("%d", len(items)))
	case []model.Task:
		w.Header().Set("X-Returned-Count", fmt.Sprintf("%d", len(items)))
	case []model.Object:
		w.Header().Set("X-Returned-Count", fmt.Sprintf("%d", len(items)))
	}
	writeJSON(w, status, value)
}

// writeError serializes a CoreError (wrapping unknown errors as InternalError)
// into the standard ErrorEnvelope and writes it. Any underlying cause attached
// to the CoreError is logged server-side, correlated with the generated
// error_id and the request_id from context, and is intentionally NOT included
// in the response to avoid leaking implementation details to API clients.
func (r *Router) writeError(w http.ResponseWriter, req *http.Request, err error) {
	coreErr, ok := model.IsCoreError(err)
	if !ok {
		coreErr = model.InternalError("internal server error", err)
	}
	errorID := newID()
	if r.deps.Logger != nil && coreErr.Cause() != nil {
		r.deps.Logger.Component("api").ErrorContext(req.Context(), "request error",
			slog.String("event", "api.error"),
			slog.String("request_id", logging.RequestID(req.Context())),
			slog.String("error_id", errorID),
			slog.Int("status", coreErr.StatusCode),
			slog.String("error_code", coreErr.ErrorCode),
			slog.String("cause", coreErr.Cause().Error()),
		)
	}
	envelope := model.ErrorEnvelope{Success: false, Message: coreErr.Message, ErrorCode: coreErr.ErrorCode, ErrorID: errorID, Timestamp: time.Now().UTC(), Path: req.URL.Path, Details: coreErr.Details}
	writeJSON(w, coreErr.StatusCode, envelope)
}

func newID() string {
	buffer := make([]byte, 8)
	if _, err := rand.Read(buffer); err != nil {
		return fmt.Sprintf("err-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buffer)
}
