# Objects And Object Files

Objects are file containers and payload metadata records. Object files are metadata rows for bytes stored on the filesystem volume.

Record-family context lives in [`record-families.md`](./record-families.md). API behavior lives in [`../core-api/objects.md`](../core-api/objects.md). Storage shape is decided in [`../../decisions/0002-core-storage-shape.md`](../../decisions/0002-core-storage-shape.md).

## Object Promoted Fields

| Field | Purpose |
| --- | --- |
| `object_id` | Creator-supplied ID, max 50 characters |
| `type` | Object purpose, such as `command_catalog`, `observation_media`, `task_result`, or `heatmap` |
| `owner_type` | Owning record family: `entity`, `observation`, `task`, or `system` |
| `owner_id` | Owning record ID |
| `json` | Object-level metadata |
| `created_at` | Core-created timestamp |
| `updated_at` | Core-updated timestamp |

Promoted fields should not be duplicated inside `json`.

## Object JSON Shape

Object JSON may contain:

```json
{
  "usage_hints": ["evidence_frame"],
  "extra": {}
}
```

`owner_type` and `owner_id` are promoted fields, not JSON metadata.

## Object Ownership

Objects are the authoritative owner of relationship links to records that use them.

Supported owner types:

| Owner type | Owner ID meaning |
| --- | --- |
| `entity` | `entity_id` |
| `observation` | `observation_id` |
| `task` | `task_id` |
| `system` | system-owned identifier such as active command catalog |

Owning records should not duplicate object references unless a later contract defines a specific exception and drift-prevention rule.

## Object File Promoted Fields

| Field | Purpose |
| --- | --- |
| `file_id` | Creator-supplied globally unique ID, max 50 characters |
| `object_id` | Parent object ID |
| `path` | Logical storage path under the managed storage root |
| `content_type` | MIME type or internal content type |
| `size_bytes` | Byte size |
| `usage_hint` | Optional per-file role or hint |
| `created_at` | Core-created timestamp |
| `updated_at` | Core-updated timestamp |

The filesystem volume owns the bytes. PostgreSQL owns metadata and logical paths.

Object file IDs are globally unique even though file API paths are nested under objects. The nested path keeps the API clear about parent ownership; the database identity remains `file_id`.

File bytes should not be mutated in place. A byte-content change should be represented by deleting and uploading an object file, or by a later explicit replace operation that updates metadata and preserves event behavior.

## Expected Object Uses

- observation media
- command catalog payloads
- task attachments or results
- geofeature media such as heatmaps
- evidence artifacts
- captured sensor dumps

## Command Catalog Object

The command catalog source lives as checked-in JSON in Atlas Core.

At startup, Atlas Core materializes that JSON into an object with an object file containing the catalog payload. The service descriptor exposes the active command catalog object ID.

Command catalog details live in [`command-catalog/overview.md`](./command-catalog/overview.md).

## Validation Boundaries

API/runtime validation should enforce:

- max 50-character `object_id`
- max 50-character `file_id`
- valid `owner_type`
- required `owner_id`
- safe logical file paths
- upload body size limits once configured
- no in-place byte mutation path

Database constraints should enforce:

Objects:

- primary key on `object_id`
- non-null `type`
- non-null `json`

Object files:

- primary key on `file_id`
- non-null `object_id`
- object file parent relationship to object
- non-null `path`
- unique logical file path
- non-null `content_type`
- non-null `size_bytes`

## Delete Behavior

Object delete is allowed.

Deleting an object should delete its object file metadata. File byte cleanup is owned by Atlas Core object-store behavior.

