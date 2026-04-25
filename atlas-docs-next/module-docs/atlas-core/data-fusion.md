# Data Fusion

This document defines the current data fusion boundary.

Data fusion turns observation evidence into authoritative track entities. The fusion algorithm and observation evidence inner shapes remain deferred, but the system boundary is settled enough to guide implementation.

## Current Direction

Data fusion should run as a separate trusted worker container.

Atlas Core remains the source of truth. Data fusion is a Core API client:

- reads observations through Atlas Core API or Atlas SDK
- reads any needed current track/entity state through Atlas Core API or Atlas SDK
- writes or updates track entities through Atlas Core API or Atlas SDK
- listens to the Core stream when useful

Data fusion should not:

- run inside the Atlas Core process
- bypass Atlas Core storage by writing to PostgreSQL
- mutate object file bytes directly
- introduce a separate source of truth for tracks
- require changes to the Core storage shape before the fusion algorithm is designed

## Container Boundary

The worker container should be swappable.

First-party and third-party fusion containers may exist as long as they follow the shared Core contracts. This lets ATLAS-C3 expose a common system outline while allowing different fusion implementations for different deployments.

The expected local service name is `atlas-data-fusion` when the worker is included in Docker Compose.

## Inputs And Outputs

Inputs:

- observations from [`../../contracts/core-api/observations.md`](../../contracts/core-api/observations.md)
- current entities and tracks from [`../../contracts/core-api/entities.md`](../../contracts/core-api/entities.md)
- object metadata and file content through the object API when fusion needs attached evidence payloads

Outputs:

- track entities written through the entity API
- optional logs explaining fusion decisions

Fusion-created or fusion-updated tracks should use normal entity records with `type: "track"`.

## Deferred Design Areas

The following remain deferred:

- how observations become tracks
- which observation evidence fields fusion consumes
- track creation and update rules
- confidence and uncertainty handling
- association rules for matching observations to existing tracks
- whether a specific fusion implementation polls, streams, or combines both
- debugging and explainability details for fusion decisions
- first implementation algorithm scope

These are real design areas, not implementation details to guess from the current docs.
