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

## Fields

GeoJSON-style fields:

| Field | Type | Required | Constraint | Description |
| --- | --- | --- | --- | --- |
| `type` | string | yes | `Point` / `LineString` / `Polygon` | GeoJSON geometry type |
| `coordinates` | array | yes | valid GeoJSON coordinate nesting, max 10000 points | Coordinates in `[longitude, latitude]` order |

Atlas fields:

| Field | Type | Required | Constraint | Description |
| --- | --- | --- | --- | --- |
| `point_lat` | number | for points/circles | `-90 <= x <= 90` | Latitude for a point or circle |
| `point_lng` | number | for points/circles | `-180 <= x <= 180` | Longitude for a point or circle |
| `radius_m` | number | for circles | `x > 0` | Circle radius in meters |
| `polygon` | array | for polygons | at least 3 valid `[lat, lng]` pairs | Polygon vertices |
| `line` | array | for lines | at least 2 valid `[lat, lng]` pairs | Line vertices |

## Usage Notes

- The entity `subtype` is advisory. The geometry component is authoritative for spatial shape.
- GeoJSON uses `[longitude, latitude]`; Atlas arrays use explicit fields or `[lat, lng]` pairs.
- Overlay payloads such as heatmaps or imagery belong in objects, not inside this component.
