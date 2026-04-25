# Observation Service

The observation service owns API behavior for observation endpoints.

API contract: [`../../../contracts/core-api/observations.md`](../../../contracts/core-api/observations.md)

## Responsibilities

- validate observation create, patch, and delete requests
- require caller-supplied `observation_id` on create
- enforce first-class observation behavior
- coordinate observation persistence through the regular record store
- allow observations to update over time with `PATCH`
- publish observation mutation events after successful writes

## Store Usage

Uses [`../stores/regular-record-store.md`](../stores/regular-record-store.md).

Observation files are handled through objects. The observation service should not own file upload, file metadata, or file byte access.

## Notes

There is no finalize or close endpoint. Observation lifecycle is represented through observation state updates.

Observation-related objects are queried through the object API using `owner_type=observation` and `owner_id={observation_id}`.

