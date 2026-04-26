# Task Service

The task service owns API behavior for task endpoints.

API contract: [`../../../contracts/core-api/tasks.md`](../../../contracts/core-api/tasks.md)

## Responsibilities

- validate task create, patch, delete, and status-transition requests
- require caller-supplied `task_id` on create
- enforce that tasks target assets
- validate requested commands against the active in-memory command catalog
- validate command parameters against the active command catalog rules
- check that the target asset has `json.components.supported_commands`
- check that the target asset's `supported_commands.commands` includes the requested command
- reject caller-supplied `command_catalog_object_id` on create
- pin the active command catalog object ID on created tasks
- coordinate task persistence through the regular record store
- publish task mutation events after successful writes

## Store Usage

Uses [`../stores/regular-record-store.md`](../stores/regular-record-store.md).

Task-related objects are queried through object service/store interfaces that implement owner filters from the object API contract.

## Command Catalog Dependency

The task service depends on the active command catalog loaded during startup for task creation.

The command catalog is not a service or store. It is checked-in JSON materialized as an object during startup and held in memory for validation.

Once a task exists, command validation and retries for that task should resolve command definitions through the task's stored `command_catalog_object_id`, not through a later global active catalog value.

If the stored `command_catalog_object_id` cannot be resolved, validation should fail deterministically with a non-retriable domain error, `TASK_CATALOG_OBJECT_UNRESOLVABLE`, including `task_id` and `command_catalog_object_id`. The task should remain in its current status and should not re-run command validation until an operator or admin process repairs the missing catalog object or re-pins the task through an explicit future repair path.

Optimistic concurrency should only be added where realistic task multi-writer collisions exist.
