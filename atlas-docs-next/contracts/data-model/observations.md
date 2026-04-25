# Observations

Observations are first-class evidence records produced by assets or sensors.

Record-family context lives in [`record-families.md`](./record-families.md). API behavior lives in [`../core-api/observations.md`](../core-api/observations.md). The first-class observation decision is [`../../decisions/0001-first-class-observations.md`](../../decisions/0001-first-class-observations.md).

## Purpose

An observation captures evidence about something detected, measured, or reported over time.

Observations are evidence. Tracks are authoritative current truth.

## Promoted Fields

| Field | Purpose |
| --- | --- |
| `observation_id` | Creator-supplied ID, max 50 characters |
| `source_asset_id` | Asset that produced the observation |
| `first_observed_at` | When the observed event started |
| `last_observed_at` | Most recent observed timestamp for the event |
| `json` | Flexible evidence metadata |
| `created_at` | Core-created timestamp |
| `updated_at` | Core-updated timestamp |

Promoted fields should not be duplicated inside `json`.

## JSON Shape

Observation JSON may contain:

```json
{
  "confidence": 0.85,
  "evidence": {
    "kinematics": {},
    "classification": {},
    "identity": {},
    "uncertainty": {}
  },
  "extra": {}
}
```

`evidence` is observation-specific. It is not the same component catalog used by entities and tasks.

## Evidence Boundary

Observations are evidence, not fused truth. Tracks are authoritative fused or current truth.

`json.evidence` remains an opaque named-section container until the fusion model and evidence contracts are designed. Known section names such as `kinematics`, `classification`, `identity`, and `uncertainty` may be used as broad buckets, but this document does not define their inner fields.

Data fusion may ignore evidence sections it does not understand.

Evidence sections should include their own measurement timestamp once their inner shapes are defined.

## Relationships

- An observation is produced by an asset through `source_asset_id`.
- Observation media and large payloads are stored through objects.
- Objects relate to observations with `owner_type=observation` and `owner_id={observation_id}`.
- Data fusion may consume observations to create or update tracks.

## Update Behavior

Observations may be updated over time with `PATCH`.

There is no finalize or close state in the API plan. If an asset stops updating an observation, the last state remains available until deleted or reset.

Patch behavior for nested `evidence` sections follows the Core API named-section replacement rule. A patch that replaces `json.evidence.kinematics` replaces that section as a whole and does not deep-merge into the existing object.

## Validation Boundaries

API/runtime validation should enforce:

- max 50-character `observation_id`
- valid `source_asset_id`
- valid RFC 3339 observation timestamps over the API
- `last_observed_at` should not be earlier than `first_observed_at`
- required `source_asset_id`, `first_observed_at`, and `last_observed_at` on create

Database constraints should enforce:

- primary key on `observation_id`
- non-null `source_asset_id`
- non-null `first_observed_at`
- non-null `last_observed_at`
- non-null `json`
