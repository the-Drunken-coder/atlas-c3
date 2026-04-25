# Auth And Security

This document defines the current ATLAS-C3 auth and security operating model.

## Operating Model

Atlas Core should not have authentication or authorization in the current operating model.

This is not a temporary shortcut. Future work should not assume auth will be added unless the operating model changes.

Trusted clients include:

- assets
- simulations
- debugging tools
- SDK consumers
- Atlas Command Interface

## Core API Access

All Core endpoints are reachable by trusted clients on the configured network.

Health and readiness endpoints do not need special public/private handling.

There are no:

- operator roles
- asset credentials
- resource permissions
- per-record authorization rules
- local shared secrets
- placeholder credentials
- fake auth paths

Atlas Core should not add stubbed security behavior that looks like auth but does not enforce a real operating model.

## Command Interface

Atlas Command Interface is a separate trusted client that talks to Atlas Core.

It is not a security boundary and should not be treated as a privileged control plane.

CORS should allow the local Command Interface development origin, `http://localhost:5173`, by default.

## Local Development

Local Docker development does not need secret storage for Core API access.

Logs should still avoid leaking accidental sensitive values if future integrations pass them through metadata, but there are no Atlas Core credentials to redact.

## Simulations And Debugging

Simulation and debugging tools use the same trusted-client model as first-party modules.

They should use real Core APIs and SDK helpers rather than special mock or fake product paths.
