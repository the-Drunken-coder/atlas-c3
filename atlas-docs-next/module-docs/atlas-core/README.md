# Atlas Core

Atlas Core is the central server and system of record described in [`../../system-docs/overview.md`](../../system-docs/overview.md).

This folder owns Atlas Core implementation planning. Shared API and data contracts belong in [`../../contracts/`](../../contracts/), and settled architecture choices belong in [`../../decisions/`](../../decisions/).

## Current Planning Docs

- [`container-system.md`](./container-system.md) - high-level Docker/container shape for running Atlas Core locally.
- [`runtime-shape.md`](./runtime-shape.md) - high-level local lifecycle flow and Python terminal UI expectations.
- [`implementation-structure.md`](./implementation-structure.md) - planned file and package structure for the Atlas Core implementation repo.
- [`storage-schema.md`](./storage-schema.md) - initial PostgreSQL table, constraint, index, and delete behavior plan.
- [`internal-interfaces.md`](./internal-interfaces.md) - high-level internal boundaries for database, object metadata, and object file access.
- [`data-fusion.md`](./data-fusion.md) - data fusion worker boundary for observation-to-track processing.
- [`stores/overview.md`](./stores/overview.md) - persistence-facing store boundaries and capabilities.
- [`services/overview.md`](./services/overview.md) - service-layer responsibilities between HTTP handlers and stores.

## Related Contracts And Decisions

- [`../../contracts/data-model/record-families.md`](../../contracts/data-model/record-families.md) - high-level Atlas Core record families and relationships.
- [`../../contracts/data-model/command-catalog/overview.md`](../../contracts/data-model/command-catalog/overview.md) - command catalog source, runtime object materialization, and task relationship.
- [`../../contracts/asset-core-protocol/overview.md`](../../contracts/asset-core-protocol/overview.md) - asset-facing behavior over the normal Core API and SDK helpers.
- [`../../contracts/auth-security/overview.md`](../../contracts/auth-security/overview.md) - trusted-client and no-auth operating model.
- [`../../contracts/core-api/overview.md`](../../contracts/core-api/overview.md) - Core API contract scope and planning order.
- [`../../contracts/core-api/endpoint-layout.md`](../../contracts/core-api/endpoint-layout.md) - planned Core API URL layout.
- [`../../contracts/core-api/conventions.md`](../../contracts/core-api/conventions.md) - shared Core API response, error, ID, pagination, update, and delete rules.
- [`../../contracts/core-api/planned-endpoints.md`](../../contracts/core-api/planned-endpoints.md) - high-level planned Core API endpoint groups.
- [`../../decisions/0001-first-class-observations.md`](../../decisions/0001-first-class-observations.md) - observations are first-class records, not object conventions.
- [`../../decisions/0002-core-storage-shape.md`](../../decisions/0002-core-storage-shape.md) - Core uses PostgreSQL for structured state and a filesystem volume for object bytes.
- [`../../decisions/0003-short-lived-operational-data.md`](../../decisions/0003-short-lived-operational-data.md) - Core data is short-lived operational state.
- [`../../decisions/0004-interactive-core-cli.md`](../../decisions/0004-interactive-core-cli.md) - Core uses an interactive Python CLI for local lifecycle management.
- [`../../decisions/0005-no-auth-current-operating-model.md`](../../decisions/0005-no-auth-current-operating-model.md) - Core does not implement auth in the current operating model.
