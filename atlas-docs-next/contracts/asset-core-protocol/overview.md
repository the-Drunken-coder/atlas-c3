# Asset-Core Protocol

This folder defines cross-system behavior between assets, Atlas SDK, and Atlas Core.

The protocol should not expand the Core API unless a new endpoint is clearly needed. Core owns a small HTTP surface; Atlas SDK owns asset-facing helper functions.

## Related Docs

- [`../sdk/overview.md`](../sdk/overview.md) defines the public SDK helper surface.
- [`../core-api/overview.md`](../core-api/overview.md) defines the underlying Core HTTP API.
- [`../data-model/entities.md`](../data-model/entities.md) defines assets as entities.
- [`../data-model/components/overview.md`](../data-model/components/overview.md) defines component rules.
- [`../data-model/command-catalog/overview.md`](../data-model/command-catalog/overview.md) defines command catalog behavior.

## Identity And Registration

Assets should not identify themselves on every API call.

Assets identify themselves by registering when they start up, then heartbeat continuously.

Asset registration should use entity create/read behavior for asset entities. The SDK can wrap that behavior as a `register` helper instead of requiring a dedicated Core registration endpoint.

Assets should be able to check whether they are already registered before trying to register again after reconnecting or restarting.

## Heartbeat And State

Heartbeat timing is asset-owned.

Different asset kinds and third-party integrations may send heartbeats at different intervals. Atlas Core should allow that instead of enforcing one global fixed interval.

Supported command capabilities live in the [`supported_commands`](../data-model/components/supported_commands.md) asset component. Asset entities without that component are invalid.

SDK helpers should expose permanent functions for heartbeat, telemetry, location, state, and supported command capability updates.

## Task Delivery

Task delivery should support both simple SDK requests and server-sent event subscriptions.

Assets should use SDK helpers for reading assigned tasks and updating task status.

Core remains the authority for task records and task status. Asset-side execution internals belong to asset implementers.

## Observations And Files

Assets should submit observations through SDK helpers that wrap the first-class observation API.

Observation files should be submitted through SDK helpers that wrap object and object file APIs.

## Trust Model

Assets are trusted clients.

Authentication and authorization between assets and Atlas Core are not part of the current operating model. The shared security contract is [`../auth-security/overview.md`](../auth-security/overview.md).

## Retry And Reconnect

Atlas Core should not own durable offline queues for assets.

SDK calls should surface failures. Assets may retry idempotent SDK helpers where appropriate.

When an asset reconnects, it should restore current state through normal SDK/Core behavior rather than relying on Core-held per-client replay state.

## Reconnect State Recovery

After any disconnect or missed live stream, assets should treat Atlas Core as the source of truth and **refresh current state** using normal reads (`GET /queries/full` where appropriate, or targeted `GET` calls), then resume heartbeats and incremental updates.

The live stream does not replay history. Recovery rules are shared with [`../core-api/stream.md`](../core-api/stream.md) and [`../../system-docs/operational-assumptions.md`](../../system-docs/operational-assumptions.md).

## Idempotency And Duplicate Writes

These rules are intentionally simple for trusted clients. They are not a substitute for authentication.

**Safe to retry**

- `GET` reads
- `PATCH` updates where the asset repeats the same intended final state (Core should apply last-write semantics per field rules; duplicate identical patches are acceptable)
- `POST /tasks/{task_id}/status` when the transition is already satisfied: Core should respond with success and the current task representation as an **idempotent no-op**

**Creates with creator-supplied IDs**

- Duplicate `POST` creates that collide on the same primary ID should return **`409 Conflict`**. The client should follow with **`GET`** on that ID to reconcile local state after ambiguous network failures.
- Assets should prefer **read-then-create** for registration-style flows (for example `GET /entities/{entity_id}` before deciding to `POST /entities`), but `409` plus `GET` remains the recovery path.

**Object file uploads**

- Reusing the same `file_id` for a second upload should fail with **`409`** (or another explicit client error) unless a later contract adds a deliberate replace operation. Clients must not assume silent overwrite.
- Object file append is not intrinsically idempotent. SDK helpers that append sighting JSONL after an ambiguous network failure should reconcile by reading current observation/file state rather than blindly repeating the same append.

**Observations and tasks**

- Observation `PATCH` and task `PATCH`/`status` behavior should follow the data model and Core API contracts. Retries after timeouts should preserve creator-supplied IDs and rely on `GET` plus `409` handling where creates collide.
