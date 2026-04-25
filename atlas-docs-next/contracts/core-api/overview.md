# Core API Overview

This folder defines the Atlas Core HTTP API contract.

Atlas Core implements this API. Atlas SDK wraps it. First-party clients should depend on Atlas SDK rather than duplicating raw HTTP behavior, but the API remains the shared contract underneath.

Authentication and authorization are not part of the current operating model. The shared auth/security contract is [`../auth-security/overview.md`](../auth-security/overview.md).

## Contract Scope

The Core API is responsible for exposing Atlas Core's current operational state and write behavior over HTTP.

The API should cover:

- service identity and dependency status
- entities
- observations
- tasks
- objects and object files
- current-state queries
- live change stream
- command catalog access

The current high-level endpoint inventory is [`planned-endpoints.md`](./planned-endpoints.md). The planned URL layout is [`endpoint-layout.md`](./endpoint-layout.md). Shared API rules are defined in [`conventions.md`](./conventions.md).

## Data Model Source

API resource groups should follow the record families defined in [`../data-model/record-families.md`](../data-model/record-families.md).

The API docs should not redefine record ownership, object/file storage behavior, or command catalog storage. They should link to the data-model contracts and define only the HTTP surface.

## API Shape

The API should stay simple and resource-oriented.

Expected direction:

- JSON for normal request and response bodies
- multipart upload for object file uploads
- raw byte streaming for object file download and view responses
- simple pagination for list endpoints
- clear error envelopes
- health and readiness endpoints
- live-only server-sent events for change notification
- service descriptor discovery for system-owned object IDs, including the active command catalog object ID

The API should not introduce GraphQL, a separate command/query API split, or a second backend layer for first-party clients.

## Endpoint Planning Order

Plan API docs in this order:

1. Endpoint layout and URL shape.
2. Shared request, response, pagination, and error conventions.
3. Resource-specific endpoints.
4. Request and response payloads.
5. Validation and status codes.

Internal Atlas Core stores should be shaped after the endpoint layout and resource contracts are clear.

## Resource Docs

Planned resource docs:

- `endpoint-layout.md`
- `conventions.md`
- `system.md`
- `entities.md`
- `observations.md`
- `tasks.md`
- `objects.md`
- `queries.md`
- `stream.md`

These files should link to this overview and to the relevant data-model contracts.

