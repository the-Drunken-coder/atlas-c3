package service

import (
	"encoding/json"
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
		return model.MergeNamedSections(current, patch), nil
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
	pComp, ok := patch["components"].(map[string]any)
	if !ok {
		return out, model.ValidationError(model.FieldError{Field: "json.components", Code: "invalid_type", Message: "components must be an object"})
	}
	orig, _ := out["components"].(map[string]any)
	mergedComp := deepCloneStringAnyMap(orig)
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

func deepCloneStringAnyMap(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	raw, err := json.Marshal(m)
	if err != nil {
		return nil
	}
	var out map[string]any
	_ = json.Unmarshal(raw, &out)
	if out == nil {
		return map[string]any{}
	}
	return out
}

// taskCommandAndParametersEqual returns true if json.components.command and
// json.components.parameters are deeply equal in both document roots. Missing
// parameters are treated as an empty object.
func taskCommandAndParametersEqual(a, b model.JSONMap) bool {
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
