# Command Catalog

This document defines the command catalog model.

The command catalog is authored as a checked-in JSON file in the Atlas Core codebase. At runtime Atlas Core validates that JSON, materializes it as a normal object plus JSON object file, and exposes the active catalog object ID through the service descriptor.

## Role

The command catalog is the system-wide list of commands Atlas Core knows how to validate and expose to clients.

It answers:

- what command types exist
- what each command means
- what parameters each command accepts
- which command catalog object a task was validated against during the current run

Asset-specific command support is separate. Assets advertise which catalog commands they support through their entity state.

## Authored JSON Shape

The checked-in catalog file should use:

```json
{
  "catalog_id": "default-command-catalog",
  "version": "2026-01-01",
  "authored_at": "2026-01-01T00:00:00Z",
  "commands": [
    {
      "type": "move_to_location",
      "display_name": "Move to Location",
      "description": "Direct an asset to move toward a target location.",
      "parameters_schema": {
        "type": "object",
        "required": ["latitude", "longitude"],
        "additionalProperties": false,
        "properties": {
          "latitude": {
            "type": "number",
            "minimum": -90,
            "maximum": 90
          },
          "longitude": {
            "type": "number",
            "minimum": -180,
            "maximum": 180
          }
        }
      }
    }
  ]
}
```

Required catalog fields:

- `catalog_id`
- `version`
- `commands`

Optional catalog fields:

- `authored_at`
- `description`
- `metadata`

Required command fields:

- `type`
- `display_name`
- `description`
- `parameters_schema`

Command types must be lowercase `snake_case`, max 50 characters, and unique within the catalog.

## Parameter Schema Subset

Command parameter validation uses a restricted JSON Schema subset.

Supported schema keywords:

- `type`: `object`, `array`, `string`, `number`, `integer`, `boolean`, or `null`
- `properties` for object schemas
- `required` for object schemas
- `additionalProperties` as `true` or `false`
- `items` for array schemas
- `enum`
- `minimum` and `maximum` for numbers and integers
- `minLength`, `maxLength`, and `pattern` for strings
- `minItems` and `maxItems` for arrays

Unsupported schema keywords must make catalog validation fail. Atlas Core should not silently ignore unknown validation rules.

Initial command schemas should prefer `additionalProperties: false` for object parameters so misspelled parameters fail clearly.

## Runtime Materialization

When Atlas Core starts:

1. Load the checked-in command catalog JSON.
2. Validate the catalog shape and every command schema.
3. Store the parsed catalog in memory for task validation.
4. Materialize the exact authored JSON as a normal object with `type: "command_catalog"`, `owner_type: "system"`, and `owner_id: "active_command_catalog"`.
5. Store the catalog JSON as an object file with content type `application/json`.
6. Expose the materialized object ID as `active_command_catalog_object_id` from `GET /`.

The object and file use the normal object API shapes. There is no command-catalog-specific table or endpoint.

If the checked-in command catalog is invalid, Atlas Core should fail startup/readiness with `catalog_unavailable` and log enough context to identify the invalid command or schema.

## Object Relationship

The command catalog object is a normal object-backed payload.

Expected object values:

```json
{
  "object_id": "command-catalog-20260101",
  "type": "command_catalog",
  "owner_type": "system",
  "owner_id": "active_command_catalog",
  "json": {
    "catalog_id": "default-command-catalog",
    "version": "2026-01-01"
  }
}
```

The object file bytes contain the full authored catalog JSON. The object `json` may contain only summary metadata needed for listing and debugging.

## Relationship To Tasks

When a task is created:

1. Validate `command_catalog_object_id` equals the active command catalog object ID.
2. Validate `json.components.command.type` exists in the active in-memory catalog.
3. Validate `json.components.parameters` against that command's `parameters_schema`.
4. Check that the target asset has `json.components.supported_commands`.
5. Check that the requested command type is listed in the target asset's `supported_commands.commands`.
6. Store the active command catalog object's ID on the task.

After creation, validation and retries for that task should resolve command definitions through the task's stored `command_catalog_object_id`, not through a later globally active catalog value.

Task command and parameter edit rules are defined in [`../tasks.md`](../tasks.md). Task API behavior is defined in [`../../core-api/tasks.md`](../../core-api/tasks.md).

## Store Ownership

The command catalog does not need a dedicated store.

Atlas Core needs:

- startup loading from the checked-in JSON file
- validation helpers that read the active in-memory catalog
- object creation through the object store
- task creation logic that pins the active catalog object

Those responsibilities belong to Core startup, services, and object-store behavior. They should not create a separate command catalog database or store boundary.
