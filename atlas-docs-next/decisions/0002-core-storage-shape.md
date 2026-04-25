# 0002: Keep Core Storage Simple

## Status

Accepted

## Context

Atlas Core needs to store structured operational records and file bytes.

Structured records need queryable fields, relationships, validation, and transactional updates. File bytes need storage, streaming access, and predictable paths, but they should not be stored directly in PostgreSQL rows.

Core data is short-lived operational state per [`0003-short-lived-operational-data.md`](./0003-short-lived-operational-data.md), so this storage shape does not imply long-term archival guarantees.

## Decision

Atlas Core will use one PostgreSQL database for structured state and one filesystem volume for object file bytes.

PostgreSQL owns structured records and metadata, including:

- entities
- observations
- tasks
- objects
- object file metadata
- command catalog materialization records

The filesystem volume owns object file bytes.

Objects are the file-container system. Files associated with observations, tasks, command catalogs, or other records should be stored through objects and linked back to the owning record.

## Consequences

- Object metadata and object file bytes are separate concerns.
- PostgreSQL stores logical file paths and metadata, not file bytes.
- Atlas Core readiness should fail when required database or filesystem storage dependencies are unavailable.
- File write consistency needs a simple, explicit approach because filesystem writes do not roll back automatically with database transactions.
- There is no separate object database service in the initial architecture.

## Rejected Direction

Do not introduce a separate object storage service, object database, or blob database for the initial system. The filesystem volume is the storage backend until a real requirement justifies changing it.

