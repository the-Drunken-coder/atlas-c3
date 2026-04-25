# Regular Record Store

The regular record store owns structured operational records that are not object file containers.

It supports the API docs for entities, observations, tasks, and full-state queries.

## Owned Record Families

- entities
- observations
- tasks

Record-family ownership is defined in [`../../../contracts/data-model/record-families.md`](../../../contracts/data-model/record-families.md).

## Entity Capabilities

Required capabilities:

- create entity with caller-supplied `entity_id`
- read entity by `entity_id`
- list entities with pagination
- patch entity
- delete entity
- list tasks assigned to an asset

The store should enforce database-level constraints and return enough information for the service layer to produce API responses and errors.

## Observation Capabilities

Required capabilities:

- create observation with caller-supplied `observation_id`
- read observation by `observation_id`
- list observations with pagination
- patch observation
- delete observation

Observation files are not owned by this store. Files are represented through objects and handled by the object store.

## Task Capabilities

Required capabilities:

- create task with caller-supplied `task_id`
- read task by `task_id`
- list tasks with pagination
- patch task
- delete task
- update task status
- list tasks assigned to an asset

Tasks target assets. The service layer should own task validation, command catalog validation, and task lifecycle rules.

## Query Support

The regular record store should support broad reads of:

- entities
- observations
- tasks

These reads are needed by [`query-support.md`](./query-support.md).

## Transactions

The store should support explicit transaction boundaries for operations that need multiple structured writes.

Examples:

- task status transition plus derived task state updates
- task create plus validation-dependent metadata writes
- observation update plus related structured bookkeeping

Object file byte writes do not belong to this store and cannot be rolled back by database transactions.

## Readiness

The regular record store should expose a database readiness check for Atlas Core readiness.

