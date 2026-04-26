# Atlas Core

Atlas Core is the Go implementation of the Atlas C3 core service.

## Run locally

```bash
cp .env.example .env
python3 tools/atlas-core-cli/main.py start --yes
```

Core defaults to `http://localhost:8080` and PostgreSQL defaults to `localhost:5432`.

## Test

```bash
go test ./...
```

Set `ATLAS_CORE_TEST_DATABASE_URL` to enable PostgreSQL integration tests.

## Lifecycle CLI

```bash
python3 tools/atlas-core-cli/main.py
```

Actions:
- `start`
- `restart`
- `shutdown`

`restart` and `shutdown` remove only Docker resources labeled for the `atlas-core` compose project.

## Optional data fusion harness

Enable the fusion profile by setting `ATLAS_DATA_FUSION_STACK=baseline` and starting with the CLI. The baseline worker reads the full query, listens to the live stream, and creates a simple track entity through the Core HTTP API.
