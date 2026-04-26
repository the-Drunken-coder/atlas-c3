# Communications

Current network-link state for an asset.

**Applies to:** assets only.

## Shape

```json
{
  "components": {
    "communications": {
      "observed_at": "2026-01-01T00:00:00Z",
      "link_state": "healthy"
    }
  }
}
```

## Fields

| Field | Type | Required | Constraint | Description |
| --- | --- | --- | --- | --- |
| `observed_at` | RFC 3339 timestamp | yes | valid timestamp | When this link state was observed |
| `link_state` | enum | yes | `healthy` / `degraded` / `disconnected` / `unknown` | Current connectivity state for the asset |

## Usage Notes

- `degraded` covers intermittent connectivity or unreliable transport, not merely slow data.
- `unknown` means Core has no current basis to judge link state.
- A gateway may write this on an asset's behalf when it owns connectivity knowledge.
