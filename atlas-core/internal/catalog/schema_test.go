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

// Issue #6: a non-array enum (e.g., a bare string) should report invalid_type,
// not invalid_value, matching the convention used for minimum/maximum/etc.
func TestValidateSchemaEnumNonArrayReportsInvalidType(t *testing.T) {
	err := ValidateSchema(map[string]any{"type": "string", "enum": "not-an-array"}, "schema")
	if err == nil {
		t.Fatal("expected non-array enum to be rejected")
	}
	assertHasFieldError(t, err, "schema.enum", "invalid_type")
}

// Issue #7: int64 values just above 2^53 must be compared in int64. Float64
// coercion would round the value down to the maximum and silently accept it.
func TestValidateValueInt64BeyondSafeIntegerRange(t *testing.T) {
	const maxSafe = int64(1) << 53 // 9_007_199_254_740_992
	schema := map[string]any{"type": "integer", "maximum": maxSafe}
	if err := ValidateValue(schema, maxSafe+1, "v"); err == nil {
		t.Fatal("expected value above maximum (2^53 + 1) to be rejected")
	} else {
		assertHasFieldError(t, err, "v", "out_of_range")
	}
	if err := ValidateValue(schema, maxSafe, "v"); err != nil {
		t.Fatalf("expected boundary value (2^53) to be accepted: %v", err)
	}
}

// Issue #7: int64 boundaries set as min == max must include only the exact value.
func TestValidateValueInt64EqualMinMax(t *testing.T) {
	pinned := int64(1) << 60
	schema := map[string]any{"type": "integer", "minimum": pinned, "maximum": pinned}
	if err := ValidateValue(schema, pinned, "v"); err != nil {
		t.Fatalf("expected pinned int64 value to validate: %v", err)
	}
	if err := ValidateValue(schema, pinned-1, "v"); err == nil {
		t.Fatal("expected pinned-1 to be rejected as below minimum")
	}
	if err := ValidateValue(schema, pinned+1, "v"); err == nil {
		t.Fatal("expected pinned+1 to be rejected as above maximum")
	}
}

// Issue #7: float64 schema constraints whose magnitude exceeds the safe-int
// range silently lose precision and must be rejected at schema-validation
// time so authors specify them as integers.
func TestValidateSchemaRejectsLossyFloatConstraint(t *testing.T) {
	err := ValidateSchema(map[string]any{"type": "integer", "minimum": 1e20}, "schema")
	if err == nil {
		t.Fatal("expected float64 minimum past safe-int range to be rejected")
	}
	assertHasFieldError(t, err, "schema.minimum", "invalid_value")
}

func TestValidateSchemaRejectsNegativeIntegerBounds(t *testing.T) {
	for _, key := range []string{"minItems", "maxItems", "minLength", "maxLength"} {
		t.Run(key, func(t *testing.T) {
			err := ValidateSchema(map[string]any{"type": "string", key: -1}, "schema")
			if err == nil {
				t.Fatalf("expected %s to reject negative values", key)
			}
			assertHasFieldError(t, err, "schema."+key, "invalid_value")
		})
	}
}

func TestValidateSchemaRejectsNonStringRequiredEntries(t *testing.T) {
	err := ValidateSchema(map[string]any{"type": "object", "required": []any{"ok", 1}}, "schema")
	if err == nil {
		t.Fatal("expected required typing error")
	}
	assertHasFieldError(t, err, "schema.required", "invalid_type")
}

func TestValidateSchemaRejectsItemsOnObjectWhenNull(t *testing.T) {
	err := ValidateSchema(map[string]any{"type": "object", "items": nil}, "schema")
	if err == nil {
		t.Fatal("expected items-on-object error")
	}
	assertHasFieldError(t, err, "schema.items", "invalid_value")
}
