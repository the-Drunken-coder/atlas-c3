package sightingcatalog

import (
	"testing"

	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/model"
)

func TestValidateSightingRejectsNonStringKind(t *testing.T) {
	active := Catalog{
		ByKind: map[string]Kind{
			"analysis": {Kind: "analysis", DataSchema: map[string]any{"type": "object", "additionalProperties": true}},
		},
	}
	err := ValidateSighting(active, map[string]any{
		"observed_at": "2026-01-01T00:00:00Z",
		"kind":        1,
		"data":        map[string]any{},
	})
	if err == nil {
		t.Fatal("expected invalid_type")
	}
	coreErr, ok := model.IsCoreError(err)
	if !ok {
		t.Fatalf("expected CoreError, got %v", err)
	}
	fields, _ := coreErr.Details["fields"].([]model.FieldError)
	if len(fields) != 1 || fields[0].Field != "json.latest_sighting.kind" || fields[0].Code != "invalid_type" {
		t.Fatalf("unexpected fields: %#v", fields)
	}
}

func TestValidateSightingRequiresKind(t *testing.T) {
	active := Catalog{
		ByKind: map[string]Kind{
			"analysis": {Kind: "analysis", DataSchema: map[string]any{"type": "object", "additionalProperties": true}},
		},
	}
	err := ValidateSighting(active, map[string]any{
		"observed_at": "2026-01-01T00:00:00Z",
		"data":        map[string]any{},
	})
	if err == nil {
		t.Fatal("expected required error")
	}
	coreErr, ok := model.IsCoreError(err)
	if !ok {
		t.Fatalf("expected CoreError, got %v", err)
	}
	fields, _ := coreErr.Details["fields"].([]model.FieldError)
	if len(fields) != 1 || fields[0].Field != "json.latest_sighting.kind" || fields[0].Code != "required" {
		t.Fatalf("unexpected fields: %#v", fields)
	}
}

func TestValidateSightingRejectsNullKind(t *testing.T) {
	active := Catalog{
		ByKind: map[string]Kind{
			"analysis": {Kind: "analysis", DataSchema: map[string]any{"type": "object", "additionalProperties": true}},
		},
	}
	err := ValidateSighting(active, map[string]any{
		"observed_at": "2026-01-01T00:00:00Z",
		"kind":        nil,
		"data":        map[string]any{},
	})
	if err == nil {
		t.Fatal("expected invalid_type")
	}
	coreErr, ok := model.IsCoreError(err)
	if !ok {
		t.Fatalf("expected CoreError, got %v", err)
	}
	fields, _ := coreErr.Details["fields"].([]model.FieldError)
	if len(fields) != 1 || fields[0].Field != "json.latest_sighting.kind" || fields[0].Code != "invalid_type" {
		t.Fatalf("unexpected fields: %#v", fields)
	}
}
