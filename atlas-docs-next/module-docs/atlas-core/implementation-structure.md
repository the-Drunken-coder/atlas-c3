# Atlas Core Implementation Structure

This document plans the file and folder structure for the Atlas Core implementation repo.

It translates the existing Core API, data model, store, service, runtime, and container plans into a concrete source layout. It does not define request/response payload fields or database table DDL.

## Goals

The implementation structure should:

- keep HTTP handlers thin
- keep business rules in services
- keep persistence behind store interfaces
- keep PostgreSQL metadata and filesystem bytes coordinated without pretending the filesystem is transactional
- keep command catalog bootstrap separate from normal request handling
- avoid database migrations
- avoid duplicate logic between API handlers, services, stores, and SDK-facing behavior

## Proposed Repository Layout

```text
atlas-core/
  cmd/
    atlas-core/
      main.go
  internal/
    app/
      app.go
      dependencies.go
      readiness.go
      shutdown.go
    config/
      config.go
      env.go
    httpapi/
      router.go
      middleware.go
      errors.go
      responses.go
      system_handlers.go
      entity_handlers.go
      observation_handlers.go
      task_handlers.go
      object_handlers.go
      query_handlers.go
      stream_handlers.go
      dto/
        system.go
        entity.go
        observation.go
        task.go
        object.go
        query.go
        stream.go
    model/
      entity.go
      observation.go
      task.go
      object.go
      command_catalog.go
      errors.go
      pagination.go
    service/
      entity_service.go
      observation_service.go
      task_service.go
      object_service.go
      query_service.go
      validation.go
      servicetest/
        stores.go
        events.go
        catalog.go
        fixtures.go
    store/
      regular_record_store.go
      object_store.go
      tx.go
    postgres/
      db.go
      schema.go
      sql/
        schema.sql
        entities.sql
        observations.sql
        tasks.sql
        objects.sql
        queries.sql
      regular_record_store.go
      object_metadata_store.go
      query_support.go
    objectfiles/
      store.go
      paths.go
      metadata.go
    catalog/
      loader.go
      validator.go
      materializer.go
      active_catalog.go
    sightingcatalog/
      loader.go
      validator.go
      active_catalog.go
    events/
      publisher.go
      sse.go
      event.go
      subscriber.go
    logging/
      logging.go
  command-catalog/
    catalog.json
  sighting-catalog/
    catalog.json
  data-fusion/
    harness/
      Dockerfile
      docker-compose.fragment.yml
      runner/
      config/
    stacks/
      baseline/
    tests/
      fixtures/
      scenarios/
  tools/
    atlas-core-cli/
      main.py
      atlas_core_cli/
        compose.py
        menu.py
        confirm.py
  docker-compose.yml
  Dockerfile
  .env.example
  go.mod
  go.sum
  README.md
```

Exact file names may change during implementation if a file becomes too large or a package boundary proves awkward. The package boundaries should stay stable unless a later planning update changes them.

## Top-Level Folders

### `cmd/atlas-core/`

Owns the Go process entrypoint.

`main.go` should stay small:

- load configuration
- initialize logging
- call the application startup path
- handle process shutdown

It should not contain API routing, database logic, command catalog validation, or business rules.

### `internal/`

Owns private Go packages used by the Atlas Core process.

Atlas Core should not expose reusable Go packages as a public API in the first implementation. Cross-module integration happens through HTTP contracts and the SDK, not direct Go imports.

### `command-catalog/`

Owns the checked-in command catalog JSON source.

Atlas Core loads this file at startup, validates it, materializes it as an object-backed payload, and keeps the active catalog available for task validation.

Command catalog behavior is defined in [`../../contracts/data-model/command-catalog/overview.md`](../../contracts/data-model/command-catalog/overview.md).

### `sighting-catalog/`

Owns the checked-in sighting catalog JSON source.

Atlas Core loads this file at startup, validates it, and keeps the active catalog available for observation and sighting validation. Unlike the command catalog, the sighting catalog is not materialized as an object and is not exposed through a public Core API.

Sighting catalog behavior is defined in [`../../contracts/data-model/sighting-catalog.md`](../../contracts/data-model/sighting-catalog.md).

