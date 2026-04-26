# Observation Service

The observation service owns API behavior for observation endpoints.

API contract: [`../../../contracts/core-api/observations.md`](../../../contracts/core-api/observations.md)

## Responsibilities

- validate observation create, patch, and delete requests
- validate `source_asset_id` references an entity whose `type` is `asset`
- validate observation JSON shape, including required `json.state`
- validate `json.latest_sighting` against the active sighting catalog when present
- when `json.latest_sighting` is present, validate `json.sightings_object_id` atomically with it: the object must exist, use `type=observation_sighting_history`, use `owner_type=observation`, and use `owner_id={observation_id}`
- require caller-supplied `observation_id` on create
- enforce first-class observation behavior
- coordinate observation persistence through the regular record store
- allow observations to update over time with `PATCH`
- publish observation mutation events after successful writes

## Store Usage

Uses [`../stores/regular-record-store.md`](../stores/regular-record-store.md).

Observation files and sighting history JSONL are handled through objects. The observation service should not own file upload, append, file metadata, or file byte access.

## Notes

There is no finalize or close endpoint. Observation lifecycle is represented through observation state updates.

Observation-related objects are queried through object service/store interfaces using `owner_type=observation` and `owner_id={observation_id}`.
