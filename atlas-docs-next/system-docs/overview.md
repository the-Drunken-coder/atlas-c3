# System Overview

ATLAS-C3 is a command, control, and communications system for coordinating field assets, maintaining current operational state, dispatching work, and collecting evidence from the field.

Shared vocabulary is defined in [`glossary.md`](./glossary.md). This overview should stay focused on whole-system shape and should not duplicate contract details.

## System Purpose

ATLAS-C3 exists to give an operator and supporting software a reliable current picture of field activity:

- what assets exist and what state they are in
- what tasks have been assigned and how execution is progressing
- what observations, objects, media, and evidence have been collected
- what tracks and geofeatures are currently known
- what system state changed recently enough that clients should refresh

The system favors clear operational behavior over architectural complexity. It should be understandable, inspectable, and runnable on a simple local deployment before distributed concerns are introduced.

## Core Shape

Atlas Core is the system of record. Other modules read from it, write through it, or wrap its contracts, but they do not replace it as the authority for current operational state.

```text
Operator
   |
   v
Atlas Command Interface
   |
   v
Atlas SDK
   |
   v
Atlas Core
   |
   +--> PostgreSQL
   |
   +--> Filesystem volume (object bytes)

Assets interact with Atlas Core through the asset-side protocol chosen for the deployment.
```

The exact asset transport is a system-level design area, but transport choice should not change the ownership of operational state: current state lands in Atlas Core, and tasking decisions originate from Atlas Core.

## Modules

### Atlas Core

Atlas Core owns the central control-plane service. It persists operational records, validates writes, manages observation state, manages object metadata and file storage, materializes the command catalog, exposes broad reads, and publishes live change notifications.

Implementation details belong in `../module-docs/`. Shared API and data shapes belong in `../contracts/`.

### Atlas SDK

Atlas SDK is the first-party integration path for tools that talk to Atlas Core. It should centralize request construction, response parsing, runtime validation, error handling, and higher-level client helpers.

The public SDK surface belongs in `../contracts/` because both SDK implementation and SDK consumers depend on it.

### Atlas Command Interface

Atlas Command Interface is the operator-facing web application. It should present current state, support inspection, provide task authoring flows, and reflect confirmed server state rather than becoming a separate authority.

Screen-level and component-level design belongs in `../module-docs/`.

### Asset Runtime

The asset runtime reports state, receives assigned work, executes tasks, updates task progress, and produces observations or other payloads.

The shared asset-to-core protocol belongs in `../contracts/`. Runtime internals belong in `../module-docs/`.

### Data Fusion

Data fusion turns observation evidence into authoritative system tracks. It should preserve the distinction between evidence reported by assets and truth maintained by the system.

Fusion algorithms and implementation planning belong in `../module-docs/` unless they define shared data contracts.

## System Boundaries

ATLAS-C3 should keep these boundaries clear:

- Atlas Core is authoritative for current operational state.
- The command interface is a client, not a backend.
- First-party clients use Atlas SDK instead of duplicating raw API logic.
- Assets do not directly redefine task or object semantics.
- Observations are first-class evidence records; system tracks are authoritative current truth.
- Objects store file containers and payload metadata; observation files are stored through objects rather than making the observation itself an object.
- Object metadata and object bytes are related but distinct concerns.

## Simplicity Constraints

The system should stay simple unless a real requirement forces more complexity:

- no separate UI backend while Atlas Core can serve the needed API
- no message broker for the initial live-change path
- no distributed cache as part of the core design
- no premature microservice split
- no fake or mocked product data in development or production behavior

These constraints can be revised by a decision record in `../decisions/`, but they should not be bypassed casually in module docs.

## Documentation Ownership

Use this overview for cross-system shape only.

- Store exact shared interfaces in `../contracts/`.
- Document module implementation details in `../module-docs/`.
- Record durable architecture choices in `../decisions/`.
- Link to authoritative docs instead of repeating their rules.

