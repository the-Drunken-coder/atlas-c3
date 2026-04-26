# Asset Reports Observation Sightings

This workflow describes how an asset creates or updates a first-class observation and reports timestamped sightings over time.

Observation API behavior is defined in [`../../contracts/core-api/observations.md`](../../contracts/core-api/observations.md). Sighting shapes are defined in [`../../contracts/data-model/sighting-catalog.md`](../../contracts/data-model/sighting-catalog.md). Object API behavior is defined in [`../../contracts/core-api/objects.md`](../../contracts/core-api/objects.md).

## Flow

1. Asset creates an observation with `POST /observations`, including `json.state`.
2. Asset reports each sighting through the SDK helper, such as `observations.reportSighting(observationId, sighting, options?)`.
3. The SDK fills `observed_at` when omitted and preserves caller-provided timestamps when present.
4. The SDK creates or uses an observation-owned object/file for append-only sighting history.
5. The SDK appends one JSON Lines sighting entry with `POST /objects/{object_id}/files/{file_id}/append`.
6. The SDK patches the observation with the new `json.latest_sighting` and `json.sightings_object_id`.
7. File bytes, such as images, are uploaded through normal observation-owned objects and referenced by separate `file` sightings.
8. Atlas Core publishes observation update events for observation patches and object update events for file upload or append changes.

## Notes

Sightings are atomic entries. If a camera produces line-of-sight, file, and analysis evidence at the same source moment, it reports separate sightings with the same `observed_at`.

Clients that receive stream events should use the SDK replica or read the observation, object, or full query to refresh current state. Stream events do not include file bytes.
