# Event Publisher

The event publisher supports live change notifications for the stream API.

API contract: [`../../../contracts/core-api/stream.md`](../../../contracts/core-api/stream.md)

## Responsibilities

- publish live create, update, and delete events after successful mutations
- support the live server-sent events stream
- avoid durable event-log behavior
- avoid replay requirements
- keep stream payloads aligned with API response shapes once those are defined

## Store Usage

The event publisher does not need a durable store.

Services should call the event publisher after successful writes. If event publication fails, the service behavior should be defined later before implementation.

## Notes

The stream is live-only. Clients recover missed events by performing fresh reads.

Exact event envelope, event types, and payload rules belong in the stream API contract before implementation.

