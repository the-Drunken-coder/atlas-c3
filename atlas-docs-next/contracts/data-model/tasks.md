# Tasks

Tasks are work items assigned to assets.

Record-family context lives in [`record-families.md`](./record-families.md). API behavior lives in [`../core-api/tasks.md`](../core-api/tasks.md). Command catalog behavior lives in [`command-catalog/overview.md`](./command-catalog/overview.md).

## Promoted Fields

| Field | Purpose |
| --- | --- |
| `task_id` | Creator-supplied ID, max 50 characters |
| `status` | Current task status |
| `asset_id` | Target asset entity ID |
| `json` | Command, parameters, progress, result, and task metadata |
| `created_at` | Core-created timestamp |
| `updated_at` | Core-updated timestamp |

Promoted fields should not be duplicated inside `json`.

## JSON Shape

Task JSON may contain:

```json
{
  "description": "Move to specified location",
  "created_by": "operator-001",
  "command_catalog_object_id": "catalog-object-id",
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

The task service validates command type and parameters against the active command catalog during creation.

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

## Relationships

- A task targets an asset through `asset_id`.
- A task pins the command catalog object used for validation.
- Objects may be owned by a task for attachments, results, evidence, or generated payloads.

Task-owned objects use `owner_type=task` and `owner_id={task_id}`.

## Validation Boundaries

API/runtime validation should enforce:

- max 50-character `task_id`
- target entity exists and is an asset
- valid status transitions
- `command.type` exists in the active command catalog
- `parameters` match the command catalog rules
- target asset supports the requested command

Database constraints should enforce:

- primary key on `task_id`
- non-null `status`
- non-null `asset_id`
- non-null `json`

## Delete Behavior

Task delete is allowed.

Tasks are operational data and may be removed during simulation, debugging, or reset workflows.

