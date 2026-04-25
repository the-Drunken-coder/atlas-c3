# Query Service

The query service owns API behavior for broad current-state query endpoints.

API contract: [`../../../contracts/core-api/queries.md`](../../../contracts/core-api/queries.md)

## Responsibilities

- assemble `GET /queries/full`
- include entities, observations, tasks, and object metadata
- exclude object file bytes
- coordinate broad reads across the regular record store and object store
- keep query behavior current-state focused, not historical replay

## Store Usage

Uses:

- [`../stores/regular-record-store.md`](../stores/regular-record-store.md)
- [`../stores/object-store.md`](../stores/object-store.md)
- [`../stores/query-support.md`](../stores/query-support.md)

## Notes

The query service does not need a separate persistence store by default.

Pagination or chunking for full queries should be defined in the API contract before implementation if broad reads become too large for a single response.

