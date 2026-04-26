package catalog

import (
    "context"
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "os"
    "sort"
    "strings"
    "sync"
    "time"

    "github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/model"
    "github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/store"
)

type Command struct {
    Type            string         `json:"type"`
    DisplayName     string         `json:"display_name"`
    Description     string         `json:"description"`
    ParametersSchema map[string]any `json:"parameters_schema"`
}

type Catalog struct {
    CatalogID   string         `json:"catalog_id"`
    Version     string         `json:"version"`
    AuthoredAt  string         `json:"authored_at,omitempty"`
    Description string         `json:"description,omitempty"`
    Metadata    map[string]any `json:"metadata,omitempty"`
    Commands    []Command      `json:"commands"`
    Raw         []byte         `json:"-"`
    ByType      map[string]Command `json:"-"`
    ContentHash string         `json:"-"`
    ObjectID    string         `json:"-"`
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
    var catalog Catalog
    if err := json.Unmarshal(raw, &catalog); err != nil {
        return Catalog{}, err
    }
    catalog.Raw = raw
    if err := Validate(catalog); err != nil {
        return Catalog{}, err
    }
    sum := sha256.Sum256(raw)
    catalog.ContentHash = hex.EncodeToString(sum[:8])
    catalog.ObjectID = fmt.Sprintf("command-catalog-%s", catalog.ContentHash)
    catalog.ByType = map[string]Command{}
    for _, command := range catalog.Commands {
        catalog.ByType[command.Type] = command
    }
    return catalog, nil
}

func Validate(catalog Catalog) error {
    fields := []model.FieldError{}
    if catalog.CatalogID == "" {
        fields = append(fields, model.FieldError{Field: "catalog_id", Code: "required", Message: "catalog_id is required"})
    }
    if catalog.Version == "" {
        fields = append(fields, model.FieldError{Field: "version", Code: "required", Message: "version is required"})
    }
    seen := map[string]struct{}{}
    for idx, command := range catalog.Commands {
        prefix := fmt.Sprintf("commands[%d]", idx)
        if err := model.ValidateSnakeCase(prefix+".type", command.Type); err != nil {
            fields = append(fields, *err)
        }
        if _, exists := seen[command.Type]; exists {
            fields = append(fields, model.FieldError{Field: prefix + ".type", Code: "conflict", Message: "duplicate command type"})
        }
        seen[command.Type] = struct{}{}
        if command.DisplayName == "" {
            fields = append(fields, model.FieldError{Field: prefix + ".display_name", Code: "required", Message: "display_name is required"})
        }
        if command.Description == "" {
            fields = append(fields, model.FieldError{Field: prefix + ".description", Code: "required", Message: "description is required"})
        }
        if err := ValidateSchema(command.ParametersSchema, prefix+".parameters_schema"); err != nil {
            if coreErr, ok := model.IsCoreError(err); ok {
                if raw, ok := coreErr.Details["fields"].([]model.FieldError); ok {
                    fields = append(fields, raw...)
                }
            }
        }
    }
    if len(fields) > 0 {
        return model.CatalogUnavailable("command catalog validation failed", map[string]any{"fields": fields})
    }
    return nil
}

func (a *Active) Set(catalog Catalog) {
    a.mu.Lock()
    defer a.mu.Unlock()
    a.catalog = catalog
    a.ready = true
}

func (a *Active) Clear() {
    a.mu.Lock()
    defer a.mu.Unlock()
    a.ready = false
    a.catalog = Catalog{}
}

func (a *Active) Ready() bool {
    a.mu.RLock()
    defer a.mu.RUnlock()
    return a.ready
}

func (a *Active) Get() (Catalog, bool) {
    a.mu.RLock()
    defer a.mu.RUnlock()
    return a.catalog, a.ready
}

func Materialize(ctx context.Context, stores store.ObjectStore, catalog Catalog) (Catalog, error) {
    objectJSON := model.JSONMap{"catalog_id": catalog.CatalogID, "version": catalog.Version}
    now := time.Now().UTC()
    object := model.Object{ObjectID: catalog.ObjectID, Type: "command_catalog", OwnerType: "system", OwnerID: "active_command_catalog", JSON: objectJSON, CreatedAt: now, UpdatedAt: now}
    if _, err := stores.GetObject(ctx, catalog.ObjectID); err != nil {
        if coreErr, ok := model.IsCoreError(err); ok && coreErr.ErrorCode == "not_found" {
            if _, err := stores.CreateObject(ctx, object); err != nil {
                return Catalog{}, err
            }
        } else {
            return Catalog{}, err
        }
    }
    files, err := stores.ListObjectFilesForObject(ctx, catalog.ObjectID)
    if err != nil {
        return Catalog{}, err
    }
    needUpload := true
    for _, file := range files {
        if file.FileID == "catalog-json" {
            needUpload = false
            break
        }
    }
    if needUpload {
        if _, err := stores.CreateObjectFile(ctx, store.ObjectUploadInput{File: model.ObjectFile{FileID: "catalog-json", ObjectID: catalog.ObjectID, ContentType: "application/json"}, Reader: strings.NewReader(string(catalog.Raw)), MaxBytes: int64(len(catalog.Raw)) + 1}); err != nil {
            if coreErr, ok := model.IsCoreError(err); !ok || coreErr.ErrorCode != "conflict" {
                return Catalog{}, err
            }
        }
    }
    return catalog, nil
}

func ValidateParameters(command Command, parameters any) error {
    return ValidateValue(command.ParametersSchema, parameters, "parameters")
}

func SortedCommands(catalog Catalog) []Command {
    commands := append([]Command(nil), catalog.Commands...)
    sort.Slice(commands, func(i, j int) bool { return commands[i].Type < commands[j].Type })
    return commands
}
