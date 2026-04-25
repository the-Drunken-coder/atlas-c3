# Entities API

Entity endpoints manage current-state records for assets, tracks, and geofeatures.

Entity record ownership is defined in [`../data-model/entities.md`](../data-model/entities.md). Shared API behavior is defined in [`conventions.md`](./conventions.md). Error codes are defined in [`errors.md`](./errors.md).

## Resource Shape

```json
{
  "entity_id": "asset-001",
  "type": "asset",
  "subtype": "quadrotor",
  "alias": "North Field Drone",
  "json": {
    "published_at": "2026-01-01T00:00:00Z",
    "components": {
      "supported_commands": {
        "observed_at": "2026-01-01T00:00:00Z",
        "commands": ["move_to_location"]
      }
    },
    "extra": {}
  },
  "created_at": "2026-01-01T00:00:00Z",
  "updated_at": "2026-01-01T00:00:00Z"
}
```

`json.components` uses the component contracts in [`../data-model/components/overview.md`](../data-model/components/overview.md). Unknown component keys are rejected unless they use the `custom_*` prefix.

## Endpoints

| Method | Path | Success | Purpose |
| --- | --- | --- | --- |
| `GET` | `/entities` | `200` array | List entities |
| `POST` | `/entities` | `201` resource | Create an entity |
| `GET` | `/entities/{entity_id}` | `200` resource | Read an entity |
| `PATCH` | `/entities/{entity_id}` | `200` resource | Update an entity |
| `DELETE` | `/entities/{entity_id}` | `204` empty | Delete an entity |
| `GET` | `/entities/{entity_id}/tasks` | `200` task array | List tasks assigned to an asset |

## List Entities

`GET /entities` returns a paginated JSON array.

Supported filters:

- `type`

Default order: `updated_at` descending, then `entity_id` ascending.

## Create Entity

Request body:

```json
{
  "entity_id": "asset-001",
  "type": "asset",
  "subtype": "quadrotor",
  "alias": "North Field Drone",
  "json": {
    "components": {
      "supported_commands": {
        "observed_at": "2026-01-01T00:00:00Z",
        "commands": ["move_to_location"]
      }
    },
    "extra": {}
  }
}
```

Required fields: `entity_id`, `type`, `json`.

Asset entities must include `json.components.supported_commands`. Assets without this component are invalid.

`created_at` and `updated_at` are Core-owned and must not be sent.

Failures:

- `400 validation_failed` for invalid type, missing required fields, missing asset `supported_commands`, ID length, or unknown top-level fields.
- `409 conflict` when `entity_id` already exists.

## Read Entity

`GET /entities/{entity_id}` returns the entity resource or `404 not_found`.

## Patch Entity

Mutable fields:

- `subtype`
- `alias`
- named top-level sections under `json`
- named components under `json.components`

Immutable fields:

- `entity_id`
- `type`
- `created_at`
- `updated_at`

PATCH follows the global named-section replacement rules in [`conventions.md`](./conventions.md). Component inner shapes are not defined by this doc.
Component inner shapes are defined by the component contracts.

## Delete Entity

Entity delete returns `204 No Content` when successful.

Delete must be rejected with `409 conflict` while dependent tasks, observations, or entity-owned objects exist.

## List Entity Tasks

`GET /entities/{entity_id}/tasks` is valid only for asset entities. It returns the same task resource shape as [`tasks.md`](./tasks.md), filtered to `asset_id={entity_id}`.

Failures:

- `404 not_found` when the entity does not exist.
- `400 validation_failed` when the entity exists but is not an asset.
