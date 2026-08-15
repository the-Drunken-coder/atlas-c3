# Atlas Core — Implementation vs Documentation Audit

Generated from a 16-agent cross-validation comparing `copilot/atlas-core-follow-up` implementation against all contracts and build plan docs. This is a point-in-time audit snapshot, not a live source of truth.

## Methodology

For each topic area, two subagents independently compared the work from opposite directions:
- **Impl→Docs**: read the implementation first, then checked against documentation
- **Docs→Impl**: read the documentation first, then checked against implementation

This cross-validation catches gaps that a single-direction comparison might miss.

Reproducibility / counting notes:
- Re-verify every cited line against current `HEAD` before making changes; some findings may already be fixed or may have moved.
- Counts below are issue entries, not deduplicated root causes. A single defect may appear in multiple handlers, files, or contract sections.
- Severity is triage guidance, not proof of exploitability. Prioritize items that are still reproducible on the current branch.

---

## HIGH SEVERITY (Contract violations / missing required behavior)

### API / HTTP Handlers

| # | Gap | Source |
|---|-----|--------|
| H1 | `GET /` returns `"status": "ok"` but docs require `"status": "ready"` | `system.md:30` vs `app.go:125` |
| H2 | PATCH immutable fields get `unknown_field` code instead of `immutable_field` | `conventions.md:96` vs router handlers |
| H3 | Invalid `content_type` form field returns 400 instead of 415 `unsupported_media_type` | `objects.md:153` vs `router.go:740-751` |
| H4 | `GET /readiness` 503 returns plain `ReadinessResponse` JSON, not `ErrorEnvelope` | `system.md:78` vs `router.go:162-165` |

### Service Layer

| # | Gap | Source |
|---|-----|--------|
| H5 | Task `json.description` and `json.created_by` not immutable on pending PATCH — `mergeTaskJSONForPatch` performs full key replacement | `tasks.md:119-120` vs `task_json.go:13-18` |
| H6 | Task `json.extra.*` not mutable in `acknowledged` (non-terminal) state — `mergeTaskComponentsOnlyProgressResultError` rejects anything outside `progress/result/error` | `tasks.md:106-111` vs `task_json.go:36-68` |
| H7 | File upload/append/delete don't persist parent object `updated_at` — object fetched for event but never written back | `objects.md:165-170,201` vs `service.go:403-447` |
| H8 | No `EnsureNoPromotedFieldDuplication` for Observation and Task creates — only Entity and Object have this guard | observation-service.md, task-service.md |
| H9 | Caller can supply `command_catalog_object_id` on task create — value silently stored in JSON blob, should be rejected | `task-service.md:16` vs `service.go` |
| H10 | `json.latest_sighting` and `json.sightings_object_id` not validated atomically — they're checked independently, either can exist without the other | `observation-service.md:13` vs `service.go:639-656` |
| H11 | `TASK_CATALOG_OBJECT_UNRESOLVABLE` error code never returned — all catalog resolution failures return `catalog_unavailable` | `task-service.md:35` vs `service.go:482-538` |
| H12 | Observation delete doesn't cascade-delete owned objects at service layer — relies entirely on store-level cascading | `observations.md:114` vs `service.go:205-211` |

### SSE / Events

| # | Gap | Source |
|---|-----|--------|
| H13 | `data.resource` wrapper missing on create/update events — fields placed directly in `data` instead of nested under `resource` | `stream.md:89-96` vs `service.go` publish calls |
| H14 | `affected_files` capped at 8 instead of omitted when >8 files — docs say omit, code slices `files[:8]` | `stream.md:145` vs `service.go:475-477` |
| H15 | Slow subscriber eviction not logged — docs require logging with context to identify the missed event | `event-publisher.md:30-31` vs `publisher.go:52-55` |

### Storage Schema

| # | Gap | Source |
|---|-----|--------|
| H16 | `object_files` primary key is composite `(object_id, file_id)` — docs specify `file_id` as solo PK | `storage-schema.md` vs `schema.go:68-78` |
| H17 | `object_files_updated_at_idx` has extra `object_id ASC` column not in documented index | `storage-schema.md` vs `schema.go:80` |

### Stores / Persistence

