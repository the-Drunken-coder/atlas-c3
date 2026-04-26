# Tasks API

Task endpoints manage work assigned to assets.

Task record ownership is defined in [`../data-model/tasks.md`](../data-model/tasks.md). Command catalog behavior is defined in [`../data-model/command-catalog/overview.md`](../data-model/command-catalog/overview.md). Shared API behavior is defined in [`conventions.md`](./conventions.md). Error codes are defined in [`errors.md`](./errors.md).

## Resource Shape

```json
{
  "task_id": "task-001",
  "status": "pending",
  "asset_id": "asset-001",
  "command_catalog_object_id": "command-catalog-20260101",
  "json": {
    "description": "Move to specified location",
    "created_by": "operator-001",
    "components": {
      "command": { "type": "move_to_location" },
      "parameters": {},
      "progress": {},
      "result": {},
      "error": {}
    },
    "extra": {}
  },
  "created_at": "2026-01-01T00:00:00Z",
  "updated_at": "2026-01-01T00:00:00Z"
}
```

## Endpoints

| Method | Path | Success | Purpose |
| --- | --- | --- | --- |
| `GET` | `/tasks` | `200` array | List tasks |
| `POST` | `/tasks` | `201` resource | Create a task |
| `GET` | `/tasks/{task_id}` | `200` resource | Read a task |
| `PATCH` | `/tasks/{task_id}` | `200` resource | Update a task |
| `DELETE` | `/tasks/{task_id}` | `204` empty | Delete a task |
| `POST` | `/tasks/{task_id}/status` | `200` resource | Transition task status |

## List Tasks

`GET /tasks` returns a paginated JSON array.

Supported filters:

- `asset_id`
- `status`

Default order: `updated_at` descending, then `task_id` ascending.

## Create Task

Request body:

```json
{
  "task_id": "task-001",
  "asset_id": "asset-001",
  "json": {
    "description": "Move to specified location",
    "created_by": "operator-001",
    "components": {
      "command": { "type": "move_to_location" },
      "parameters": {}
    },
    "extra": {}
  }
}
```

Required fields: `task_id`, `asset_id`, `json.components.command.type`.

Core sets initial `status` to `pending` and resolves `command_catalog_object_id` from the active command catalog. Create requests must not set `status`, `command_catalog_object_id`, `created_at`, or `updated_at`.

Validation sequence:

1. `asset_id` must exist and reference an entity whose `type` is `asset`.
2. The active command catalog object ID must be available.
3. `json.components.command.type` must exist in the active in-memory command catalog.
4. `json.components.parameters` must satisfy that command's `parameters_schema`.
5. The target asset must include `json.components.supported_commands`.
6. The target asset's `supported_commands.commands` must include the requested command type.

Failures:

- `400 validation_failed` for missing fields, unknown fields, invalid request shape, or caller-supplied `command_catalog_object_id`.
- `400 command_validation_failed` for invalid command type, parameters, missing asset `supported_commands`, or unsupported asset command.
- `404 not_found` when `asset_id` does not exist.
- `409 conflict` when `task_id` already exists.
- `503 catalog_unavailable` when no active catalog is available.

## Read Task

`GET /tasks/{task_id}` returns the task resource or `404 not_found`.

## Patch Task

Mutable while `status` is `pending`:

- `json.components.command`
- `json.components.parameters`

Mutable after creation in all non-terminal states:

- `json.components.progress`
- `json.components.result`
- `json.components.error`
- named top-level sections under `json.extra`

Immutable after create:

- `task_id`
- `asset_id`
- `command_catalog_object_id`
- `status` through this endpoint
- `json.description`
- `json.created_by`
- `created_at`
- `updated_at`

Command and parameter edits must re-run command validation against the task's pinned `command_catalog_object_id`.

Use `POST /tasks/{task_id}/status` for lifecycle transitions.

## Transition Task Status

Request body:

```json
{
  "status": "acknowledged",
  "json": {
    "components": {
      "progress": {}
    }
  }
}
```

Required fields: `status`.

Allowed transitions:

- `pending` -> `acknowledged`
- `acknowledged` -> `completed`
- `acknowledged` -> `failed`

Repeating the current status is an idempotent no-op and returns `200` with the current task.

`completed` and `failed` are terminal. Terminal statuses must not transition back to active statuses.

Failures:

- `400 invalid_status_transition` for disallowed transitions.
- `404 not_found` when the task does not exist.

## Delete Task

Task delete returns `204 No Content` when successful.

Delete must be rejected with `409 conflict` while task-owned objects exist.
