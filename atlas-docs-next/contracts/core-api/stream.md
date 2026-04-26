# Stream API

The stream endpoint publishes live changes so clients can update local state.

Shared API behavior is defined in [`conventions.md`](./conventions.md). Error codes are defined in [`errors.md`](./errors.md).

## Endpoints

| Method | Path | Success | Purpose |
| --- | --- | --- | --- |
| `GET` | `/stream/changes` | `200` SSE stream | Live change stream |

## Stream Behavior

The stream uses server-sent events with `Content-Type: text/event-stream`.

The stream is live-only:

- it does not replay missed history
- it is not the system of record
- clients recover missed state by performing a fresh read

Atlas Core may send comment keepalives such as `: keepalive` to keep connections open. Clients must ignore comment events.

If the connection closes or a client suspects missed events, the client should call `GET /queries/full` before treating local state as fully current again.

## SSE Message Format

Each mutation event uses:

```text
event: task.updated
id: evt-001
data: {"event_id":"evt-001","type":"task.updated","resource_type":"task","resource_id":"task-001","mutation":"update","occurred_at":"2026-01-01T00:00:00Z","data":{"resource":{"task_id":"task-001","status":"acknowledged"}}}
```

`id` should match `event_id`. Clients must not treat event IDs as durable replay cursors in the current operating model.

## Event Envelope

```json
{
  "event_id": "evt-001",
  "type": "task.updated",
  "resource_type": "task",
  "resource_id": "task-001",
  "mutation": "update",
  "occurred_at": "2026-01-01T00:00:00Z",
  "data": {
    "resource": {}
  }
}
```

Fields:

| Field | Meaning |
| --- | --- |
| `event_id` | Server-generated event identifier for logging and local de-duplication |
| `type` | Stable event type |
| `resource_type` | `entity`, `observation`, `task`, or `object` |
| `resource_id` | ID of the changed resource |
| `mutation` | `create`, `update`, or `delete` |
| `occurred_at` | RFC 3339 server timestamp |
| `data` | Event-specific payload, including resource data needed by SDK replica mode |

## Event Types

| Event type | Resource type | Mutation |
| --- | --- | --- |
| `entity.created` | `entity` | `create` |
| `entity.updated` | `entity` | `update` |
| `entity.deleted` | `entity` | `delete` |
| `observation.created` | `observation` | `create` |
| `observation.updated` | `observation` | `update` |
| `observation.deleted` | `observation` | `delete` |
| `task.created` | `task` | `create` |
| `task.updated` | `task` | `update` |
| `task.deleted` | `task` | `delete` |
| `object.created` | `object` | `create` |
| `object.updated` | `object` | `update` |
| `object.deleted` | `object` | `delete` |

## Event Data Defaults

Create and update events must include enough payload for Atlas SDK replica mode to update its in-memory replica without fetching the changed resource.

Entity, observation, task, and object create/update events should use:

```json
{
  "resource": {}
}
```

`resource` is the full current resource JSON using the relevant Core API resource shape:

- entities use [`entities.md`](./entities.md)
- observations use [`observations.md`](./observations.md)
- tasks use [`tasks.md`](./tasks.md)
- objects use [`objects.md`](./objects.md)

Delete events should use:

```json
{
  "deleted_at": "2026-01-01T00:00:00Z"
}
```

Delete events do not include the deleted resource snapshot by default. The envelope `resource_type` and `resource_id` are sufficient for replica removal.

## Object And File Events

Object metadata create, update, and delete operations emit object events.

File uploads, appends, and deletions also emit `object.updated` because they change object/file metadata visible through the object API.

Object event `data` should use:

```json
{
  "resource": {},
  "file_count": 3,
  "affected_files": [
    {
      "file_id": "file-001",
      "object_id": "object-001",
      "path": "objects/object-001/file-001",
      "content_type": "image/jpeg",
      "size_bytes": 12345,
      "usage_hint": "evidence_frame",
      "created_at": "2026-01-01T00:00:00Z",
      "updated_at": "2026-01-01T00:00:00Z"
    }
  ]
}
```

Rules:

- Events must not include object file bytes.
- Object create/update events include the full current object resource.
- `file_count` should be included when cheap to compute.
- `affected_files` should be included only as a short, capped list of up to 8 metadata records when the mutation touches a known small set of files.
- If more than 8 files are affected, omit `affected_files` and rely on a full-query refresh after reconnect or a later targeted object/file read outside replica event handling.
- Object file content bytes are never included in stream payloads.

The 8-file cap is a pragmatic payload-size limit. It keeps event bytes and latency bounded while still covering typical small file changes, and maintainers may revisit it if telemetry shows a different common batch size.

Atlas Core does not support mutating file bytes in place as a separate event category. If file bytes need to change under the current contract, callers should delete and upload a file or use the documented append operation.

## Publication Failures

Stored state remains the source of truth. If a mutation succeeds but event publication fails, the mutation remains committed, Atlas Core logs the publication failure, and clients recover through reads.
