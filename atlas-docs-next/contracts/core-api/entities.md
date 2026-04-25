# Entities API

Entity endpoints manage current-state records for assets, tracks, and geofeatures.

Entity record ownership is defined in [`../data-model/record-families.md`](../data-model/record-families.md). Shared API behavior is defined in [`conventions.md`](./conventions.md).

## Endpoints

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/entities` | List entities |
| `POST` | `/entities` | Create an entity |
| `GET` | `/entities/{entity_id}` | Read an entity |
| `PATCH` | `/entities/{entity_id}` | Update an entity |
| `DELETE` | `/entities/{entity_id}` | Delete an entity |
| `GET` | `/entities/{entity_id}/tasks` | List tasks assigned to an asset |

## Notes

Create requests must include `entity_id`.

Tasks target assets. `GET /entities/{entity_id}/tasks` should be valid for asset entities.

Entity-related objects should be queried through [`objects.md`](./objects.md) using `owner_type=entity` and `owner_id={entity_id}`.

