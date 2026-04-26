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

If command catalog materialization or validation fails, Atlas Core should fail startup/readiness with `catalog_unavailable` and log context. Modules must use that exact code for catalog startup/readiness failures.

Command catalog behavior is defined in [`../../contracts/data-model/command-catalog/overview.md`](../../contracts/data-model/command-catalog/overview.md).

## Local Lifecycle

Local destructive restart and shutdown workflows are decided by [`../../decisions/0004-interactive-core-cli.md`](../../decisions/0004-interactive-core-cli.md).

Core runtime behavior is described in [`../../module-docs/atlas-core/runtime-shape.md`](../../module-docs/atlas-core/runtime-shape.md).

## Workflow Areas

Detailed workflow docs:

- [`asset-online.md`](./asset-online.md) - asset comes online and becomes visible to operators.
- [`operator-creates-task.md`](./operator-creates-task.md) - operator creates a task for an asset.
- [`asset-executes-task.md`](./asset-executes-task.md) - asset receives, executes, and reports task lifecycle state.
- [`asset-submits-observation-files.md`](./asset-submits-observation-files.md) - asset reports observation sightings and related object files.
- [`core-startup-command-catalog.md`](./core-startup-command-catalog.md) - Atlas Core restarts and rematerializes the command catalog.

Workflow docs still needed later:

- UI starts up through SDK replica mode and becomes live
- local destructive restart and shutdown
- debugging and simulation workflows

## Deferred

Observation evidence updating a track is deferred with data fusion planning.

Data fusion worker boundaries are documented in [`../../module-docs/atlas-core/data-fusion.md`](../../module-docs/atlas-core/data-fusion.md). Fusion algorithms and observation-to-track association choices remain deferred.