### `data-fusion/`

Owns the data fusion worker harness and selectable fusion stacks.

Data fusion remains outside the Atlas Core process and must communicate with Atlas Core through the Core API or SDK. This folder may live in the Atlas Core implementation repo so local development, Docker wiring, and tests are easy to run together, but fusion algorithm code must not be imported into the Core server process.

Expected shape:

- `harness/` - Docker/container runner, startup wiring, shared worker runtime, and stack selection.
- `harness/config/` - configuration that selects the active fusion stack.
- `stacks/{stack_name}/` - one complete fusion algorithm/behavior stack per folder.
- `tests/` - fusion harness tests, fixtures, and scenarios shared across stacks.

Only one stack should be active at a time. Switching stacks should be a configuration change, not a Core code change. The first implementation should include one `baseline` stack and keep test harness code outside individual stack folders unless a test is truly stack-private.

Data fusion boundaries are defined in [`data-fusion.md`](./data-fusion.md).

### `tools/atlas-core-cli/`

Owns the interactive Python CLI for local lifecycle management.

The CLI should manage Docker Compose start, destructive restart, and destructive shutdown behavior described in [`runtime-shape.md`](./runtime-shape.md). It should not become a general Docker cleanup tool.

Expected files:

- `main.py` - CLI entrypoint
- `atlas_core_cli/menu.py` - interactive menu rendering and action selection
- `atlas_core_cli/confirm.py` - typed destructive confirmation behavior
- `atlas_core_cli/compose.py` - Docker Compose command execution and progress reporting

The CLI should not contain Atlas Core API business logic.

### Docker Files

`docker-compose.yml` and `Dockerfile` belong at the Atlas Core repo root.

The compose project should match [`container-system.md`](./container-system.md) and [`../../system-docs/deployment-topology.md`](../../system-docs/deployment-topology.md).

`.env.example` should document local defaults without storing secrets. The current no-auth operating model means it should not contain API credentials.

## Internal Packages

### `internal/app/`

Owns application assembly and startup orchestration.

Responsibilities:

- create database connection
- initialize filesystem object storage
- initialize stores
- initialize services
- load and materialize the command catalog
- create HTTP router/server
- expose readiness state
- coordinate graceful shutdown

This package wires dependencies together. It should not own resource-specific business rules.

### `internal/config/`

Owns environment configuration parsing.

Configuration should cover:

- Core API bind address and port
- PostgreSQL connection details
- object file storage root
- command catalog file path
- sighting catalog file path
- active data fusion stack name when the optional worker is enabled
- CORS defaults for the Command Interface
- logging settings

Defaults should align with [`../../system-docs/deployment-topology.md`](../../system-docs/deployment-topology.md).

Expected environment variables:

- `ATLAS_CORE_HOST`
- `ATLAS_CORE_PORT`
- `ATLAS_CORE_DATABASE_URL`
- `ATLAS_CORE_OBJECT_STORAGE_ROOT`
- `ATLAS_CORE_COMMAND_CATALOG_PATH`
- `ATLAS_CORE_SIGHTING_CATALOG_PATH`
- `ATLAS_DATA_FUSION_STACK`
- `ATLAS_CORE_ALLOWED_ORIGINS`
- `ATLAS_CORE_LOG_LEVEL`

The final names can change during implementation, but configuration should stay explicit and documented in `.env.example`.

### `internal/httpapi/`

Owns HTTP routing, request decoding, response encoding, and API error serialization.

Handlers should:

- decode path/query/body input
- call the appropriate service
- serialize service results
- use shared error handling

Handlers should not:

- call PostgreSQL directly
- write file bytes directly
- contain lifecycle rules
- validate command catalog parameters directly
- publish stream events directly

The route groups should follow [`../../contracts/core-api/endpoint-layout.md`](../../contracts/core-api/endpoint-layout.md).

#### `internal/httpapi/dto/`

Owns HTTP-facing request and response structs.

DTO structs are allowed to differ from internal models when that keeps API parsing clear. They should stay aligned with the Core API contracts and should not leak PostgreSQL implementation details.

Expected DTO files:

