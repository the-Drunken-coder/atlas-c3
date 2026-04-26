# Sighting Catalog

The sighting catalog defines the allowed sighting payloads that assets can report inside observations.

The catalog is checked-in JSON loaded by Atlas Core at startup. It is not materialized as a runtime object, does not have public API access, and does not use catalog versioning. Atlas Core uses the loaded catalog to validate incoming sighting payloads. SDK helpers may use the same shape for local validation, but Core remains authoritative.

The catalog should:

- define allowed sighting `kind` values
- define the data shape for each kind
- provide a stable input vocabulary for data fusion workers
- keep asset and third-party integrations consistent
- keep Core generic while letting the SDK expose ergonomic helpers

The catalog should not:

- define observation lifecycle behavior
- define data fusion association rules
- define track output shape
- require public discovery through Core API
- be materialized as an object
- create a version negotiation system

## Sighting Shape

All sightings share a small wrapper. Each sighting is one atomic entry of one kind.

```json
{
  "observed_at": "2026-01-01T10:00:05Z",
  "kind": "line_of_sight",
  "data": {},
  "extra": {}
}
```

Fields:

| Field | Required | Meaning |
| --- | --- | --- |
| `observed_at` | yes | Source or measurement time as RFC 3339 |
| `observed_at_inaccuracy_ms` | no | Directional time inaccuracy in milliseconds, encoded as a string such as `-10000`, `+10000`, or `+-10000` |
| `kind` | yes | One of the allowed catalog kind values |
| `data` | yes | Payload shape for the sighting kind |
| `extra` | no | Integration-specific metadata outside the shared contract |

SDK helpers should fill `observed_at` automatically when the caller omits it and preserve caller-provided timestamps when present.

## Inaccuracy Values

There is no universal top-level `confidence` field. Inaccuracy lives with the value it qualifies.

`inaccuracy` is a single string value. The unit is implied by the measurement value:

- `azimuth.inaccuracy: "+-1.5"` means plus/minus 1.5 degrees
- `elevation.inaccuracy: "+2"` means up to 2 degrees higher
- `range.inaccuracy: "-25"` means up to 25 meters shorter
- `position.inaccuracy: "+-15"` means plus/minus 15 meters
- `observed_at_inaccuracy_ms: "-10000"` means up to 10 seconds older

Omit `inaccuracy` entirely when the value has no meaningful inaccuracy. Use `+-` for symmetric inaccuracy, `+` for plus-only inaccuracy, and `-` for minus-only inaccuracy.

## Allowed Kinds

Initial allowed sighting kinds:

- `position`
- `line_of_sight`
- `file`
- `analysis`

Do not use a generic `location` kind. `position` and `line_of_sight` represent different evidence:

- `position` is a resolved target position such as ADS-B, GPS, or another lat/lon source.
- `line_of_sight` is a known observer position plus azimuth/elevation toward the observed subject. Range may be included when available, but it is not required.

If spatial, file, and analysis data are produced at the same source time, report separate sightings with the same `observed_at`.

Unknown `kind` values must be rejected unless a future contract explicitly allows `custom_*` kinds. The initial catalog does not allow custom sighting kinds.

## Position Sighting

```json
{
  "observed_at": "2026-01-01T10:00:05Z",
  "observed_at_inaccuracy_ms": "-10000",
  "kind": "position",
  "data": {
    "position": {
      "latitude": 40.7128,
      "longitude": -74.006,
      "altitude_m": 1200,
      "inaccuracy": "+-15"
    },
    "speed_m_s": 92,
    "heading_deg": 180
  },
  "extra": {}
}
```

Required `data` fields: `position.latitude`, `position.longitude`.

Optional `data` fields: `position.altitude_m`, `position.inaccuracy`, `speed_m_s`, `heading_deg`.

## Line-Of-Sight Sighting

```json
{
  "observed_at": "2026-01-01T10:00:05Z",
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
}
```

Required `data` fields: `observer_latitude`, `observer_longitude`, `azimuth.deg`, `elevation.deg`.

