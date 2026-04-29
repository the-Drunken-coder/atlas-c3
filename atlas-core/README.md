# Atlas Core

Atlas Core is the Go implementation of the Atlas C3 core service.

## Run locally

```bash
cp .env.example .env
python3 tools/atlas-core-cli/main.py start --yes
```

From the host, the API is at `http://localhost:${ATLAS_CORE_HOST_PORT:-8080}` (see `.env.example`). Inside Docker Compose the `atlas-core` service always listens on **8080**; `ATLAS_CORE_HOST_PORT` only changes the published host port. PostgreSQL defaults to `localhost:${ATLAS_CORE_POSTGRES_PORT:-5432}`. For Compose, set **`ATLAS_CORE_DATABASE_URL`** (with a URL-encoded password if needed) and **`POSTGRES_PASSWORD`** in `.env`; both are required (see `docker-compose.yml`).

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

Enable the fusion profile by setting `ATLAS_DATA_FUSION_STACK=baseline` and starting with the CLI. The baseline worker polls the full-query endpoint on a timer and creates a simple track entity through the Core HTTP API.
