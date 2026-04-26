# Atlas Core Build Plan

This plan breaks Atlas Core into six ordered implementation stages. Each stage is meant to be one reviewable PR that can be built, reviewed, adjusted, merged, and then used as the base for the next stage.

The intended builder is a strong coding agent. The plan therefore spells out system boundaries, sequencing, invariants, and verification work, while leaving ordinary code writing decisions to the implementation model.

## Source Of Truth

Do not invent API shapes, database fields, component rules, stream event payloads, error envelopes, or workflow behavior in this plan. Use these docs as the authoritative contract set:

- Core implementation layout: [`atlas-docs-next/module-docs/atlas-core/implementation-structure.md`](./atlas-docs-next/module-docs/atlas-core/implementation-structure.md)
- Internal boundaries: [`atlas-docs-next/module-docs/atlas-core/internal-interfaces.md`](./atlas-docs-next/module-docs/atlas-core/internal-interfaces.md)
- Storage schema: [`atlas-docs-next/module-docs/atlas-core/storage-schema.md`](./atlas-docs-next/module-docs/atlas-core/storage-schema.md)
- Core API contracts: [`atlas-docs-next/contracts/core-api/overview.md`](./atlas-docs-next/contracts/core-api/overview.md)
- Data model contracts: [`atlas-docs-next/contracts/data-model/record-families.md`](./atlas-docs-next/contracts/data-model/record-families.md)
- Runtime/container docs: [`atlas-docs-next/module-docs/atlas-core/runtime-shape.md`](./atlas-docs-next/module-docs/atlas-core/runtime-shape.md), [`atlas-docs-next/module-docs/atlas-core/container-system.md`](./atlas-docs-next/module-docs/atlas-core/container-system.md)
- Operational assumptions: [`atlas-docs-next/system-docs/operational-assumptions.md`](./atlas-docs-next/system-docs/operational-assumptions.md)
- Accepted decisions: [`atlas-docs-next/decisions/`](./atlas-docs-next/decisions/)

## Global Build Rules

- Build Atlas Core as a Go service in an `atlas-core/` implementation root unless the repository structure changes before implementation starts.
- Keep HTTP handlers thin. Business rules belong in services. Persistence belongs behind store interfaces.
- Do not add database migrations. Startup creates or verifies the current expected schema.
- Do not add auth, API keys, roles, credential scaffolding, or fake security paths.
- Do not add mock/dev product paths. Test fakes are allowed only in tests.
- Do not add durable event replay, message brokers, archival history, backups, or schema compatibility layers.
- Keep object file bytes on the filesystem volume and object/object-file metadata in PostgreSQL.
- Treat committed PostgreSQL object-file metadata as the source of truth, per ADR 0006.
- Publish stream events after successful mutations, but never roll back committed writes because event publication failed.
- Make logs JSON Lines and include the required observability fields from [`atlas-docs-next/system-docs/observability.md`](./atlas-docs-next/system-docs/observability.md).

## Stage 1: Core Foundation And Schema Bootstrap

### Goal

Create the runnable Atlas Core process skeleton, configuration, logging, common models, error types, PostgreSQL connection, schema bootstrap, and basic health/readiness surface.

By the end of this stage, Atlas Core should start locally against PostgreSQL, create or verify its schema, initialize readiness state, and answer the system endpoints needed to prove the process is alive and dependency-aware.

### Primary Docs

- [`implementation-structure.md`](./atlas-docs-next/module-docs/atlas-core/implementation-structure.md)
- [`storage-schema.md`](./atlas-docs-next/module-docs/atlas-core/storage-schema.md)
- [`system.md`](./atlas-docs-next/contracts/core-api/system.md)
- [`conventions.md`](./atlas-docs-next/contracts/core-api/conventions.md)
- [`errors.md`](./atlas-docs-next/contracts/core-api/errors.md)
- [`observability.md`](./atlas-docs-next/system-docs/observability.md)

### System Design

Create these package boundaries first:

