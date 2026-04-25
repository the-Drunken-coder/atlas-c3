# Status

Operator-visible posture for an entity.

**Applies to:** assets, tracks, and geofeatures.

## Shape

```json
{
  "components": {
    "status": {
      "observed_at": "2026-01-01T00:00:00Z",
      "value": "idle"
    }
  }
}
```

## Fields

| Field | Type | Required | Constraint | Description |
| --- | --- | --- | --- | --- |
| `observed_at` | RFC 3339 timestamp | yes | valid timestamp | When this posture was set or observed |
| `value` | enum | yes | `idle` / `on_mission` / `rtb` / `unknown` | High-level operator-visible posture |

## Usage Notes

- Status is display-oriented metadata, not behavioral authority.
- Use task records for assigned work and communications for reachability.
- `unknown` is valid when the UI should avoid implying a posture.
