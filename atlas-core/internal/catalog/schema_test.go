package catalog

import (
	"testing"

	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/model"
)

func fieldErrors(t *testing.T, err error) []model.FieldError {
	t.Helper()
	core, ok := model.IsCoreError(err)
	if !ok {
		t.Fatalf("expected CoreError: %v", err)
	}
	fields, _ := core.Details["fields"].([]model.FieldError)
	return fields
}

func assertHasFieldError(t *testing.T, err error, field, code string) {
	t.Helper()
	for _, f := range fieldErrors(t, err) {
		if f.Field == field && f.Code == code {
			return
		}
	}
	t.Fatalf("expected field %q with code %q among %#v", field, code, fieldErrors(t, err))
}

func TestValidateSchemaRejectsUnsupportedKeyword(t *testing.T) {
	err := ValidateSchema(map[string]any{"type": "object", "oneOf": []any{}}, "schema")
	if err == nil {
		t.Fatal("expected unsupported keyword error")
	}
	assertHasFieldError(t, err, "schema.oneOf", "invalid_value")
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
	assertHasFieldError(t, err, "schema.type", "required")
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
	assertHasFieldError(t, err, "schema.additionalProperties", "invalid_type")
}

func TestValidateSchemaRejectsNonNumericMinimum(t *testing.T) {
	err := ValidateSchema(map[string]any{"type": "number", "minimum": "nope"}, "schema")
	if err == nil {
		t.Fatal("expected minimum type error")
	}
	assertHasFieldError(t, err, "schema.minimum", "invalid_type")
}

func TestValidateSchemaRejectsEmptyEnum(t *testing.T) {
	err := ValidateSchema(map[string]any{"type": "string", "enum": []any{}}, "schema")
	if err == nil {
		t.Fatal("expected empty enum error")
	}
	assertHasFieldError(t, err, "schema.enum", "invalid_value")
}

func TestValidateSchemaAcceptsNonEmptyStringEnumSlice(t *testing.T) {
	err := ValidateSchema(map[string]any{"type": "string", "enum": []string{"a", "b"}}, "schema")
	if err != nil {
		t.Fatalf("expected []string enum to validate: %v", err)
	}
}

func TestValidateValueAppliesInt64SchemaBounds(t *testing.T) {
	schema := map[string]any{"type": "number", "minimum": int64(10), "maximum": int64(20)}
	if err := ValidateValue(schema, int64(15), "v"); err != nil {
		t.Fatalf("expected int64 value in bounds: %v", err)
	}
	errBelow := ValidateValue(schema, int64(5), "v")
	if errBelow == nil {
		t.Fatal("expected below minimum")
	}
	assertHasFieldError(t, errBelow, "v", "out_of_range")
}
