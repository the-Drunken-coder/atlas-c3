package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/catalog"
	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/events"
	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/model"
	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/sightingcatalog"
	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/store"
)

type Services struct {
	stores    store.Stores
	commands  *catalog.Active
	sightings *sightingcatalog.Active
	publisher *events.Hub
	now       func() time.Time
}

type EntityCreateInput struct {
	EntityID string
	Type     string
	Subtype  string
	Alias    string
	JSON     model.JSONMap
}

type EntityPatchInput struct {
	Subtype *string
	Alias   *string
	JSON    model.JSONMap
}

type ObservationCreateInput struct {
	ObservationID string
	SourceAssetID string
	JSON          model.JSONMap
}

type ObservationPatchInput struct {
	JSON model.JSONMap
}

type TaskCreateInput struct {
	TaskID  string
	AssetID string
	JSON    model.JSONMap
}

type TaskPatchInput struct {
	JSON model.JSONMap
}

type TaskStatusInput struct {
	Status string
	JSON   model.JSONMap
}

type ObjectCreateInput struct {
	ObjectID  string
	Type      string
	OwnerType string
	OwnerID   string
	JSON      model.JSONMap
}

type ObjectPatchInput struct {
	Type *string
	JSON model.JSONMap
}

func New(stores store.Stores, commands *catalog.Active, sightings *sightingcatalog.Active, publisher *events.Hub) *Services {
	return &Services{stores: stores, commands: commands, sightings: sightings, publisher: publisher, now: func() time.Time { return time.Now().UTC() }}
}

func (s *Services) CreateEntity(ctx context.Context, input EntityCreateInput) (model.Entity, error) {
	if err := validateEntityInput(input.Type, input.JSON, true); err != nil {
		return model.Entity{}, err
	}
	if err := requireValidID("entity_id", input.EntityID); err != nil {
		return model.Entity{}, err
	}
	entity := model.Entity{EntityID: input.EntityID, Type: input.Type, Subtype: input.Subtype, Alias: input.Alias, JSON: model.NormalizeJSONMap(input.JSON), CreatedAt: s.now(), UpdatedAt: s.now()}
	created, err := s.stores.CreateEntity(ctx, entity)
	if err != nil {
		return model.Entity{}, err
	}
	s.publish(ctx, "entity", "created", created.EntityID, created)
	return created, nil
}

func (s *Services) GetEntity(ctx context.Context, id string) (model.Entity, error) {
	return s.stores.GetEntity(ctx, id)
}

func (s *Services) ListEntities(ctx context.Context, filter store.EntityListFilter, pagination model.Pagination) ([]model.Entity, int, error) {
	return s.stores.ListEntities(ctx, filter, pagination)
}

func (s *Services) PatchEntity(ctx context.Context, id string, patch EntityPatchInput) (model.Entity, error) {
	current, err := s.stores.GetEntity(ctx, id)
	if err != nil {
		return model.Entity{}, err
	}
	if patch.Subtype != nil {
		current.Subtype = *patch.Subtype
	}
	if patch.Alias != nil {
		current.Alias = *patch.Alias
	}
	current.JSON = model.MergeNamedSections(current.JSON, patch.JSON)
	if err := validateEntityInput(current.Type, current.JSON, false); err != nil {
		return model.Entity{}, err
	}
	current.UpdatedAt = s.now()
	updated, err := s.stores.UpdateEntity(ctx, current)
	if err != nil {
		return model.Entity{}, err
	}
	s.publish(ctx, "entity", "updated", updated.EntityID, updated)
	return updated, nil
}

func (s *Services) DeleteEntity(ctx context.Context, id string) error {
	dependents, err := s.stores.CountEntityDependents(ctx, id)
	if err != nil {
		return err
	}
	if dependents["tasks"] > 0 || dependents["observations"] > 0 || dependents["objects"] > 0 {
		return model.Conflict("entity", id, "dependent_records", map[string]any{"dependents": dependents})
	}
	if err := s.stores.DeleteEntity(ctx, id); err != nil {
		return err
	}
	s.publish(ctx, "entity", "deleted", id, model.DeleteEventData{DeletedAt: s.now()})
	return nil
}