- `cmd/atlas-core`: tiny process entrypoint.
- `internal/config`: environment parsing and defaults.
- `internal/logging`: JSONL logger setup, run IDs, common log fields.
- `internal/model`: record structs, JSON metadata containers, pagination, and Core error type.
- `internal/postgres`: database connection and schema creation/verification.
- `internal/app`: dependency assembly, readiness state, shutdown wiring.
- `internal/httpapi`: router, middleware, system handlers, response/error serialization.

Schema bootstrap should create the current tables and indexes from [`storage-schema.md`](./atlas-docs-next/module-docs/atlas-core/storage-schema.md):

- `entities`
- `observations`
- `objects`
- `tasks`
- `object_files`

Create `objects` before `tasks` because `tasks.command_catalog_object_id` references `objects.object_id`.

### Build Scope

- Add `go.mod`, application entrypoint, configuration, logger, and graceful shutdown shell.
- Add PostgreSQL connection handling and readiness checks.
- Add schema creation/verification without migrations.
- Add common response and error envelope serialization.
- Add `GET /health`, `GET /readiness`, and a temporary or partial `GET /` implementation that reports `catalog_unavailable` until Stage 3 materializes the command catalog.
- Add pagination parsing helpers and validation helpers for IDs, timestamps, and unknown fields.

### Out Of Scope

- Resource CRUD endpoints.
- Command catalog materialization.
- Sighting catalog validation.
- Object file byte storage.
- SSE stream.
- Docker Compose and lifecycle CLI.

### Verification

- Unit tests for config parsing, error serialization, pagination, ID validation, and timestamp serialization.
- Integration test against PostgreSQL proving schema bootstrap creates all tables, constraints, and indexes.
- System endpoint tests:
  - `GET /health` returns process health.
  - `GET /readiness` returns `503 storage_unavailable` when PostgreSQL is unreachable.
  - `GET /` returns `503 catalog_unavailable` before command catalog materialization exists.
- Log test or snapshot proving required JSONL fields are present.

## Stage 2: Persistence Stores And Object File Bytes

### Goal

Implement the persistence layer behind store interfaces, including PostgreSQL CRUD/list behavior, object file metadata, filesystem byte storage, upload staging, append, content streaming, and readiness mismatch handling.

By the end of this stage, services are still mostly absent, but stores should be testable directly and should enforce the data storage rules the API will later depend on.

### Primary Docs

- [`stores/overview.md`](./atlas-docs-next/module-docs/atlas-core/stores/overview.md)
- [`regular-record-store.md`](./atlas-docs-next/module-docs/atlas-core/stores/regular-record-store.md)
- [`object-store.md`](./atlas-docs-next/module-docs/atlas-core/stores/object-store.md)
- [`query-support.md`](./atlas-docs-next/module-docs/atlas-core/stores/query-support.md)
- [`objects.md`](./atlas-docs-next/contracts/data-model/objects.md)
- [`0006-object-file-write-ordering.md`](./atlas-docs-next/decisions/0006-object-file-write-ordering.md)
- [`0007-core-referential-integrity.md`](./atlas-docs-next/decisions/0007-core-referential-integrity.md)

### System Design

Create store interfaces in `internal/store` before implementing PostgreSQL-specific details:

- Regular record store for entities, observations, tasks, task status updates, entity-task listing, and broad structured reads.
- Object store for objects, object file metadata, file upload, append, streaming/opening, deletion, file listing, owner filtering, and storage readiness.
- Transaction abstraction for structured metadata changes that must stay consistent.

Split implementation details:

- `internal/postgres` owns SQL and metadata transactions.
- `internal/objectfiles` owns path safety, byte staging, byte append, stat, open, and delete.
- The object store composes PostgreSQL metadata and filesystem byte operations.

File upload ordering must follow ADR 0006:

1. Stage bytes under the managed storage root.
2. Validate size, path safety, and content type.
3. Commit metadata.
4. Promote bytes to the final logical path.
5. If byte promotion fails after metadata commit, log a serious metadata/bytes mismatch, return `503 storage_unavailable`, and degrade readiness until repair.

Append is not intrinsically idempotent. Do not add `Idempotency-Key` handling unless the contract changes.

