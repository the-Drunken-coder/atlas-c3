# Deployment Topology

This document defines the current local-first deployment topology for ATLAS-C3.

Module internals belong in [`../module-docs/`](../module-docs/). Storage decisions belong in [`../decisions/0002-core-storage-shape.md`](../decisions/0002-core-storage-shape.md).

## Current Topology

The first deployment topology is local-first Docker Compose for Atlas Core.

The Atlas Core Compose project should include:

- Atlas Core API process
- PostgreSQL
- a filesystem volume for object file bytes

The Compose project may also include an optional data fusion worker container:

- `atlas-data-fusion`

Atlas Command Interface should run outside the Atlas Core Compose project as a separate dev app/client.

SDK consumers, simulations, debugging tools, asset runtimes, and Atlas Command Interface connect to Atlas Core over the host-exposed Core API port.

Data fusion is also a trusted Core API client. It should connect to Atlas Core over HTTP/SSE and should not use PostgreSQL as an integration surface.

PostgreSQL should be reachable by Atlas Core on the Compose network. It does not need to be part of the public client contract.

Object file bytes live in the project-owned filesystem volume, not in S3 or a separate object storage service.

## Fixed Default Ports

- Atlas Core API: `8080`
- PostgreSQL: `5432`
- Atlas Command Interface dev server: `5173`

These are defaults for planning and local development. Implementation may allow overrides through normal environment configuration, but docs and examples should use these defaults.

## Service Discovery

Inside Docker Compose, Atlas Core should connect to PostgreSQL by Compose service name.

Inside Docker Compose, `atlas-data-fusion` should connect to Atlas Core by Compose service name when present.

Outside Docker Compose, trusted clients should use `http://localhost:8080` for Atlas Core by default.

Atlas Command Interface should use `http://localhost:8080` as its default Core API base URL during local development.

## Reset And Persistence

Local data is disposable.

Restart and shutdown flows may remove containers, images, and volumes according to [`../decisions/0004-interactive-core-cli.md`](../decisions/0004-interactive-core-cli.md).

## Non-Goals

This topology should not introduce:

- database migrations
- backups
- multi-node deployment
- cloud orchestration
- Kubernetes
- service mesh
- production hardening

The optional data fusion worker does not change these non-goals. It is a client container around Core contracts, not a second backend or storage owner.

Future topology changes that alter module boundaries or storage shape should become decision records.
