# Data Fusion

This document defines the current data fusion boundary.

Data fusion turns observation sightings and related evidence files into authoritative track entities. The fusion algorithm, observation-to-track association rules, and detailed confidence model remain deferred, but the system boundary is settled enough to guide implementation.

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

The data fusion implementation may live in the Atlas Core implementation repository for development convenience, Docker packaging, and tests. It must remain a separate worker boundary and must not be imported into the Core server process.

## Container Boundary

The worker container should be swappable.

First-party and third-party fusion containers may exist as long as they follow the shared Core contracts. This lets ATLAS-C3 expose a common system outline while allowing different fusion implementations for different deployments.

The expected local service name is `atlas-data-fusion` when the worker is included in Docker Compose.

## Harness And Stack Layout

Data fusion should use a harness plus selectable stack layout:

```text
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
```

The harness owns:

- worker startup and shutdown
- Docker container packaging
- Core API/SDK connection setup
- shared logging and run IDs
- stack loading and configuration
- common test harness behavior

`stacks/` owns the selectable fusion behavior. Each child folder is one complete fusion algorithm/behavior stack. Only one stack should be active at a time.

The active stack should be selected by explicit configuration, such as `ATLAS_DATA_FUSION_STACK=baseline`. Switching to a different fusion approach should require adding a new stack folder and changing configuration, not rewriting the harness or Atlas Core.

The shared testing system belongs outside individual stack folders. Stack-specific tests may live inside a stack only when they do not apply to other stacks.

## Inputs And Outputs

Inputs:

- observations from [`../../contracts/core-api/observations.md`](../../contracts/core-api/observations.md)
- current entities and tracks from [`../../contracts/core-api/entities.md`](../../contracts/core-api/entities.md)
- object metadata and file content through the object API when fusion needs attached evidence payloads

Outputs:

- track entities written through the entity API
- track-owned `fusion_provenance` objects containing detailed reasoning
- logs explaining fusion decisions

Fusion-created or fusion-updated tracks should use normal entity records with `type: "track"`.

Track entities should stay small. Fusion workers should write compact current state and [`fusion_summary`](../../contracts/data-model/components/fusion_summary.md) onto the track entity, then store detailed reasoning as an object with `type: "fusion_provenance"`, `owner_type: "entity"`, and `owner_id` set to the track ID.

The fusion provenance object file should contain structured JSON. Atlas Core should not add a fusion-specific provenance endpoint until normal object and SDK helper access proves insufficient.

## Deferred Design Areas

The following remain deferred:

- how observations become tracks
- which sighting fields fusion consumes
- track creation and update rules
- confidence and inaccuracy handling inside fusion decisions
- association rules for matching observations to existing tracks
- whether a specific fusion implementation polls, streams, or combines both
- debugging and explainability details for fusion decisions
- first implementation algorithm scope
- exact algorithm package internals inside each stack

These are real design areas, not implementation details to guess from the current docs.
