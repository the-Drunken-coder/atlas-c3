# Operator Creates Task

This workflow describes how an operator creates work for an asset.

Task API behavior is defined in [`../../contracts/core-api/tasks.md`](../../contracts/core-api/tasks.md). Command catalog behavior is defined in [`../../contracts/data-model/command-catalog/overview.md`](../../contracts/data-model/command-catalog/overview.md).

## Flow

1. Client reads `GET /` and stores `active_command_catalog_object_id`.
2. Client reads the active command catalog object and JSON file through the object API if it needs command authoring metadata.
3. Client reads the target asset entity with `GET /entities/{asset_id}`.
4. Client builds a `POST /tasks` request with `asset_id`, `command_catalog_object_id`, `json.components.command.type`, and `json.components.parameters`.
5. Atlas Core validates the target asset, active catalog pin, command type, command parameters, and required supported command capability.
6. Atlas Core creates the task with `status: "pending"`.
7. Atlas Core publishes `task.created`.

## Failure Handling

- Invalid command type or parameters return `400 command_validation_failed`.
- A stale or wrong `command_catalog_object_id` returns `400 command_validation_failed`.
- A missing asset returns `404 not_found`.
- A duplicate `task_id` returns `409 conflict`; clients should read the existing task to reconcile.
