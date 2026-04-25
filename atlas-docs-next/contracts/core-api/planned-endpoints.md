# Planned Core API Endpoints

This document is a high-level endpoint inventory for Atlas Core. It names the planned API groups without defining request and response shapes.

Detailed endpoint contracts should be written in separate files later and linked from here. The planned URL layout is [`endpoint-layout.md`](./endpoint-layout.md).

Atlas Core internal stores should be shaped to support these resource groups, not designed independently first.

## System Endpoints

Purpose: identify the service and expose process/dependency status.

Planned groups:

- service descriptor
- health
- readiness

The service descriptor should expose important system-owned identifiers, including the active command catalog object ID.

## Entity Endpoints

Purpose: manage current-state records for assets, tracks, and geofeatures.

Planned groups:

- create entity
- read entity
- list entities
- update entity
- delete entity
- read entity-related tasks

## Observation Endpoints

Purpose: manage first-class observation evidence records.

Planned groups:

- create observation
- read observation
- list observations
- update observation

Observation records are first-class by decision: [`../../decisions/0001-first-class-observations.md`](../../decisions/0001-first-class-observations.md).

## Task Endpoints

Purpose: manage work assigned to assets or other taskable entities.

Planned groups:

- create task
- read task
- list tasks
- update task
- delete task
- transition task status

## Object And File Endpoints

Purpose: manage object metadata and files stored through the Core object file system.

Planned groups:

- create object
- read object
- list objects
- update object metadata
- delete object
- upload object file
- read object file metadata
- download or view object file content
- delete object file
- list objects by related entity, observation, task, or system owner

Objects store file containers and payload metadata. Observation lifecycle rules belong to observation contracts, not object contracts.

## Query Endpoints

Purpose: support broad current-state reads for clients that need to bootstrap or refresh local state.

Planned groups:

- full current-state query

Query endpoints are for current state, not historical replay.

## Stream Endpoints

Purpose: publish live changes so clients can refresh or update local state.

Planned groups:

- live change stream

The stream is live-only. It is not the system of record and should not be treated as a durable event log.

## Command Catalog Access

The command catalog is stored and exposed through the object API. It should not have command-catalog-specific API endpoints.

The command catalog source is checked-in Atlas Core JSON, and its runtime representation is an object-backed payload. Clients discover the active command catalog object ID from the service descriptor, then read it through object and file endpoints. The data contract is [`../data-model/command-catalog/overview.md`](../data-model/command-catalog/overview.md).

