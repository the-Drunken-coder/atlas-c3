# Core API Errors

This document defines stable Atlas Core API error codes.

Shared response envelope rules live in [`conventions.md`](./conventions.md). Resource docs should use the codes here instead of defining local error names.

## Error Envelope

Error responses use this shape:

```json
{
  "success": false,
  "message": "Human-readable summary",
  "error_code": "validation_failed",
  "error_id": "01HZXEXAMPLE000000000000000",
  "timestamp": "2026-01-01T00:00:00Z",
  "path": "/tasks",
  "details": {}
}
```

Fields:

| Field | Meaning |
| --- | --- |
| `success` | Always `false` for errors |
| `message` | Short human-readable summary, not a stable machine contract |
| `error_code` | Stable machine-readable code from this document |
| `error_id` | Server-generated ID for logs and debugging |
| `timestamp` | RFC 3339 server timestamp |
| `path` | Request path that failed |
| `details` | Error-specific JSON object |

## Validation Details

Validation errors should use:

```json
{
  "fields": [
    {
      "field": "asset_id",
      "code": "required",
      "message": "asset_id is required"
    }
  ]
}
```

`field` uses request JSON field paths such as `json.components.command.type`. For query parameters, use names such as `query.limit`.

Common field codes:

- `required`
- `invalid_type`
- `invalid_value`
- `too_long`
- `out_of_range`
- `unknown_field`
- `immutable`
- `not_found`
- `conflict`

## Error Codes

| HTTP status | Error code | Meaning |
| --- | --- | --- |
| `400` | `validation_failed` | Request body, path, query, or multipart fields are invalid |
| `400` | `immutable_field` | Request attempts to change an immutable field |
| `400` | `invalid_status_transition` | Task status transition is not allowed |
| `400` | `command_validation_failed` | Task command or parameters do not match the pinned command catalog, or the asset does not support the command |
| `404` | `not_found` | Requested resource does not exist |
| `409` | `conflict` | Create ID already exists, duplicate file ID, or delete is blocked by dependents |
| `413` | `payload_too_large` | Request or file upload exceeds configured limit |
| `415` | `unsupported_media_type` | Request content type is not supported |
| `500` | `internal_error` | Unexpected server failure |
| `503` | `storage_unavailable` | PostgreSQL or object file storage is unavailable or inconsistent |
| `503` | `catalog_unavailable` | Active command catalog is missing, invalid, or not materialized |

## Conflict Details

Conflict responses should include enough detail for caller recovery:

```json
{
  "resource_type": "task",
  "resource_id": "task-001",
  "reason": "already_exists"
}
```

Delete conflicts should use `reason: "dependent_records"` and include a small count summary when cheap to compute.
