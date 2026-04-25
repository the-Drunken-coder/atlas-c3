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

Supported command capabilities should live in an asset component that lists command catalog command types the asset can perform.

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
