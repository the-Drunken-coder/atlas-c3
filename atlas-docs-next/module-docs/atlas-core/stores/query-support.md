# Query Support

Query support describes how Atlas Core should serve `GET /queries/full`.

It does not need to be a separate persistence store by default.

## Source API

The API contract is [`../../../contracts/core-api/queries.md`](../../../contracts/core-api/queries.md).

## Full Query Capability

`GET /queries/full` should assemble broad current-state data for clients that need to bootstrap or refresh local state.

Expected contents:

- entities
- observations
- tasks
- object metadata

Object file bytes should not be included.

## Store Usage

The full query can be assembled from:

- [`regular-record-store.md`](./regular-record-store.md) for entities, observations, and tasks
- [`object-store.md`](./object-store.md) for object metadata

The assembled response should represent one logical read snapshot for PostgreSQL-backed structured records and object metadata. Atlas Core should use one read-only database transaction where possible so entities, observations, tasks, objects, and object file metadata are mutually consistent at `generated_at`.

Object file bytes are excluded, so the full query does not provide byte-content atomicity. Clients should treat missed stream events or questionable replica state by replacing local structured state from `GET /queries/full`, then reading object file bytes separately when needed.

This should stay simple unless implementation shows a need for a dedicated query path.

## Pagination

The full query may need pagination as data grows, but exact cursor or paging shape should be defined in the API contract before implementation.

## Non-Goals

Query support is not:

- a historical replay system
- a durable event log
- a reporting database
- a replacement for normal list endpoints
