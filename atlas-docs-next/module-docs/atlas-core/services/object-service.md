# Object Service

The object service owns API behavior for object and object file endpoints.

API contract: [`../../../contracts/core-api/objects.md`](../../../contracts/core-api/objects.md)

## Responsibilities

- validate object create, patch, and delete requests
- require caller-supplied `object_id` on create
- validate object owner links
- list objects by `owner_type` and `owner_id`
- coordinate object metadata through the object store
- coordinate object file uploads, metadata, byte storage, streaming, and deletion
- require caller-supplied `file_id` for object file upload
- publish object mutation events after successful writes

## Store Usage

Uses [`../stores/object-store.md`](../stores/object-store.md).

## File Upload Coordination

File upload must coordinate:

- multipart request validation
- object existence
- caller-supplied `file_id`
- file metadata creation
- byte write to the filesystem volume
- error cleanup when a normal runtime step fails

The system does not need migration or rollback machinery, but normal uploads should avoid leaving obvious contradictory metadata and bytes.

## Notes

Objects are the authoritative owners of relationship links to entities, observations, tasks, and system-owned records.

The command catalog is materialized through the object service or object store during startup, but command catalog loading itself is bootstrap behavior.