| # | Gap | Source |
|---|-----|--------|
| H18 | Entity delete doesn't reject when polymorphic `owner_type=entity` objects exist — no DB-level FK, store-level check not called | ADR 0007:39-40 vs `store.go:90-99` |
| H19 | Task delete doesn't reject when polymorphic `owner_type=task` objects exist — same gap as above | ADR 0007:55-56 vs `store.go:248-257` |
| H20 | ADR 0006: raw PG errors returned instead of `StorageUnavailable` on upload commit failure and append DB-update failure | ADR 0006 vs `store.go:462,514,520` |

### Catalogs

| # | Gap | Source |
|---|-----|--------|
| H21 | Sighting catalog failure doesn't fail readiness — `app.go` only logs a warning, `Readiness()` has no sighting catalog gate | `sighting-catalog.md:236` vs `app.go:81-88,132-154` |
| H22 | No startup audit log for command catalog reuse vs. creation — `Materialize()` has zero logging | `core-startup-command-catalog.md:25` vs `catalog.go:211-243` |
| H23 | Empty `commands` array passes command catalog validation — `len(catalog.Commands) == 0` never checked | `overview.md:58` vs `catalog.go:139` |

### Observability / Logging

| # | Gap | Source |
|---|-----|--------|
| H24 | `event` field absent from base logger — observability.md requires it on every log line, only HTTP middleware adds it manually | `observability.md` vs `logging.go` |
| H25 | No `correlation_id` infrastructure anywhere — no field, helper, or context plumbing | `observability.md` |
| H26 | Event publication failures silently discarded — `_ = s.publisher.Publish()` never logs failures | `operational-assumptions.md` vs `service.go:469` |

### Data Fusion

| # | Gap | Source |
|---|-----|--------|
| H27 | Stack loading mechanism not implemented — baseline behavior hardcoded in `runner/main.py`, `stacks/baseline/` is a README-only placeholder | `data-fusion.md` vs `harness/runner/main.py` |
| H28 | Harness violates entire observability contract — no `timestamp`, `run_id`, `component`, or `level` in log lines | `observability.md` vs `harness/runner/main.py:113,115` |

---

## MEDIUM SEVERITY

### API / HTTP Handlers

| # | Gap |
|---|-----|
| M1 | ID length validation (50 chars) delegated to service, no model-layer constant |

### Services

| # | Gap |
|---|-----|
| M2 | File parent ownership not re-validated on append — `AppendObjectFile` delegates directly to store |
| M3 | `sightings_object_id` validated independently of `latest_sighting` — stricter than docs but not wrong |

### SSE / Events

| # | Gap |
|---|-----|
| M4 | `EventEnvelope` has typo: field name `Mutaion` (missing 't'); JSON tag is `mutation` |
| M5 | `publishObject` includes all object files for every `object.updated` event, not just affected ones |

### Data Model

| # | Gap |
|---|-----|
| M6 | No entity type constants (`asset`/`track`/`geofeature`) — `Type` is raw `string` |
| M7 | No observation state constants (`active`/`inactive`/`ended`) |
| M8 | No task status constants or transition-rules map (`pending`/`acknowledged`/`completed`/`failed`) |
| M9 | No object `owner_type` constants (`entity`/`observation`/`task`/`system`) |
| M10 | No component name validation (known components vs `custom_*` prefix) |
| M11 | No canonical object type constants (`observation_sighting_history`, `observation_media`, `fusion_provenance`, `command_catalog`) |
| M12 | No pagination response metadata struct — `Pagination` is input-only, no `TotalCount`/`ReturnedCount` |
| M13 | No pagination response header constants (`X-Total-Count`, `X-Limit`, `X-Offset`, `X-Returned-Count`) |
| M14 | No unknown query parameter rejection helper per conventions |
| M15 | `TopLevelUnknownFields` returns `[]string` instead of `[]FieldError{Code: "unknown_field"}` |
| M16 | `invalid_type` and `unknown_field` field error codes documented but never emitted as `FieldError` |

### Stores

| # | Gap |
|---|-----|
| M17 | No explicit transaction API exposed to callers — `regular-record-store.md` requires it |
| M18 | `StorageStatus` doesn't report PostgreSQL readiness, only filesystem |
| M19 | No database readiness check on regular record store |

### Catalogs

