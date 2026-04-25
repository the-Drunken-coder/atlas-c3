# System API

System endpoints identify the service and expose process/dependency status.

Shared API behavior is defined in [`conventions.md`](./conventions.md).

## Endpoints

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/` | Service descriptor |
| `GET` | `/health` | Process health |
| `GET` | `/readiness` | Dependency readiness |

## Service Descriptor

`GET /` should return basic service information and discoverable system-owned identifiers.

It should include the active command catalog object ID so clients can read the catalog through the object API.

## Health

`GET /health` answers whether the Atlas Core process is alive.

It should not require every dependency to be usable.

## Readiness

`GET /readiness` answers whether Atlas Core can serve real requests that depend on required storage.

Readiness should fail when required PostgreSQL or object file storage dependencies are unavailable.

