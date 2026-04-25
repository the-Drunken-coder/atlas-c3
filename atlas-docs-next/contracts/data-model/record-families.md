# Record Families

This document is the high-level source of truth for Atlas Core record families. It defines the major kinds of records, what each owns, and how they relate.

Shared vocabulary lives in [`../../system-docs/glossary.md`](../../system-docs/glossary.md). This document should not redefine those terms beyond the data-model context.

## Storage Shape

Atlas Core uses one PostgreSQL database for structured records and metadata, plus one filesystem volume for object file bytes.

The storage decision is recorded in [`../../decisions/0002-core-storage-shape.md`](../../decisions/0002-core-storage-shape.md). The short-lived data assumption is recorded in [`../../decisions/0003-short-lived-operational-data.md`](../../decisions/0003-short-lived-operational-data.md).

## Planned Record Families

| Record family | Structured storage | Owns |
| --- | --- | --- |
| Entity | `entities` | Current state for assets, tracks, and geofeatures |
| Observation | `observations` | First-class sensor evidence over time |
| Task | `tasks` | Work assigned to assets |
| Object | `objects` | File-container metadata and payload grouping |
| Object File | `object_files` | Per-file metadata for bytes stored on the filesystem volume |

Detailed contracts:

- [`entities.md`](./entities.md)
- [`observations.md`](./observations.md)
- [`tasks.md`](./tasks.md)
- [`objects.md`](./objects.md)

## Shared Conventions

Unless a later contract overrides them, record families should follow these conventions:

- IDs are creator-supplied strings up to 50 characters.
- Timestamps are stored as PostgreSQL timestamp values and serialized as RFC 3339 strings.
- Queryable, stable fields should be promoted into columns.
- Evolving or record-specific state may live in a JSON metadata shape.
- Promoted fields should not be duplicated inside JSON metadata.
- Coordinates use WGS84. Latitude and longitude are decimal degrees; distances are meters; speeds are meters per second; angles are degrees.

## Entity Records

Entities represent current-state things the system knows about. The detailed entity contract is [`entities.md`](./entities.md).

Entity types:

- `asset` - taskable field systems such as drones, rovers, cameras, sensors, or other systems the operator can direct.
- `track` - system-known subjects in the environment, usually produced or maintained from observation evidence.
- `geofeature` - spatial map features such as points, lines, polygons, routes, zones, or markers.

Entities should share one record family rather than splitting assets, tracks, and geofeatures into unrelated tables. Type and subtype fields distinguish the kind of entity.

Entity state may use components for structured state that evolves over time. Component rules should be defined in component contracts rather than copied into every entity-related doc.

Important entity relationships:

- Tasks target assets.
- Objects may be linked to an entity when they contain related media or payloads.
- Tracks are authoritative current truth and may be influenced by observations.
- Assets may produce observations.

## Observation Records

Observations are first-class evidence records. They are not objects with `type = observation`. The detailed observation contract is [`observations.md`](./observations.md).

The first-class observation decision is recorded in [`../../decisions/0001-first-class-observations.md`](../../decisions/0001-first-class-observations.md).

Observations should capture source-owned evidence about something detected, measured, or reported over time. A single observation may start when an asset detects something and then update as the event develops.

Observation state should support:

- source asset identity
- first observed time
- most recent observed time
- kinematic evidence
- classification evidence
- identity evidence
- confidence or uncertainty information
- related object records that store files for the observation

Observation files should be stored through objects. The observation owns the evidence semantics; the object owns file-container metadata, link metadata, and file bytes.

Important observation relationships:

- An observation is produced by an asset or sensor.
- An observation may have one or more related objects containing files such as images, clips, sensor captures, or sidecar data.
- Data fusion may consume observations to create or update tracks.
- An observation is evidence, while a track is authoritative current system truth.

## Task Records

Tasks represent work assigned to an asset. The detailed task contract is [`tasks.md`](./tasks.md).

Tasks should own:

- task identity
- current status
- target asset reference
- command request
- command parameters
- progress or result metadata
- command catalog pin

The task record is the authority for task lifecycle state. Derived views such as an asset task queue should not override task records.

The starting lifecycle from the old docs remains a good baseline:

```text
pending -> acknowledged -> completed
                      \-> failed
```

Important task relationships:

- A task targets an asset.
- A task pins the command catalog object used to validate and interpret its command.
- Objects may be linked to tasks for attachments, results, evidence, or generated payloads.

## Object Records

Objects are file containers and payload metadata records. The detailed object and object file contract is [`objects.md`](./objects.md).

Objects should own:

- object identity
- object type or purpose
- object-level metadata
- authoritative links back to records that use the object
- grouping for one or more object files

Objects do not own observation lifecycle rules. If an object stores files for an observation, the observation remains the source of truth for observation semantics.

Expected object uses:

- observation media
- command catalog payloads
- task attachments or results
- geofeature media such as heatmaps
- evidence artifacts
- captured sensor dumps

Important object relationships:

- An object may be owned by an entity, observation, task, or system-owned record.
- An object may contain zero, one, or many object files.
- File bytes live outside PostgreSQL on the object file volume.

Object relationship links should have one authoritative direction: the object records which entity, observation, task, or system-owned record uses it. Owning records should not duplicate object references in their own JSON unless a later contract identifies a specific reason and defines how drift is prevented.

Task command catalog pinning is an explicit exception. The task contract requires `command_catalog_object_id` as a pinned object reference so task validation can keep using the catalog object captured at task creation time; see [`tasks.md`](./tasks.md).

## Object File Records

Object file records store metadata for bytes belonging to an object.

Object file metadata should own:

- file identity
- parent object reference
- logical storage path
- content type
- size
- usage hint or role
- timestamps

The filesystem volume owns the bytes. PostgreSQL owns the metadata and logical path.

## Command Catalog Records

The command catalog remains stored through the object system.

The source command catalog should live in the Atlas Core codebase as a checked-in JSON file. At startup, Atlas Core should materialize that JSON file into an object. Tasks should pin the command catalog object used to validate and interpret their command during that run.

The command catalog contract is [`command-catalog/overview.md`](./command-catalog/overview.md).

## Concurrency Scope

Optimistic concurrency should be used only where realistic multi-writer collisions can happen. It should not be applied broadly to every endpoint by default.

## Validation Boundaries

Later contracts should identify whether each rule is enforced by:

- database constraint
- API/runtime validation
- writer obligation
- reader assumption

This distinction from the old docs is worth keeping. It makes clear which rules are hard guarantees and which rules require writers or readers to behave correctly.