| # | Gap |
|---|-----|
| M20 | Sighting catalog `catalog_id` never validated (command catalog validates it, sighting doesn't) |
| M21 | Sighting `data` field is required per docs but has no explicit presence check |
| M22 | Empty `sighting_kinds` array passes sighting catalog validation |

### Runtime / Containers

| # | Gap |
|---|-----|
| M23 | `docker-compose.fragment.yml` documented but file doesn't exist |
| M24 | `data-fusion/harness/config/` directory documented but doesn't exist |
| M25 | `data-fusion/tests/fixtures/` and `tests/scenarios/` documented but don't exist |
| M26 | CLI label mismatch: checks `atlas.core.project` vs docs require `com.docker.compose.project` or `atlas.project` |
| M27 | CLI image removal doesn't check external container references before force-removing |
| M28 | Container healthcheck uses `/readiness` instead of `/health` — conflates health vs readiness |
| M29 | PostgreSQL port exposed to host (docs say private Compose network only) |
| M30 | `atlas-data-fusion` service has no healthcheck |
| M31 | No explicit named private Docker network |

### Observability / Logging

| # | Gap |
|---|-----|
| M32 | `request_id` not returned in HTTP responses — only `error_id` is, operators can't correlate client failures |
| M33 | Write operations have zero lifecycle logging — entities, tasks, observations, objects, files |
| M34 | Shutdown not logged |
| M35 | Configuration summary not logged at startup |
| M36 | Dependency/readiness failures not logged server-side |
| M37 | Missing component categories in logs: storage, tasks, observations, stream, data fusion |
| M38 | No duration logging for PG operations, file I/O, stream publication, or fusion processing |
| M39 | Resource context fields (`entity_id`, `task_id`, etc.) not logged |

### Data Fusion

| # | Gap |
|---|-----|
| M40 | Harness has no SSE stream listener — only polls `/queries/full` |
| M41 | `fusion_provenance` objects never created |
| M42 | `stacks/baseline/` is README-only — no actual algorithm code in any stack |
| M43 | Harness doesn't share `run_id` with Atlas Core |
| M44 | No common test harness infrastructure |

### Configuration

| # | Gap |
|---|-----|
| M45 | `ATLAS_CORE_MAX_UPLOAD_BYTES` (16 MB default) undocumented |
| M46 | `ATLAS_CORE_VERSION` (default `"dev"`) undocumented |
| M47 | `ATLAS_CORE_ROOT_DIR` undocumented |
| M48 | `ATLAS_CORE_BASE_URL` missing from `.env.example` |

---

## LOW SEVERITY

| # | Gap |
|---|-----|
| L1 | Duplicated `observed_at` validation in `validateSupportedCommands` |
| L2 | `sightings_object_id` validated independently of `latest_sighting` (stricter, harmless) |
| L3 | Dead `ObjectStore` dependency in router — `StorageStatus` declared, never called |
| L4 | CLI `--yes`/`--confirm` flag lacks `help=` text |
| L5 | CLI no step-by-step progress output during Docker operations |
| L6 | CLI errors bubble as raw Python tracebacks, not user-friendly messages |
| L7 | CLI selected menu action not echoed back to user |
| L8 | CLI shutdown doesn't report what was removed |
| L9 | CLI restart uses `docker rm -f` skipping explicit stop step |
| L10 | Documentation: `object_id` example in `overview.md:123` is misleading (version-based vs hash-based) |

---

## Totals

These totals are raw audit-entry counts from the snapshot above. They are useful for rough scoping, but not for measuring remaining work after fixes unless the list is first re-triaged against the current branch and deduplicated.

| Severity | Count |
|----------|-------|
| High | 28 |
| Medium | 48 |
| Low | 10 |
| **Total** | **86** |

## Highest-Concentration Problem Areas

Suggested triage order:
1. Confirm which high-severity items still reproduce on the current branch.
2. Group fixes by shared root cause (for example: HTTP field parsing, store race/error mapping, catalog/schema validation).
3. Recount after each triage pass instead of treating the snapshot totals as a live backlog.

1. **Service Layer** — Task immutability, PATCH semantics, file→object `updated_at` cascading, validation gaps
2. **Observability** — Entire contract barely followed; missing `event`, `correlation_id`, lifecycle logging, duration fields
3. **SSE Events** — Missing `resource` wrapper, wrong `affected_files` semantics, no eviction logging
4. **Stores** — Polymorphic referential integrity gaps, ADR 0006 error type mismatches
5. **Data Fusion** — Stack loading not implemented, observability contract entirely violated
6. **Data Model** — No validation constants for any documented allowed values
