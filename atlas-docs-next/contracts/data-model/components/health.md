# Health

Reported asset vitals.

**Applies to:** assets only.

## Shape

```json
{
  "components": {
    "health": {
      "observed_at": "2026-01-01T00:00:00Z",
      "battery_percent": 76
    }
  }
}
```

## Fields

| Field | Type | Required | Constraint | Description |
| --- | --- | --- | --- | --- |
| `observed_at` | RFC 3339 timestamp | yes | valid timestamp | When the health values were observed |
| `battery_percent` | integer | no | `0 <= x <= 100` | Remaining battery charge as a whole-number percentage |

## Usage Notes

- `battery_percent` is the canonical health value currently shared across assets.
- Additional vitals may be added deliberately to this component when they become shared contract fields.
- Asset-specific vitals that are not shared should use a `custom_*` component.
