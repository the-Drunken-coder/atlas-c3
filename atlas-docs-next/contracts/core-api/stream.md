# Stream API

The stream endpoint publishes live changes so clients can refresh or update local state.

Shared API behavior is defined in [`conventions.md`](./conventions.md).

## Endpoints

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/stream/changes` | Live change stream |

## Stream Behavior

The stream should use server-sent events.

The stream is live-only:

- it does not replay missed history
- it is not the system of record
- clients should recover missed state by performing a fresh read

## Object And File Events

Object metadata create, update, and delete operations should emit object events.

Object file upload and delete operations should also emit object events because they change the object/file metadata visible through the object API.

Object events should not include file bytes. Clients that need file content should fetch it through the object file content endpoint.

Atlas Core should not support mutating file bytes in place as a separate event category. If file bytes need to change, callers should use delete/upload or a later replace-style operation that updates object file metadata and emits an object event.

## Event Detail

Detailed event envelope and payload rules belong in this document later.

