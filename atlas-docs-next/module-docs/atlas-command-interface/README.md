# Atlas Command Interface Module

This folder documents Atlas Command Interface implementation planning.

Shared contract expectations live in [`../../contracts/command-interface/overview.md`](../../contracts/command-interface/overview.md).

## Role

Atlas Command Interface is a separate trusted client of Atlas Core.

It should use Atlas SDK rather than raw Core API calls.

## SDK Usage

The Command Interface should use Atlas SDK replica mode for normal operator workflows.

SDK replica mode should handle:

- current-state bootstrap
- stream subscription
- disconnect recovery
- local state invalidation
- full-state refresh

The UI should read active command catalog information through SDK helpers.

## State And Authority

Atlas Core remains authoritative.

UI-derived display state is not authoritative.

UI-specific view models belong inside this module. Shared contracts should stay based on SDK/Core shapes unless another module needs a UI-specific shape.

Disconnected Core, stale stream state, and questionable local state should be handled through SDK replica mode invalidation and refresh behavior.

## Deferred UI Design

The following details are intentionally deferred until Command Interface planning starts in depth:

- primary operator screens
- navigation structure
- detailed task authoring workflow
- asset, track, observation, object, and task inspection workflows
- map and spatial display expectations
- testing approach for operator workflows
