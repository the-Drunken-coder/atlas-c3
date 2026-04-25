# Component Contracts

This folder defines shared component contract rules used inside entity and task JSON.

Component contracts should define:

- component name
- owning record family
- allowed shape
- validation boundaries
- applicability by entity or task type
- update semantics

## Update Semantics

`PATCH` should replace named components intentionally rather than deep-merging arbitrary nested fields.

This avoids accidental partial updates that corrupt nested component state. Resource-specific API docs may define the exact patch shape, but they should preserve the rule that named component replacement is explicit.

## Unknown Components

Unknown component keys should be rejected unless they use the `custom_*` prefix.

`custom_*` components:

- are allowed as extension data
- must be JSON objects
- receive only lightweight validation for basic JSON shape and size limits
- are not indexed or queried directly
- are not stable shared contracts

## Timestamp Convention

Component timestamps should use RFC 3339 strings when present.

## Object References

Object relationships should stay in object ownership fields. Components should not duplicate object links unless a later contract defines a specific exception and drift-prevention rule.

## Promotion Rule

Component fields should only be promoted out of JSON when repeated querying or indexing proves it is necessary.

## Initial Component Areas

Initial component contracts should be planned for:

- asset heartbeat
- asset health
- asset supported command capabilities
- asset location and state
- track kinematics
- track classification
- geofeature geometry
- task progress and status details

Until individual component contracts are written, docs may reference this overview as the canonical location for component rules.

