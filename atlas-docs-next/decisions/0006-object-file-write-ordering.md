# 0006: Object File Write Ordering And Failure Behavior

## Status

Accepted

## Context

Atlas Core stores object file metadata in PostgreSQL and stores object file bytes on a filesystem volume.

Filesystem writes are not part of PostgreSQL transactions. A naive implementation can leave the system in inconsistent states such as:

- database rows exist that point at missing bytes
- bytes exist on disk that are not referenced by committed metadata

The storage shape decision notes this risk in [`0002-core-storage-shape.md`](./0002-core-storage-shape.md).

## Decision

Atlas Core should treat **committed PostgreSQL metadata as the source of truth for what files exist**.

Object file uploads should follow a **staging-first** sequence:

1. Write incoming bytes to a **temporary path** under the configured object storage root.
2. Commit object file metadata in PostgreSQL **only after** the bytes are fully written and validated for basic invariants (size limits, path safety, content type presence).
3. Move or rename the staged bytes to the final logical path as part of the same successful upload operation after metadata commit, or write directly to the final path only if the implementation can guarantee atomic publish semantics without exposing partial state.

If PostgreSQL commit fails after bytes were staged, Atlas Core should delete the staged bytes and return a failure. No durable metadata row should reference missing bytes.

If byte finalization fails after metadata commit succeeds, Atlas Core should treat this as a **serious operational failure**: log with enough context to repair, surface readiness degradation where appropriate, and avoid claiming a healthy storage subsystem.

Atlas Core should not expose partially uploaded files as readable object file metadata.

## Consequences

- Upload implementations must include explicit cleanup paths for staged bytes.
- Readiness and dependency checks should treat “metadata/bytes mismatch” as unhealthy rather than silently continuing.
- Clients still recover from missed stream events by refreshing current state, but the system should not pretend bytes exist when they do not.

## Rejected Alternatives

- **Best-effort metadata without byte guarantees** - hides failures and breaks clients that trust object file metadata.
- **Commit metadata before bytes exist** - creates durable dangling references.
- **Two visible states (`pending`/`ready`) in the current contract** - adds API and query complexity before it is proven necessary.
