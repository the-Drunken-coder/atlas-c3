# Atlas SDK Contract

This folder defines the public Atlas SDK contract.

Atlas Core owns the HTTP API. Atlas SDK owns the stable helper surface that makes that API easy and hard to misuse.

## Relationship To Core API

SDK docs should reference Core API contracts for raw endpoint behavior instead of redefining endpoint rules.

The SDK should wrap Core API calls rather than requiring extra Core endpoints for every asset, UI, simulation, or debugging workflow.

## Client Shape

The SDK should expose one `AtlasClient` with grouped namespaces:

- `entities`
- `observations`
- `tasks`
- `objects`
- `queries`
- `stream`
- `assetRuntime`

Assets are entities in the Core data model. Do not add a separate `assets` resource client unless it is clearly documented as a convenience wrapper over `entities`, not a separate Core resource.

Helpers meant for software running on an asset should live under the asset-runtime-focused namespace.

## Resource And Workflow Helpers

The SDK should expose both resource clients and workflow helpers.

Workflow helpers are the preferred developer-facing path when they remove repeated raw API sequences.

Expected asset-runtime helpers include:

- registration
- heartbeat
- current-state updates
- task reads
- task status updates
- observation submission
- object file submission

Task status transitions should be exposed as clear helpers:

- acknowledge
- reject
- start
- update progress
- complete
- fail

Observation submission should support creating an observation alone and submitting an observation with related files.

Object and file helpers should hide multipart upload details from SDK callers.

## Asset Helper Rules

Asset registration should be idempotent at the SDK level: read first, create if missing, and return or update the existing asset when already registered.

Heartbeat should be a dedicated SDK helper that uses Core entity update behavior underneath.

Asset state updates should start with a generic update helper. Convenience helpers such as location updates can be added when they remove real repetition.

Asset task access should support both request-based reads and server-sent event subscriptions.

## Streams And Recovery

Stream helpers should expose high-level subscription helpers.

The Core stream remains live-only and does not replay missed events. SDK recovery from missed stream events should use a fresh current-state read, not replay.

## Errors And Validation

SDK calls should throw typed SDK errors by default.

SDK errors should preserve Core error details, including:

- HTTP status
- Core error code
- Core error ID
- message
- details

SDK input validation should stay lightweight: required fields, ID length, obvious enum values, and similarly cheap checks. Atlas Core remains authoritative.

SDK response validation should exist where it is cheap and useful, especially for stream events and error envelopes.

## Raw HTTP Escape Hatch

The SDK should include a raw HTTP escape hatch for advanced tools and debugging.

The escape hatch is secondary to typed resource clients and workflow helpers.

## Versioning

The SDK and Atlas Core always target the current system version together.

Do not add API version negotiation, compatibility matrices, or multi-version support.
