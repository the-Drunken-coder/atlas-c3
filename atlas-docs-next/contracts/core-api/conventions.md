# Core API Conventions

This document defines shared Core API behavior. Resource-specific docs should link here instead of repeating these rules.

## Success Responses

Successful responses should return the resource JSON directly.

Do not wrap successful responses in a generic `{ "success": true, "data": ... }` envelope.

Default success statuses:

| Operation | Status | Body |
| --- | --- | --- |
| Create | `201 Created` | Created resource JSON |
| Read | `200 OK` | Resource JSON |
| List | `200 OK` | JSON array of resource JSON |
| Update | `200 OK` | Updated resource JSON |
| Status transition | `200 OK` | Updated resource JSON |
| Delete | `204 No Content` | No response body |

Resource docs may define a narrower rule when an endpoint is unusual, such as file content streaming.

## Error Responses

Error responses should use a consistent JSON envelope:

```json
{
  "success": false,
  "message": "string",
  "error_code": "string",
  "error_id": "string",
  "timestamp": "2026-01-01T00:00:00Z",
  "path": "string",
  "details": {}
}
```

Stable error codes, status mappings, and validation detail shapes are defined in [`errors.md`](./errors.md). Resource docs should reference those codes instead of inventing local error names.

## IDs

Create requests must provide IDs explicitly.

Atlas Core should reject create requests that omit the ID for the record being created. This applies to entities, observations, tasks, objects, and object files.

IDs are creator-supplied strings up to 50 characters, matching [`../data-model/record-families.md`](../data-model/record-families.md).

Atlas Core should not silently generate IDs for normal resource creation. Callers need to know what they are creating and how to reference it later.

## Pagination

List endpoints should use offset pagination:

- `limit` defaults to `50`
- `limit` max is `100`
- `offset` defaults to `0`

Pagination response metadata should be exposed through headers:

- `X-Total-Count`
- `X-Limit`
- `X-Offset`
- `X-Returned-Count`

## List Query Parameters

List endpoints (`GET` on plural resources) share pagination headers and limits from the previous section.

Query parameters used for **filtering or sorting** should be **explicitly documented per resource**. Parameters that are not documented for that endpoint should be **rejected** with `400` and a clear validation error, rather than silently ignored. That keeps busy-system clients from accidentally running unbounded or misspelled queries.

Each resource doc should define:

- which filters exist
- which combinations are supported
- default ordering when the client omits sort parameters
- any parameters that are intentionally forward-compatible (rare; call them out explicitly)

Unfiltered list reads return a **bounded page** of rows in the resource’s default order. They are not a guarantee of “everything important on this page” in a hot system. Interactive clients and assets should prefer documented filters (for example by `asset_id`, `status`, or time window) instead of relying on defaults alone.

List responses return a JSON array directly. Pagination metadata stays in headers rather than a response wrapper.

When a resource doc does not define an explicit sort parameter, clients cannot request arbitrary sorting. The endpoint should use its documented default ordering and reject unknown sort parameters with `400 validation_failed`.

## Updates

Updates should use `PATCH`.

`PATCH` means partial update of the targeted record.

Global PATCH rules:

- Unknown top-level request fields are rejected with `400 validation_failed`.
- Omitted fields are unchanged.
- Immutable fields are rejected with `400 immutable_field`.
- Nullable fields must be explicitly documented by the resource. Sending `null` for a non-nullable field is `400 validation_failed`.
- Promoted fields update only when explicitly present in the request body.
- `json` updates replace the provided named top-level JSON sections; they do not deep-merge arbitrary nested objects.
- `json.components.<name>` updates replace that named component as a whole. They do not recursively merge into the existing component.
- Observation `json.evidence.<name>` sections follow the same named-section replacement rule, but the inner evidence shapes remain deferred to the observation evidence contract.
- Updating a record should advance that record's `updated_at`.
- A PATCH that makes no material change may return `200` with the current resource.

Resource-specific docs define which promoted fields and JSON sections are mutable.

## Deletes

Deletes should use `DELETE`.

Atlas Core data is short-lived operational state, so hard delete is allowed for the planned resource families. Exact cascading or cleanup behavior belongs in each resource doc.

## Timestamps

API timestamps should be serialized as RFC 3339 strings.

Core-created timestamps are server-owned. Create and patch requests must not set `created_at` or `updated_at`; those fields are returned in responses.

Required timestamp request fields must be RFC 3339 strings. Optional timestamp fields may be omitted unless a resource doc explicitly permits `null`.

## Create Conflicts

Create requests use caller-supplied IDs. If a create request uses an ID that already exists, Atlas Core should return `409 conflict`.

Clients that need idempotent create behavior should follow the duplicate-create recovery rule from the asset protocol: on `409`, perform `GET` on the existing resource and reconcile.

## File Content

Object file metadata is JSON. Object file content endpoints stream bytes directly.

File upload endpoints should use multipart form data.
