# Core Startup Command Catalog Materialization

This workflow describes how Atlas Core prepares the active command catalog.

Command catalog behavior is defined in [`../../contracts/data-model/command-catalog/overview.md`](../../contracts/data-model/command-catalog/overview.md). Object behavior is defined in [`../../contracts/core-api/objects.md`](../../contracts/core-api/objects.md).

## Flow

1. Atlas Core starts and reads the checked-in command catalog JSON file.
2. Core validates the catalog shape and every command's restricted JSON Schema subset.
3. Core keeps the parsed catalog in memory for task validation.
4. Core materializes the exact authored JSON as a normal object with `type: "command_catalog"`, `owner_type: "system"`, and `owner_id: "active_command_catalog"`.
5. Core stores the catalog JSON as an object file with content type `application/json`.
6. Core exposes the materialized object ID as `active_command_catalog_object_id` from `GET /`.
7. Readiness reports `command_catalog: "ready"`.

## Failure Handling

If catalog loading, validation, object creation, or file materialization fails, Atlas Core should fail readiness with `catalog_unavailable` and log enough context to diagnose the startup failure.

Atlas Core must not start with a partial or best-effort command catalog.
