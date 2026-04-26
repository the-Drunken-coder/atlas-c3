# Observations

Observations are first-class evidence records produced by assets or sensors.

Record-family context lives in [`record-families.md`](./record-families.md). API behavior lives in [`../core-api/observations.md`](../core-api/observations.md). Sighting payloads live in [`sighting-catalog.md`](./sighting-catalog.md). The first-class observation decision is [`../../decisions/0001-first-class-observations.md`](../../decisions/0001-first-class-observations.md).

## Purpose

An observation captures source-owned evidence about something detected, measured, or reported over time.

Observations are evidence. Tracks are authoritative current truth.

Atlas Core should not store asset-local subject or tracker references in observation records by default. The `observation_id` is the durable handle an asset uses to update the same observation over time, and `source_asset_id` identifies who produced it. Asset-internal tracker IDs should stay inside the asset unless a future contract defines a specific shared need.

## Promoted Fields

| Field | Purpose |
| --- | --- |
| `observation_id` | Creator-supplied ID, max 50 characters |
| `source_asset_id` | Asset that produced the observation |
| `json` | Current observation summary and flexible metadata |
| `created_at` | Core-created timestamp |
| `updated_at` | Core-updated timestamp |

Promoted fields should not be duplicated inside `json`.

## JSON Shape

Observation JSON uses a small current-state envelope:

```json
{
  "state": "active",
  "latest_sighting": {
    "observed_at": "2026-01-01T00:00:10Z",
    "kind": "position",
    "data": {
      "position": {
        "latitude": 40.7128,
        "longitude": -74.006,
        "altitude_m": 1200,
        "inaccuracy": "+-15"
      }
    },
    "extra": {}
  },
  "sightings_object_id": "obj-obs-001-sightings",
  "extra": {}
}
```

`state` is required and must be one of:

- `active`
- `inactive`
- `ended`

`latest_sighting` and `sightings_object_id` may be absent until the first sighting is reported. `latest_sighting` must match the sighting shape defined by [`sighting-catalog.md`](./sighting-catalog.md).

## Sighting History

A sighting is a timestamped measurement or report within an observation.

Full sighting history should live in an observation-owned object file as append-only JSON Lines. The observation JSON stores only the latest sighting and the ID of the object that owns the sighting history file.

The sighting-history object must use `type: "observation_sighting_history"`, `owner_type: "observation"`, and `owner_id: "{observation_id}"`. The object relationship remains authoritative through `objects.owner_type` and `objects.owner_id`.

Example sighting history JSONL:

```jsonl
{"observed_at":"2026-01-01T10:00:00Z","kind":"line_of_sight","data":{"observer_latitude":40.7128,"observer_longitude":-74.006,"observer_altitude_m":35,"azimuth":{"deg":42.1,"inaccuracy":"+-1.5"},"elevation":{"deg":8.5}},"extra":{}}
{"observed_at":"2026-01-01T10:00:00Z","kind":"analysis","data":{"classification":"black_suv","classification_confidence":0.76},"extra":{}}
{"observed_at":"2026-01-01T10:00:05Z","kind":"file","data":{"object_id":"obj-obs-001-frame-001","file_id":"file-frame-001","role":"source_image"},"extra":{}}
```

## Evidence Boundary

Observations are evidence, not fused truth. Tracks are authoritative fused or current truth.

The sighting catalog defines the initial input vocabulary for observation evidence. Data fusion may still ignore sighting kinds or fields it does not understand.

Observation-to-track association rules, detailed fusion confidence models, and detailed track provenance remain deferred. Detailed fusion reasoning should be stored as track-owned `fusion_provenance` objects rather than inside observation records.

## Relationships

- An observation is produced by an asset through `source_asset_id`.
- Observation media and large payloads are stored through objects.
- Objects relate to observations with `owner_type=observation` and `owner_id={observation_id}`.
- Data fusion may consume observations and their sighting history to create or update tracks.

## Update Behavior

Observations may be updated over time with `PATCH`.

There is no finalize or close endpoint. Observation lifecycle is represented through `json.state`. If an asset stops updating an observation, the last state remains available until deleted or reset.

Patch behavior for named JSON sections follows the Core API named-section replacement rule. A patch that replaces `json.latest_sighting` replaces that sighting as a whole and does not deep-merge into the existing object.

## Validation Boundaries

API/runtime validation should enforce:

- max 50-character `observation_id`
- valid `source_asset_id`
- `source_asset_id` must refer to an entity whose `type` is `asset`
- required `json.state` on create
- valid `json.state`
- valid `latest_sighting` shape when present

Database constraints should enforce:

- primary key on `observation_id`
- non-null `source_asset_id`
- non-null `json`
