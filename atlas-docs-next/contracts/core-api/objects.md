# Objects API

Object endpoints manage file-container metadata and object file bytes.

Object record ownership is defined in [`../data-model/objects.md`](../data-model/objects.md). Shared API behavior is defined in [`conventions.md`](./conventions.md). Error codes are defined in [`errors.md`](./errors.md).

## Object Resource Shape

```json
{
  "object_id": "object-001",
  "type": "observation_media",
  "owner_type": "observation",
  "owner_id": "obs-001",
  "json": {
    "usage_hints": ["evidence_frame"],
    "extra": {}
  },
  "created_at": "2026-01-01T00:00:00Z",
  "updated_at": "2026-01-01T00:00:00Z"
}
```

## Object File Resource Shape

```json
{
  "file_id": "file-001",
  "object_id": "object-001",
  "path": "objects/object-001/file-001",
  "content_type": "image/jpeg",
  "size_bytes": 12345,
  "usage_hint": "evidence_frame",
  "created_at": "2026-01-01T00:00:00Z",
  "updated_at": "2026-01-01T00:00:00Z"
}
```

`path`, `content_type`, and `size_bytes` are Core-owned outcomes of upload and storage validation.

## Endpoints

| Method | Path | Success | Purpose |
| --- | --- | --- | --- |
| `GET` | `/objects` | `200` array | List objects |
| `POST` | `/objects` | `201` resource | Create an object |
| `GET` | `/objects/{object_id}` | `200` resource | Read an object |
| `PATCH` | `/objects/{object_id}` | `200` resource | Update object metadata |
| `DELETE` | `/objects/{object_id}` | `204` empty | Delete an object |
| `POST` | `/objects/{object_id}/files` | `201` file resource | Upload an object file |
| `POST` | `/objects/{object_id}/files/{file_id}/append` | `200` file resource | Append bytes to an object file |
| `GET` | `/objects/{object_id}/files/{file_id}` | `200` file resource | Read object file metadata |
| `GET` | `/objects/{object_id}/files/{file_id}/content` | `200` byte stream | Stream object file content |
| `DELETE` | `/objects/{object_id}/files/{file_id}` | `204` empty | Delete an object file |

## List Objects

Supported filters:

- `owner_type`
- `owner_id`
- `type`

`owner_type` and `owner_id` must be supplied together when filtering by owner.

Owner filter examples:

- `GET /objects?owner_type=entity&owner_id={entity_id}`
- `GET /objects?owner_type=observation&owner_id={observation_id}`
- `GET /objects?owner_type=task&owner_id={task_id}`
- `GET /objects?owner_type=system&owner_id={system_owned_id}`

Default order: `updated_at` descending, then `object_id` ascending.

## Create Object

Request body:

```json
{
  "object_id": "object-001",
  "type": "observation_media",
  "owner_type": "observation",
  "owner_id": "obs-001",
  "json": {
    "usage_hints": ["evidence_frame"],
    "extra": {}
  }
}
```

Required fields: `object_id`, `type`, `owner_type`, `owner_id`, `json`.

The owner must exist unless `owner_type` is `system`. System owners are limited to Core-defined identifiers such as `active_command_catalog`.

For `type: "observation_sighting_history"`, the owner must be the observation whose sighting history is stored in the object: `owner_type: "observation"` and `owner_id: "{observation_id}"`. This object type is intended for append-only JSON Lines sighting history.

For `type: "fusion_provenance"`, the owner should be a track entity. The object file content should be structured JSON containing detailed fusion reasoning.

Failures:

- `400 validation_failed` for invalid owner type, missing fields, unknown fields, or invalid ID length.
- `404 not_found` when the owner does not exist.
- `409 conflict` when `object_id` already exists.

## Patch Object

Mutable fields:

- `type`
- named top-level sections under `json`

Immutable fields:

- `object_id`
- `owner_type`
- `owner_id`
- `created_at`
- `updated_at`

Object ownership changes are intentionally not part of the initial API.

## Delete Object

Object delete returns `204 No Content` when successful.

Deleting an object deletes its object file metadata and file bytes according to [`../data-model/objects.md`](../data-model/objects.md) and [`../../decisions/0006-object-file-write-ordering.md`](../../decisions/0006-object-file-write-ordering.md).

## Upload Object File

`POST /objects/{object_id}/files` uses `multipart/form-data`.

Required multipart fields:

- `file_id` - caller-supplied globally unique file ID, max 50 characters.
- `file` - file content bytes.

Optional multipart fields:

- `usage_hint`
- `content_type` only when the multipart part does not supply a usable content type.

Core derives `path`, `size_bytes`, and final `content_type`. Upload follows the staging-first write ordering decision.

Failures:

- `400 validation_failed` for missing `file_id`, missing file bytes, invalid ID length, or unsafe multipart fields.
- `404 not_found` when `object_id` does not exist.
- `409 conflict` when `file_id` already exists.
- `413 payload_too_large` when upload exceeds configured limit.
- `415 unsupported_media_type` when content type is not accepted.
- `503 storage_unavailable` for PostgreSQL or object file storage failure.

## Append Object File

`POST /objects/{object_id}/files/{file_id}/append` appends bytes to an existing object file. It does not replace existing bytes.

Append is intended for append-only payloads such as observation sighting history JSON Lines. Callers should append complete records and include their own newline delimiters when writing JSONL.

The request body is a raw byte stream. Response body is the updated object file metadata.

Append is not intrinsically idempotent in the current contract. Atlas Core does not de-duplicate append requests by `Idempotency-Key` or append sequence. After an ambiguous network failure, clients should reconcile by reading current observation/file state rather than blindly repeating the append when duplicate bytes would be harmful.

Successful append behavior:

- append bytes to the existing file content
- update the file's `size_bytes`
- update the file's `updated_at`
- update the parent object's `updated_at`
- publish an object update event

Failures:

- `400 validation_failed` for an empty append body or unsafe request fields.
- `404 not_found` when `object_id` or `file_id` does not exist, or when the file does not belong to the object.
- `413 payload_too_large` when append exceeds configured per-request or resulting-file limits.
- `503 storage_unavailable` for PostgreSQL or object file storage failure.

## Read Object File Metadata

`GET /objects/{object_id}/files/{file_id}` returns file metadata or `404 not_found`.

The path must match the file's parent object. A known `file_id` under the wrong `object_id` should return `404 not_found`.

## Stream Object File Content

`GET /objects/{object_id}/files/{file_id}/content` streams bytes directly. It does not return the JSON success envelope.

Response headers should include:

- `Content-Type`
- `Content-Length` when known

If committed metadata points at missing bytes, return `503 storage_unavailable` and log the mismatch.

## Delete Object File

Object file delete returns `204 No Content` when successful.

Deleting a file deletes metadata and bytes, and advances the parent object's `updated_at`.
