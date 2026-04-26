# 0001: Make Observations First-Class Records

## Status

Accepted

## Context

Earlier ATLAS-C3 planning treated observations as a special convention on top of objects. Under that model, an observation was an object with `type = observation`, observation state lived in the object metadata, and media lived as files on that same object.

That kept the early data model small, but it blurred two different concepts:

- an observation is sensor evidence about something observed over time
- an object is a container for stored files and payload metadata

Data fusion, asset reporting, operator inspection, and observation lifecycle rules are important enough that observations should not be hidden inside a generic object convention.

## Decision

ATLAS-C3 will treat observations as first-class operational records with their own persistence model and contracts.

Objects remain the mechanism for storing files and payload containers. Observation media or other large files should be stored through objects and linked back to the owning observation.

## Consequences

- Observation contracts should be documented separately from object contracts.
- Atlas Core should eventually own an observations table or equivalent first-class persistence surface.
- The Core API should expose observation behavior directly instead of relying only on generic object endpoints.
- Data fusion should consume observation records as evidence and use linked objects only when it needs associated files.
- Object docs should not define observation lifecycle, observation identity, or observation-specific validation rules.

## Rejected Direction

Do not model observations only as `objects` with `type = observation`. That makes querying, lifecycle rules, validation, and fusion semantics depend on object conventions that are too generic for observation behavior.