- `system.go` - service descriptor, health, and readiness payloads
- `entity.go` - entity create, patch, list, and response DTOs
- `observation.go` - observation create, patch, list, and response DTOs
- `task.go` - task create, patch, status transition, list, and response DTOs
- `object.go` - object create, patch, file metadata, upload, and response DTOs
- `query.go` - full current-state query response DTOs
- `stream.go` - SSE event payload DTOs once the stream contract is detailed

DTO packages should not contain service logic. Conversion between DTOs and models should stay simple and local to handlers or small helper functions.

### `internal/model/`

Owns shared Go structs used inside Atlas Core.

This package should include internal representations for:

- entities
- observations
- tasks
- objects
- object files
- pagination metadata
- Core error envelopes

Keep this package boring. It should not become a service layer or persistence layer.

Expected model files:

- `entity.go` - entity and entity component containers
- `observation.go` - observation record and current sighting summary
- `task.go` - task record, status, command, progress, result, and error containers
- `object.go` - object and object file metadata
- `command_catalog.go` - parsed catalog definitions used by task validation
- `errors.go` - internal Core error type and error codes
- `pagination.go` - pagination request and response metadata

### `internal/service/`

Owns business behavior between handlers and stores.

Expected services:

- entity service
- observation service
- task service
- object service
- query service

Services should own:

- validation that depends on Core rules
- lifecycle transitions
- command catalog checks for task creation
- cross-store coordination
- event publication after successful mutations
- response-ready orchestration

Detailed service responsibilities live in [`services/overview.md`](./services/overview.md).

`validation.go` may hold small shared validation helpers used by multiple services, such as ID length checks or common timestamp checks.

Validation that is specific to one service should stay with that service instead of becoming a broad validation framework.

#### `internal/service/servicetest/`

Owns test-only helpers for service tests.

Expected helpers:

- fake regular record store
- fake object store
- event publisher recorder
- active command catalog fixtures
- transaction recorder
- record fixture builders

This package should only be used by tests. It should not be imported by production code.

Service testing expectations are documented in [`services/testing.md`](./services/testing.md).

### `internal/store/`

Owns store interfaces and transaction abstractions.

Expected interfaces:

- regular record store
- object store
- transaction boundary helpers

This package defines what services need from persistence. It should not know PostgreSQL SQL strings or filesystem path details.

Detailed store boundaries live in [`stores/overview.md`](./stores/overview.md).

### `internal/postgres/`

Owns PostgreSQL-backed store implementations.

Responsibilities:

- open and verify database connections
- ensure expected schema exists without a migration framework
- implement regular record persistence
- implement object and object file metadata persistence
- support query assembly for `GET /queries/full`
- provide transaction support for structured metadata changes

`schema.go` may create or verify the current expected schema at startup. It is not a migration system and should not grow rollback/versioning behavior.

#### `internal/postgres/sql/`

Owns SQL text organized by resource area.

Expected SQL files:

- `schema.sql` - current expected schema creation or verification statements
- `entities.sql` - entity CRUD and list queries
- `observations.sql` - observation CRUD and list queries
- `tasks.sql` - task CRUD, list, and status queries
- `objects.sql` - object and object file metadata queries
- `queries.sql` - broad current-state query support

These files are not migrations. They represent the current schema and current query set for the short-lived operational database.

If embedding SQL in Go proves simpler for a small query, implementation may do that. The rule is that SQL should remain organized by resource area and should not be duplicated across stores.

### `internal/objectfiles/`

Owns filesystem byte storage.

Responsibilities:

- validate logical file paths
- keep writes under the configured storage root
- write file bytes
- open or stream file bytes
- delete file bytes
- stat file bytes
- report storage availability for readiness

Object file metadata stays in PostgreSQL. This package owns bytes and storage-root safety.

Expected files:

- `store.go` - filesystem byte store implementation
- `paths.go` - logical path validation and storage-root safety
- `metadata.go` - helpers for byte size, content type, and file stat behavior when needed

### `internal/catalog/`

Owns command catalog bootstrap behavior.

Responsibilities:

- load checked-in catalog JSON
- validate catalog structure
- expose the active in-memory catalog for task validation
- materialize the catalog as an object-backed payload during startup
- fail startup/readiness when the catalog is invalid