func (s *Services) ListEntityTasks(ctx context.Context, entityID string, pagination model.Pagination) ([]model.Task, int, error) {
	entity, err := s.stores.GetEntity(ctx, entityID)
	if err != nil {
		return nil, 0, err
	}
	if entity.Type != "asset" {
		return nil, 0, model.ValidationError(model.FieldError{Field: "entity_id", Code: "invalid_value", Message: "entity must be an asset"})
	}
	return s.stores.ListTasksForAsset(ctx, entityID, pagination)
}

func (s *Services) CreateObservation(ctx context.Context, input ObservationCreateInput) (model.Observation, error) {
	if err := requireValidID("observation_id", input.ObservationID); err != nil {
		return model.Observation{}, err
	}
	asset, err := s.stores.GetEntity(ctx, input.SourceAssetID)
	if err != nil {
		return model.Observation{}, err
	}
	if asset.Type != "asset" {
		return model.Observation{}, model.ValidationError(model.FieldError{Field: "source_asset_id", Code: "invalid_value", Message: "source_asset_id must reference an asset"})
	}
	if err := validateObservationJSON(ctx, s.stores, s.sightings, input.ObservationID, input.JSON); err != nil {
		return model.Observation{}, err
	}
	observation := model.Observation{ObservationID: input.ObservationID, SourceAssetID: input.SourceAssetID, JSON: model.NormalizeJSONMap(input.JSON), CreatedAt: s.now(), UpdatedAt: s.now()}
	created, err := s.stores.CreateObservation(ctx, observation)
	if err != nil {
		return model.Observation{}, err
	}
	s.publish(ctx, "observation", "created", created.ObservationID, created)
	return created, nil
}

func (s *Services) GetObservation(ctx context.Context, id string) (model.Observation, error) {
	return s.stores.GetObservation(ctx, id)
}

func (s *Services) ListObservations(ctx context.Context, filter store.ObservationListFilter, pagination model.Pagination) ([]model.Observation, int, error) {
	return s.stores.ListObservations(ctx, filter, pagination)
}

func (s *Services) PatchObservation(ctx context.Context, id string, patch ObservationPatchInput) (model.Observation, error) {
	current, err := s.stores.GetObservation(ctx, id)
	if err != nil {
		return model.Observation{}, err
	}
	current.JSON = model.MergeNamedSections(current.JSON, patch.JSON)
	if err := validateObservationJSON(ctx, s.stores, s.sightings, current.ObservationID, current.JSON); err != nil {
		return model.Observation{}, err
	}
	current.UpdatedAt = s.now()
	updated, err := s.stores.UpdateObservation(ctx, current)
	if err != nil {
		return model.Observation{}, err
	}
	s.publish(ctx, "observation", "updated", updated.ObservationID, updated)
	return updated, nil
}

func (s *Services) DeleteObservation(ctx context.Context, id string) error {
	if err := s.stores.DeleteObservation(ctx, id); err != nil {
		return err
	}
	s.publish(ctx, "observation", "deleted", id, model.DeleteEventData{DeletedAt: s.now()})
	return nil
}

func (s *Services) CreateTask(ctx context.Context, input TaskCreateInput) (model.Task, error) {
	if err := requireValidID("task_id", input.TaskID); err != nil {
		return model.Task{}, err
	}
	asset, err := s.stores.GetEntity(ctx, input.AssetID)
	if err != nil {
		return model.Task{}, err
	}
	if asset.Type != "asset" {
		return model.Task{}, model.ValidationError(model.FieldError{Field: "asset_id", Code: "invalid_value", Message: "asset_id must reference an asset"})
	}
	activeCatalog, ok := s.commands.Get()
	if !ok {
		return model.Task{}, model.CatalogUnavailable("active command catalog is unavailable", nil)
	}
	if err := validateTaskCommand(input.JSON, activeCatalog, asset); err != nil {
		return model.Task{}, err
	}
	task := model.Task{TaskID: input.TaskID, Status: "pending", AssetID: input.AssetID, CommandCatalogObjectID: activeCatalog.ObjectID, JSON: model.NormalizeJSONMap(input.JSON), CreatedAt: s.now(), UpdatedAt: s.now()}
	created, err := s.stores.CreateTask(ctx, task)
	if err != nil {
		return model.Task{}, err
	}
	s.publish(ctx, "task", "created", created.TaskID, created)
	return created, nil
}

