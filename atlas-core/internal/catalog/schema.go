package catalog

import (
	"fmt"
	"regexp"
	"unicode/utf8"

	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/model"
)

var supportedSchemaKeywords = map[string]struct{}{
	"type": {}, "properties": {}, "required": {}, "additionalProperties": {}, "items": {}, "enum": {}, "minimum": {}, "maximum": {}, "minLength": {}, "maxLength": {}, "pattern": {}, "minItems": {}, "maxItems": {},
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
	switch schemaType {
	case "object":
		if properties, ok := schema["properties"].(map[string]any); ok {
			for name, raw := range properties {
				nested, ok := raw.(map[string]any)
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
		if items := schema["items"]; items != nil {
			fields = append(fields, model.FieldError{Field: path + ".items", Code: "invalid_value", Message: "items not allowed on object schema"})
		}
	case "array":
		if rawItems, ok := schema["items"]; ok {
			nested, ok := rawItems.(map[string]any)
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
	if len(fields) > 0 {
		return model.ValidationError(fields...)
	}
	return nil
}

func ValidateValue(schema map[string]any, value any, path string) error {
	schemaType, _ := schema["type"].(string)
	switch schemaType {
	case "object":
		obj, ok := value.(map[string]any)
		if !ok {
			return model.ValidationError(model.FieldError{Field: path, Code: "invalid_type", Message: "must be an object"})
		}
		required, _ := toStringSlice(schema["required"])
		for _, field := range required {
			if _, ok := obj[field]; !ok {
				return model.ValidationError(model.FieldError{Field: path + "." + field, Code: "required", Message: "field is required"})
			}
		}
		properties, _ := schema["properties"].(map[string]any)
		additional, hasAdditional := schema["additionalProperties"].(bool)
		for key, child := range obj {
			rawSchema, ok := properties[key]
			if !ok {
				if hasAdditional && !additional {
					return model.ValidationError(model.FieldError{Field: path + "." + key, Code: "unknown_field", Message: "field is not allowed"})
				}
				continue
			}
			nested, ok := rawSchema.(map[string]any)
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
		nested, _ := schema["items"].(map[string]any)
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
		actual, ok := value.(float64)
		if !ok {
			intValue, ok := value.(int)
			if !ok {
				return model.ValidationError(model.FieldError{Field: path, Code: "invalid_type", Message: "must be numeric"})
			}
			actual = float64(intValue)
		}
		if schemaType == "integer" && actual != float64(int64(actual)) {
			return model.ValidationError(model.FieldError{Field: path, Code: "invalid_type", Message: "must be an integer"})
		}
		if min, ok := numberToFloat(schema["minimum"]); ok && actual < min {
			return model.ValidationError(model.FieldError{Field: path, Code: "out_of_range", Message: "value is below minimum"})
		}
		if max, ok := numberToFloat(schema["maximum"]); ok && actual > max {
			return model.ValidationError(model.FieldError{Field: path, Code: "out_of_range", Message: "value is above maximum"})
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
	switch v := value.(type) {
	case float64:
		if v == float64(int(v)) {
			return int(v), true
		}
		return 0, false
	case int:
		return v, true
	default:
		return 0, false
	}
}

func numberToFloat(value any) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case int:
		return float64(v), true
	default:
		return 0, false
	}
}
