# 0004: Use An Interactive Core Lifecycle CLI

## Status

Accepted

## Context

Atlas Core should be easy to run without remembering Docker Compose commands or command-line flags. The system is local-first, short-lived, and allowed to use destructive reset behavior.

## Decision

Atlas Core will provide a Python CLI script with an interactive terminal UI for local lifecycle management.

The CLI should present a small menu of actions:

- start
- restart
- shutdown

The user should be able to choose from the menu instead of supplying command-line arguments.

Restart and shutdown are destructive for the Atlas Core Docker Compose project.

Restart should stop the Atlas Core containers, remove the project-owned containers, images, and volumes, then start the project again from a clean state.

Shutdown should stop the Atlas Core containers and remove the project-owned containers, images, and volumes.

This destructive cleanup is scoped to Atlas Core project resources, not unrelated Docker resources on the developer's machine.

## Consequences

- Local operation is menu-driven and does not require memorized command flags.
- Restart and shutdown clear stored Core data and object file bytes by removing project volumes.
- The next start may need to rebuild or pull images again.
- The CLI must clearly label restart and shutdown as destructive before running them.
- Runtime docs and implementation should avoid adding migration or rollback paths to compensate for shutdown cleanup.

## Rejected Direction

Do not make developers manage normal Atlas Core lifecycle through raw Docker commands as the primary workflow.

