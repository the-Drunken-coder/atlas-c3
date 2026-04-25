# Data Fusion

This document records the current Atlas Core planning boundary for future data fusion work.

## Current Direction

Data fusion is deferred.

Data fusion is part of Atlas Core planning, not a separate top-level module in these docs.

Atlas Core remains useful without data fusion. Observations can exist without immediately producing or updating tracks.

When data fusion exists, it should consume observations through Core contracts and write track entities through Core contracts.

Data fusion should not change the first Core storage or API design right now.

## Deferred Design Areas

The following are deferred:

- how observations become tracks
- which observation evidence fields fusion consumes
- track creation and update rules
- confidence and uncertainty handling
- association rules for matching observations to existing tracks
- whether fusion runs synchronously on writes or asynchronously
- how fusion publishes resulting entity changes
- debugging and explainability for fusion decisions
- first implementation scope
