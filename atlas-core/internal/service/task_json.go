package service

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/model"
)

// mergeTaskJSONForPatch merges client patch into a task. When the task is still
// pending, full top-level key replacement follows model.MergeNamedSections. After
// pending, only json.components.(progress|result|error) may be updated.
func mergeTaskJSONForPatch(current model.JSONMap, patch model.JSONMap, pending bool) (model.JSONMap, error) {
	if pending {
		if len(patch) == 0 {
			return model.CloneJSONMap(current), nil
		}
		return model.MergeNamedSections(current, model.CloneJSONMap(patch)), nil
	}
	if len(patch) == 0 {
		return model.CloneJSONMap(current), nil
	}
	return mergeTaskComponentsOnlyProgressResultError(current, patch)
}

// mergeTaskJSONForStatusTransition is used for status transitions. Only
// json.components.(progress|result|error) may be updated; command and parameters
// are never taken from the transition body.
func mergeTaskJSONForStatusTransition(current model.JSONMap, patch model.JSONMap) (model.JSONMap, error) {
	if len(patch) == 0 {
		return model.CloneJSONMap(current), nil
	}
	return mergeTaskComponentsOnlyProgressResultError(current, patch)
}

func mergeTaskComponentsOnlyProgressResultError(current, patch model.JSONMap) (model.JSONMap, error) {
	for k := range patch {
		if k != "components" {
			return model.JSONMap{}, model.ImmutableFieldError("json." + k)
		}
	}
	out := model.CloneJSONMap(current)
	// Normalize patch so nested values are map[string]any regardless of whether
	// the caller built them with map[string]any or model.JSONMap (the latter has
	// the same underlying type but is a distinct named type, so the assertion
	// below would otherwise reject a structurally valid patch).
	patch = model.CloneJSONMap(patch)
	pComp, ok := patch["components"].(map[string]any)
	if !ok {
		return out, model.ValidationError(model.FieldError{Field: "json.components", Code: "invalid_type", Message: "components must be an object"})
	}
	orig, ok := out["components"].(map[string]any)
	if !ok && out["components"] != nil {
		return out, model.ValidationError(model.FieldError{Field: "json.components", Code: "invalid_type", Message: "components must be an object"})
	}
	mergedComp, err := deepCloneStringAnyMap(orig)
	if err != nil {
		return model.JSONMap{}, model.InternalError("task json clone failed", err)
	}
	if mergedComp == nil {
		mergedComp = map[string]any{}
	}
	for k, v := range pComp {
		if k == "progress" || k == "result" || k == "error" {
			mergedComp[k] = v
			continue
		}
		return model.JSONMap{}, model.ImmutableFieldError("json.components." + k)
	}
	out["components"] = mergedComp
	return out, nil
}

func deepCloneStringAnyMap(m map[string]any) (map[string]any, error) {
	if m == nil {
		return nil, nil
	}
	raw, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("marshal json map: %w", err)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("unmarshal json map: %w", err)
	}
	return out, nil
}

// taskCommandAndParametersEqual returns true if json.components.command and
// json.components.parameters are deeply equal in both document roots. Missing
// parameters are treated as an empty object.
//
// Both inputs are normalized through CloneJSONMap so nested values are always
// map[string]any. Without this, reflect.DeepEqual would treat a model.JSONMap
// and an equivalently-shaped map[string]any as different (distinct named
// types), producing false negatives for callers that build patches
// programmatically.
func taskCommandAndParametersEqual(a, b model.JSONMap) bool {
	a = model.CloneJSONMap(a)
	b = model.CloneJSONMap(b)
	c1, _ := a["components"].(map[string]any)
	c2, _ := b["components"].(map[string]any)
	if c1 == nil && c2 == nil {
		return true
	}
	if c1 == nil || c2 == nil {
		return false
	}
	return reflect.DeepEqual(c1["command"], c2["command"]) && parametersEqual(c1["parameters"], c2["parameters"])
}

func parametersEqual(p1, p2 any) bool {
	if p1 == nil {
		p1 = map[string]any{}
	}
	if p2 == nil {
		p2 = map[string]any{}
	}
	return reflect.DeepEqual(p1, p2)
}
