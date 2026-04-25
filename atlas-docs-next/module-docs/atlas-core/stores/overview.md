# Store Overview

This folder plans Atlas Core's persistence-facing store boundaries.

These docs define capabilities and ownership, not final Go interfaces or function names. Exact signatures should come later, after API payloads and table contracts are stable.

## Inputs

Store planning follows these contracts:

- [`../../../contracts/core-api/endpoint-layout.md`](../../../contracts/core-api/endpoint-layout.md)
- [`../../../contracts/core-api/conventions.md`](../../../contracts/core-api/conventions.md)
- [`../../../contracts/data-model/record-families.md`](../../../contracts/data-model/record-families.md)

## Store Boundaries

Atlas Core should have two persistence-facing store boundaries:

- [`regular-record-store.md`](./regular-record-store.md) - entities, observations, and tasks.
- [`object-store.md`](./object-store.md) - objects, object file metadata, and object file bytes.

[`query-support.md`](./query-support.md) describes how `GET /queries/full` should be assembled. It does not need to become a separate database store unless implementation pressure proves otherwise.

## Non-Stores

The command catalog does not need its own store. The source catalog is a version-controlled command catalog JSON file stored in the repository and loaded at application startup, then materialized through the object store.

The live stream does not need its own durable store. It is live-only and should publish mutation events from service-layer behavior.

## Service Layer Relationship

Stores should provide persistence capabilities. Services should own API behavior, validation, lifecycle rules, derived state, cross-store coordination, and event publication.

HTTP handlers should call services, not stores directly.

