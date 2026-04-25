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
- `replica`
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

Task status transitions should be exposed as clear helpers only where they map to the current task status lifecycle defined by Atlas Core.

Current SDK contract:

- acknowledge
- complete
- fail

Deferred until the task status model and API expand:

- reject
- start
- update progress

Observation submission should support creating an observation alone and submitting an observation with related files.

Object and file helpers should hide multipart upload details from SDK callers.

## Asset Helper Rules

Asset registration should be idempotent at the SDK level: read first, create if missing, and return or update the existing asset when already registered.

Heartbeat should be a dedicated SDK helper that uses Core entity update behavior underneath.

Asset state updates should start with a generic update helper. Convenience helpers such as location updates can be added when they remove real repetition.

Asset task access should support both request-based reads and server-sent event subscriptions.

## Replica Mode

Replica mode is the SDK-owned in-memory current-state cache for clients such as Atlas Command Interface.

Replica mode is always in-memory. It should not persist state to browser storage, files, IndexedDB, or any other durable local store.

Startup flow:

1. Enter `connecting` health state.
2. Call `GET /queries/full`.
3. Hydrate the in-memory replica from the full-query response.
4. Subscribe to `GET /stream/changes`.
5. Enter `healthy` after the full query and stream subscription are both active.

Read behavior:

- Replica reads return from memory.
- Replica reads do not trigger per-read Core API calls.
- Calling the same replica read many times is acceptable because it is local memory access.
- Non-replica raw/resource reads should not deduplicate simultaneous identical HTTP requests by default.

Write behavior:

- Writes go through SDK resource clients or workflow helpers.
- Successful write responses update and validate the in-memory replica.
- UI code must not mutate replica state directly.
- If a write succeeds but the replica cannot apply the returned resource, the SDK should mark replica health as `stale` and perform a full refresh.

Stream behavior:

- Replica mode applies stream event payloads directly.
- Core stream create/update events must include enough resource payload for direct replica updates.
- Missed stream events still require a fresh `GET /queries/full`.

Subscriptions:

- The SDK should expose framework-agnostic subscriptions for replica state changes.
- UI/framework packages may wrap those subscriptions.
- Subscription callbacks receive current derived state from the SDK, not mutable cache internals.

## Replica Health

Replica mode should expose a health primitive instead of returning stale flags from every read.

Health states:

| State | Meaning |
| --- | --- |
| `connecting` | Initial full-query or stream setup is in progress |
| `healthy` | Full-query state is loaded and stream is connected |
| `reconnecting` | Stream connection was lost and the SDK is trying to reconnect |
| `stale` | Cached reads are still served, but the SDK knows the replica may be outdated |
| `failed` | The SDK cannot currently refresh or reconnect |

When disconnected, the SDK should keep serving cached reads. Consumers that care about freshness should subscribe to the replica health primitive.

## Streams And Recovery

Stream helpers should expose high-level subscription helpers.

The Core stream remains live-only and does not replay missed events. SDK recovery from missed stream events should use a fresh full-state read, not replay.

## Errors And Validation

SDK calls should throw typed SDK errors by default.

SDK errors should preserve Core error details, including:

- HTTP status
- Core error code
- Core error ID
- message
- timestamp
- path
- details

SDK input validation should stay lightweight: required fields, ID length, obvious enum values, and similarly cheap checks. Atlas Core remains authoritative.

SDK response validation should exist where it is cheap and useful, especially for stream events and error envelopes.

## Raw HTTP Escape Hatch

The SDK should include a raw HTTP escape hatch for advanced tools and debugging.

The escape hatch is secondary to typed resource clients and workflow helpers.

## Versioning

The SDK and Atlas Core always target the current system version together.

Do not add API version negotiation, compatibility matrices, or multi-version support.
