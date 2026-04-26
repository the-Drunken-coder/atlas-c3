# Core Startup Command Catalog Materialization

This workflow describes how Atlas Core prepares the active command catalog.

Command catalog behavior is defined in [`../../contracts/data-model/command-catalog/overview.md`](../../contracts/data-model/command-catalog/overview.md). Object behavior is defined in [`../../contracts/core-api/objects.md`](../../contracts/core-api/objects.md).

## Flow

1. Atlas Core starts and reads the checked-in command catalog JSON file.
2. Core validates the catalog shape and every command's restricted JSON Schema subset.
3. Core keeps the parsed catalog in memory for task validation.
4. Core computes a stable content hash for the exact authored JSON.
5. Core materializes the JSON as a normal object with `type: "command_catalog"`, `owner_type: "system"`, and `owner_id: "active_command_catalog"`.
6. If the same content-hash object already exists, startup reuses it. If the content changed, Core creates a new object and rotates `active_command_catalog_object_id`.
7. Core stores the catalog JSON as an object file with content type `application/json`.
8. Core exposes the active materialized object ID as `active_command_catalog_object_id` from `GET /`.
9. Readiness reports `command_catalog: "ready"`.

## Retention And Audit

Identical catalog content is idempotent and should not create duplicate objects. Changed catalog content creates a new object ID so existing tasks keep their pinned catalog relationship.

Atlas Core's default local operating model may rely on reset/rebuild for cleanup. Longer-running deployments should keep the active catalog object and any catalog objects still referenced by tasks; older unreferenced catalog objects may be deleted by an explicit maintenance workflow.

Startup logs should record whether the catalog object was reused or created, including the active object ID, catalog ID, and content hash.

## Failure Handling

If catalog loading, validation, object creation, or file materialization fails, Atlas Core should fail readiness with `catalog_unavailable` and log enough context to diagnose the startup failure.

Atlas Core must not start with a partial or best-effort command catalog.