Optional `data` fields: `observer_altitude_m`, `azimuth.inaccuracy`, `elevation.inaccuracy`, `range.m`, `range.inaccuracy`.

`range` is optional and only present when the source actually measured or estimated range.

## File Sighting

```json
{
  "observed_at": "2026-01-01T10:00:05Z",
  "kind": "file",
  "data": {
    "object_id": "obj-obs-001-frame-001",
    "file_id": "file-frame-001",
    "role": "source_image"
  },
  "extra": {}
}
```

Required `data` fields: `object_id`, `file_id`, `role`.

File bytes are stored through normal objects owned by the observation. File sightings reference object/file IDs and do not contain bytes.

Initial file role values:

- `source_image`
- `source_video_clip`
- `sensor_payload`
- `analysis_artifact`

## Analysis Sighting

```json
{
  "observed_at": "2026-01-01T10:00:05Z",
  "kind": "analysis",
  "data": {
    "classification": "black_suv",
    "classification_confidence": 0.76
  },
  "extra": {}
}
```

`analysis.data` may contain classification or identity hints. Detailed analysis and fusion confidence models remain deferred.

## Same-Moment Evidence

A source may report several kinds of evidence at the same source timestamp. For example, a camera may report a line-of-sight angular sighting, capture an image, and run local analysis that classifies the image.

Do not package spatial, file/media, and analysis data into one sighting by default. Report separate sightings with the same `observed_at`:

```jsonl
{"observed_at":"2026-01-01T10:00:00Z","kind":"line_of_sight","data":{"observer_latitude":40.7128,"observer_longitude":-74.006,"observer_altitude_m":35,"azimuth":{"deg":42.1,"inaccuracy":"+-1.5"},"elevation":{"deg":8.5}},"extra":{}}
{"observed_at":"2026-01-01T10:00:00Z","kind":"analysis","data":{"classification":"black_suv","classification_confidence":0.76},"extra":{}}
{"observed_at":"2026-01-01T10:00:00Z","kind":"file","data":{"object_id":"obj-obs-001-frame-001","file_id":"file-frame-001","role":"source_image"},"extra":{}}
```

Fusion consumes the full sequence of sightings for an observation and decides how to combine position, line-of-sight, file, and analysis evidence.

## Catalog JSON Shape

The authored catalog JSON should use this structure:

```json
{
  "catalog_id": "default-sighting-catalog",
  "sighting_kinds": [
    {
      "kind": "line_of_sight",
      "display_name": "Line Of Sight",
      "description": "Observer-relative azimuth/elevation measurement from a known observer position.",
      "data_schema": {
        "type": "object",
        "required": ["observer_latitude", "observer_longitude", "azimuth", "elevation"],
        "additionalProperties": false,
        "properties": {
          "observer_latitude": { "type": "number", "minimum": -90, "maximum": 90 },
          "observer_longitude": { "type": "number", "minimum": -180, "maximum": 180 },
          "observer_altitude_m": { "type": "number" },
          "azimuth": {
            "type": "object",
            "required": ["deg"],
            "additionalProperties": false,
            "properties": {
              "deg": { "type": "number", "minimum": 0, "maximum": 360 },
              "inaccuracy": { "type": "string", "pattern": "^(\\+-|\\+|-)[0-9]+(\\.[0-9]+)?$" }
            }
          },
          "elevation": {
            "type": "object",
            "required": ["deg"],
            "additionalProperties": false,
            "properties": {
              "deg": { "type": "number", "minimum": -90, "maximum": 90 },
              "inaccuracy": { "type": "string", "pattern": "^(\\+-|\\+|-)[0-9]+(\\.[0-9]+)?$" }
            }
          }
        }
      }
    }
  ]
}
```

Startup must fail readiness if the checked-in catalog is missing or invalid.

## Deferred Catalog Choices

The initial contract intentionally defers:

- whether `analysis.data` should receive stricter shared schemas
- whether kind aliases should exist or integrations must normalize before reporting
- whether any inaccuracy values should become required for `position` or `line_of_sight`
