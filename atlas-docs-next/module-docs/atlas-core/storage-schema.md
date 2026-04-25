# Atlas Core Storage Schema

This document defines the initial PostgreSQL schema Atlas Core creates at startup.

Atlas Core does not use database migrations in the current operating model. When the schema no longer matches the expected local or scenario state, reset and rebuild are preferred over compatibility layers.

## General Rules

- IDs are `text` with `length(id) <= 50`.
- Flexible resource state uses `jsonb`.
- Timestamps use `timestamptz`.
- `created_at` and `updated_at` are Core-owned.
- Promoted fields are not duplicated inside `json`.
- Schema bootstrap should create the tables, constraints, and indexes below if the database is empty.
- Bootstrap should create `objects` before `tasks` because `tasks.command_catalog_object_id` references `objects.object_id`.

## Tables

### `entities`

| Column | Type | Constraints |
| --- | --- | --- |
| `entity_id` | `text` | Primary key, length 1-50 |
| `type` | `text` | Not null, one of `asset`, `track`, `geofeature` |
| `subtype` | `text` | Nullable |
| `alias` | `text` | Nullable |
| `json` | `jsonb` | Not null, default `{}` |
| `created_at` | `timestamptz` | Not null |
| `updated_at` | `timestamptz` | Not null |

Indexes:

- `entities_type_idx` on `type`
- `entities_updated_at_idx` on `updated_at desc, entity_id asc`

### `observations`

| Column | Type | Constraints |
| --- | --- | --- |
| `observation_id` | `text` | Primary key, length 1-50 |
| `source_asset_id` | `text` | Not null, references `entities(entity_id)` |
| `first_observed_at` | `timestamptz` | Not null |
| `last_observed_at` | `timestamptz` | Not null, greater than or equal to `first_observed_at` |
| `json` | `jsonb` | Not null, default `{}` |
| `created_at` | `timestamptz` | Not null |
| `updated_at` | `timestamptz` | Not null |

Indexes:

- `observations_source_asset_idx` on `source_asset_id`
- `observations_last_observed_idx` on `last_observed_at desc, observation_id asc`
- `observations_updated_at_idx` on `updated_at desc, observation_id asc`

Service validation must ensure `source_asset_id` references an entity whose `type` is `asset`.

### `tasks`

| Column | Type | Constraints |
| --- | --- | --- |
| `task_id` | `text` | Primary key, length 1-50 |
| `status` | `text` | Not null, one of `pending`, `acknowledged`, `completed`, `failed` |
| `asset_id` | `text` | Not null, references `entities(entity_id)` |
| `command_catalog_object_id` | `text` | Not null, references `objects(object_id)` |
| `json` | `jsonb` | Not null, default `{}` |
| `created_at` | `timestamptz` | Not null |
| `updated_at` | `timestamptz` | Not null |

Indexes:

- `tasks_asset_idx` on `asset_id`
- `tasks_status_idx` on `status`
- `tasks_asset_status_updated_idx` on `asset_id, status, updated_at desc, task_id asc`
- `tasks_updated_at_idx` on `updated_at desc, task_id asc`

Service validation must ensure `asset_id` references an entity whose `type` is `asset`.

### `objects`

| Column | Type | Constraints |
| --- | --- | --- |
| `object_id` | `text` | Primary key, length 1-50 |
| `type` | `text` | Not null |
| `owner_type` | `text` | Not null, one of `entity`, `observation`, `task`, `system` |
| `owner_id` | `text` | Not null |
| `json` | `jsonb` | Not null, default `{}` |
| `created_at` | `timestamptz` | Not null |
| `updated_at` | `timestamptz` | Not null |

Indexes:

- `objects_owner_idx` on `owner_type, owner_id`
- `objects_type_idx` on `type`
- `objects_updated_at_idx` on `updated_at desc, object_id asc`

`owner_type` and `owner_id` are polymorphic. Atlas Core must enforce owner existence in the service layer.

### `object_files`

| Column | Type | Constraints |
| --- | --- | --- |
| `file_id` | `text` | Primary key, length 1-50 |
| `object_id` | `text` | Not null, references `objects(object_id)` on delete cascade |
| `path` | `text` | Not null, unique |
| `content_type` | `text` | Not null |
| `size_bytes` | `bigint` | Not null, greater than or equal to 0 |
| `usage_hint` | `text` | Nullable |
| `created_at` | `timestamptz` | Not null |
| `updated_at` | `timestamptz` | Not null |

Indexes:

- `object_files_object_idx` on `object_id`
- `object_files_updated_at_idx` on `updated_at desc, file_id asc`

The filesystem volume owns file bytes. PostgreSQL owns file metadata and the logical path.

## Delete Behavior

Delete behavior follows [`../../decisions/0007-core-referential-integrity.md`](../../decisions/0007-core-referential-integrity.md):

- Entity delete is rejected with `409 conflict` while tasks, observations, or entity-owned objects still reference the entity.
- Observation delete should delete observation-owned objects through service behavior; deleting those objects deletes object file metadata and bytes.
- Task delete is rejected with `409 conflict` while task-owned objects exist.
- Object delete deletes object file metadata and file bytes.

## Object File Consistency

Object file upload and delete behavior follows [`../../decisions/0006-object-file-write-ordering.md`](../../decisions/0006-object-file-write-ordering.md).

Committed PostgreSQL metadata is the source of truth for what files exist. Atlas Core should not expose metadata for partially uploaded files.
