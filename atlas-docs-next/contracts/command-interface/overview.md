# Command Interface Contract

This folder defines shared contract expectations for Atlas Command Interface.

Screen layout, navigation, UI state machines, and internal view models belong in [`../../module-docs/atlas-command-interface/`](../../module-docs/atlas-command-interface/).

## Data Contract

Atlas Command Interface should use Atlas SDK instead of hand-building raw Core API calls.

The shared data contract is the SDK/Core resource shape. UI-specific view models are internal to the Command Interface unless another module later depends on them.

## SDK Mode

Atlas Command Interface should run Atlas SDK in replica mode for normal operation.

SDK replica mode should handle:

- current-state bootstrap
- live stream subscription
- stream disconnect recovery
- questionable state invalidation
- full-state refresh

The UI should not duplicate this synchronization behavior locally.

## Command Catalog Discovery

The Command Interface should discover the active command catalog through SDK helpers that wrap the Core service descriptor and object API.

The command catalog contract is [`../data-model/command-catalog/overview.md`](../data-model/command-catalog/overview.md).

## Task Authoring

Task authoring should produce normal Core task-create payloads through SDK helpers unless a later contract proves a separate shared task-authoring payload is needed.

Task API behavior is [`../core-api/tasks.md`](../core-api/tasks.md). Task data shape is [`../data-model/tasks.md`](../data-model/tasks.md).

## Authority Boundaries

Atlas Core remains authoritative for:

- records
- object metadata
- object file bytes

Atlas SDK may validate and transform data for convenience.

Atlas Command Interface may derive display state, but derived UI state is not authoritative.

## Module-Owned Behavior

UI loading states, error presentation, map display shapes, operator workflow state, and internal view models are module behavior unless they become observable shared contracts.
