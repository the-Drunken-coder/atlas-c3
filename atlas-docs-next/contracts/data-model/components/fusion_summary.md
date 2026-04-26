# Fusion Summary

Compact provenance summary for a fused track.

**Applies to:** tracks only.

## Shape

```json
{
  "components": {
    "fusion_summary": {
      "observed_at": "2026-01-01T00:00:00Z",
      "fusion_run_id": "fusion-run-001",
      "source_observation_ids": ["obs-001", "obs-002"],
      "confidence": 0.82,
      "provenance_object_id": "obj-track-001-fusion-run-001"
    }
  }
}
```

## Fields

| Field | Type | Required | Constraint | Description |
| --- | --- | --- | --- | --- |
| `observed_at` | RFC 3339 timestamp | yes | valid timestamp | When the fused track state summarized here was produced |
| `fusion_run_id` | string | yes | non-empty, max 100 characters | Fusion worker run or decision identifier |
| `source_observation_ids` | array of strings | yes | observation IDs, may be empty only for manually created tracks | Observations used by the fusion decision |
| `confidence` | number | no | `0 <= x <= 1` | Summary confidence for the fused track state |
| `provenance_object_id` | string | no | object ID | Track-owned object containing detailed fusion reasoning |

## Usage Notes

- This component keeps normal track reads small and operationally useful.
- Full reasoning belongs in a `fusion_provenance` object owned by the track entity.
- `source_observation_ids` is a compact summary, not a complete long-term audit log.
- Assets that only need current track state can ignore `provenance_object_id`.
