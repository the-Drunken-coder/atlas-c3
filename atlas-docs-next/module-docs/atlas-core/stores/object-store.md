# Object Store

The object store owns object metadata, object file metadata, and object file bytes as one coordinated internal boundary.

It supports the API docs for objects and object files.

## Owned Record Families

- objects
- object files
- file bytes on the filesystem volume

Record-family ownership is defined in [`../../../contracts/data-model/record-families.md`](../../../contracts/data-model/record-families.md). Storage shape is defined in [`../../../decisions/0002-core-storage-shape.md`](../../../decisions/0002-core-storage-shape.md).

## Object Capabilities

Required capabilities:

- create object with caller-supplied `object_id`
- read object by `object_id`
- list objects with pagination
- list objects by `owner_type` and `owner_id`
- patch object metadata
- delete object

Objects are the authoritative owners of relationship links to records that use them.

## Object File Capabilities

Required capabilities:

- upload object file with caller-supplied `file_id`
- append bytes to an existing object file
- create object file metadata
- read object file metadata
- stream object file bytes
- delete object file metadata
- delete object file bytes
- list files for an object

File upload and append should coordinate metadata update and byte write steps carefully.

Success is reported only after both the byte write and the PostgreSQL metadata update are complete. The public API should not report partial success.

For file upload, the object store should write bytes to a staging path first, then create object file metadata and update the parent object metadata in one PostgreSQL transaction, then promote the staged bytes to the committed path. If the metadata update fails, delete the staged bytes and return `503 storage_unavailable`. If byte promotion fails after metadata commit, committed PostgreSQL metadata remains the source of truth per ADR 0006; the store should log a serious metadata/bytes mismatch, return `503 storage_unavailable` for the failed operation, and degrade object storage readiness until the mismatch is repaired. It should not silently delete committed metadata as a normal recovery path.

For append, the byte write and metadata update must stay ordered so `size_bytes` reflects committed content. If the byte write fails, leave metadata unchanged and return `503 storage_unavailable`. If the metadata update fails after bytes append, the store should make one immediate compensating attempt to truncate or remove the appended bytes; if that cleanup cannot be proven, log a storage mismatch and return `503 storage_unavailable`.

Retries follow the API-level semantics: uploads are create operations with caller-supplied `file_id`, while append is not intrinsically idempotent. Callers should not blindly retry append after an ambiguous transport failure unless they can tolerate duplicate appended bytes or use a higher-level SDK reconciliation path.

## Filesystem Capabilities

Required capabilities:

- write bytes under the managed storage root
- append bytes to existing managed files
- open bytes for streaming
- stat bytes
- delete bytes
- reject unsafe paths
- report storage readiness

PostgreSQL stores logical paths and metadata. The filesystem volume stores bytes.

## Command Catalog Materialization

The command catalog is materialized through the object store at startup.

The command catalog source is checked-in JSON. Atlas Core loads it, resolves the content-hash object ID, creates or reuses the catalog object and file, and exposes the active command catalog object ID through the service descriptor.

There is no command catalog store.

## Readiness

The object store should expose readiness for both:

- object metadata access through PostgreSQL
- filesystem byte storage access
