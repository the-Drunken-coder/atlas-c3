# Observations API

Observation endpoints manage first-class sensor evidence records.

Observation record ownership is defined in [`../data-model/record-families.md`](../data-model/record-families.md), and the first-class observation decision is recorded in [`../../decisions/0001-first-class-observations.md`](../../decisions/0001-first-class-observations.md). Shared API behavior is defined in [`conventions.md`](./conventions.md).

## Endpoints

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/observations` | List observations |
| `POST` | `/observations` | Create an observation |
| `GET` | `/observations/{observation_id}` | Read an observation |
| `PATCH` | `/observations/{observation_id}` | Update an observation |
| `DELETE` | `/observations/{observation_id}` | Delete an observation |

## Notes

Create requests must include `observation_id`.

Observations can be updated over time with `PATCH`. There is no separate finalize or close endpoint.

Observation updates should use a natural observation payload. Clients should not need to know or care how observation data is stored internally.

Patch semantics for nested observation evidence must be explicit before implementation. The current planning preference is to update named evidence sections intentionally rather than accidentally shallow-merging nested data.

Observation files are stored through objects. Observation-related objects should be queried through [`objects.md`](./objects.md) using `owner_type=observation` and `owner_id={observation_id}`.