### Build Scope

- Implement regular record CRUD/list methods for entities, observations, and tasks.
- Implement task status update storage and `listTasksForAsset(asset_id)`.
- Implement object metadata CRUD/list/filter methods.
- Implement object file metadata CRUD/list methods.
- Implement file byte staging, safe path construction, upload, append, open/stream, stat, and delete.
- Implement full-query support from one read-only PostgreSQL transaction where possible.
- Return store-level errors that preserve enough structure for services/handlers to map to API errors.

### Out Of Scope

- HTTP endpoint handlers for resource CRUD.
- Business validation such as asset `supported_commands` or sighting catalog checks.
- Event publication.
- Command catalog materialization.

### Verification

- PostgreSQL integration tests for all table constraints, indexes needed by documented reads, create conflicts, list ordering, and owner filters.
- Object file tests for path traversal rejection, staging cleanup on metadata failure, readiness degradation on metadata/bytes mismatch, append size updates, and delete cleanup.
- Store tests for delete rules:
  - reject entity delete with dependent tasks, observations, or entity-owned objects.
  - cascade observation-owned objects through service later; store should expose the capabilities needed.
  - reject task delete with task-owned objects.
  - delete object files with object delete.
- Query-support test proving `GET /queries/full` data can be assembled from the store layer without file bytes.

## Stage 3: Catalog Bootstrap And Validation Engines

### Goal

Implement startup catalog handling and reusable validation engines for command payloads, sighting payloads, entity components, object types, and Core-owned JSON shape rules.

By the end of this stage, Atlas Core can load both checked-in catalogs, materialize the command catalog as an object-backed payload, hold the active command/sighting catalogs in memory, and expose a ready service descriptor when dependencies and catalogs are healthy.

### Primary Docs

- [`command-catalog/overview.md`](./atlas-docs-next/contracts/data-model/command-catalog/overview.md)
- [`sighting-catalog.md`](./atlas-docs-next/contracts/data-model/sighting-catalog.md)
- [`components/overview.md`](./atlas-docs-next/contracts/data-model/components/overview.md)
- [`core-startup-command-catalog.md`](./atlas-docs-next/system-docs/workflows/core-startup-command-catalog.md)
- [`system.md`](./atlas-docs-next/contracts/core-api/system.md)

### System Design

Create startup-owned packages:

- `internal/catalog`: command catalog load, shape validation, restricted JSON Schema validation, content hash, active in-memory catalog, object materialization.
- `internal/sightingcatalog`: sighting catalog load, shape validation, active in-memory catalog.

The command catalog is special:

- It starts as checked-in JSON.
- It validates at startup.
- It materializes into a normal object with `type: "command_catalog"`, `owner_type: "system"`, `owner_id: "active_command_catalog"`.
- Its object ID should include a stable content hash.
- Identical content should reuse the existing object/file.
- `GET /` should expose `active_command_catalog_object_id`.

The sighting catalog is different:

- It starts as checked-in JSON.
- It validates at startup.
- It stays in memory.
- It is not materialized as an object.
- It has no public API endpoint.

### Build Scope

- Add default checked-in `command-catalog/catalog.json` and `sighting-catalog/catalog.json`.
- Implement the command catalog restricted JSON Schema subset.
- Implement command validation helpers for command type and parameters.
- Implement supported-command validation helper for assets.
- Implement sighting validation helpers for `position`, `line_of_sight`, `file`, and `analysis`.
- Implement entity component validation for known component names, `custom_*` limits, and asset-required `supported_commands`.
- Implement object type validation for canonical types such as `command_catalog`, `observation_sighting_history`, `observation_media`, and `fusion_provenance`.
- Finish `GET /` so it returns `200 OK` with `active_command_catalog_object_id` only after materialization succeeds.
- Finish readiness dependency reporting for `postgres`, `object_storage`, and `command_catalog`.

### Out Of Scope

- Resource services and HTTP CRUD behavior.
- SSE event publishing.
- SDK helper implementation.
- Data fusion algorithms.

### Verification

