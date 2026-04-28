---
# Trigger - when should this workflow run?
on:
  workflow_dispatch:  # Manual trigger

# Alternative triggers (uncomment to use):
# on:
#   issues:
#     types: [opened, reopened]
#   pull_request:
#     types: [opened, synchronize]
#   schedule: daily  # Fuzzy daily schedule (scattered execution time)
#   # schedule: weekly on monday  # Fuzzy weekly schedule

# Permissions - what can this workflow access?
# Write operations (creating issues, PRs, comments, etc.) are handled
# automatically by the safe-outputs job with its own scoped permissions.
permissions:
  contents: read
  issues: read
  pull-requests: read

# AI engine to use for this workflow
engine:
  id: copilot
  model: gpt-5.4

# Tools - GitHub API access via toolsets (context, repos, issues, pull_requests)
# tools:
#   github:
#     toolsets: [default]

# Network access
network: defaults

# Outputs - what APIs and tools can the AI use?
safe-outputs:
  create-issue:
    max: 1

---

# repo-analyzer

Review the repository structure, recent commits, and overall codebase health. Produce a concise analysis as a GitHub issue.

## Instructions

1. Examine the repository's directory structure, top-level config files, and README
2. Check recent commits for patterns (refactoring, bug fixes, feature work)
3. Note any obvious areas for improvement in code organization, documentation, or CI
4. Create a single GitHub issue summarizing your findings with:
   - **Overview**: What the repo does and its general state
   - **Highlights**: Things done well
   - **Suggestions**: 2-3 actionable improvements
   - Keep it concise — 300 words max
