package model

import (
	"errors"
	"fmt"
	"net/http"
	"time"
)

type FieldError struct {
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ErrorEnvelope struct {
	Success   bool           `json:"success"`
	Message   string         `json:"message"`
	ErrorCode string         `json:"error_code"`
	ErrorID   string         `json:"error_id"`
	Timestamp time.Time      `json:"timestamp"`
	Path      string         `json:"path"`
	Details   map[string]any `json:"details,omitempty"`
}

type CoreError struct {
	StatusCode int
	ErrorCode  string
	Message    string
	Details    map[string]any
	// cause is the underlying error, kept server-side only. It is never
	// serialized into the API response — the response contains a generated
	// error_id which the router logs alongside the cause for correlation.
	cause error
}

func (e *CoreError) Error() string {
	return fmt.Sprintf("%s: %s", e.ErrorCode, e.Message)
}

// Cause returns the underlying error captured server-side. Callers (the HTTP
// router) use this to log the real failure correlated with the response
// error_id, without leaking it to API consumers.
func (e *CoreError) Cause() error {
	return e.cause
}

// Unwrap exposes the cause for errors.Is / errors.As.
func (e *CoreError) Unwrap() error {
	return e.cause
}

func NewCoreError(status int, code, message string, details map[string]any) *CoreError {
	return &CoreError{StatusCode: status, ErrorCode: code, Message: message, Details: details}
}

func ValidationError(fields ...FieldError) *CoreError {
	return NewCoreError(http.StatusBadRequest, "validation_failed", "request validation failed", map[string]any{"fields": fields})
}

func ImmutableFieldError(field string) *CoreError {
	return NewCoreError(http.StatusBadRequest, "immutable_field", "immutable field update rejected", map[string]any{"fields": []FieldError{{Field: field, Code: "immutable", Message: "field is immutable"}}})
}

func ImmutableFieldsError(fields ...string) *CoreError {
	fieldErrors := make([]FieldError, 0, len(fields))
	for _, field := range fields {
		fieldErrors = append(fieldErrors, FieldError{Field: field, Code: "immutable", Message: "field is immutable"})
	}
	return NewCoreError(http.StatusBadRequest, "immutable_field", "immutable field update rejected", map[string]any{"fields": fieldErrors})
}

func CommandValidationError(message string, fields ...FieldError) *CoreError {
	details := map[string]any{}
	if len(fields) > 0 {
		details["fields"] = fields
	}
	return NewCoreError(http.StatusBadRequest, "command_validation_failed", message, details)
}

func InvalidStatusTransition(current, target string) *CoreError {
	return NewCoreError(http.StatusBadRequest, "invalid_status_transition", "task status transition is not allowed", map[string]any{"current_status": current, "target_status": target})
}

func NotFound(resourceType, resourceID string) *CoreError {
	return NewCoreError(http.StatusNotFound, "not_found", fmt.Sprintf("%s not found", resourceType), map[string]any{"resource_type": resourceType, "resource_id": resourceID})
}

func Conflict(resourceType, resourceID, reason string, extra map[string]any) *CoreError {
	details := map[string]any{}
	for k, v := range extra {
		details[k] = v
	}
	details["resource_type"] = resourceType
	details["resource_id"] = resourceID
	details["reason"] = reason
	return NewCoreError(http.StatusConflict, "conflict", "resource conflict", details)
}

func PayloadTooLarge(message string) *CoreError {
	return NewCoreError(http.StatusRequestEntityTooLarge, "payload_too_large", message, nil)
}

func UnsupportedMediaType(message string) *CoreError {
	return NewCoreError(http.StatusUnsupportedMediaType, "unsupported_media_type", message, nil)
}

func StorageUnavailable(message string, details map[string]any) *CoreError {
	return NewCoreError(http.StatusServiceUnavailable, "storage_unavailable", message, details)
}

func CatalogUnavailable(message string, details map[string]any) *CoreError {
	return NewCoreError(http.StatusServiceUnavailable, "catalog_unavailable", message, details)
}

func InternalError(message string, err error) *CoreError {
	ce := NewCoreError(http.StatusInternalServerError, "internal_error", message, nil)
	ce.cause = err
	return ce
}

func IsCoreError(err error) (*CoreError, bool) {
	var coreErr *CoreError
	if errors.As(err, &coreErr) {
		return coreErr, true
	}
	return nil, false
}