- Catalog validation tests:
  - valid command catalog loads.
  - unsupported schema keyword fails readiness.
  - duplicate command type fails readiness.
  - valid sighting catalog loads.
  - unknown sighting kind validation fails.
- Command materialization tests:
  - first startup creates object/file.
  - identical startup reuses object/file.
  - changed catalog rotates object ID.
  - invalid catalog makes `GET /readiness` return `503 catalog_unavailable`.
- Component validation tests for canonical components and `custom_*` size/depth/field limits.
- Sighting validation tests for each allowed kind and inaccuracy string format.

## Stage 4: Service Layer Business Behavior

### Goal

Implement the service layer that turns stores and validators into Core behavior: entity lifecycle, observation lifecycle, task lifecycle, object/file behavior, full query assembly, delete rules, and event publication calls through an interface.

By the end of this stage, the Core behavior should be testable without HTTP by calling services directly with fake stores and event recorders.

### Primary Docs

- [`services/overview.md`](./atlas-docs-next/module-docs/atlas-core/services/overview.md)
- [`entity-service.md`](./atlas-docs-next/module-docs/atlas-core/services/entity-service.md)
- [`observation-service.md`](./atlas-docs-next/module-docs/atlas-core/services/observation-service.md)
- [`task-service.md`](./atlas-docs-next/module-docs/atlas-core/services/task-service.md)
- [`object-service.md`](./atlas-docs-next/module-docs/atlas-core/services/object-service.md)
- [`query-service.md`](./atlas-docs-next/module-docs/atlas-core/services/query-service.md)
- [`event-publisher.md`](./atlas-docs-next/module-docs/atlas-core/services/event-publisher.md)
- [`services/testing.md`](./atlas-docs-next/module-docs/atlas-core/services/testing.md)

### System Design

Create services in `internal/service`:

- Entity service owns entity create/read/list/patch/delete and asset task listing.
- Observation service owns observation create/read/list/patch/delete and sighting-summary validation.
- Task service owns task create/read/list/patch/delete/status transitions and command catalog checks.
- Object service owns object create/read/list/patch/delete plus file upload/append/read/delete behavior.
- Query service owns full current-state assembly.

Services should:

- validate request meaning before writing.
- call stores, not PostgreSQL directly.
- publish mutation events after successful writes through an event publisher interface.
- treat event publication failure as logged best-effort failure, not as API failure.
- return structured Core errors that handlers can serialize.

Create `internal/service/servicetest` with small test-only fakes and fixtures.

### Build Scope

- Implement all service methods needed by the planned HTTP endpoints.
- Enforce entity rules:
  - caller-supplied IDs.
  - valid types.
  - asset entities require `json.components.supported_commands`.
  - no unknown components unless `custom_*`.
  - delete rejects dependents.
- Enforce observation rules:
  - `source_asset_id` must reference an asset.
  - `json.state` is required and valid.
  - `json.latest_sighting` validates against active sighting catalog.
  - `json.sightings_object_id` references an observation-owned `observation_sighting_history` object when present.
- Enforce task rules:
  - create rejects caller-supplied `command_catalog_object_id`.
  - create resolves active command catalog object ID and pins it.
  - command type and parameters validate against active/pinned catalog.
  - target asset must support the command.
  - status transitions are only `pending -> acknowledged -> completed|failed`.
  - repeating current status is idempotent.
- Enforce object rules:
  - owner exists except system owners.
  - `observation_sighting_history` ownership is observation-only.
  - `fusion_provenance` is track-entity-owned.
  - file upload/append delegates byte coordination to object store.
- Implement query service snapshot assembly over structured records and object metadata.

### Out Of Scope

- HTTP request decoding and multipart parsing.
- SSE transport mechanics.
- Docker Compose and CLI.

### Verification

- Service tests with fakes for every success/failure mode documented in resource API docs.
- Event recorder tests proving create/update/delete services publish the right event intent after successful writes.
- Tests proving event publication failure does not fail the original service call.
- Task tests for catalog pinning, unsupported command rejection, invalid parameters, and status idempotency.
- Observation tests for sighting validation and sighting-history object coupling.
- Object tests for owner validation and canonical object type behavior.
- Delete conflict tests matching ADR 0007.

