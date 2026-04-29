package catalog

import (
	"fmt"
	"math"
	"regexp"
	"unicode/utf8"

	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/model"
)

var supportedSchemaKeywords = map[string]struct{}{
	"type": {}, "properties": {}, "required": {}, "additionalProperties": {}, "items": {}, "enum": {}, "minimum": {}, "maximum": {}, "minLength": {}, "maxLength": {}, "pattern": {}, "minItems": {}, "maxItems": {},
}

func asJSONObject(v any) (map[string]any, bool) {
	switch m := v.(type) {
	case map[string]any:
		return m, true
	case model.JSONMap:
		return map[string]any(m), true
	default:
		return nil, false
	}
}

// asJSONNumberAndInt returns the float64 and int64 representations of a numeric
// JSON value. isInt is true only when the value can be represented losslessly
// as int64 — int/int64 always set isInt; float64 sets isInt only when it has
// no fractional part and lies within the safe-integer range [-2^53, 2^53].
// ok is false for non-numeric inputs.
func asJSONNumberAndInt(value any) (f float64, i int64, isInt bool, ok bool) {
	switch v := value.(type) {
	case float64:
		f = v
		ok = true
		if v == math.Trunc(v) && v >= -float64(1<<53) && v <= float64(1<<53) {
			i = int64(v)
			isInt = true
		}
		return
	case int:
		return float64(v), int64(v), true, true
	case int64:
		return float64(v), v, true, true
	default:
		return 0, 0, false, false
	}
}

// compareNumericBound checks a value against a single minimum/maximum
// constraint. When both the value and the constraint can be represented as
// int64 the comparison happens in int64 to avoid float64 precision loss for
// magnitudes above 2^53. Otherwise it falls back to float64.
func compareNumericBound(rawBound any, actualF float64, actualI int64, isInt, isMax bool, path string) error {
	if isInt {
		if boundI, ok := numberToInt64(rawBound); ok {
			if isMax && actualI > boundI {
				return model.ValidationError(model.FieldError{Field: path, Code: "out_of_range", Message: "value is above maximum"})
			}
			if !isMax && actualI < boundI {
				return model.ValidationError(model.FieldError{Field: path, Code: "out_of_range", Message: "value is below minimum"})
			}
			return nil
		}
	}
	boundF, ok := numberToFloat(rawBound)
	if !ok {
		return nil
	}
	if isMax && actualF > boundF {
		return model.ValidationError(model.FieldError{Field: path, Code: "out_of_range", Message: "value is above maximum"})
	}
	if !isMax && actualF < boundF {
		return model.ValidationError(model.FieldError{Field: path, Code: "out_of_range", Message: "value is below minimum"})
	}
	return nil
}

// numberToInt64 is the int64 companion of numberToInt: full int64 range,
// rejecting float64 values that cannot be represented losslessly as int64.
func numberToInt64(value any) (int64, bool) {
	switch v := value.(type) {
	case float64:
		if v != math.Trunc(v) {
			return 0, false
		}
		if v < -float64(1<<53) || v > float64(1<<53) {
			return 0, false
		}
		return int64(v), true
	case int:
		return int64(v), true
	case int64:
		return v, true
	default:
		return 0, false
	}
}

func isJSONIntegerKeyword(v any) bool {
	switch x := v.(type) {
	case int:
		return true
	case int64:
		return true
	case float64:
		return x == math.Trunc(x) && x >= -float64(1<<53) && x <= float64(1<<53)
	default:
		return false
	}
}

func isJSONNumberKeyword(v any) bool {
	switch v.(type) {
	case int, int64, float64:
		return true
	default:
		return false
	}
}

func appendSchemaKeywordValueErrors(schema map[string]any, path string, fields *[]model.FieldError) {
	if v, ok := schema["minimum"]; ok {
		if !isJSONNumberKeyword(v) {
			*fields = append(*fields, model.FieldError{Field: path + ".minimum", Code: "invalid_type", Message: "minimum must be a number"})
		} else if !isLosslessNumericConstraint(v) {
			*fields = append(*fields, model.FieldError{Field: path + ".minimum", Code: "invalid_value", Message: "minimum exceeds lossless integer range; specify as integer"})
		}
	}
	if v, ok := schema["maximum"]; ok {
		if !isJSONNumberKeyword(v) {
			*fields = append(*fields, model.FieldError{Field: path + ".maximum", Code: "invalid_type", Message: "maximum must be a number"})
		} else if !isLosslessNumericConstraint(v) {
			*fields = append(*fields, model.FieldError{Field: path + ".maximum", Code: "invalid_value", Message: "maximum exceeds lossless integer range; specify as integer"})
		}
	}
	for _, key := range []string{"minItems", "maxItems", "minLength", "maxLength"} {
		if v, ok := schema[key]; ok && !isJSONIntegerKeyword(v) {
			*fields = append(*fields, model.FieldError{Field: path + "." + key, Code: "invalid_type", Message: key + " must be an integer"})
		}
	}
	if v, ok := schema["additionalProperties"]; ok {
		if _, ok := v.(bool); !ok {
			*fields = append(*fields, model.FieldError{Field: path + ".additionalProperties", Code: "invalid_type", Message: "additionalProperties must be a boolean"})
		}
	}
	if v, ok := schema["enum"]; ok {
		switch arr := v.(type) {
		case []any:
			if len(arr) == 0 {
				*fields = append(*fields, model.FieldError{Field: path + ".enum", Code: "invalid_value", Message: "enum must be a non-empty array"})
			}
			for _, member := range arr {
				if _, ok := member.(string); !ok {
					*fields = append(*fields, model.FieldError{Field: path + ".enum", Code: "invalid_type", Message: "enum values must be strings"})
					break
				}
			}
		case []string:
			if len(arr) == 0 {
				*fields = append(*fields, model.FieldError{Field: path + ".enum", Code: "invalid_value", Message: "enum must be a non-empty array"})
			}
		default:
			*fields = append(*fields, model.FieldError{Field: path + ".enum", Code: "invalid_type", Message: "enum must be an array"})
		}
	}
}

