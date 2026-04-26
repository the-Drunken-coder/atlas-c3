# Atlas Core Runtime Shape

This document describes how Atlas Core should be operated locally at a high level. Container details live in [`container-system.md`](./container-system.md), and the interactive CLI decision is recorded in [`../../decisions/0004-interactive-core-cli.md`](../../decisions/0004-interactive-core-cli.md).

## Goal

Atlas Core should be easy to start, restart, and shut down without requiring the operator or developer to remember Docker Compose commands.

The normal local entry point should be a Python CLI script with an interactive terminal UI.

## Local CLI

The CLI should show a small menu of lifecycle actions:

- start
- restart
- shutdown

The CLI should not require command-line arguments for normal use.

## Start

Start should bring up the Atlas Core Docker Compose project.

Expected behavior:

- start the PostgreSQL container
- start the Atlas Core container
- mount the object file volume
- wait for Atlas Core readiness
- report a clear success or failure result

Start should leave existing project volumes intact unless the user has explicitly chosen a destructive action.

## Restart

Restart should perform the same destructive cleanup as shutdown, then start the Atlas Core Docker Compose project again.

Expected behavior:

- stop Atlas Core project containers
- remove Atlas Core project containers
- remove Atlas Core project images
- remove Atlas Core project volumes
- start the PostgreSQL container
- start the Atlas Core container
- mount a fresh object file volume
- wait for readiness
- report a clear success or failure result

Restart should clear stored Core database state and object file bytes before starting again.

## Shutdown

Shutdown is destructive for the Atlas Core Docker Compose project.

Expected behavior:

- stop Atlas Core project containers
- remove Atlas Core project containers
- remove Atlas Core project images
- remove Atlas Core project volumes
- report what was removed

Shutdown should clear stored Core database state and object file bytes by removing the project volumes.

The destructive scope is Atlas Core project resources only. The CLI should not delete unrelated Docker containers, images, or volumes from the developer's machine.

## Terminal UI Expectations

The terminal UI should be simple:

- show the available actions as a selectable list
- make the selected action obvious
- require typed confirmation before destructive restart or shutdown, such as `Type YES to continue`
- support an explicit non-interactive confirmation flag such as `--yes` or `--confirm` for automation
- print progress for each Docker step
- show clear errors when Docker is unavailable or a step fails

This does not need to be a complex application. The goal is a reliable local control surface.

## Runtime Startup Sequence

After containers start, Atlas Core itself should:

1. Load configuration.
2. Connect to PostgreSQL.
3. Ensure required database structures exist.
4. Verify object file storage is available.
5. Materialize startup records such as the command catalog.
6. Load and validate startup catalogs that are not materialized as objects, such as the sighting catalog.
7. Start the HTTP server.
8. Report readiness only after required dependencies and startup catalogs are usable.

## Non-Goals

The local CLI should not become:

- a general Docker cleanup utility
- a migration runner
- a data backup tool
- a production deployment system
