# Geometry

Spatial shape for a geofeature.

**Applies to:** geofeatures only.

## Shape

GeoJSON-style shape:

```json
{
  "components": {
    "geometry": {
      "type": "Point",
      "coordinates": [-74.006, 40.7128]
    }
  }
}
```

Atlas point/circle shape:

```json
{
  "components": {
    "geometry": {
      "point_lat": 40.7128,
      "point_lng": -74.006,
      "radius_m": 500
    }
  }
}
```

## Schema Variants

The `geometry` component supports two representations:

- GeoJSON-style: `type` plus `coordinates`
- Atlas-native: `point_lat`, `point_lng`, optional `radius_m`, `line`, or `polygon`

Writers MUST use exactly one representation in a single `geometry` component. New Atlas-authored data SHOULD prefer the Atlas-native representation because its fields are explicit and easy to validate. Implementations MUST read both supported representations, but create and patch validation MUST reject a component that mixes `type`/`coordinates` with Atlas-native fields such as `point_lat`, `point_lng`, `radius_m`, `line`, or `polygon`.

There is no normal precedence rule because mixed representations are invalid. If an implementation encounters legacy stored data containing both representations, it should treat the Atlas-native fields as authoritative for display, log the inconsistency, and require cleanup on the next write.

## Fields

GeoJSON-style fields:

| Field | Type | Required | Constraint | Description |
| --- | --- | --- | --- | --- |
| `type` | string | yes | `Point` / `LineString` / `Polygon` | GeoJSON geometry type; `MultiPoint`, `MultiLineString`, `MultiPolygon`, and `GeometryCollection` are not supported |
| `coordinates` | array | yes | valid GeoJSON coordinate nesting, max 10,000 coordinate positions total | Coordinates in `[longitude, latitude]` order. The limit counts one `Point` as one position, all positions in a `LineString`, and all positions across all rings of a `Polygon`. |

Atlas fields:

| Field | Type | Required | Constraint | Description |
| --- | --- | --- | --- | --- |
| `point_lat` | number | for points/circles | `-90 <= x <= 90` | Latitude for a point or circle |
| `point_lng` | number | for points/circles | `-180 <= x <= 180` | Longitude for a point or circle |
| `radius_m` | number | for circles | `x > 0` | Circle radius in meters |
| `polygon` | array | for polygons | at least 3 valid `[lat, lng]` pairs; do not repeat the first vertex as a closing coordinate | Polygon vertices. Atlas-native polygons are auto-closed for rendering and geometry operations by repeating the first vertex internally. Example: `[[40.0, -74.0], [40.1, -74.0], [40.1, -73.9]]`. |
| `line` | array | for lines | at least 2 valid `[lat, lng]` pairs | Line vertices |

## Usage Notes

- The entity `subtype` is advisory. The geometry component is authoritative for spatial shape.
- GeoJSON uses `[longitude, latitude]`; Atlas arrays use explicit fields or `[lat, lng]` pairs.
- `Point` maps to Atlas `point_lat` and `point_lng`, with `radius_m` for circles; `LineString` maps to the `line` array; `Polygon` maps to the `polygon` array after removing the GeoJSON closing coordinate from each ring.
- Multi-part geometries should be represented as separate geofeature entities rather than one `geometry` component.
- Overlay payloads such as heatmaps or imagery must be stored as separate object records in the Objects component/object schema and created or retrieved through the `/objects` API. See [`../objects.md`](../objects.md) and [`../../core-api/objects.md`](../../core-api/objects.md) for implementation details.
