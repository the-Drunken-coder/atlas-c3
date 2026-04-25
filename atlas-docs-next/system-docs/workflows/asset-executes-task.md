# Asset Executes Task

This workflow describes how an asset receives and reports task lifecycle state.

Task API behavior is defined in [`../../contracts/core-api/tasks.md`](../../contracts/core-api/tasks.md). Stream behavior is defined in [`../../contracts/core-api/stream.md`](../../contracts/core-api/stream.md).

## Flow

1. Asset discovers assigned work by calling `GET /tasks?asset_id={asset_id}&status=pending`, `GET /entities/{asset_id}/tasks`, or by listening for task stream events.
2. Asset reads `GET /tasks/{task_id}` before acting if local state is stale or incomplete.
3. Asset accepts work with `POST /tasks/{task_id}/status` and `status: "acknowledged"`.
4. Atlas Core returns the updated task and publishes `task.updated`.
5. Asset finishes with `POST /tasks/{task_id}/status` and `status: "completed"` or `status: "failed"`.
6. Atlas Core returns the updated terminal task and publishes `task.updated`.

## Idempotency

Repeating the current status is an idempotent no-op and returns `200` with the current task.

Terminal statuses are final. `completed` and `failed` do not transition back to active statuses.

## Out Of Scope

Reject, start, and progress-update lifecycle states are intentionally out of scope until the task status model expands.
