# Tasks API

Task endpoints manage work assigned to assets.

Task record ownership is defined in [`../data-model/record-families.md`](../data-model/record-families.md). Shared API behavior is defined in [`conventions.md`](./conventions.md).

## Endpoints

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/tasks` | List tasks |
| `POST` | `/tasks` | Create a task |
| `GET` | `/tasks/{task_id}` | Read a task |
| `PATCH` | `/tasks/{task_id}` | Update a task |
| `DELETE` | `/tasks/{task_id}` | Delete a task |
| `POST` | `/tasks/{task_id}/status` | Transition task status |

## Notes

Create requests must include `task_id`.

Tasks target assets.

Task-related objects should be queried through [`objects.md`](./objects.md) using `owner_type=task` and `owner_id={task_id}`.

The active command catalog is exposed through objects. Tasks should pin the active command catalog object used for validation during the current run.