func (s *Services) GetTask(ctx context.Context, id string) (model.Task, error) {
	return s.stores.GetTask(ctx, id)
}

func (s *Services) ListTasks(ctx context.Context, filter store.TaskListFilter, pagination model.Pagination) ([]model.Task, int, error) {
	return s.stores.ListTasks(ctx, filter, pagination)
}

func (s *Services) PatchTask(ctx context.Context, id string, patch TaskPatchInput) (model.Task, error) {
	current, err := s.stores.GetTask(ctx, id)
	if err != nil {
		return model.Task{}, err
	}
	var nextJSON model.JSONMap
	var mErr error
	if current.Status == "pending" {
		if patch.JSON != nil {
			nextJSON, mErr = mergeTaskJSONForPatch(current.JSON, patch.JSON, true)
		} else {
			nextJSON = model.CloneJSONMap(current.JSON)
		}
	} else {
		if patch.JSON != nil {
			nextJSON, mErr = mergeTaskJSONForPatch(current.JSON, patch.JSON, false)
		} else {
			nextJSON = model.CloneJSONMap(current.JSON)
		}
	}
	if mErr != nil {
		return model.Task{}, mErr
	}
	if current.Status != "pending" && !taskCommandAndParametersEqual(current.JSON, nextJSON) {
		return model.Task{}, model.ImmutableFieldsError("json.components.command", "json.components.parameters")
	}
	cat, err := s.loadPinnedCatalog(ctx, current.CommandCatalogObjectID)
	if err != nil {
		return model.Task{}, err
	}
	if current.Status == "pending" {
		asset, err := s.stores.GetEntity(ctx, current.AssetID)
		if err != nil {
			return model.Task{}, err
		}
		if err := validateTaskCommand(nextJSON, cat, asset); err != nil {
			return model.Task{}, err
		}
	} else if _, err := validateTaskCommandCatalog(nextJSON, cat); err != nil {
		return model.Task{}, err
	}
	current.JSON = nextJSON
	current.UpdatedAt = s.now()
	updated, err := s.stores.UpdateTask(ctx, current)
	if err != nil {
		return model.Task{}, err
	}
	s.publish(ctx, "task", "updated", updated.TaskID, updated)
	return updated, nil
}

func (s *Services) TransitionTaskStatus(ctx context.Context, id string, input TaskStatusInput) (model.Task, error) {
	task, err := s.stores.GetTask(ctx, id)
	if err != nil {
		return model.Task{}, err
	}
	if task.Status == input.Status {
		return task, nil
	}
	if !allowedTaskTransition(task.Status, input.Status) {
		return model.Task{}, model.InvalidStatusTransition(task.Status, input.Status)
	}
	merged, mErr := mergeTaskJSONForStatusTransition(task.JSON, input.JSON)
	if mErr != nil {
		return model.Task{}, mErr
	}
	if !taskCommandAndParametersEqual(task.JSON, merged) {
		return model.Task{}, model.ImmutableFieldsError("json.components.command", "json.components.parameters")
	}
	cat, err := s.loadPinnedCatalog(ctx, task.CommandCatalogObjectID)
	if err != nil {
		return model.Task{}, err
	}
	if _, err := validateTaskCommandCatalog(merged, cat); err != nil {
		return model.Task{}, err
	}
	task.Status = input.Status
	task.JSON = merged
	task.UpdatedAt = s.now()
	updated, err := s.stores.UpdateTask(ctx, task)
	if err != nil {
		return model.Task{}, err
	}
	s.publish(ctx, "task", "updated", updated.TaskID, updated)
	return updated, nil
}

