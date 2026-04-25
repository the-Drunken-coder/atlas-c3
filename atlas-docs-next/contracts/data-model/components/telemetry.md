# Telemetry

Position and motion state for an asset or track.

**Applies to:** assets and tracks. Geofeatures use [`geometry`](./geometry.md).

## Shape

```json
{
  "components": {
    "telemetry": {
      "observed_at": "2026-01-01T00:00:00Z",
      "latitude": 40.7128,
      "longitude": -74.006,
      "altitude_m": 120,
      "speed_m_s": 8.2,
      "heading_deg": 165
    }
  }
}
```

## Fields

| Field | Type | Required | Constraint | Description |
| --- | --- | --- | --- | --- |
| `observed_at` | RFC 3339 timestamp | yes | valid timestamp | When this position or motion state was observed |
| `latitude` | number | yes | `-90 <= x <= 90` | Latitude in decimal degrees using WGS84 |
| `longitude` | number | yes | `-180 <= x <= 180` | Longitude in decimal degrees using WGS84 |
| `altitude_m` | number | no | finite | Altitude above sea level in meters |
| `speed_m_s` | number | no | `x >= 0` | Speed over ground in meters per second |
| `heading_deg` | number | no | `0 <= x < 360` | Heading in degrees clockwise from true north |

## Usage Notes

- Assets usually report their own telemetry.
- Tracks receive telemetry from data fusion or other trusted track-producing systems.
- Omit unknown optional values rather than sending `null`, `NaN`, or placeholder numbers.
