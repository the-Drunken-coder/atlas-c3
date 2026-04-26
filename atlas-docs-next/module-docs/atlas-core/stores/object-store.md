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

File upload and append should coordinate metadata and byte writes carefully. The system does not need migration or rollback machinery, but normal runtime operations should avoid leaving obvious contradictory state when possible.

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

The command catalog source is checked-in JSON. Atlas Core loads it, creates an object, stores the catalog payload as an object file, and exposes the active command catalog object ID through the service descriptor.

There is no command catalog store.

## Readiness

The object store should expose readiness for both:

- object metadata access through PostgreSQL
- filesystem byte storage access