The command catalog does not need a store or service package of its own.

### `internal/sightingcatalog/`

Owns sighting catalog bootstrap behavior.

Responsibilities:

- load checked-in sighting catalog JSON
- validate catalog structure
- expose the active in-memory catalog for observation validation
- fail startup/readiness when the catalog is invalid

The sighting catalog does not need a store, service package, object materialization path, or public API endpoint.

### `internal/events/`

Owns live change publication and SSE support.

Responsibilities:

- define internal event envelope
- publish create, update, and delete events after successful mutations
- support connected SSE subscribers
- avoid durable event replay

Event publication should follow commit-first, best-effort behavior described in [`services/event-publisher.md`](./services/event-publisher.md).

Expected files:

- `event.go` - internal event type and resource/action naming
- `publisher.go` - in-process publisher interface and implementation
- `subscriber.go` - subscriber registration and cleanup
- `sse.go` - SSE connection writing and keepalive behavior

### `internal/logging/`

Owns logging setup and shared helpers.

Logs should make startup, readiness failures, validation failures, storage failures, event publication failures, task transitions, observations, and object file operations diagnosable.

Logging expectations are described in [`../../system-docs/operational-assumptions.md`](../../system-docs/operational-assumptions.md).

## Implementation Sequence

Build Atlas Core in dependency order so each layer can be tested before the next layer depends on it.

Recommended order:

1. Create `go.mod`, configuration loading, logging setup, and process entrypoint.
2. Add PostgreSQL connection handling, schema creation or verification, and readiness checks.
3. Add filesystem object storage initialization and readiness checks.
4. Define internal models, Core errors, pagination, and store interfaces.
5. Add PostgreSQL stores for entities, observations, tasks, objects, object files, and full-query support.
6. Provide object file byte storage and path safety.
7. Load and validate the command catalog, active catalog state, and startup materialization through the object store.
8. Introduce sighting catalog loading, validation, and active in-memory catalog state.
9. Provide services with test-only store fakes and event publisher recorders.
10. Define HTTP DTOs, handlers, shared error serialization, and router wiring.
11. Enable in-process event publishing and SSE stream handling.
12. Configure Dockerfile, Docker Compose, `.env.example`, and the Python lifecycle CLI.
13. Add the optional data fusion harness and baseline stack wiring.
14. Add integration tests that run against real PostgreSQL and a temporary object file root.

This order keeps most behavior testable before the HTTP surface is complete and avoids treating Docker or the CLI as the first proof that Core works.

## Tests

Tests should live next to the package being tested unless a full integration test needs a separate folder.

Expected test areas:

- HTTP handler request decoding and error serialization
- service validation and lifecycle behavior
- PostgreSQL store behavior
- object file path safety
- command catalog validation and materialization
- sighting catalog validation
- SSE event publication behavior
- data fusion harness stack selection
- `GET /queries/full` snapshot assembly
- readiness behavior when PostgreSQL or object file storage is unavailable

Expected test file placement:

- handler tests next to `internal/httpapi` files
- service tests next to `internal/service` files
- store integration tests next to `internal/postgres` files
- object file tests next to `internal/objectfiles` files
- catalog tests next to `internal/catalog` files
- sighting catalog tests next to `internal/sightingcatalog` files
- event publisher tests next to `internal/events` files
- fusion harness tests under `data-fusion/tests`
- CLI tests next to `tools/atlas-core-cli` files if the CLI grows enough logic to justify them

Integration tests may use real PostgreSQL and a temporary filesystem root. Mocking data is for tests only and should not create fake dev/prod paths.

Service-specific testing guidance is [`services/testing.md`](./services/testing.md).

## Explicit Non-Goals

The implementation structure should not add:

- database migrations
- rollback systems
- a separate object storage service
- a message broker for live changes
- a UI backend
- auth middleware
- durable asset offline queues
- data fusion algorithm implementation files in the Atlas Core server process

Data fusion runs as a separate worker boundary when present. Algorithm stacks may live under the top-level `data-fusion/stacks/` folder for local development and Docker packaging, but they must not become packages imported by the Core server process. See [`data-fusion.md`](./data-fusion.md).