func (s *Services) DeleteTask(ctx context.Context, id string) error {
	count, err := s.stores.CountTaskOwnedObjects(ctx, id)
	if err != nil {
		return err
	}
	if count > 0 {
		return model.Conflict("task", id, "dependent_records", map[string]any{"objects": count})
	}
	if err := s.stores.DeleteTask(ctx, id); err != nil {
		return err
	}
	s.publish(ctx, "task", "deleted", id, model.DeleteEventData{DeletedAt: s.now()})
	return nil
}

func (s *Services) CreateObject(ctx context.Context, input ObjectCreateInput) (model.Object, error) {
	if err := requireValidID("object_id", input.ObjectID); err != nil {
		return model.Object{}, err
	}
	objectType := strings.TrimSpace(input.Type)
	if err := validateObjectInput(ctx, s.stores, input.ObjectID, objectType, input.OwnerType, input.OwnerID, input.JSON); err != nil {
		return model.Object{}, err
	}
	object := model.Object{ObjectID: input.ObjectID, Type: objectType, OwnerType: input.OwnerType, OwnerID: input.OwnerID, JSON: model.NormalizeJSONMap(input.JSON), CreatedAt: s.now(), UpdatedAt: s.now()}
	created, err := s.stores.CreateObject(ctx, object)
	if err != nil {
		return model.Object{}, err
	}
	s.publishObject(ctx, "created", created.ObjectID, created)
	return created, nil
}

func (s *Services) GetObject(ctx context.Context, id string) (model.Object, error) {
	return s.stores.GetObject(ctx, id)
}

func (s *Services) ListObjects(ctx context.Context, filter store.ObjectListFilter, pagination model.Pagination) ([]model.Object, int, error) {
	return s.stores.ListObjects(ctx, filter, pagination)
}

func (s *Services) PatchObject(ctx context.Context, id string, patch ObjectPatchInput) (model.Object, error) {
	object, err := s.stores.GetObject(ctx, id)
	if err != nil {
		return model.Object{}, err
	}
	if patch.Type != nil {
		object.Type = strings.TrimSpace(*patch.Type)
	}
	object.JSON = model.MergeNamedSections(object.JSON, patch.JSON)
	if err := validateObjectInput(ctx, s.stores, object.ObjectID, object.Type, object.OwnerType, object.OwnerID, object.JSON); err != nil {
		return model.Object{}, err
	}
	object.UpdatedAt = s.now()
	updated, err := s.stores.UpdateObject(ctx, object)
	if err != nil {
		return model.Object{}, err
	}
	s.publishObject(ctx, "updated", updated.ObjectID, updated)
	return updated, nil
}

func (s *Services) DeleteObject(ctx context.Context, id string) error {
	if err := s.commands.RunWhileObjectInactive(id, func() error {
		return s.stores.DeleteObject(ctx, id)
	}); err != nil {
		return err
	}
	s.publish(ctx, "object", "deleted", id, model.DeleteEventData{DeletedAt: s.now()})
	return nil
}

func (s *Services) UploadObjectFile(ctx context.Context, input store.ObjectUploadInput) (model.ObjectFile, error) {
	if err := requireValidID("file_id", input.File.FileID); err != nil {
		return model.ObjectFile{}, err
	}
	file, err := s.stores.CreateObjectFile(ctx, input)
	if err != nil {
		return model.ObjectFile{}, err
	}
	object, err := s.stores.GetObject(ctx, file.ObjectID)
	if err == nil {
		s.publishObject(ctx, "updated", object.ObjectID, object)
	}
	return file, nil
}

func (s *Services) GetObjectFile(ctx context.Context, objectID, fileID string) (model.ObjectFile, error) {
	return s.stores.GetObjectFile(ctx, objectID, fileID)
}

