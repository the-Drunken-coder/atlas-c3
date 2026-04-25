# Observability And Run Logs

This document defines the ATLAS-C3 logging and debugging contract.

Logging is intended to reveal unknown inefficiencies as well as explain known failures. Logs should support both a combined timeline view and focused per-category inspection.

## Log Format

Logs should be JSON Lines.

Each line is one JSON object. Required fields:

| Field | Meaning |
| --- | --- |
| `timestamp` | RFC 3339 timestamp |
| `run_id` | Identifier shared by all services in one run |
| `service` | Service or process name, such as `atlas-core`, `atlas-sdk`, or `atlas-data-fusion` |
| `component` | Internal component or subsystem |
| `level` | `debug`, `info`, `warn`, or `error` |
| `event` | Stable event name |
| `message` | Human-readable summary |

Optional common fields:

| Field | Meaning |
| --- | --- |
| `correlation_id` | Cross-service operation identifier |
| `request_id` | Single HTTP request identifier |
| `duration_ms` | Operation duration in milliseconds |
| `error_code` | Stable Core or SDK error code |
| `details` | Event-specific structured context |

## Resource Context

Logs should include relevant resource IDs when present:

- `entity_id`
- `task_id`
- `observation_id`
- `object_id`
- `file_id`
- `event_id`

Resource IDs should live at the top level when they are central to the event. Extra context belongs in `details`.

## Required Categories

Implementations should support these categories as `component` or `event` prefixes:

- startup/shutdown
- API
- storage
- object files
- stream/events
- assets
- tasks
- observations
- data fusion
- SDK replica
- simulation/debugging

## API Request Logs

Request-handling logs should include:

- `request_id`
- HTTP method
- path
- status
- duration
- error code when present

Request IDs should be returned or exposed in errors where practical so an operator can connect client-visible failures to logs.

## Correlation IDs

Cross-service workflows should use `correlation_id`.

Examples:

- asset registration and first heartbeat
- task creation through execution
- observation creation through object file upload
- data fusion processing an observation into a track update
- SDK replica full refresh after stream disconnect

## Combined And Category Logs

The system should support one combined run timeline and optional category or service logs.

The combined timeline should preserve enough ordering to reconstruct what happened during a run. Category logs may duplicate entries from the combined timeline when that makes focused inspection easier.

## Duration Fields

Include duration fields for:

- API requests
- PostgreSQL operations when visible at service boundaries
- object file writes, reads, and deletes
- stream publication
- data fusion processing
- SDK replica full-query hydration and refresh

Durations are important because logs should help find inefficiencies even when the system appears to work.

## Failure Honesty

Logs should avoid fake success states.

If a write, file operation, event publish, task update, stream application, or fusion step fails, the log should make the failure visible enough to diagnose the sequence that led to it.
