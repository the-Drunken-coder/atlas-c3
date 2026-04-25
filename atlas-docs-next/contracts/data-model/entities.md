# Entities

Entities are current-state records for things Atlas Core knows about.

Record-family context lives in [`record-families.md`](./record-families.md). API behavior lives in [`../core-api/entities.md`](../core-api/entities.md).

## Entity Types

| Type | Meaning |
| --- | --- |
| `asset` | Taskable field system such as a drone, rover, camera, sensor, or other directed system |
| `track` | Authoritative current representation of a real-world subject |
| `geofeature` | Spatial feature such as a point, line, polygon, route, zone, or marker |

## Promoted Fields

| Field | Purpose |
| --- | --- |
| `entity_id` | Creator-supplied ID, max 50 characters |
| `type` | Entity type: `asset`, `track`, or `geofeature` |
| `subtype` | Optional lower-level type |
| `alias` | Optional human-readable name |
| `json` | Flexible metadata and component state |
| `created_at` | Core-created timestamp |
| `updated_at` | Core-updated timestamp |

Promoted fields should not be duplicated inside `json`.

## JSON Shape

Entity JSON may contain:

```json
{
  "published_at": "2026-01-01T00:00:00Z",
  "components": {
    "supported_commands": {
      "observed_at": "2026-01-01T00:00:00Z",
      "commands": []
    }
  },
  "extra": {}
}
```

`components` holds structured current state. `extra` is for non-promoted metadata that does not belong to a known component.

## Components

Entity components are documented in [`components/overview.md`](./components/overview.md).

Canonical components:

- [`heartbeat`](./components/heartbeat.md)
- [`communications`](./components/communications.md)
- [`health`](./components/health.md)
- [`telemetry`](./components/telemetry.md)
- [`geometry`](./components/geometry.md)
- [`sensor_refs`](./components/sensor_refs.md)
- [`status`](./components/status.md)
- [`supported_commands`](./components/supported_commands.md)
- `custom_*`

Asset entities must include `json.components.supported_commands`. Assets without this component are invalid. An asset with an empty command list is valid but cannot receive tasks.

Object references should not be duplicated as entity components by default. Objects own their relationship links through the object data contract.

## Relationships

- Tasks target asset entities.
- Assets may produce observations.
- Tracks may be created or updated from observation evidence.
- Objects may be owned by an entity through the object owner fields.

## Validation Boundaries

API/runtime validation should enforce:

- valid entity type
- max 50-character `entity_id`
- component applicability from [`components/overview.md`](./components/overview.md)
- no unknown component keys except `custom_*`
- `supported_commands` presence for assets

Database constraints should enforce:

- primary key on `entity_id`
- non-null `type`
- non-null `json`

## Delete Behavior

Entity delete is allowed.

Assets are not expected to be deleted often in normal operation, but delete remains useful for debugging, simulation, and local reset workflows.