func (s *Services) AppendObjectFile(ctx context.Context, objectID, fileID string, reader io.Reader, maxBytes int64) (model.ObjectFile, error) {
	file, err := s.stores.AppendObjectFile(ctx, objectID, fileID, reader, maxBytes)
	if err != nil {
		return model.ObjectFile{}, err
	}
	object, err := s.stores.GetObject(ctx, objectID)
	if err == nil {
		s.publishObject(ctx, "updated", objectID, object)
	}
	return file, nil
}

func (s *Services) OpenObjectFile(ctx context.Context, objectID, fileID string) (model.ObjectFile, io.ReadCloser, error) {
	return s.stores.OpenObjectFile(ctx, objectID, fileID)
}

func (s *Services) DeleteObjectFile(ctx context.Context, objectID, fileID string) error {
	if err := s.stores.DeleteObjectFile(ctx, objectID, fileID); err != nil {
		return err
	}
	object, err := s.stores.GetObject(ctx, objectID)
	if err == nil {
		s.publishObject(ctx, "updated", objectID, object)
	}
	return nil
}

func (s *Services) FullQuery(ctx context.Context) (model.QuerySnapshot, error) {
	active, ok := s.commands.Get()
	if !ok {
		return model.QuerySnapshot{}, model.CatalogUnavailable("active command catalog is unavailable", nil)
	}
	state, err := s.stores.GetFullQueryState(ctx)
	if err != nil {
		return model.QuerySnapshot{}, err
	}
	return model.QuerySnapshot{GeneratedAt: s.now(), Service: model.ServiceBrief{Service: model.ServiceName, ActiveCommandCatalogObjectID: active.ObjectID}, Entities: state.Entities, Observations: state.Observations, Tasks: state.Tasks, Objects: state.Objects, ObjectFiles: state.ObjectFiles}, nil
}

func (s *Services) publish(ctx context.Context, resourceType, mutation, resourceID string, data any) {
	if s.publisher == nil {
		return
	}
	event := model.EventEnvelope{EventID: s.publisher.NextID(), Type: fmt.Sprintf("%s.%s", resourceType, mutation), ResourceType: resourceType, ResourceID: resourceID, Mutation: strings.TrimSuffix(mutation, "d"), OccurredAt: s.now(), Data: data}
	if mutation == "deleted" {
		event.Mutation = "delete"
	}
	_ = s.publisher.Publish(ctx, event)
}

func (s *Services) publishObject(ctx context.Context, mutation, objectID string, object model.Object) {
	files, _ := s.stores.ListObjectFilesForObject(ctx, objectID)
	totalFiles := len(files)
	if len(files) > 8 {
		files = files[:8]
	}
	payload := model.ObjectEventResource{Object: object, FileCount: totalFiles, AffectedFiles: files}
	s.publish(ctx, "object", mutation, objectID, payload)
}

