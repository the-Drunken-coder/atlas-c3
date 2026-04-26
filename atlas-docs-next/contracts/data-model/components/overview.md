# Component Contracts

This folder defines shared component contracts used inside entity JSON.

Components are named, structured current-state sections under `entity.json.components`. They are part of the shared data contract when their key is listed here.

## Update Semantics

`PATCH` should replace named components intentionally rather than deep-merging arbitrary nested fields.

This avoids accidental partial updates that corrupt nested component state. Resource-specific API docs may define the exact patch shape, but they should preserve the rule that named component replacement is explicit.

## Timestamp Convention

Components that represent reported or observed state should include `observed_at` as an RFC 3339 timestamp.

`observed_at` means when the represented state was observed by the writer or source system. It is not the same as the entity row's `updated_at`, which advances when Atlas Core stores any entity change.

## Unknown Components

Unknown component keys should be rejected unless they use the `custom_*` prefix.

`custom_*` components:

- are allowed as extension data
- must be JSON objects
- receive only lightweight validation for basic JSON shape and size limits
- are not indexed or queried directly
- are not stable shared contracts

## Object References

Object relationships should stay in object ownership fields. Components should not duplicate object links unless a later contract defines a specific exception and drift-prevention rule.

The old `media_refs` component does not carry forward into this contract. Object ownership through `objects.owner_type` and `objects.owner_id` is the authoritative relationship.

## Promotion Rule

Component fields should only be promoted out of JSON when repeated querying or indexing proves it is necessary.

## Canonical Entity Components

| Component | Applies to | Contract |
| --- | --- | --- |
| `heartbeat` | assets | [`heartbeat.md`](./heartbeat.md) |
| `communications` | assets | [`communications.md`](./communications.md) |
| `health` | assets | [`health.md`](./health.md) |
| `telemetry` | assets, tracks | [`telemetry.md`](./telemetry.md) |
| `geometry` | geofeatures | [`geometry.md`](./geometry.md) |
| `sensor_refs` | assets, tracks | [`sensor_refs.md`](./sensor_refs.md) |
| `status` | assets, tracks, geofeatures | [`status.md`](./status.md) |
| `supported_commands` | assets | [`supported_commands.md`](./supported_commands.md) |
| `fusion_summary` | tracks | [`fusion_summary.md`](./fusion_summary.md) |

## Asset Validity

Asset entities must include `json.components.supported_commands`. Assets without this component are invalid.

An asset with an empty `supported_commands.commands` array is valid, but Atlas Core must reject task creation for that asset because it cannot perform any commands.

## Derived Task Queue

Task records are the authoritative source for assigned, active, and completed work.

If Atlas Core or a client exposes a task queue view, it should be derived from task records. Assets should not write a `task_queue` component as independent authority.
