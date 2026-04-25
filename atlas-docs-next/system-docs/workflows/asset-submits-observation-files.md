# Asset Submits Observation With Files

This workflow describes how an asset submits first-class observation evidence and related object files.

Observation API behavior is defined in [`../../contracts/core-api/observations.md`](../../contracts/core-api/observations.md). Object API behavior is defined in [`../../contracts/core-api/objects.md`](../../contracts/core-api/objects.md).

## Flow

1. Asset creates an observation with `POST /observations`.
2. Asset updates the observation over time with `PATCH /observations/{observation_id}` as evidence changes.
3. Asset creates an object with `POST /objects` using `owner_type: "observation"` and `owner_id: "{observation_id}"`.
4. Asset uploads files with `POST /objects/{object_id}/files`.
5. Atlas Core stores file bytes and metadata using staging-first object file write ordering.
6. Atlas Core publishes `observation.created` or `observation.updated` for observation changes.
7. Atlas Core publishes `object.created` for object creation and `object.updated` for file uploads.

## Notes

Observation evidence inner shapes are intentionally not defined by this workflow. The workflow relies only on the observation API's promoted fields and opaque `json.evidence` container.

Clients that receive stream events should read the observation, object, or full query to refresh current state. Stream events do not include file bytes.