func (s *Services) loadPinnedCatalog(ctx context.Context, objectID string) (catalog.Catalog, error) {
	if active, ok := s.commands.Get(); ok && active.ObjectID == objectID {
		if len(active.Raw) == 0 {
			return catalog.Catalog{}, model.CatalogUnavailable("active command catalog has no raw bytes for verification", nil)
		}
		contentHash := active.ContentHash
		if contentHash == "" {
			// Older in-memory catalog values may not have ContentHash populated, so
			// derive it once from the raw bytes before verifying the object ID.
			contentHash = catalog.ContentHashOfBytes(active.Raw)
		}
		if catalog.ObjectIDFromContentHash(contentHash) != objectID {
			return catalog.Catalog{}, model.CatalogUnavailable("command catalog object id does not match catalog bytes", nil)
		}
		return active, nil
	}
	meta, rc, err := s.stores.OpenObjectFile(ctx, objectID, "catalog-json")
	if err != nil {
		return catalog.Catalog{}, err
	}
	defer rc.Close()
	if meta.SizeBytes <= 0 {
		return catalog.Catalog{}, model.CatalogUnavailable("stored command catalog has invalid size metadata", nil)
	}
	raw, err := readAllCapped(rc, maxPinnedCatalogBytes)
	if err != nil {
		if coreErr, ok := model.IsCoreError(err); ok && coreErr.ErrorCode == "payload_too_large" {
			return catalog.Catalog{}, model.CatalogUnavailable("stored command catalog exceeds read limit", nil)
		}
		return catalog.Catalog{}, err
	}
	if int64(len(raw)) != meta.SizeBytes {
		return catalog.Catalog{}, model.CatalogUnavailable("stored command catalog byte length does not match object metadata", nil)
	}
	contentHash := catalog.ContentHashOfBytes(raw)
	if catalog.ObjectIDFromContentHash(contentHash) != objectID {
		return catalog.Catalog{}, model.CatalogUnavailable("command catalog object id does not match catalog bytes", nil)
	}
	var parsed catalog.Catalog
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return catalog.Catalog{}, model.CatalogUnavailable("stored command catalog is not valid JSON", nil)
	}
	parsed.Raw = raw
	if err := catalog.Validate(parsed); err != nil {
		if _, ok := model.IsCoreError(err); ok {
			return catalog.Catalog{}, model.CatalogUnavailable("stored command catalog failed validation", nil)
		}
		return catalog.Catalog{}, err
	}
	parsed.ObjectID = objectID
	parsed.ContentHash = contentHash
	parsed.ByType = map[string]catalog.Command{}
	for _, command := range parsed.Commands {
		parsed.ByType[command.Type] = command
	}
	return parsed, nil
}

func requireValidID(name, value string) error {
	if err := model.ValidateID(name, value); err != nil {
		return model.ValidationError(*err)
	}
	return nil
}

func allowedTaskTransition(current, target string) bool {
	return (current == "pending" && target == "acknowledged") || (current == "acknowledged" && (target == "completed" || target == "failed"))
}

var componentRules = map[string][]string{
	"heartbeat":          {"asset"},
	"communications":     {"asset"},
	"health":             {"asset"},
	"telemetry":          {"asset", "track"},
	"geometry":           {"geofeature"},
	"sensor_refs":        {"asset", "track"},
	"status":             {"asset", "track", "geofeature"},
	"supported_commands": {"asset"},
	"fusion_summary":     {"track"},
}

func validateEntityInput(entityType string, jsonMap model.JSONMap, create bool) error {
	if !model.ContainsString([]string{"asset", "track", "geofeature"}, entityType) {
		return model.ValidationError(model.FieldError{Field: "type", Code: "invalid_value", Message: "type must be asset, track, or geofeature"})
	}
	if err := model.EnsureNoPromotedFieldDuplication(jsonMap, "entity_id", "type", "subtype", "alias", "created_at", "updated_at"); err != nil {
		return model.ValidationError(*err)
	}
	componentsRaw, ok := jsonMap["components"]
	if !ok && entityType == "asset" {
		return model.ValidationError(model.FieldError{Field: "json.components.supported_commands", Code: "required", Message: "asset entities require supported_commands"})
	}
	if ok {
		components, ok := componentsRaw.(map[string]any)
		if !ok {
			return model.ValidationError(model.FieldError{Field: "json.components", Code: "invalid_type", Message: "components must be an object"})
		}
		for name, value := range components {
			if strings.HasPrefix(name, "custom_") {
				if err := model.ValidateCustomComponent("json.components."+name, value); err != nil {
					return model.ValidationError(*err)
				}
				continue
			}
			allowedTypes, exists := componentRules[name]
			if !exists {
				return model.ValidationError(model.FieldError{Field: "json.components." + name, Code: "unknown_field", Message: "unknown component"})
			}
			if !model.ContainsString(allowedTypes, entityType) {
				return model.ValidationError(model.FieldError{Field: "json.components." + name, Code: "invalid_value", Message: "component is not allowed for this entity type"})
			}
			component, ok := value.(map[string]any)
			if !ok {
				return model.ValidationError(model.FieldError{Field: "json.components." + name, Code: "invalid_type", Message: "component must be an object"})
			}
			if observedAt, ok := component["observed_at"].(string); ok {
				if err := model.ValidateRFC3339Field("json.components."+name+".observed_at", observedAt); err != nil {
					return model.ValidationError(*err)
				}
			}
			if name == "supported_commands" {
				if err := validateSupportedCommands(component); err != nil {
					return err
				}
			}
		}
		if entityType == "asset" {
			if _, ok := components["supported_commands"]; !ok {
				return model.ValidationError(model.FieldError{Field: "json.components.supported_commands", Code: "required", Message: "asset entities require supported_commands"})
			}
		}
	}
	return nil
}

