# Command Catalog

This document defines the high-level command catalog model. Exact initial command definitions can be added after the first real command set is chosen.

## Role

The command catalog is the system-wide list of commands Atlas Core knows how to validate and expose to clients.

It answers:

- what command types exist
- what each command means
- what parameters each command accepts
- which command catalog object a task was validated against during the current run

Asset-specific command support is separate. Assets advertise which catalog commands they support through their entity state.

## Source Of Truth

The source command catalog lives in the Atlas Core codebase as a checked-in JSON file.

The checked-in JSON file is the authored source. It is changed with code, reviewed with code, and loaded by Atlas Core at startup.

## Catalog Shape Direction

The catalog should use a simple authored JSON structure with:

- catalog-level metadata
- command definitions
- command type
- display name
- description
- parameter schema

Command parameter validation should use a restricted JSON Schema subset rather than a custom validation language.

Command types should be lowercase `snake_case`, max 50 characters, and stable for the current system version.

## Runtime Materialization

When Atlas Core starts, it should load the checked-in command catalog JSON and materialize it into an object.

That object is the runtime catalog object for the current Core run.

Tasks should pin the runtime catalog object's `object_id` when they are created. This lets task validation and interpretation refer to the catalog version active for that run.

If the checked-in command catalog is invalid, Atlas Core should fail startup/readiness with a clear error code and log context. It should not start with a partial or best-effort command catalog.

## Store Ownership

The command catalog does not need a dedicated store.

Atlas Core needs:

- startup loading from the checked-in JSON file
- validation helpers that read the active in-memory catalog
- object creation through the object store
- task creation logic that pins the active catalog object

Those responsibilities belong to Core services and the object store. They should not create a separate command catalog database or store boundary.

## Relationship To Objects

The command catalog object is a normal object-backed payload.

Expected object usage:

- object type or purpose identifies it as a command catalog
- object file bytes contain the catalog JSON payload
- object file content type identifies the payload as JSON
- tasks pin the command catalog object used for validation

The object stores the catalog payload; the catalog contract defines what the payload means.

## Relationship To Tasks

When a task is created:

1. Validate the requested command against the active in-memory catalog.
2. Validate command parameters against the command's parameter rules.
3. Check that the target asset supports the command.
4. Store the active command catalog object's ID on the task.

Exact task fields belong in the task data contract.

Asset-supported command capabilities should reference catalog command types by string.

