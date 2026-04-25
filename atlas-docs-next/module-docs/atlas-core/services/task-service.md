# Task Service

The task service owns API behavior for task endpoints.

API contract: [`../../../contracts/core-api/tasks.md`](../../../contracts/core-api/tasks.md)

## Responsibilities

- validate task create, patch, delete, and status-transition requests
- require caller-supplied `task_id` on create
- enforce that tasks target assets
- validate requested commands against the active in-memory command catalog
- validate command parameters against the active command catalog rules
- check that the target asset supports the requested command
- require `command_catalog_object_id` on create and ensure it matches the active catalog used for command validation
- pin the active command catalog object ID on created tasks
- coordinate task persistence through the regular record store
- publish task mutation events after successful writes

## Store Usage

Uses [`../stores/regular-record-store.md`](../stores/regular-record-store.md).

Task-related objects are queried through the object API using owner filters.

## Command Catalog Dependency

The task service depends on the active command catalog loaded during startup for task creation.

The command catalog is not a service or store. It is checked-in JSON materialized as an object during startup and held in memory for validation.

Once a task exists, command validation and retries for that task should resolve command definitions through the task's stored `command_catalog_object_id`, not through a later global active catalog value.

## Notes

The task status lifecycle should be defined in the task data/API contract before implementation.

Optimistic concurrency should only be added where realistic task multi-writer collisions exist.

