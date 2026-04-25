# Atlas Core Container System

This document is a high-level plan for how Atlas Core should run in containers. It does not define API contracts, database table shapes, or implementation package names.

Storage shape is settled in [`../../decisions/0002-core-storage-shape.md`](../../decisions/0002-core-storage-shape.md). Core data is short-lived operational state per [`../../decisions/0003-short-lived-operational-data.md`](../../decisions/0003-short-lived-operational-data.md).

Local lifecycle management is described in [`runtime-shape.md`](./runtime-shape.md).

## Goal

The Atlas Core container system should make the full core service easy to run locally with real persistence and real object file storage.

The system should be simple enough to understand at a glance:

- one Atlas Core application container
- one PostgreSQL container for structured operational state
- one managed volume for object file bytes
- one private Docker network connecting the service to its storage dependencies
- optional trusted worker containers, such as data fusion, that connect through the Core API

## Planned Containers

### `atlas-core`

Runs the Core server process.

Responsibilities:

- load configuration from environment
- connect to PostgreSQL
- ensure the expected schema exists without a migration framework
- initialize object file storage access
- materialize startup-owned system records such as the command catalog
- expose HTTP endpoints
- report health and readiness

### `postgres`

Stores structured Core state.

Expected record families include:

- entities
- observations
- tasks
- objects
- object file metadata
- command catalog materialization records

Table and field contracts should live in `../../contracts/`, not in this container document.

### Object File Volume

Stores object file bytes outside PostgreSQL.

Atlas Core should treat stored file paths as logical paths managed by the application, not host-specific absolute paths. PostgreSQL should store metadata and logical references; the mounted volume should store the bytes.

Objects are the system's file-container abstraction. A file associated with an observation, task, command catalog, or other record should be stored through an object; the owning record links to that object rather than storing bytes directly.

### Optional `atlas-data-fusion`

Runs a data fusion worker when the deployment includes one.

Responsibilities:

- read observations through Atlas Core API or SDK
- read current entity/track state through Atlas Core API or SDK
- write track entities through Atlas Core API or SDK
- listen to Core stream events when useful
- log fusion decisions and processing durations

The worker should not use PostgreSQL as an integration surface or mutate object file bytes. Atlas Core remains the source of truth.

## Startup Flow

At a high level, startup should happen in this order:

1. Load configuration.
2. Connect to PostgreSQL.
3. Ensure required database structures exist.
4. Verify object file storage is available.
5. Materialize startup records that Core owns.
6. Start the HTTP server.
7. Mark readiness only after required dependencies are usable.

Optional worker containers should start after Atlas Core is reachable, but Atlas Core readiness should not depend on data fusion being present.

## Readiness Model

Health and readiness should be separate.

- Health answers whether the Core process is alive.
- Readiness answers whether Core can serve real requests that depend on PostgreSQL and object file storage.

Readiness should fail if Core cannot reach required persistence or storage dependencies.

## Non-Goals

The initial container system should not include:

- Redis
- a message broker
- a separate object storage service
- a separate UI backend
- a distributed deployment topology
- database migration or rollback services
- long-term archival backup machinery
- global Docker cleanup outside Atlas Core project resources

Those choices can be revisited through a decision record if the system later needs them.
