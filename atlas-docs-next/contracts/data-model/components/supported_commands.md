# Supported Commands

Commands an asset can execute.

**Applies to:** assets only.

## Shape

```json
{
  "components": {
    "supported_commands": {
      "observed_at": "2026-01-01T00:00:00Z",
      "commands": ["move_to_location", "survey_grid"]
    }
  }
}
```

## Fields

| Field | Type | Required | Constraint | Description |
| --- | --- | --- | --- | --- |
| `observed_at` | RFC 3339 timestamp | yes | valid timestamp | When this command support list was reported |
| `commands` | array of strings | yes | each entry is a command type from the active command catalog | Command types this asset can execute |

## Usage Notes

- Asset entities must include this component to be valid task targets.
- An empty `commands` array is valid and means the asset currently cannot perform any commands.
- Atlas Core must reject task creation when the requested command type is not listed in the target asset's `commands`.
- Do not confuse this with the system command catalog. The command catalog defines what commands mean; this component defines what one asset can do.
