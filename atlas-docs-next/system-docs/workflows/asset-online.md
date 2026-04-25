# Asset Comes Online

This workflow describes how an asset becomes visible to operators.

Exact entity payloads are defined in [`../../contracts/core-api/entities.md`](../../contracts/core-api/entities.md). Asset/Core protocol rules are defined in [`../../contracts/asset-core-protocol/overview.md`](../../contracts/asset-core-protocol/overview.md).

## Flow

1. Asset chooses a stable `entity_id` for the current operating model.
2. Asset reads `GET /entities/{entity_id}`.
3. If the entity exists, the asset treats it as the current server record.
4. If the entity does not exist, the asset creates it with `POST /entities` using `type: "asset"`.
5. If create returns `409 conflict`, the asset reads `GET /entities/{entity_id}` and reconciles.
6. Asset updates heartbeat and state through `PATCH /entities/{entity_id}`.
7. Clients observe the change through `entity.created` or `entity.updated` stream events, then read the entity if they need current state.

## Notes

Heartbeat and state component inner shapes are intentionally not defined by this workflow. They belong to future entity component contracts.

Atlas Core does not require an asset-specific registration endpoint. Registration is normal entity create/read behavior.
