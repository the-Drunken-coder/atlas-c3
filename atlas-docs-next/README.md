# ATLAS-C3 Docs Next

This folder is the new planning structure for ATLAS-C3 documentation. It is meant to keep whole-system planning, shared contracts, subsystem-specific docs, and decision records separate from each other.

## Structure

- `system-docs/` - Cross-system docs for product goals, whole-system architecture, shared vocabulary, MVP workflows, and constraints that affect multiple modules.
- `contracts/` - Shared interfaces for API contracts, data shapes, SDK surfaces, asset-to-core protocols, stream formats, error envelopes, and any interface that more than one module depends on.
- `module-docs/` - Module-specific docs owned by one module, such as `atlas-core`, `atlas-sdk`, `atlas-command-interface`, or future asset/runtime modules.
- `decisions/` - Settled choices recorded as short decision notes that explain what was chosen, why it was chosen, and what alternatives were rejected.

## Placement Rules

- If changing a doc would affect the whole project, put it in `system-docs/`.
- If two or more modules must agree on the same shape, put it in `contracts/`.
- If the doc only explains how one module works internally, put it in `module-docs/`.
- If the doc records a choice that should not be re-litigated casually, put it in `decisions/`.

When in doubt, start with the highest shared level. A module doc can link to a contract, but it must not redefine the contract locally.

## Reference Rules

- Each rule, field definition, workflow, or decision should have one authoritative home.
- Prefer links over repeated explanations. If another doc already owns the detail, summarize only enough context to make the link useful.
- Do not copy request/response shapes, validation tables, enum values, or lifecycle rules into multiple docs.
- If a detail is needed in several places, promote it into `contracts/` or `system-docs/`, then reference it from module docs.
- If a referenced doc changes, update the source doc first and then check the linking docs for stale summaries.
- If duplication seems necessary for readability, keep the duplicate text clearly non-authoritative and point to the source of truth.

## Writing Conventions

- Write docs so they can guide implementation, not just describe intent.
- Prefer concrete rules, examples, and acceptance criteria over vague goals.
- Keep system docs stable and concise. Move implementation detail into module docs.
- Keep contracts precise. Name fields, types, validation rules, request/response shapes, and ownership boundaries.
- Keep module docs scoped to one module. Do not invent cross-system behavior there.
- Keep decisions short: context, decision, consequences, and rejected alternatives are usually enough.
- Mark uncertainty directly with `Open Question:` instead of hiding it in broad language.
- Avoid duplicating the same rule in multiple places. Put the authoritative version in one doc and link to it.
- Update docs when implementation changes the contract or invalidates an old assumption.
- Do not use mock or fake data as product truth. Examples should be clearly illustrative.