func validateSupportedCommands(component map[string]any) error {
	observedAt, _ := component["observed_at"].(string)
	if err := model.ValidateRFC3339Field("json.components.supported_commands.observed_at", observedAt); err != nil {
		return model.ValidationError(*err)
	}
	commands, ok := model.ReadStringSlice(component["commands"])
	if !ok {
		return model.ValidationError(model.FieldError{Field: "json.components.supported_commands.commands", Code: "invalid_type", Message: "commands must be an array of strings"})
	}
	for _, command := range commands {
		if err := model.ValidateSnakeCase("json.components.supported_commands.commands", command); err != nil {
			return model.ValidationError(*err)
		}
	}
	return nil
}

func validateObservationJSON(ctx context.Context, stores store.Stores, sightings *sightingcatalog.Active, observationID string, jsonMap model.JSONMap) error {
	state, _ := jsonMap["state"].(string)
	if !model.ContainsString([]string{"active", "inactive", "ended"}, state) {
		return model.ValidationError(model.FieldError{Field: "json.state", Code: "invalid_value", Message: "state must be active, inactive, or ended"})
	}
	if latest, ok := jsonMap["latest_sighting"]; ok {
		active, ready := sightings.Get()
		if !ready {
			return model.CatalogUnavailable("sighting catalog is unavailable", nil)
		}
		if err := sightingcatalog.ValidateSighting(active, latest); err != nil {
			return err
		}
	}
	if sightingsObjectID, ok := jsonMap["sightings_object_id"].(string); ok && sightingsObjectID != "" {
		object, err := stores.GetObject(ctx, sightingsObjectID)
		if err != nil {
			return err
		}
		if object.Type != "observation_sighting_history" || object.OwnerType != "observation" || object.OwnerID != observationID {
			return model.ValidationError(model.FieldError{Field: "json.sightings_object_id", Code: "invalid_value", Message: "sightings_object_id must reference an observation-owned observation_sighting_history object"})
		}
	}
	return nil
}

func validateTaskCommand(jsonMap model.JSONMap, cat catalog.Catalog, asset model.Entity) error {
	commandType, err := validateTaskCommandCatalog(jsonMap, cat)
	if err != nil {
		return err
	}
	return validateTaskCommandSupport(commandType, asset)
}

func validateTaskCommandCatalog(jsonMap model.JSONMap, cat catalog.Catalog) (string, error) {
	componentsValue, ok := jsonMap["components"]
	if !ok {
		return "", model.ValidationError(model.FieldError{Field: "json.components", Code: "required", Message: "components are required"})
	}
	components, ok := componentsValue.(map[string]any)
	if !ok {
		return "", model.ValidationError(model.FieldError{Field: "json.components", Code: "invalid_type", Message: "components must be an object"})
	}
	commandValue, ok := components["command"]
	if !ok {
		return "", model.ValidationError(model.FieldError{Field: "json.components.command", Code: "required", Message: "command is required"})
	}
	commandSection, ok := commandValue.(map[string]any)
	if !ok {
		return "", model.ValidationError(model.FieldError{Field: "json.components.command", Code: "invalid_type", Message: "command must be an object"})
	}
	commandType, _ := commandSection["type"].(string)
	if commandType == "" {
		return "", model.ValidationError(model.FieldError{Field: "json.components.command.type", Code: "required", Message: "command.type is required"})
	}
	commandDef, ok := cat.ByType[commandType]
	if !ok {
		return "", model.CommandValidationError("command type is not defined in the catalog", model.FieldError{Field: "json.components.command.type", Code: "invalid_value", Message: "unknown command type"})
	}
	parameters, ok := components["parameters"]
	if !ok {
		parameters = map[string]any{}
	}
	if err := catalog.ValidateParameters(commandDef, parameters); err != nil {
		if coreErr, ok := model.IsCoreError(err); ok {
			return "", model.CommandValidationError("command parameters failed validation", flattenFieldErrors(coreErr)...)
		}
		return "", err
	}
	return commandType, nil
}

