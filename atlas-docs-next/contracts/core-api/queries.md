# Queries API

Query endpoints support broad current-state reads.

Shared API behavior is defined in [`conventions.md`](./conventions.md). Record ownership is defined in [`../data-model/record-families.md`](../data-model/record-families.md).

## Endpoints

| Method | Path | Success | Purpose |
| --- | --- | --- | --- |
| `GET` | `/queries/full` | `200` full-state object | Fetch broad current system state |

## Full Query

`GET /queries/full` returns current-state data needed by clients that need to bootstrap or refresh local state.

Response body:

```json
{
  "generated_at": "2026-01-01T00:00:00Z",
  "service": {
    "service": "atlas-core",
    "active_command_catalog_object_id": "command-catalog-20260101"
  },
  "entities": [],
  "observations": [],
  "tasks": [],
  "objects": [],
  "object_files": []
}
```

Array entries use the same resource shapes defined in the resource API docs:

- `entities` uses [`entities.md`](./entities.md)
- `observations` uses [`observations.md`](./observations.md)
- `tasks` uses [`tasks.md`](./tasks.md)
- `objects` and `object_files` use [`objects.md`](./objects.md)

The full query must not include object file bytes.

The response should represent one logical read snapshot for structured state and object metadata. Since these records live in PostgreSQL, Atlas Core should assemble the response from a single read-only database transaction where possible.

This endpoint is not a historical replay API.

Failures:

- `503 storage_unavailable` when PostgreSQL is unavailable.
- `503 catalog_unavailable` when the active command catalog identifier is unavailable.
