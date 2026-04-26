# Service Testing

This document defines how Atlas Core services should be tested.

It is about test-only support for service behavior. It must not introduce fake data paths, mock product behavior, or stubbed runtime behavior into development or production code.

## Testing Scope

Service tests should prove Core behavior between HTTP handlers and stores.

They should cover:

- validation rules
- lifecycle transitions
- cross-store coordination
- command catalog validation usage
- event publication calls
- error behavior
- transaction behavior at the service boundary
- behavior when a store or object file operation fails

HTTP request decoding belongs in handler tests. SQL behavior belongs in PostgreSQL store tests. Filesystem path safety belongs in object file store tests.

## Test-Only Dependencies

Service tests may use test-only implementations of service dependencies:

- regular record store fake
- object store fake
- event publisher recorder
- active command catalog fixture
- transaction recorder

These helpers should live in test files or a test-only helper package such as `internal/service/servicetest`.

They should not be imported by production packages.

## Store Fakes

Store fakes should model only the behavior needed by the service under test.

They may record calls, return configured records, and return configured errors. They should not become a second implementation of PostgreSQL or filesystem behavior.

Use real PostgreSQL integration tests for SQL constraints, query behavior, and transaction behavior that cannot be represented honestly with a small fake.

## Event Publisher Recorder

Service tests should verify that services publish mutation events after successful writes.

The test event publisher should record attempted events and allow configured publish failures.

Commit-first, best-effort behavior should be tested:

- if the write succeeds and publish succeeds, the service succeeds
- if the write succeeds and publish fails, the service still succeeds and exposes no partial failure to the caller
- publish failures should be observable to logging behavior where practical

## Command Catalog Fixtures

Task service tests should use small command catalog fixtures that represent real catalog rules.

Fixtures should cover:

- valid command type
- invalid command type
- valid parameters
- invalid parameters
- asset supports command
- asset does not support command
- task pins the active command catalog object ID

Do not use command names or parameter shapes as product truth unless they are also in the command catalog contract.

## Error Assertions

Service tests should assert Core errors by stable fields, not by free-form message text when possible.

Use the Core API error envelope fields from [`../../../contracts/core-api/conventions.md`](../../../contracts/core-api/conventions.md):

- `message`
- `error_code`
- `error_id`
- `timestamp`
- `path`
- `details`

Service-layer errors may not know HTTP path or status yet, but the error type should preserve enough detail for handlers to serialize the API envelope consistently.

## Fixture Builders

Fixture builders are allowed in tests to reduce repetition.

They should be boring and explicit:

- `NewAssetEntityFixture`
- `NewObservationFixture`
- `NewTaskFixture`
- `NewObjectFixture`
- `NewCommandCatalogFixture`

Avoid large hidden defaults that make tests hard to understand.

## What Not To Test In Service Tests

Service tests should not:

- verify exact SQL strings
- test Docker Compose behavior
- test Python CLI behavior
- test SSE socket mechanics
- test SDK behavior
- create fake dev/prod data paths
- preserve compatibility with unimplemented historical schemas

Those belong in their own package tests, integration tests, or are outside the current operating model.
