package catalog

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/model"
	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/store"
)

type Command struct {
	Type             string         `json:"type"`
	DisplayName      string         `json:"display_name"`
	Description      string         `json:"description"`
	ParametersSchema map[string]any `json:"parameters_schema"`
}

type Catalog struct {
	CatalogID   string             `json:"catalog_id"`
	Version     string             `json:"version"`
	AuthoredAt  string             `json:"authored_at,omitempty"`
	Description string             `json:"description,omitempty"`
	Metadata    map[string]any     `json:"metadata,omitempty"`
	Commands    []Command          `json:"commands"`
	Raw         []byte             `json:"-"`
	ByType      map[string]Command `json:"-"`
	ContentHash string             `json:"-"`
	ObjectID    string             `json:"-"`
}

func cloneJSONValue(v any) any {
	switch x := v.(type) {
	case map[string]any:
		return cloneJSONMap(x)
	case []any:
		out := make([]any, len(x))
		for i, e := range x {
			out[i] = cloneJSONValue(e)
		}
		return out
	default:
		return v
	}
}

func cloneJSONMap(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = cloneJSONValue(v)
	}
	return out
}

func (c Catalog) Clone() Catalog {
	out := c
	if c.Commands != nil {
		out.Commands = make([]Command, len(c.Commands))
		copy(out.Commands, c.Commands)
		for i := range out.Commands {
			out.Commands[i].ParametersSchema = cloneJSONMap(out.Commands[i].ParametersSchema)
		}
	}
	if c.Metadata != nil {
		out.Metadata = make(map[string]any, len(c.Metadata))
		for k, v := range c.Metadata {
			out.Metadata[k] = cloneJSONValue(v)
		}
	}
	if c.Raw != nil {
		out.Raw = make([]byte, len(c.Raw))
		copy(out.Raw, c.Raw)
	}
	if c.ByType != nil {
		out.ByType = make(map[string]Command, len(c.ByType))
		for k, v := range c.ByType {
			cmd := v
			cmd.ParametersSchema = cloneJSONMap(v.ParametersSchema)
			out.ByType[k] = cmd
		}
	}
	return out
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
	catalog.ContentHash = ContentHashOfBytes(raw)
	catalog.ObjectID = ObjectIDFromContentHash(catalog.ContentHash)
	catalog.ByType = map[string]Command{}
	for _, command := range catalog.Commands {
		catalog.ByType[command.Type] = command
	}
	return catalog, nil
}

// ContentHashOfBytes returns the first 8 bytes of the SHA-256 of raw as hex, matching [Load] and [ObjectIDFromContentBytes].
func ContentHashOfBytes(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:8])
}

// ObjectIDFromContentHash returns the object_id command-catalog-<8-hex> for a content hash.
func ObjectIDFromContentHash(hash string) string {
	return fmt.Sprintf("command-catalog-%s", hash)
}

// ObjectIDFromContentBytes returns the object_id command-catalog-<8-hex> for a catalog JSON body, matching [Load].
func ObjectIDFromContentBytes(raw []byte) string {
	return ObjectIDFromContentHash(ContentHashOfBytes(raw))
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
			appended := false
			if coreErr, ok := model.IsCoreError(err); ok {
				if raw, ok := coreErr.Details["fields"].([]model.FieldError); ok {
					fields = append(fields, raw...)
					appended = true
				}
			}
			if !appended {
				fields = append(fields, model.FieldError{Field: prefix + ".parameters_schema", Code: "invalid_value", Message: err.Error()})
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
	a.catalog = catalog.Clone()
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
	if !a.ready {
		return Catalog{}, false
	}
	return a.catalog.Clone(), true
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
			if err := verifyMaterializedCatalogFile(ctx, stores, catalog); err != nil {
				return Catalog{}, err
			}
			needUpload = false
			break
		}
	}
	if needUpload {
		input := store.ObjectUploadInput{
			File: model.ObjectFile{
				FileID:      "catalog-json",
				ObjectID:    catalog.ObjectID,
				ContentType: "application/json",
			},
			Reader:   bytes.NewReader(catalog.Raw),
			MaxBytes: int64(len(catalog.Raw)) + 1,
		}
		if _, err := stores.CreateObjectFile(ctx, input); err != nil {
			if coreErr, ok := model.IsCoreError(err); !ok || coreErr.ErrorCode != "conflict" {
				return Catalog{}, err
			}
		}
	}
	return catalog, nil
}

func verifyMaterializedCatalogFile(ctx context.Context, stores store.ObjectStore, catalog Catalog) error {
	meta, rc, err := stores.OpenObjectFile(ctx, catalog.ObjectID, "catalog-json")
	if err != nil {
		return err
	}
	defer rc.Close()
	if meta.SizeBytes != int64(len(catalog.Raw)) {
		return model.CatalogUnavailable("materialized command catalog does not match checked-in catalog bytes", nil)
	}
	raw, err := io.ReadAll(io.LimitReader(rc, meta.SizeBytes+1))
	if err != nil {
		return err
	}
	if !bytes.Equal(raw, catalog.Raw) {
		return model.CatalogUnavailable("materialized command catalog does not match checked-in catalog bytes", nil)
	}
	return nil
}

func ValidateParameters(command Command, parameters any) error {
	return ValidateValue(command.ParametersSchema, parameters, "parameters")
}

func SortedCommands(catalog Catalog) []Command {
	commands := append([]Command(nil), catalog.Commands...)
	sort.Slice(commands, func(i, j int) bool { return commands[i].Type < commands[j].Type })
	return commands
}
