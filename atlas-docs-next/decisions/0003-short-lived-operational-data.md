# 0003: Treat Core Data As Short-Lived Operational State

## Status

Accepted

## Context

ATLAS-C3 is not planned as a long-lived enterprise data system. The system runs for bounded operational windows, and the data stored by Atlas Core is not expected to require long-term schema evolution, rollback, archival protection, or migration management.

This changes the amount of persistence complexity the system should carry.

## Decision

Atlas Core data is treated as short-lived operational state.

Atlas Core should not include a database migration framework, rollback system, archival data-protection layer, or long-term data-retention machinery under the current operating model.

Schema setup should remain simple and startup-owned. During development or bounded deployments, destructive reset is acceptable when the stored data no longer matches the expected schema.

## Consequences

- Schema changes do not need migration files.
- Rollback planning for stored operational data is out of scope.
- Long-term retention, archival backups, and historical database compatibility are out of scope.
- Implementation should favor simple reset/rebuild workflows over compatibility layers.
- Contracts should still be precise, because clients and modules depend on them even if stored data is short-lived.

## Rejected Direction

Do not add migration frameworks, schema rollback tooling, or long-term data protection systems unless the project explicitly changes away from the short-lived operational-state model.

