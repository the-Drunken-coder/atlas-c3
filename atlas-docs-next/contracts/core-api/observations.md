# Observations API

Observation endpoints manage first-class sensor evidence records.

Observation record ownership is defined in [`../data-model/observations.md`](../data-model/observations.md), and the first-class observation decision is recorded in [`../../decisions/0001-first-class-observations.md`](../../decisions/0001-first-class-observations.md). Shared API behavior is defined in [`conventions.md`](./conventions.md). Error codes are defined in [`errors.md`](./errors.md).

## Resource Shape

```json
{
  "observation_id": "obs-001",
  "source_asset_id": "asset-001",
  "first_observed_at": "2026-01-01T00:00:00Z",
  "last_observed_at": "2026-01-01T00:00:10Z",
  "json": {
    "confidence": 0.85,
    "evidence": {
      "kinematics": {},
      "classification": {},
      "identity": {},
      "uncertainty": {}
    },
    "extra": {}
  },
  "created_at": "2026-01-01T00:00:00Z",
  "updated_at": "2026-01-01T00:00:10Z"
}
```

The inner shapes of `json.evidence` sections are intentionally deferred. This API still defines how those named sections are created, replaced, and returned.

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
- `observed_after` filters records where `last_observed_at` is greater than or equal to the RFC 3339 value.
- `updated_after` filters records where `updated_at` is greater than or equal to the RFC 3339 value.

Default order: `last_observed_at` descending, then `observation_id` ascending.

## Create Observation

Request body:

```json
{
  "observation_id": "obs-001",
  "source_asset_id": "asset-001",
  "first_observed_at": "2026-01-01T00:00:00Z",
  "last_observed_at": "2026-01-01T00:00:10Z",
  "json": {
    "evidence": {},
    "extra": {}
  }
}
```

Required fields: `observation_id`, `source_asset_id`, `first_observed_at`, `last_observed_at`, `json`.

Failures:

- `400 validation_failed` for missing required fields, invalid timestamps, `last_observed_at` before `first_observed_at`, unknown top-level fields, or non-asset source entities.
- `404 not_found` when `source_asset_id` does not exist.
- `409 conflict` when `observation_id` already exists.

## Read Observation

`GET /observations/{observation_id}` returns the observation resource or `404 not_found`.

## Patch Observation

Mutable fields:

- `last_observed_at`
- named top-level sections under `json`
- named evidence sections under `json.evidence`

Immutable fields:

- `observation_id`
- `source_asset_id`
- `first_observed_at`
- `created_at`
- `updated_at`

PATCH follows named-section replacement rules. Replacing `json.evidence.classification` replaces that section as a whole and does not deep-merge into the existing object.

There is no finalize or close endpoint. If an asset stops updating an observation, the last state remains available until deleted or reset.

## Delete Observation

Observation delete returns `204 No Content` when successful.

Observation-owned objects should be deleted through service behavior as defined by [`../../decisions/0007-core-referential-integrity.md`](../../decisions/0007-core-referential-integrity.md).
