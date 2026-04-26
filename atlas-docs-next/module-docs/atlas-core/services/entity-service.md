# Entity Service

The entity service owns API behavior for entity endpoints.

API contract: [`../../../contracts/core-api/entities.md`](../../../contracts/core-api/entities.md)

## Responsibilities

- validate entity create, patch, and delete requests
- require caller-supplied `entity_id` on create
- enforce entity type rules for assets, tracks, and geofeatures
- coordinate entity persistence through the regular record store
- list tasks assigned to an asset
- publish entity mutation events after successful writes

## Store Usage

Uses [`../stores/regular-record-store.md`](../stores/regular-record-store.md).

The entity service should not access object files directly. Entity-related objects are queried through object service/store interfaces that implement the object API contract, such as methods accepting `owner_type=entity` and `owner_id={entity_id}` filters.

## Notes

Tasks target assets, so task listing should be meaningful for asset entities.

Exact component validation rules belong in data-model contracts when those are written.
