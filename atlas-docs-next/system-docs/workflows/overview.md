# System Workflows

This folder describes cross-system workflows.

Workflow docs should explain behavior across modules without redefining API request and response shapes. Link to contracts for exact shapes.

## Shared Workflow Rules

Atlas Core remains the source of truth.

Clients recover missed or questionable state by refreshing current state through SDK behavior, not through event replay.

Atlas Command Interface should use Atlas SDK replica mode for startup, current-state reads, live updates, and disconnect recovery.

SDK replica mode should own:

- full-state query
- stream subscription
- stream reconnect
- local replica invalidation
- full refresh

## Command Catalog Startup

Atlas Core restart should rematerialize the checked-in command catalog as an object before reporting ready.

If command catalog materialization or validation fails, Atlas Core should fail startup/readiness with a clear error code and log context.

Command catalog behavior is defined in [`../../contracts/data-model/command-catalog/overview.md`](../../contracts/data-model/command-catalog/overview.md).

## Local Lifecycle

Local destructive restart and shutdown workflows are decided by [`../../decisions/0004-interactive-core-cli.md`](../../decisions/0004-interactive-core-cli.md).

Core runtime behavior is described in [`../../module-docs/atlas-core/runtime-shape.md`](../../module-docs/atlas-core/runtime-shape.md).

## Workflow Areas

Detailed workflow docs can be written for:

- asset comes online and becomes visible to operators
- operator creates a task for an asset
- asset receives, executes, and reports task progress
- asset produces an observation and related object files
- UI starts up through SDK replica mode and becomes live
- Atlas Core restarts and rematerializes the command catalog
- local destructive restart and shutdown
- debugging and simulation workflows

## Deferred

Observation evidence updating a track is deferred with data fusion planning.

Data fusion planning is part of Atlas Core and is documented in [`../../module-docs/atlas-core/data-fusion.md`](../../module-docs/atlas-core/data-fusion.md).
