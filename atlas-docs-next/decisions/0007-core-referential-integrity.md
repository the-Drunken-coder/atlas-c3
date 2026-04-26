# 0007: Core Referential Integrity And Delete Rules

## Status

Accepted

## Context

ATLAS-C3 uses multiple record families that reference each other:

- tasks reference assets
- observations reference assets
- objects reference owners through `owner_type` and `owner_id`
- object files reference objects

The current operating model treats clients as trusted, but trusted clients still benefit from **hard invariants** that prevent accidental impossible states and make debugging deterministic.

## Decision

Atlas Core should use **PostgreSQL foreign keys** for relationships that must never dangle in normal operation, and should pair them with explicit delete behavior that matches the contracts.

### Foreign keys

Atlas Core should enforce at least:

- `tasks.asset_id` references `entities.entity_id`
- `observations.source_asset_id` references `entities.entity_id`
- `object_files.object_id` references `objects.object_id`

`objects.owner_type` and `objects.owner_id` are a polymorphic reference. PostgreSQL should enforce `objects.owner_type` with a `CHECK (owner_type in ('entity', 'observation', 'task', 'system'))` constraint and an index on `(owner_type, owner_id)`. Atlas Core must still enforce owner existence for `owner_id` in the service layer because it points at different tables depending on `owner_type`.

### Delete behavior

**Entity delete**

Deleting an entity should be **rejected** if any of the following still exist:

- tasks with `asset_id` equal to that entity
- observations with `source_asset_id` equal to that entity
- objects with `owner_type=entity` and `owner_id` equal to that entity

Callers should delete or reassign dependents first, or use destructive reset workflows for local environments.

**Observation delete**

Deleting an observation should **cascade delete** objects owned by that observation:

- delete all `objects` rows with `owner_type=observation` and `owner_id` equal to the deleted observation id
- deleting those objects should delete their object file metadata and object file bytes per object delete rules

**Task delete**

Task delete remains allowed as operational data cleanup.

If objects exist with `owner_type=task` and `owner_id` equal to the task id, task delete should be **rejected** until those objects are deleted. This avoids leaving ownerless task objects around.

**Object delete**

Object delete should delete object file metadata and bytes per [`../contracts/data-model/objects.md`](../contracts/data-model/objects.md).

## Consequences

- Core cannot rely on “writers always clean up” as the only guarantee for the relationships above.
- Some deletes become two-step operations for clients, which is acceptable for short-lived operational data and debugging workflows.
- Schema bootstrap remains non-migratory, but foreign keys still add valuable guardrails.

## Rejected Alternatives

- **No foreign keys, only service checks** - cheaper to write initially, but easier to accidentally violate invariants under concurrency and partial failures.
- **Silent cascade deletes for assets** - surprising in operations and dangerous for simulations unless explicitly requested.
