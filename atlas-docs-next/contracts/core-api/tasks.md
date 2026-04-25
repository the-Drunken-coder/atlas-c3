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

The active command catalog is exposed through objects. Task creation must capture and persist an immutable reference to the active command catalog object by storing `command_catalog_object_id` on the task.

For task creation, the provided `command_catalog_object_id` must match the active command catalog object used to validate `command.type` and parameters. After creation, validation and retries for that task should resolve commands through the task's stored `command_catalog_object_id`, not whatever catalog is globally active later in the process.

