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

This should stay simple unless implementation shows a need for a dedicated query path.

## Pagination

The full query may need pagination as data grows, but exact cursor or paging shape should be defined in the API contract before implementation.

## Non-Goals

Query support is not:

- a historical replay system
- a durable event log
- a reporting database
- a replacement for normal list endpoints