## Stage 5: HTTP API, SSE Stream, And Contract Integration

### Goal

Expose the full Core HTTP API, including JSON resources, multipart uploads, raw byte append/streaming, pagination headers, error envelopes, full query, and live SSE change stream.

By the end of this stage, a client can use Atlas Core through HTTP according to the contracts, and stream events contain enough resource payload for SDK replica mode.

### Primary Docs

- [`endpoint-layout.md`](./atlas-docs-next/contracts/core-api/endpoint-layout.md)
- [`planned-endpoints.md`](./atlas-docs-next/contracts/core-api/planned-endpoints.md)
- Resource contracts under [`atlas-docs-next/contracts/core-api/`](./atlas-docs-next/contracts/core-api/)
- [`stream.md`](./atlas-docs-next/contracts/core-api/stream.md)
- [`sdk/overview.md`](./atlas-docs-next/contracts/sdk/overview.md)
- Workflow docs under [`atlas-docs-next/system-docs/workflows/`](./atlas-docs-next/system-docs/workflows/)

### System Design

Implement `internal/httpapi` around DTOs and services:

- Router defines the exact planned paths.
- Middleware adds request IDs, request logging, CORS defaults, panic/internal error handling, and error envelope serialization.
- DTOs decode only documented fields and reject unknown top-level fields where the contract requires it.
- Handlers convert DTOs to service inputs and service results to API resource JSON.
- Handlers set pagination headers from the pagination helpers.

Implement `internal/events`:

- In-process publisher and subscriber registry.
- SSE endpoint at `GET /stream/changes`.
- Keepalive comments.
- Event IDs.
- Full resource payloads for create/update events.
- Delete event payloads.
- Object/file affected-file metadata with the documented cap.
- Live-only behavior with no replay.

### Build Scope

- Implement all system, entity, observation, task, object/file, query, and stream endpoints.
- Implement multipart file upload parsing and raw byte append handling.
- Implement raw byte content streaming with correct content headers.
- Implement standard error envelope fields from [`errors.md`](./atlas-docs-next/contracts/core-api/errors.md).
- Implement list pagination headers and documented filters:
  - entities by `type`
  - observations by `source_asset_id` and `updated_after`
  - tasks by `asset_id` and `status`
  - objects by `owner_type`, `owner_id`, and `type`
- Implement `GET /queries/full` with `generated_at`, service descriptor summary, entities, observations, tasks, objects, and object files.
- Implement SSE event publication wiring from services through the event publisher.

### Out Of Scope

- SDK implementation.
- Atlas Command Interface.
- Data fusion algorithm behavior.
- Durable stream replay.
- Auth middleware.

### Verification

- Handler tests for every endpoint in [`planned-endpoints.md`](./atlas-docs-next/contracts/core-api/planned-endpoints.md).
- Contract tests for:
  - success statuses and response bodies.
  - error envelopes and stable error codes.
  - pagination headers.
  - unknown query/body field rejection.
  - multipart upload behavior.
  - byte append behavior.
  - byte content streaming.
- SSE tests proving:
  - create/update events include full resource JSON.
  - delete events include deletion metadata.
  - object file upload/append/delete emits `object.updated`.
  - keepalives are emitted.
  - no replay cursor behavior is implied.
- Workflow-level HTTP tests:
  - asset comes online.
  - operator creates task.
  - asset executes task.
  - asset creates observation, appends sighting JSONL, patches latest sighting.
  - startup materializes command catalog.

## Stage 6: Local Runtime, Containers, CLI, And Data Fusion Harness

### Goal

Package Atlas Core for local operation with Docker Compose, object file volume, lifecycle CLI, integration tests, and an optional data fusion worker harness with a baseline selectable stack.

By the end of this stage, a developer should be able to start, restart, and shut down the full local Core system, run integration tests against real dependencies, and optionally run a data fusion worker container that talks to Core only through HTTP/SSE.

### Primary Docs

