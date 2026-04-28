package catalog

import (
	"testing"

	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/model"
)

func TestValidateSchemaRejectsUnsupportedKeyword(t *testing.T) {
	err := ValidateSchema(map[string]any{"type": "object", "oneOf": []any{}}, "schema")
	if err == nil {
		t.Fatal("expected unsupported keyword error")
	}
}

func TestValidateValueAcceptsConfiguredObject(t *testing.T) {
	schema := map[string]any{
		"type":                 "object",
		"required":             []any{"latitude"},
		"additionalProperties": false,
		"properties": map[string]any{
			"latitude": map[string]any{"type": "number", "minimum": -90.0, "maximum": 90.0},
		},
	}
	if err := ValidateValue(schema, map[string]any{"latitude": 42.0}, "parameters"); err != nil {
		t.Fatalf("expected value to validate: %v", err)
	}
}

func TestValidateSchemaRequiresTypeWhenConstraintsPresent(t *testing.T) {
	err := ValidateSchema(map[string]any{"minimum": 1.0}, "schema")
	if err == nil {
		t.Fatal("expected type required error for schema with constraints but no type")
	}
}

func TestValidateSchemaNonStringTypeDoesNotAddTypeRequired(t *testing.T) {
	err := ValidateSchema(map[string]any{"type": 1, "minimum": 0.0}, "schema")
	if err == nil {
		t.Fatal("expected error")
	}
	core, ok := model.IsCoreError(err)
	if !ok {
		t.Fatalf("expected CoreError: %v", err)
	}
	fields, _ := core.Details["fields"].([]model.FieldError)
	for _, f := range fields {
		if f.Field == "schema.type" && f.Code == "required" {
			t.Fatalf("non-string type should not also produce type required; fields: %#v", fields)
		}
	}
}

func TestValidateSchemaRejectsObjectAdditionalProperties(t *testing.T) {
	err := ValidateSchema(map[string]any{"type": "object", "additionalProperties": map[string]any{"type": "string"}}, "schema")
	if err == nil {
		t.Fatal("expected additionalProperties object form to be rejected")
	}
}

func TestValidateSchemaRejectsNonNumericMinimum(t *testing.T) {
	err := ValidateSchema(map[string]any{"type": "number", "minimum": "nope"}, "schema")
	if err == nil {
		t.Fatal("expected minimum type error")
	}
}

func TestValidateSchemaRejectsEmptyEnum(t *testing.T) {
	err := ValidateSchema(map[string]any{"type": "string", "enum": []any{}}, "schema")
	if err == nil {
		t.Fatal("expected empty enum error")
	}
}
