# System API

System endpoints identify the service and expose process/dependency status.

Shared API behavior is defined in [`conventions.md`](./conventions.md). Error codes are defined in [`errors.md`](./errors.md).

## Endpoints

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/` | Service descriptor |
| `GET` | `/health` | Process health |
| `GET` | `/readiness` | Dependency readiness |

## Service Descriptor

`GET /` returns basic service information and discoverable system-owned identifiers when required startup state is available.

Statuses:

| Status | Condition |
| --- | --- |
| `200 OK` | Service descriptor is available and the active command catalog object ID has been materialized |
| `503 catalog_unavailable` | The command catalog is missing, invalid, or not materialized |

Response body:

```json
{
  "service": "atlas-core",
  "status": "ready",
  "version": "0.1.0",
  "started_at": "2026-01-01T00:00:00Z",
  "active_command_catalog_object_id": "command-catalog-20260101",
  "links": {
    "health": "/health",
    "readiness": "/readiness",
    "full_query": "/queries/full",
    "stream": "/stream/changes"
  }
}
```

`active_command_catalog_object_id` must be present only after the checked-in command catalog has been validated and materialized as an object. If it is unavailable, `GET /` must return `503 catalog_unavailable`.

## Health

`GET /health` answers whether the Atlas Core process is alive. It does not require every dependency to be usable.

Response body:

```json
{
  "status": "ok",
  "service": "atlas-core",
  "timestamp": "2026-01-01T00:00:00Z"
}
```

## Readiness

`GET /readiness` answers whether Atlas Core can serve real requests that depend on required storage.

Ready response body:

```json
{
  "status": "ready",
  "timestamp": "2026-01-01T00:00:00Z",
  "dependencies": {
    "postgres": "ready",
    "object_storage": "ready",
    "command_catalog": "ready"
  }
}
```

If PostgreSQL or object file storage is unavailable or inconsistent, return `503 storage_unavailable`. If the command catalog is missing, invalid, or not materialized, return `503 catalog_unavailable`.