// isLosslessNumericConstraint reports whether a numeric schema constraint can
// be compared against int64 values without precision loss. int/int64 are always
// lossless. float64 constraints lose precision when their magnitude exceeds the
// safe-integer range; reject those at schema-validation time so authors must
// rewrite them as integers.
func isLosslessNumericConstraint(v any) bool {
	switch x := v.(type) {
	case int, int64:
		return true
	case float64:
		if x != x { // NaN
			return false
		}
		return x >= -float64(1<<53) && x <= float64(1<<53)
	default:
		return true
	}
}

func ValidateSchema(schema map[string]any, path string) error {
	fields := []model.FieldError{}
	if schema == nil {
		return model.ValidationError(model.FieldError{Field: path, Code: "required", Message: "schema is required"})
	}
	for key := range schema {
		if _, ok := supportedSchemaKeywords[key]; !ok {
			fields = append(fields, model.FieldError{Field: path + "." + key, Code: "invalid_value", Message: "unsupported schema keyword"})
		}
	}
	schemaTypeRaw, hasType := schema["type"]
	var schemaType string
	if hasType {
		var ok bool
		if schemaType, ok = schemaTypeRaw.(string); !ok {
			fields = append(fields, model.FieldError{Field: path + ".type", Code: "invalid_type", Message: "type must be a string"})
		}
	}
	hasConstraintKeywords := false
	for key := range schema {
		if key == "type" {
			continue
		}
		if _, supported := supportedSchemaKeywords[key]; supported {
			hasConstraintKeywords = true
			break
		}
	}
	if !hasType && hasConstraintKeywords {
		fields = append(fields, model.FieldError{Field: path + ".type", Code: "required", Message: "type is required when schema defines properties or constraints"})
	}
	switch schemaType {
	case "object":
		if pRaw, hasP := schema["properties"]; hasP {
			properties, ok := asJSONObject(pRaw)
			if !ok {
				fields = append(fields, model.FieldError{Field: path + ".properties", Code: "invalid_type", Message: "properties must be an object"})
			} else {
				for name, raw := range properties {
					nested, ok := asJSONObject(raw)
					if !ok {
						fields = append(fields, model.FieldError{Field: path + ".properties." + name, Code: "invalid_type", Message: "property schema must be an object"})
						continue
					}
					if err := ValidateSchema(nested, path+".properties."+name); err != nil {
						if coreErr, ok := model.IsCoreError(err); ok {
							if rawFields, ok := coreErr.Details["fields"].([]model.FieldError); ok {
								fields = append(fields, rawFields...)
							}
						}
					}
				}
			}
		}
		if items := schema["items"]; items != nil {
			fields = append(fields, model.FieldError{Field: path + ".items", Code: "invalid_value", Message: "items not allowed on object schema"})
		}
	case "array":
		if rawItems, ok := schema["items"]; ok {
			nested, ok := asJSONObject(rawItems)
			if !ok {
				fields = append(fields, model.FieldError{Field: path + ".items", Code: "invalid_type", Message: "items must be an object"})
			} else if err := ValidateSchema(nested, path+".items"); err != nil {
				if coreErr, ok := model.IsCoreError(err); ok {
					if rawFields, ok := coreErr.Details["fields"].([]model.FieldError); ok {
						fields = append(fields, rawFields...)
					}
				}
			}
		}
	case "string", "number", "integer", "boolean", "null", "":
	default:
		fields = append(fields, model.FieldError{Field: path + ".type", Code: "invalid_value", Message: "unsupported schema type"})
	}
	appendSchemaKeywordValueErrors(schema, path, &fields)
	if len(fields) > 0 {
		return model.ValidationError(fields...)
	}
	return nil
}

