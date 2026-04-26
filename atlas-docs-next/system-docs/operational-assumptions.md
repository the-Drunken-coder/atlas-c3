# Operational Assumptions

This document defines cross-system assumptions that Atlas Core, Atlas SDK, Atlas Command Interface, asset runtimes, and future modules can rely on.

Contract details belong in [`../contracts/`](../contracts/). Module implementation details belong in [`../module-docs/`](../module-docs/). Durable architecture choices belong in [`../decisions/`](../decisions/).

## Short-Lived Operational State

Atlas Core stores short-lived operational state, not permanent enterprise history.

The accepted decision is [`../decisions/0003-short-lived-operational-data.md`](../decisions/0003-short-lived-operational-data.md). Core records should support a bounded operational run: current entities, observations, tasks, objects, object file metadata, command catalog materialization, and related current-state data.

Atlas Core should not include migration frameworks, schema rollback machinery, archival retention systems, backup workflows, or historical database compatibility layers. These are not deferred v2 features; they are outside the intended operating model.

Contracts still need to be precise because modules depend on shared shapes even when stored data is short-lived.

## Reset And Destructive Local Workflows

Destructive reset is acceptable for development, simulation, debugging, and bounded deployments.

When stored state no longer matches the expected schema or scenario, the preferred path is reset and rebuild rather than migration or compatibility code. Delete operations are also acceptable for planned record families where the resource contracts allow them.

## Trusted Asset Clients

Assets, Atlas Command Interface, SDK consumers, simulations, debugging tools, and asset runtimes are trusted clients.

Authentication and authorization are not part of the current operating model. The decision is [`../decisions/0005-no-auth-current-operating-model.md`](../decisions/0005-no-auth-current-operating-model.md), and the shared contract is [`../contracts/auth-security/overview.md`](../contracts/auth-security/overview.md).

Asset-facing contracts should focus on registration, heartbeat, state reporting, task handling, observations, object file submission, retries, and reconnect behavior without adding auth assumptions.

If a future deployment has external security requirements, that should be treated as a different operating model rather than a hidden requirement in the current one.

## Client Disconnects

Atlas Core should assume clients can disconnect.

The live change stream is not durable and does not replay missed history. When a UI, SDK client, or asset reconnects after missing updates, it should refresh from Atlas Core's current state instead of expecting Core to maintain a per-client event backlog.

The stream behavior is defined in [`../contracts/core-api/stream.md`](../contracts/core-api/stream.md).

Atlas Command Interface should use SDK replica mode for current-state bootstrap, live updates, stream disconnect recovery, and full refresh. SDK behavior is defined in [`../contracts/sdk/overview.md`](../contracts/sdk/overview.md).

Asset reconnect, resubmission after ambiguous failures, and basic idempotency expectations are defined in [`../contracts/asset-core-protocol/overview.md`](../contracts/asset-core-protocol/overview.md) under **Reconnect State Recovery** and **Idempotency And Duplicate Writes**. Atlas Core should implement those HTTP-level rules consistently so SDK helpers can expose predictable retry behavior.

## Storage Availability

Atlas Core depends on PostgreSQL for structured state and a filesystem volume for object file bytes.

The storage decision is [`../decisions/0002-core-storage-shape.md`](../decisions/0002-core-storage-shape.md). Readiness should fail when required PostgreSQL or object file storage dependencies are unavailable.

Requests that need unavailable storage should fail with the standard Core API error envelope. Atlas Core should not pretend object bytes are available when the filesystem volume is unavailable.

Existing structured reads may still work when PostgreSQL is healthy, but the system should not claim readiness without object file storage.

File write ordering and failure behavior are decided in [`../decisions/0006-object-file-write-ordering.md`](../decisions/0006-object-file-write-ordering.md).

Core referential integrity and delete rules are decided in [`../decisions/0007-core-referential-integrity.md`](../decisions/0007-core-referential-integrity.md).

## Event Publication Failures

Atlas Core's stored state remains the source of truth.

If a mutation succeeds but event publication fails, the mutation should remain committed. Atlas Core should log the publication failure, and clients should recover by refreshing current state. The current operating model should not add a durable event log, message broker, or replay queue for this case.

## Logging And Debugging

Logging is a first-class operational requirement.

ATLAS-C3 behavior will include non-deterministic systems, simulations, field integrations, and task execution paths that can be difficult to evaluate from final state alone. The system should produce detailed logs that can be exported and analyzed after a run, including by large language models.

The structured logging contract is [`observability.md`](./observability.md).

Core logging should capture at minimum:

- startup, shutdown, and configuration summary
- readiness and dependency failures
- API write failures and validation failures
- storage failures, including object file operations
- event publication failures
- asset registration, heartbeat, reconnect, and disconnect-relevant state
- task lifecycle transitions and task execution updates
- observation creation and update activity
- object and object file create, delete, and upload activity

Logs should avoid fake success states. If a write, file operation, event publish, or task update fails, the logs should make the failure visible enough to diagnose the sequence that led to it.

Logs should use the shared run logging fields, correlation IDs, and category structure defined in the observability contract.

## Simulation And Debugging

Simulation and debugging workflows are first-class uses of the system.

Simulation assets must use the same real Atlas Core APIs, SDK helpers, task lifecycle rules, observation paths, and object file paths as other assets, and should not introduce fake product data paths or mock-only runtime behavior outside tests.

Reset, delete, and rebuild workflows are acceptable tools for simulation and local debugging.

## Explicit Non-Goals

The current operating model excludes:

- asset-to-Core authentication and authorization
- operator roles and resource permissions
- database migration frameworks
- schema rollback systems
- archival retention and backup machinery
- durable event replay
- message brokers for the live-change path under the current operating model
- multi-tenant permission systems
- complex distributed conflict-resolution machinery
- fake or mocked product data in development or production behavior

These are not expected follow-on features for later versions under the same operating model. Adding one would require changing the system assumptions, not simply filling in missing implementation detail.