- [`runtime-shape.md`](./atlas-docs-next/module-docs/atlas-core/runtime-shape.md)
- [`container-system.md`](./atlas-docs-next/module-docs/atlas-core/container-system.md)
- [`deployment-topology.md`](./atlas-docs-next/system-docs/deployment-topology.md)
- [`data-fusion.md`](./atlas-docs-next/module-docs/atlas-core/data-fusion.md)
- [`0004-interactive-core-cli.md`](./atlas-docs-next/decisions/0004-interactive-core-cli.md)
- [`0005-no-auth-current-operating-model.md`](./atlas-docs-next/decisions/0005-no-auth-current-operating-model.md)

### System Design

Runtime pieces:

- `Dockerfile` builds the Atlas Core process.
- `docker-compose.yml` starts `atlas-core`, `postgres`, and a project-owned object-file volume.
- `.env.example` documents local defaults and no credentials.
- `tools/atlas-core-cli` owns Python lifecycle commands for `start`, `restart`, and `shutdown`.
- Destructive CLI cleanup must be label-scoped to Atlas Core project resources only.

Data fusion pieces:

- `data-fusion/harness` owns worker startup, Core connection setup, logging, and stack selection.
- `data-fusion/stacks/baseline` is a minimal baseline stack.
- `data-fusion/tests` owns shared harness tests.
- The worker reads observations/entities and writes track entities through Core API or SDK behavior.
- The worker must not import Core server packages, write PostgreSQL directly, or mutate object file bytes directly.

### Build Scope

- Add Dockerfile, Compose file, `.env.example`, and documented local defaults:
  - Core API default port `8080`
  - PostgreSQL default port `5432`
  - object storage root volume
  - command/sighting catalog paths
  - allowed origin default for command interface dev server `http://localhost:5173`
- Add healthcheck/readiness wait behavior for local startup.
- Add Python lifecycle CLI:
  - `start`
  - destructive `restart`
  - destructive `shutdown`
  - typed confirmation for destructive actions
  - explicit non-interactive confirm flag
  - label-based selection rules from ADR 0004
- Add end-to-end integration test suite using real PostgreSQL and temporary or Compose-backed object storage.
- Add optional `atlas-data-fusion` harness and baseline stack wiring behind configuration.
- Add README instructions for local run/test workflow.

### Out Of Scope

- Production deployment hardening.
- Kubernetes, cloud orchestration, service mesh, backups, migrations, and multi-node deployment.
- Real fusion algorithm design beyond a minimal harness/baseline that proves the boundary.
- Atlas SDK and Atlas Command Interface implementation.

### Verification

- `docker compose up` brings Core and PostgreSQL up with a mounted object volume.
- `GET /readiness` becomes ready after schema, storage, command catalog, and sighting catalog initialization.
- CLI `start` waits for readiness.
- CLI `restart` and `shutdown` remove only label-matching Atlas Core project resources and refuse unlabeled/mismatched resources.
- End-to-end tests cover all workflows from [`system-docs/workflows`](./atlas-docs-next/system-docs/workflows/).
- Object bytes persist across non-destructive start and disappear after destructive restart/shutdown.
- Optional fusion worker starts, connects to Core over HTTP/SSE, logs with shared fields, and can write a test track entity without touching PostgreSQL directly.

## Final Build Definition

Atlas Core is build-ready when all six stages are merged and:

- every endpoint in [`planned-endpoints.md`](./atlas-docs-next/contracts/core-api/planned-endpoints.md) is implemented and tested.
- every table in [`storage-schema.md`](./atlas-docs-next/module-docs/atlas-core/storage-schema.md) exists with its documented constraints and indexes.
- command catalog and sighting catalog startup failures affect readiness correctly.
- object upload, append, content streaming, delete, and readiness mismatch behavior follow ADR 0006.
- services publish live events after successful mutations.
- `GET /queries/full` can hydrate an SDK replica without file bytes.
- the SSE stream can update SDK replica state without per-event resource fetches.
- local Docker and CLI workflows run the real Core system without fake product paths.
- all logs needed for debugging and inefficiency analysis use the structured run logging contract.