func validateTaskCommandSupport(commandType string, asset model.Entity) error {
	assetComponents, _ := asset.JSON["components"].(map[string]any)
	supported, ok := assetComponents["supported_commands"].(map[string]any)
	if !ok {
		return model.CommandValidationError("asset does not expose supported_commands", model.FieldError{Field: "asset.json.components.supported_commands", Code: "required", Message: "asset missing supported_commands"})
	}
	commands, ok := model.ReadStringSlice(supported["commands"])
	if !ok || !model.ContainsString(commands, commandType) {
		return model.CommandValidationError("asset does not support requested command", model.FieldError{Field: "json.components.command.type", Code: "invalid_value", Message: "asset does not support command"})
	}
	return nil
}

func validateObjectInput(ctx context.Context, stores store.Stores, objectID, objectType, ownerType, ownerID string, jsonMap model.JSONMap) error {
	if err := requireValidID("object_id", objectID); err != nil {
		return err
	}
	objectType = strings.TrimSpace(objectType)
	if objectType == "" {
		return model.ValidationError(model.FieldError{Field: "type", Code: "required", Message: "type is required"})
	}
	if ownerType == "" || ownerID == "" {
		return model.ValidationError(model.FieldError{Field: "owner", Code: "required", Message: "owner_type and owner_id are required"})
	}
	if !model.ContainsString([]string{"entity", "observation", "task", "system"}, ownerType) {
		return model.ValidationError(model.FieldError{Field: "owner_type", Code: "invalid_value", Message: "owner_type must be entity, observation, task, or system"})
	}
	if ownerType != "system" {
		var err error
		switch ownerType {
		case "entity":
			_, err = stores.GetEntity(ctx, ownerID)
		case "observation":
			_, err = stores.GetObservation(ctx, ownerID)
		case "task":
			_, err = stores.GetTask(ctx, ownerID)
		}
		if err != nil {
			return err
		}
	} else if ownerID != "active_command_catalog" {
		return model.ValidationError(model.FieldError{Field: "owner_id", Code: "invalid_value", Message: "unsupported system owner id"})
	}
	if objectType == "observation_sighting_history" && ownerType != "observation" {
		return model.ValidationError(model.FieldError{Field: "type", Code: "invalid_value", Message: "observation_sighting_history must be observation-owned"})
	}
	if objectType == "fusion_provenance" {
		if ownerType != "entity" {
			return model.ValidationError(model.FieldError{Field: "type", Code: "invalid_value", Message: "fusion_provenance must be entity-owned"})
		}
		entity, err := stores.GetEntity(ctx, ownerID)
		if err != nil {
			return err
		}
		if entity.Type != "track" {
			return model.ValidationError(model.FieldError{Field: "owner_id", Code: "invalid_value", Message: "fusion_provenance owner must be a track entity"})
		}
	}
	if err := model.EnsureNoPromotedFieldDuplication(jsonMap, "object_id", "type", "owner_type", "owner_id", "created_at", "updated_at"); err != nil {
		return model.ValidationError(*err)
	}
	return nil
}

func flattenFieldErrors(err *model.CoreError) []model.FieldError {
	raw, _ := err.Details["fields"].([]model.FieldError)
	if len(raw) > 0 {
		return raw
	}
	return nil
}
