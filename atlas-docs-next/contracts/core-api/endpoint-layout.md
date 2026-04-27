# Core API Endpoint Layout

This document defines the planned Atlas Core URL layout at a high level. It does not define request bodies, response bodies, validation rules, or exact status codes.

The endpoint inventory is [`planned-endpoints.md`](./planned-endpoints.md). Record ownership is defined in [`../data-model/record-families.md`](../data-model/record-families.md).

## General Shape

The API uses simple plural resource paths.

Normal reads and writes use JSON. Object file uploads use multipart form data. Object file content endpoints stream raw bytes.

Create requests must include caller-supplied IDs, including object file uploads. Shared rules are defined in [`conventions.md`](./conventions.md).

Use `PATCH` for updates. Atlas Core records commonly combine promoted fields with flexible metadata, so partial updates fit the model better than full replacement.

Use `DELETE` for delete operations. Core data is short-lived operational state, and deletes are useful for simulation, debugging, and reset workflows.

## System

Planned paths:

- `GET /`
- `GET /health`
- `GET /readiness`

`GET /` should return a service descriptor. It should include important system-owned identifiers that clients need to discover, including the active command catalog object ID.

## Entities

Planned paths:

- `GET /entities`
- `POST /entities`
- `GET /entities/{entity_id}`
- `PATCH /entities/{entity_id}`
- `DELETE /entities/{entity_id}`
- `GET /entities/{entity_id}/tasks`

Entity-related objects should be queried through `GET /objects` filters instead of nested entity object endpoints.

## Observations

Planned paths:

- `GET /observations`
- `POST /observations`
- `GET /observations/{observation_id}`
- `PATCH /observations/{observation_id}`
- `DELETE /observations/{observation_id}`

Observation files should be stored through objects. The observations API owns observation state; the objects API owns file containers and file bytes.

Observation-related objects should be queried through `GET /objects` filters instead of nested observation object endpoints.

## Tasks

Planned paths:

- `GET /tasks`
- `POST /tasks`
- `GET /tasks/{task_id}`
- `PATCH /tasks/{task_id}`
- `DELETE /tasks/{task_id}`
- `POST /tasks/{task_id}/status`

Task-related objects should be queried through `GET /objects` filters instead of nested task object endpoints.

## Objects And Files

Planned paths:

- `GET /objects`
- `POST /objects`
- `GET /objects/{object_id}`
- `PATCH /objects/{object_id}`
- `DELETE /objects/{object_id}`
- `POST /objects/{object_id}/files/{file_id}`
- `POST /objects/{object_id}/files/{file_id}/append`
- `GET /objects/{object_id}/files/{file_id}`
- `GET /objects/{object_id}/files/{file_id}/content`
- `DELETE /objects/{object_id}/files/{file_id}`

Append is the generic object-file operation used for append-only payloads such as observation sighting history JSON Lines. It is not nested under observation endpoints.

`GET /objects` should support filtering by owning or related record:

- `GET /objects?owner_type=entity&owner_id={entity_id}`
- `GET /objects?owner_type=observation&owner_id={observation_id}`
- `GET /objects?owner_type=task&owner_id={task_id}`
- `GET /objects?owner_type=system&owner_id={system_owned_id}`

Use `owner_type` and `owner_id` rather than separate query parameters for each record family. That keeps object lookup extensible without adding new endpoints for every owner type.

## Queries

Planned path:

- `GET /queries/full`

The full query endpoint is for broad current-state reads used by clients that need to bootstrap or refresh local state. It is not a historical replay endpoint.

## Stream

Planned path:

- `GET /stream/changes`

The stream uses server-sent events. It is live-only and does not replay missed history.

## Command Catalog

The command catalog does not have command-catalog-specific endpoints.

The active command catalog is materialized as an object at startup. Clients discover the active command catalog object ID from `GET /`, then use the object and file endpoints to read it.
