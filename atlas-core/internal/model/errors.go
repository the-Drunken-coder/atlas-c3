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
}

func (e *CoreError) Error() string {
	return fmt.Sprintf("%s: %s", e.ErrorCode, e.Message)
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
	details := map[string]any{"resource_type": resourceType, "resource_id": resourceID, "reason": reason}
	for k, v := range extra {
		details[k] = v
	}
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
	details := map[string]any{}
	if err != nil {
		details["cause"] = err.Error()
	}
	return NewCoreError(http.StatusInternalServerError, "internal_error", message, details)
}

func IsCoreError(err error) (*CoreError, bool) {
	var coreErr *CoreError
	if errors.As(err, &coreErr) {
		return coreErr, true
	}
	return nil, false
}