func ValidateValue(schema map[string]any, value any, path string) error {
	schemaType, _ := schema["type"].(string)
	switch schemaType {
	case "object":
		obj, ok := asJSONObject(value)
		if !ok {
			return model.ValidationError(model.FieldError{Field: path, Code: "invalid_type", Message: "must be an object"})
		}
		required, _ := toStringSlice(schema["required"])
		for _, field := range required {
			if _, ok := obj[field]; !ok {
				return model.ValidationError(model.FieldError{Field: path + "." + field, Code: "required", Message: "field is required"})
			}
		}
		properties, _ := asJSONObject(schema["properties"])
		additional, hasAdditional := schema["additionalProperties"].(bool)
		for key, child := range obj {
			rawSchema, ok := properties[key]
			if !ok {
				if hasAdditional && !additional {
					return model.ValidationError(model.FieldError{Field: path + "." + key, Code: "unknown_field", Message: "field is not allowed"})
				}
				continue
			}
			nested, ok := asJSONObject(rawSchema)
			if !ok {
				return model.ValidationError(model.FieldError{Field: path + "." + key, Code: "invalid_type", Message: "schema property must be object"})
			}
			if err := ValidateValue(nested, child, path+"."+key); err != nil {
				return err
			}
		}
	case "array":
		items, ok := value.([]any)
		if !ok {
			return model.ValidationError(model.FieldError{Field: path, Code: "invalid_type", Message: "must be an array"})
		}
		if minItems, ok := numberToInt(schema["minItems"]); ok && len(items) < minItems {
			return model.ValidationError(model.FieldError{Field: path, Code: "out_of_range", Message: fmt.Sprintf("must contain at least %d items", minItems)})
		}
		if maxItems, ok := numberToInt(schema["maxItems"]); ok && len(items) > maxItems {
			return model.ValidationError(model.FieldError{Field: path, Code: "out_of_range", Message: fmt.Sprintf("must contain at most %d items", maxItems)})
		}
		nested, _ := asJSONObject(schema["items"])
		for index, item := range items {
			if nested != nil {
				if err := ValidateValue(nested, item, fmt.Sprintf("%s[%d]", path, index)); err != nil {
					return err
				}
			}
		}
	case "string":
		actual, ok := value.(string)
		if !ok {
			return model.ValidationError(model.FieldError{Field: path, Code: "invalid_type", Message: "must be a string"})
		}
		length := utf8.RuneCountInString(actual)
		if minLength, ok := numberToInt(schema["minLength"]); ok && length < minLength {
			return model.ValidationError(model.FieldError{Field: path, Code: "out_of_range", Message: "string shorter than minLength"})
		}
		if maxLength, ok := numberToInt(schema["maxLength"]); ok && length > maxLength {
			return model.ValidationError(model.FieldError{Field: path, Code: "out_of_range", Message: "string longer than maxLength"})
		}
		if pattern, ok := schema["pattern"].(string); ok {
			re, err := regexp.Compile(pattern)
			if err != nil {
				return model.ValidationError(model.FieldError{Field: path, Code: "invalid_value", Message: "invalid regex pattern in schema"})
			}
			if !re.MatchString(actual) {
				return model.ValidationError(model.FieldError{Field: path, Code: "invalid_value", Message: "string does not match pattern"})
			}
		}
		if enum, ok := toStringSlice(schema["enum"]); ok && !model.ContainsString(enum, actual) {
			return model.ValidationError(model.FieldError{Field: path, Code: "invalid_value", Message: "value is not in enum"})
		}
	case "number", "integer":
		actualF, actualI, isInt, ok := asJSONNumberAndInt(value)
		if !ok {
			return model.ValidationError(model.FieldError{Field: path, Code: "invalid_type", Message: "must be numeric"})
		}
		if schemaType == "integer" && !isInt {
			return model.ValidationError(model.FieldError{Field: path, Code: "invalid_type", Message: "must be an integer"})
		}
		// Prefer int64 comparison when the value and the constraint can both be
		// represented losslessly as int64 — float64 round-trip silently mis-orders
		// values whose magnitude exceeds 2^53.
		if rawMin, present := schema["minimum"]; present {
			if err := compareNumericBound(rawMin, actualF, actualI, isInt, false, path); err != nil {
				return err
			}
		}
		if rawMax, present := schema["maximum"]; present {
			if err := compareNumericBound(rawMax, actualF, actualI, isInt, true, path); err != nil {
				return err
			}
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return model.ValidationError(model.FieldError{Field: path, Code: "invalid_type", Message: "must be a boolean"})
		}
	case "null":
		if value != nil {
			return model.ValidationError(model.FieldError{Field: path, Code: "invalid_type", Message: "must be null"})
		}
	case "":
		return nil
	default:
		return model.ValidationError(model.FieldError{Field: path, Code: "invalid_value", Message: "unsupported schema type"})
	}
	return nil
}

func toStringSlice(value any) ([]string, bool) {
	raw, ok := value.([]any)
	if !ok {
		if strings, ok := value.([]string); ok {
			return strings, true
		}
		return nil, false
	}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		str, ok := item.(string)
		if !ok {
			return nil, false
		}
		out = append(out, str)
	}
	return out, true
}

func numberToInt(value any) (int, bool) {
	i64, ok := numberToInt64(value)
	if !ok || i64 > int64(math.MaxInt) || i64 < int64(math.MinInt) {
		return 0, false
	}
	return int(i64), true
}

func numberToFloat(value any) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	default:
		return 0, false
	}
}
