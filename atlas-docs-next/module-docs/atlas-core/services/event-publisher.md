# Event Publisher

The event publisher supports live change notifications for the stream API.

API contract: [`../../../contracts/core-api/stream.md`](../../../contracts/core-api/stream.md)

## Responsibilities

- publish entity create, update, and delete events after successful entity mutations
- publish observation create, update, and delete events after successful observation mutations
- publish task create, update, and delete events after successful task mutations
- publish object create, update, and delete events after successful object mutations
- publish object events after successful object file upload or delete
- support the live server-sent events stream
- avoid durable event-log behavior
- avoid replay requirements
- keep stream payloads aligned with API response shapes once those are defined

## Store Usage

The event publisher does not need a durable store.

Services should call the event publisher after successful writes.

## Failure Policy

Atlas Core should use commit-first, best-effort publishing.

If a mutation commits successfully but event publication fails:

- the original API request should still succeed
- the failure should be logged with enough context to identify the missed event
- the response should not expose a partial failure to the caller
- no durable retry queue or dead-letter queue is required
- no rollback or compensation is required
- clients recover missed changes by performing a fresh read

The event publisher should make a single in-process publish attempt. There is no exponential backoff or retry queue in the initial design.

If a future implementation adds internal retries, events must carry a unique event ID so duplicate deliveries can be ignored by clients. Failed publications should be observable through logs at minimum; metrics or alerts can be added later if the runtime grows an observability system.

## Notes

The stream is live-only. Clients recover missed events by performing fresh reads.

Exact event envelope, event types, and payload rules belong in the stream API contract before implementation.

