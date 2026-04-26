package catalog

import "testing"

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
