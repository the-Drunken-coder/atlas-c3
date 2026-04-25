# Core API Conventions

This document defines shared Core API behavior. Resource-specific docs should link here instead of repeating these rules.

## Success Responses

Successful responses should return the resource JSON directly.

Do not wrap successful responses in a generic `{ "success": true, "data": ... }` envelope.

## Error Responses

Error responses should use a consistent JSON envelope:

```json
{
  "success": false,
  "message": "string",
  "error_code": "string",
  "error_id": "string",
  "timestamp": "RFC3339",
  "path": "string",
  "details": {}
}
```

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

## Updates

Updates should use `PATCH`.

`PATCH` means partial update of the targeted record. Exact update semantics for each resource belong in that resource's API doc.

## Deletes

Deletes should use `DELETE`.

Atlas Core data is short-lived operational state, so hard delete is allowed for the planned resource families. Exact cascading or cleanup behavior belongs in each resource doc.

## Timestamps

API timestamps should be serialized as RFC 3339 strings.

## File Content

Object file metadata is JSON. Object file content endpoints stream bytes directly.

File upload endpoints should use multipart form data.

