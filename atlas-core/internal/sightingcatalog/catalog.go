package sightingcatalog

import (
    "encoding/json"
    "fmt"
    "os"
    "sync"

    "github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/catalog"
    "github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/model"
)

type Kind struct {
    Kind        string         `json:"kind"`
    DisplayName string         `json:"display_name"`
    Description string         `json:"description"`
    DataSchema  map[string]any `json:"data_schema"`
}

type Catalog struct {
    CatalogID     string         `json:"catalog_id"`
    SightingKinds []Kind         `json:"sighting_kinds"`
    ByKind        map[string]Kind `json:"-"`
}

type Active struct {
    mu      sync.RWMutex
    catalog Catalog
    ready   bool
}

func Load(path string) (Catalog, error) {
    raw, err := os.ReadFile(path)
    if err != nil {
        return Catalog{}, err
    }
    var catalogDoc Catalog
    if err := json.Unmarshal(raw, &catalogDoc); err != nil {
        return Catalog{}, err
    }
    if err := Validate(catalogDoc); err != nil {
        return Catalog{}, err
    }
    catalogDoc.ByKind = map[string]Kind{}
    for _, item := range catalogDoc.SightingKinds {
        catalogDoc.ByKind[item.Kind] = item
    }
    return catalogDoc, nil
}

func Validate(c Catalog) error {
    fields := []model.FieldError{}
    seen := map[string]struct{}{}
    for idx, kind := range c.SightingKinds {
        prefix := fmt.Sprintf("sighting_kinds[%d]", idx)
        if err := model.ValidateSnakeCase(prefix+".kind", kind.Kind); err != nil {
            fields = append(fields, *err)
        }
        if _, ok := seen[kind.Kind]; ok {
            fields = append(fields, model.FieldError{Field: prefix + ".kind", Code: "conflict", Message: "duplicate kind"})
        }
        seen[kind.Kind] = struct{}{}
        if err := catalog.ValidateSchema(kind.DataSchema, prefix+".data_schema"); err != nil {
            if coreErr, ok := model.IsCoreError(err); ok {
                if rawFields, ok := coreErr.Details["fields"].([]model.FieldError); ok {
                    fields = append(fields, rawFields...)
                }
            }
        }
    }
    if len(fields) > 0 {
        return model.CatalogUnavailable("sighting catalog validation failed", map[string]any{"fields": fields})
    }
    return nil
}

func (a *Active) Set(c Catalog) {
    a.mu.Lock()
    defer a.mu.Unlock()
    a.catalog = c
    a.ready = true
}

func (a *Active) Get() (Catalog, bool) {
    a.mu.RLock()
    defer a.mu.RUnlock()
    return a.catalog, a.ready
}

func ValidateSighting(active Catalog, sighting any) error {
    sightingMap, ok := sighting.(map[string]any)
    if !ok {
        return model.ValidationError(model.FieldError{Field: "json.latest_sighting", Code: "invalid_type", Message: "sighting must be an object"})
    }
    observedAt, _ := sightingMap["observed_at"].(string)
    if err := model.ValidateRFC3339Field("json.latest_sighting.observed_at", observedAt); err != nil {
        return model.ValidationError(*err)
    }
    if inaccuracy, ok := sightingMap["observed_at_inaccuracy_ms"].(string); ok {
        if err := model.ValidateInaccuracyString("json.latest_sighting.observed_at_inaccuracy_ms", inaccuracy); err != nil {
            return model.ValidationError(*err)
        }
    }
    kind, _ := sightingMap["kind"].(string)
    if kind == "" {
        return model.ValidationError(model.FieldError{Field: "json.latest_sighting.kind", Code: "required", Message: "kind is required"})
    }
    def, ok := active.ByKind[kind]
    if !ok {
        return model.ValidationError(model.FieldError{Field: "json.latest_sighting.kind", Code: "invalid_value", Message: "unknown sighting kind"})
    }
    data := sightingMap["data"]
    if err := catalog.ValidateValue(def.DataSchema, data, "json.latest_sighting.data"); err != nil {
        return err
    }
    return nil
}
