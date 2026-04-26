# Tasks

Tasks are work items assigned to assets.

Record-family context lives in [`record-families.md`](./record-families.md). API behavior lives in [`../core-api/tasks.md`](../core-api/tasks.md). Command catalog behavior lives in [`command-catalog/overview.md`](./command-catalog/overview.md).

## Promoted Fields

| Field | Purpose |
| --- | --- |
| `task_id` | Creator-supplied ID, max 50 characters |
| `status` | Current task status |
| `asset_id` | Target asset entity ID |
| `command_catalog_object_id` | Pinned command catalog object used for validation |
| `json` | Command, parameters, progress, result, and task metadata |
| `created_at` | Core-created timestamp, serialized as UTC RFC 3339 such as `2026-01-01T00:00:00Z` |
| `updated_at` | Core-updated timestamp, serialized as UTC RFC 3339 such as `2026-01-01T00:00:00Z` |

Promoted fields should not be duplicated inside `json`.

## JSON Shape

Task JSON may contain:

```json
{
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
}
```

## Required Task Meaning

Every task should have:

- a target asset
- a command type
- a pinned command catalog object ID

During task creation, the client must not provide `command_catalog_object_id`. Core resolves the active command catalog object, validates `command.type` and `parameters` against the in-memory catalog, and writes the active catalog object ID onto the task.

The target asset must include [`supported_commands`](./components/supported_commands.md). Task creation must fail unless `json.components.command.type` is listed in the target asset's `supported_commands.commands`.

After creation, validation and retries for that task should resolve command definitions through the task's stored `command_catalog_object_id`, not through a later global active catalog value.

## Patch And Edit Rules

Some task fields should stay stable so clients can trust identity, provenance, and catalog pinning. Other fields should stay editable long enough for normal operator fixes, especially command parameters, before an asset commits to execution.

### Immutable After Create

These should not change after `POST /tasks`:

- `task_id`
- `asset_id`
- `command_catalog_object_id`
- `json.description`
- `json.created_by`

### Editable By Lifecycle Stage

- While `status` is `pending`, `PATCH` may replace **`json.components.command`** and/or **`json.components.parameters`** using the same **replace named component** semantics as entities (no accidental deep-merge). Each successful write must re-run the same checks as task creation: `command.type` exists in the **pinned** catalog, `parameters` match that command's rules in the pinned catalog, and the target asset still supports the command.
- After the task leaves `pending` (for example `acknowledged`, `completed`, or `failed`), **`json.components.command` and `json.components.parameters` are immutable**. Further work should flow through **`json.components.progress`**, **`json.components.result`**, **`json.components.error`**, and status transitions.
- `status` should transition through **`POST /tasks/{task_id}/status`** as the primary lifecycle path. If `PATCH` also accepts `status`, it must enforce the same transition rules.

Terminal statuses must not accept command or parameter edits.

## Status Lifecycle

Initial status lifecycle:

```text
pending -> acknowledged -> completed
                      \-> failed
```

| Status | Meaning |
| --- | --- |
| `pending` | Created, not yet accepted by the asset |
| `acknowledged` | Asset received and accepted the task |
| `completed` | Finished successfully |
| `failed` | Finished unsuccessfully |

Terminal statuses should not transition back to active statuses.

Additional intermediate transitions such as reject, start, and update progress are intentionally out of scope for the initial task status contract. They should only be added after this lifecycle and the Task API expand to define their meanings and valid transitions.

## Relationships

- A task targets an asset through `asset_id`.
- A task pins the command catalog object used for validation through `command_catalog_object_id`.
- Objects may be owned by a task for attachments, results, evidence, or generated payloads.

Task-owned objects use `owner_type=task` and `owner_id={task_id}`.

## Validation Boundaries

API/runtime validation should enforce:

- max 50-character `task_id`
- target entity exists and is an asset
- caller does not provide `command_catalog_object_id` on create
- Core writes `command_catalog_object_id` from the active catalog object used for command validation at creation time
- valid status transitions
- `command.type` exists in the pinned command catalog
- `parameters` match the pinned command catalog rules
- target asset has `json.components.supported_commands`
- target asset's `supported_commands.commands` includes the requested command

Database constraints should enforce:

- primary key on `task_id`
- non-null `status`
- non-null `asset_id`
- non-null `command_catalog_object_id`
- non-null `json`

## Delete Behavior

Task delete is allowed.

Tasks are operational data and may be removed during simulation, debugging, or reset workflows.
