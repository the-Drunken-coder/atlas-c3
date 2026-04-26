# Observations API

Observation endpoints manage first-class source-owned evidence records.

Observation record ownership is defined in [`../data-model/observations.md`](../data-model/observations.md), sighting payloads are defined in [`../data-model/sighting-catalog.md`](../data-model/sighting-catalog.md), and the first-class observation decision is recorded in [`../../decisions/0001-first-class-observations.md`](../../decisions/0001-first-class-observations.md). Shared API behavior is defined in [`conventions.md`](./conventions.md). Error codes are defined in [`errors.md`](./errors.md).

## Resource Shape

```json
{
  "observation_id": "obs-001",
  "source_asset_id": "asset-001",
  "json": {
    "state": "active",
    "latest_sighting": {
      "observed_at": "2026-01-01T00:00:10Z",
      "kind": "line_of_sight",
      "data": {
        "observer_latitude": 40.7128,
        "observer_longitude": -74.006,
        "observer_altitude_m": 35,
        "azimuth": {
          "deg": 42.8,
          "inaccuracy": "+-1.5"
        },
        "elevation": {
          "deg": 8.7,
          "inaccuracy": "+-1.0"
        }
      },
      "extra": {}
    },
    "sightings_object_id": "obj-obs-001-sightings",
    "extra": {}
  },
  "created_at": "2026-01-01T00:00:00Z",
  "updated_at": "2026-01-01T00:00:10Z"
}
```

Observation records do not include asset-local subject or tracker references by default. Assets use `observation_id` to update the same observation over time. Sighting history is stored as append-only JSON Lines in an observation-owned object file; the observation resource stores only current summary state.

## Endpoints

| Method | Path | Success | Purpose |
| --- | --- | --- | --- |
| `GET` | `/observations` | `200` array | List observations |
| `POST` | `/observations` | `201` resource | Create an observation |
| `GET` | `/observations/{observation_id}` | `200` resource | Read an observation |
| `PATCH` | `/observations/{observation_id}` | `200` resource | Update an observation |
| `DELETE` | `/observations/{observation_id}` | `204` empty | Delete an observation |

## List Observations

`GET /observations` returns a paginated JSON array.

Supported filters:

- `source_asset_id`
- `updated_after` filters records where `updated_at` is greater than or equal to the RFC 3339 value.

Default order: `updated_at` descending, then `observation_id` ascending.

## Create Observation

Request body:

```json
{
  "observation_id": "obs-001",
  "source_asset_id": "asset-001",
  "json": {
    "state": "active",
    "extra": {}
  }
}
```

Required fields: `observation_id`, `source_asset_id`, `json.state`.

`json.state` must be one of `active`, `inactive`, or `ended`. `json.latest_sighting` and `json.sightings_object_id` may be absent until the first sighting is reported. When `json.latest_sighting` is present, it must validate against the active sighting catalog. When `json.sightings_object_id` is present, it must reference an observation-owned object with `type: "observation_sighting_history"`, `owner_type: "observation"`, and `owner_id` equal to the observation ID.

Failures:

- `400 validation_failed` for missing required fields, invalid state, invalid sighting shape, unknown top-level fields, or non-asset source entities.
- `404 not_found` when `source_asset_id` does not exist.
- `409 conflict` when `observation_id` already exists.

## Read Observation

`GET /observations/{observation_id}` returns the observation resource or `404 not_found`.

## Patch Observation

Mutable fields:

- named top-level sections under `json`

Immutable fields:

- `observation_id`
- `source_asset_id`
- `created_at`
- `updated_at`

PATCH follows named-section replacement rules. Replacing `json.latest_sighting` replaces that sighting object as a whole and does not deep-merge into the existing object. Replacing `json.state` must use one of the allowed observation states. Replacing `json.latest_sighting` must validate against the active sighting catalog. Replacing `json.sightings_object_id` must reference an observation-owned `observation_sighting_history` object.

There is no finalize or close endpoint. Observation lifecycle is represented by updating `json.state`; if an asset stops updating an observation, the last state remains available until deleted or reset.

## Delete Observation

Observation delete returns `204 No Content` when successful.

Observation-owned objects should be deleted through service behavior as defined by [`../../decisions/0007-core-referential-integrity.md`](../../decisions/0007-core-referential-integrity.md).
