# 0005 No Auth In Current Operating Model

## Context

ATLAS-C3 is currently planned around trusted local and bounded operational clients.

Clients include Atlas Command Interface, Atlas SDK consumers, simulations, debugging tools, and asset runtimes.

## Decision

Atlas Core should not implement authentication or authorization in the current operating model.

This is not a temporary v1-only placeholder. Future work should not assume auth will be added unless the operating model changes.

Atlas Core should not add fake auth, placeholder credentials, local shared secrets, stubbed security paths, role scaffolding, or per-record permission machinery.

## Consequences

- All Core endpoints are reachable by trusted clients on the configured network.
- Health and readiness endpoints do not need special public/private handling.
- There are no operator roles, asset credentials, resource permissions, or per-record authorization rules.
- Local Docker development does not need secret storage for Core API access.
- Atlas Command Interface is a client, not a security boundary.
- Logs should still avoid accidental leakage of sensitive metadata from external integrations.

## Rejected Alternatives

- API keys for local clients.
- Bearer tokens for assets.
- Operator roles.
- Per-resource authorization rules.
- Stubbed auth that looks real but does not enforce a real security model.
