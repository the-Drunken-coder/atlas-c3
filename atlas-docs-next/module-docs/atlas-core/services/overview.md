# Service Overview

This folder plans Atlas Core service-layer boundaries.

Services sit between HTTP handlers and stores. They own Core behavior: validation, lifecycle rules, cross-store coordination, event publication, and response-ready orchestration.

These docs define responsibilities, not final Go package names or method signatures.

## Inputs

Service planning follows:

- [`../../../contracts/core-api/endpoint-layout.md`](../../../contracts/core-api/endpoint-layout.md)
- [`../../../contracts/core-api/conventions.md`](../../../contracts/core-api/conventions.md)
- [`../../../contracts/data-model/record-families.md`](../../../contracts/data-model/record-families.md)
- [`../stores/overview.md`](../stores/overview.md)

## Planned Services

- [`entity-service.md`](./entity-service.md)
- [`observation-service.md`](./observation-service.md)
- [`task-service.md`](./task-service.md)
- [`object-service.md`](./object-service.md)
- [`query-service.md`](./query-service.md)
- [`event-publisher.md`](./event-publisher.md)

## Non-Service Startup Work

Command catalog materialization is startup/bootstrap behavior, not a service.

At startup, Atlas Core should load the checked-in command catalog JSON, materialize it through the object store, keep the active catalog available for task validation, and expose the active catalog object ID through the service descriptor.

The command catalog contract is [`../../../contracts/data-model/command-catalog/overview.md`](../../../contracts/data-model/command-catalog/overview.md).

## Handler Relationship

HTTP handlers should:

- decode requests
- call the appropriate service
- serialize service results
- use shared error handling

Handlers should not own persistence logic, lifecycle rules, command validation, object-file consistency, or event publication.

## Store Relationship

Stores provide persistence capabilities. Services decide when and how those capabilities are used to satisfy API behavior.

