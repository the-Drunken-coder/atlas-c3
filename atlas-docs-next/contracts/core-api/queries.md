# Queries API

Query endpoints support broad current-state reads.

Shared API behavior is defined in [`conventions.md`](./conventions.md). Record ownership is defined in [`../data-model/record-families.md`](../data-model/record-families.md).

## Endpoints

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/queries/full` | Fetch broad current system state |

## Full Query

`GET /queries/full` should return current-state data needed by clients that need to bootstrap or refresh local state.

Expected contents:

- entities
- observations
- tasks
- object metadata

The full query should not include object file bytes.

This endpoint is not a historical replay API.

