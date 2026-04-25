# Atlas SDK Module

This folder documents how Atlas SDK should be built and organized.

The public SDK surface is a shared contract and lives in [`../../contracts/sdk/overview.md`](../../contracts/sdk/overview.md). Asset-facing behavior is defined in [`../../contracts/asset-core-protocol/overview.md`](../../contracts/asset-core-protocol/overview.md).

## Role

Atlas SDK is the main developer-facing way to use Atlas Core.

It should centralize Core endpoint logic instead of spreading raw HTTP calls across assets, simulations, tools, and the Command Interface.

## Package Shape

Use one package with internal modules such as:

- `client`
- `http`
- `errors`
- `resources`
- `workflows`
- `stream`
- `replica`
- `files`
- `types`

Keep public exports small:

- `AtlasClient`
- public types
- SDK errors
- client configuration types

## Types

Maintain TypeScript types manually from the docs at first.

Do not add code generation until the contracts are stable enough to justify it.

The SDK and Atlas Core always target the current system version together. Do not add multi-version compatibility machinery.

## HTTP Wrapper

Use a fetch-based internal HTTP wrapper.

The wrapper should centralize:

- base URL handling
- headers
- request construction
- response parsing
- error conversion

Allow injecting `fetch` for tests and nonstandard runtimes.

## Errors

Use one primary SDK error class with fields for:

- HTTP status
- Core error code
- Core error ID
- message
- details
- cause

Add subclasses only if implementation proves they are useful.

## Retries

Do not implement automatic retries.

If a request fails, surface the error to the caller. Workflow helpers may be idempotent where the underlying Core behavior supports it, but the SDK should not hide failed writes behind implicit retry behavior.

## Operating Modes

Carry forward the request mode and replica mode system.

Request mode is the default. It is stateless, and each SDK call maps to Core API requests.

Replica mode maintains an in-memory current-state replica by performing a full query, subscribing to server-sent events, and applying successful changes to local state.

Both modes should expose the same public SDK method surface so callers choose the mode at initialization without rewriting business logic.

Replica mode is in-memory only. It is not durable storage, not offline-first, and does not persist across process restarts.

Replica mode writes should go through Atlas Core first. The local replica should update only after a successful Core response or confirmed stream update.

If the stream disconnects, replica mode should reset or invalidate the local replica, perform a fresh full-state read, reconnect, and resume.

Replica mode should support periodic full refresh as a consistency backstop. The default interval should be 20 seconds and should be configurable.

## Testing

The SDK should have broad test coverage for:

- request construction
- response parsing
- error conversion
- request mode behavior
- replica mode cache reads
- replica refresh behavior
- stream handling
- upload and download helpers
- asset runtime helpers

## Build Expectations

Build expectations should stay simple:

- TypeScript strict mode
- normal lint and format checks
- npm package output
- no heavy framework

Runtime support should target modern Node and browser-compatible fetch.
