# Agent Notes Purpose

The role of this file is to describe common mistakes and confusion points that agents might encounter while working in this project.
If you encounter anything in the project that surprises you, alert the developer you are working with and record that note in this agents.md file so future agents can avoid the same issue.

## Core Working Rules

- Keep solutions simple and fully functional; avoid mockups.
- Do not use database migrations.
- Avoid mock or stub data in dev/prod code paths (tests only).
- Keep code modular and avoid duplicated logic.
- Avoid introducing new patterns/technologies when an existing implementation can solve the issue.
- Keep files reasonably small; refactor when files grow too large.
- Optimize for ease, clarity, and speed from the start, establishing patterns that are easy to contribute to, clear in their function, and fast to change. Small changes should touch a few files; big changes should touch many files.
- Tolerate nothing when it comes to bad patterns or code. If a bad pattern appears, it will multiply—remove it aggressively, even if it's inconvenient.
- Embrace sledgehammering or aggressively deleting and rebuilding parts of the codebase. Throw away more code and be less attached to existing lines; aggressively delete code if there's an inkling it should be gone.

## Behavioral Guidelines

These guidelines are intended to reduce common LLM coding mistakes. Merge them with project-specific instructions as needed.
Tradeoff: These guidelines bias toward caution over speed. For trivial tasks, use judgment.

### 1. Think Before Coding

- State assumptions explicitly before implementing. If uncertain, ask.
- If multiple interpretations exist, present them instead of silently picking one.
- If a simpler approach exists, say so; push back when warranted. Avoid assumptions and confusion—surface tradeoffs and ask before proceeding if unclear.

### 2. Simplicity First

- Write the minimum code that solves the problem. Nothing speculative.
- Skip features beyond what was asked.
- Avoid abstractions for single-use code.
- Resist flexibility or configurability that was not requested.
- Avoid adding error-handling for well-established invariants, but prefer explicit assertions or lightweight logging/alerts for unexpected external I/O or validation failures that may become possible over time.
- If you write 200 lines and it could be 50, rewrite it.
- Ask: "Would a senior engineer say this is overcomplicated?" If yes, simplify.

### 3. Surgical Changes

- When this document explicitly says to "tolerate nothing" or otherwise mandates aggressive removal of bad patterns, that aggressive-cleanup guidance takes precedence over the Surgical Changes constraints; in all other cases follow the Surgical Changes rules.
- Touch only what you must. Clean up only your own mess.
- Do not improve adjacent code, comments, or formatting unless the request requires it.
- Do not refactor things that are not broken.
- Match existing style, even if you would do it differently.
- Remove imports, variables, and functions that your own changes make unused; do not delete pre-existing or unrelated dead code unless explicitly requested.
- Every changed line should trace directly to the user's request.

### 4. Goal-Driven Execution

- Define success criteria that can be verified, then loop until verified.
- Translate vague tasks into checks:
  - "Add validation" -> write tests for invalid inputs, then make them pass.
  - "Fix the bug" -> write a test that reproduces it, then make it pass.
  - "Refactor X" -> ensure tests pass before and after.
- For multi-step tasks, state a brief plan with a verification step for each item.
- Strong success criteria enable independent execution. Weak criteria require clarification.

These guidelines are working if they produce fewer unnecessary diff changes, fewer rewrites due to overcomplication, and clarifying questions before implementation instead of after mistakes.

## Project Notes

- The repository root is still the workspace/docs root; the executable Atlas Core implementation now lives under `/atlas-core`, matching the build plan's implementation root.
