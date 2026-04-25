# Heartbeat

Last-seen contact timing for an asset.

**Applies to:** assets only.

## Shape

```json
{
  "components": {
    "heartbeat": {
      "observed_at": "2026-01-01T00:00:00Z"
    }
  }
}
```

## Fields

| Field | Type | Required | Constraint | Description |
| --- | --- | --- | --- | --- |
| `observed_at` | RFC 3339 timestamp | yes | valid timestamp | Most recent contact time for the asset |

## Usage Notes

- Heartbeat measures contact, not a specific kind of state update.
- Writers are the asset-facing SDK path or Core service code that handles an inbound check-in.
- To judge current reachability, use [`communications`](./communications.md). Heartbeat is historical contact timing.
