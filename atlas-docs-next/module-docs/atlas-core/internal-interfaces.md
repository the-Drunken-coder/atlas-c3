# Atlas Core Internal Interfaces

This document is a high-level plan for the internal Atlas Core boundaries that support the Core API.

The names below are planning names, not final package or function names.

Storage shape is settled in [`../../decisions/0002-core-storage-shape.md`](../../decisions/0002-core-storage-shape.md). Core data is short-lived operational state per [`../../decisions/0003-short-lived-operational-data.md`](../../decisions/0003-short-lived-operational-data.md).

## Planning Direction

Atlas Core internals should be shaped by the planned API and record-family contracts first. Store boundaries exist to support Core behavior; they should not become independent subsystems with their own product semantics.

The main input contracts are:

- [`../../contracts/core-api/planned-endpoints.md`](../../contracts/core-api/planned-endpoints.md)
- [`../../contracts/data-model/record-families.md`](../../contracts/data-model/record-families.md)

## Store Boundaries

Atlas Core should keep two persistence-facing store areas:

1. Regular record store.
2. Object store.

The service layer should compose those stores into API behavior.

Detailed store capabilities are documented in [`stores/overview.md`](./stores/overview.md).

## Regular Record Store

The regular record store owns structured operational records that are not object file containers.

Detailed capabilities are documented in [`stores/regular-record-store.md`](./stores/regular-record-store.md).

Expected responsibilities:

- create, read, update, list, and delete entities
- create, read, update, list, and delete observations
- create, read, update, list, and delete tasks
- support broad current-state reads needed by API query endpoints
- support transaction boundaries for multi-record operations
- report database health for readiness

Observation behavior should follow [`../../decisions/0001-first-class-observations.md`](../../decisions/0001-first-class-observations.md).

## Object Store

The object store owns object records, object file metadata, and file byte access as one coordinated internal area.

Detailed capabilities are documented in [`stores/object-store.md`](./stores/object-store.md).

Expected responsibilities:

- create, read, update, list, and delete objects
- create and delete object file metadata rows
- list files for an object
- track logical storage paths, content types, sizes, and usage hints
- support references from objects back to owning records, including observations
- write file bytes to a logical path
- open or stream file bytes
- delete file bytes
- stat file bytes
- enforce storage-root safety
- participate in transactions when metadata changes must stay consistent with another record
- report object storage health for readiness

Objects are the file-container system. Files for observations, command catalogs, tasks, or other records should be represented through object metadata and object file metadata, then linked to the owning record by contract.

The object store should still keep metadata and byte operations internally organized. They are one store boundary because the API treats objects and files as one resource area.

## Service Layer

The service layer coordinates stores into meaningful Core behavior.

Expected service areas:

- entity service
- observation service
- task service
- object service
- query service
- stream/event service

The service layer should own business rules such as validation, lifecycle transitions, derived state, and cross-record consistency. HTTP handlers should stay thin and should not contain persistence logic.

Detailed service responsibilities are documented in [`services/overview.md`](./services/overview.md).

The command catalog does not need its own service or store. It starts as checked-in JSON, is loaded by Atlas Core at startup, and is materialized through the object store as a normal object-backed payload.

## Transaction Model

Some operations will need to coordinate multiple stores.

Examples:

- creating an observation and linking an object that stores its media
- uploading a file and recording its object file metadata
- changing a task status and updating derived task queue state
- deleting a record and cleaning up references where the contract requires it

Atlas Core should use explicit transaction boundaries for structured database changes. File byte writes need a simple consistency strategy because filesystem writes do not roll back automatically with database transactions.

The exact transaction and cleanup rules belong in later contracts and implementation docs. They should protect normal runtime correctness, not create a long-term migration or rollback system.

## Settled Storage Shape

Atlas Core uses one PostgreSQL database for structured records and metadata, plus one filesystem volume for object file bytes. There is no separate object database service in the initial architecture.

