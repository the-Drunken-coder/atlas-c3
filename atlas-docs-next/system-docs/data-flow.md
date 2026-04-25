# Data Flow

This document describes how operational data moves through ATLAS-C3 and where authority lives.

Exact request and response shapes belong in [`../contracts/`](../contracts/). Module implementation details belong in [`../module-docs/`](../module-docs/).

## Authority

Atlas Core is authoritative for:

- records
- object metadata
- object file bytes

Atlas SDK may validate, transform, and simplify data access for callers.

Atlas Command Interface may derive display state, but derived UI state is not authoritative.

## Asset State

Asset state flows into Atlas Core through SDK helpers that wrap entity create, read, and update behavior.

Assets are entities in the Core data model. The asset-facing protocol is [`../contracts/asset-core-protocol/overview.md`](../contracts/asset-core-protocol/overview.md).

## Observations And Files

Observations flow into Atlas Core as first-class observation records.

Observation files flow into Atlas Core as objects and object files owned by the observation.

Observation data is defined in [`../contracts/data-model/observations.md`](../contracts/data-model/observations.md). Object and file ownership is defined in [`../contracts/data-model/objects.md`](../contracts/data-model/objects.md).

## Tasks

Tasks flow from operators through Atlas Command Interface and Atlas SDK into Atlas Core task records.

Assets read assigned tasks through SDK helpers over Core task/entity APIs and stream behavior.

Task status flows back into Atlas Core through SDK helpers that wrap task status/update behavior.

Task data is defined in [`../contracts/data-model/tasks.md`](../contracts/data-model/tasks.md).

## Events And Recovery

Core writes publish live stream events after successful mutations.

UI clients should use SDK replica mode for current-state bootstrap, live updates, stream disconnect recovery, and full refresh.

Missed events are recovered by refreshing current state, not by replaying durable events.

The stream contract is [`../contracts/core-api/stream.md`](../contracts/core-api/stream.md). SDK stream recovery behavior is defined in [`../contracts/sdk/overview.md`](../contracts/sdk/overview.md).

## Failure Paths

Failed uploads and object file storage failures should surface as clear failures.

Atlas Core should not claim readiness without object file storage.

Stale task state should be resolved by reading current task state from Core through the SDK.

## Data Fusion Data Flow

Observation-to-track algorithm details are deferred.

Data fusion should run as a trusted worker container when present. It consumes observations through Core contracts and writes track entities through Core contracts. Fusion boundary planning belongs in [`../module-docs/atlas-core/data-fusion.md`](../module-docs/atlas-core/data-fusion.md).
