# Glossary

This document is the source of truth for shared ATLAS-C3 vocabulary. Other docs should link here instead of redefining these terms.

## Core System Terms

### ATLAS-C3

The full command, control, and communications system for coordinating field assets, tracking operational state, dispatching work, and collecting evidence from the field.

### System of Record

The authoritative place where current operational state is stored. In ATLAS-C3, Atlas Core is the system of record for entities, observations, tasks, objects, command catalog materialization, and current control-plane state.

### Module

A major independently documented system area, such as Atlas Core, Atlas SDK, Atlas Command Interface, or an asset runtime. Module-specific implementation details belong in `../module-docs/`.

### Contract

A shared interface that multiple modules must obey. Examples include API endpoints, data models, SDK method surfaces, event stream shapes, and asset-to-core protocols. Contract details belong in `../contracts/`.

## Runtime Modules

### Atlas Core

The central server and system of record. It owns persistence, validation, object storage integration, observation state, task state, command catalog materialization, broad reads, and live change publication.

### Atlas SDK

The first-party client package used by ATLAS-C3 tools to interact with Atlas Core through shared, typed interfaces instead of raw ad hoc HTTP calls.

### Atlas Command Interface

The operator-facing web application. It presents operational state, supports inspection and task authoring, and talks to Atlas Core through Atlas SDK.

### Asset Runtime

The software running on a field asset. It reports state, receives or polls for assigned work, executes tasks, updates task status, and produces observations or other payloads.

## Operational Records

### Entity

A current-state record for something ATLAS-C3 knows about. Entities include assets, tracks, and geofeatures.

### Asset

A taskable entity that can be directed by the system. Examples include drones, rovers, fixed cameras, sensors, or other field systems.

### Track

An entity representing something observed in the environment, such as a person, vehicle, aircraft, or other detected subject.

### Geofeature

An entity representing a spatial feature on the map, such as a point, line, polygon, route, zone, or marker.

### Task

A work item assigned to an entity, normally an asset. Tasks describe what should be done, who or what it is assigned to, and the current execution status.

### Object

A metadata record for stored files and payload containers, such as media, command catalogs, evidence artifacts, heatmaps, or files attached to another record. Object bytes are stored separately from object metadata.

### Object File

One stored file belonging to an object. A single object may contain zero, one, or many files.

### Observation

Evidence produced by an asset or sensor about something it detected or measured. Observations are first-class operational records, separate from objects. Files associated with an observation are stored through objects and linked back to the observation.

### System Track

The system's authoritative current representation of a real-world subject after observation evidence has been interpreted or fused.

## Shared Data Concepts

### Component

A named piece of structured state inside an entity or task payload. Components let the data model evolve without creating a separate table or object shape for every feature.

### Custom Component

An integration-owned component whose key begins with `custom_`. Custom components are for subsystem-specific payloads that should not become first-class shared concepts yet.

### Promoted Field

A field stored as a first-class indexed column rather than only inside flexible metadata. Promoted fields are for stable values that the system queries, filters, sorts, or joins on.

### Command

A named action that an asset may execute, such as moving to a location or capturing data. The command defines the requested action; a task records a concrete request to run it.

### Command Catalog

The system-wide registry of known commands and their parameter shapes. Assets advertise which catalog commands they support.

### Task Queue

A derived view of pending and active tasks for an asset. The authoritative task state remains the task records themselves.

## Sync And Runtime Behavior

### Check-In

An asset-to-core interaction where an asset reports current state and receives or discovers relevant assigned work.

### Bulk Query

A broad read of current system state used for bootstrapping, recovery, or state refresh. A bulk query is not a durable event log.

### Live Change Stream

A live-only feed of current mutations, such as creates, updates, and deletes. The stream helps clients keep local state fresh but is not the system of record.

### Replica

An in-memory client-side copy of current state maintained from bulk reads and live updates. A replica is temporary and must recover from Atlas Core, not from local persistence.

