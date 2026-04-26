# Sensor Refs

Sensors associated with an asset or track.

**Applies to:** assets and tracks.

## Shape

```json
{
  "components": {
    "sensor_refs": {
      "observed_at": "2026-01-01T00:00:00Z",
      "sensors": [
        {
          "sensor_id": "camera-1",
          "type": "camera",
          "horizontal_fov_deg": 90,
          "vertical_fov_deg": 60,
          "yaw_deg": 45,
          "pitch_deg": 10
        }
      ]
    }
  }
}
```

## Fields

| Field | Type | Required | Constraint | Description |
| --- | --- | --- | --- | --- |
| `observed_at` | RFC 3339 timestamp | yes | valid timestamp | When this sensor state or context was observed |
| `sensors` | array | yes | entries are sensor objects | Sensors carried by the asset or associated with the track context |

Sensor entry fields:

| Field | Type | Required | Constraint | Description |
| --- | --- | --- | --- | --- |
| `sensor_id` | string | yes | non-empty | Stable sensor identifier |
| `type` | string | yes | non-empty | Sensor kind such as `camera`, `radar`, `lidar`, or `rf` |
| `horizontal_fov_deg` | number | no | `0 < x <= 180` | Horizontal field of view in degrees |
| `vertical_fov_deg` | number | no | `0 < x <= 180` | Vertical field of view in degrees |
| `yaw_deg` | number | no | `-180 <= x <= 180` | Yaw relative to the platform in degrees |
| `pitch_deg` | number | no | `-90 <= x <= 90` | Pitch relative to the platform in degrees |
| `roll_deg` | number | no | `-180 <= x <= 180` | Roll relative to the platform in degrees |

## Usage Notes

- On an asset, this component describes onboard sensors.
- On a track, this component may capture sensor context associated with the current track state.
- FOV and orientation values must stay within the bounds above so clients can render and validate sensor context consistently.
- If asset sensor inventory and track observation context diverge too much later, they should split into separate components through an explicit contract update.
