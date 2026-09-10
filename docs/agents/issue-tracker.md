# Issue tracker: GitHub

Issues and specs live in GitHub Issues for `XTLS/libXray`.

Use the `gh` CLI. Because this clone's `origin` uses `yiguo.dev`,
include `--repo XTLS/libXray` in every `gh issue` and `gh pr` command.
For `gh api`, use explicit `repos/XTLS/libXray/...` endpoints.

## Conventions

- Use English for issue and PR titles, descriptions, and GitHub comments.
- Keep PR titles, descriptions, and comments self-contained. Do not mention
  or link to another repository's PR, including companion, dependency, or
  merge-order references. Describe required interface or build behavior directly.
- Use [triage-labels.md](triage-labels.md) for canonical triage roles.
- Use a heredoc for multiline bodies.

## Issue operations

- Publish a ticket: `gh issue create --repo XTLS/libXray --title "..." --body "..."`
- Fetch a ticket: `gh issue view <number> --repo XTLS/libXray --comments`
- List tickets: `gh issue list --repo XTLS/libXray --state open --json number,title,body,labels,comments`
- Comment: `gh issue comment <number> --repo XTLS/libXray --body "..."`
- Add or remove labels: `gh issue edit <number> --repo XTLS/libXray --add-label "..."` or `--remove-label "..."`
- Close: `gh issue close <number> --repo XTLS/libXray --comment "..."`

GitHub shares one number space across issues and PRs. Resolve an ambiguous
number with `gh pr view <number> --repo XTLS/libXray`, falling back to
`gh issue view <number> --repo XTLS/libXray`.

## Pull requests as a triage surface

**PRs as a request surface: no.**

## Wayfinding operations

Used by `/wayfinder`. The map is one issue with child issues as tickets.

- Map: label it `wayfinder:map`.
- Child ticket: link it as a GitHub sub-issue. If unavailable, add it to the
  map's task list and put `Part of #<map>` at the top of the child body.
- Child labels: `wayfinder:research`, `wayfinder:prototype`,
  `wayfinder:grilling`, or `wayfinder:task`.
- Blocking: use GitHub issue dependencies. If unavailable, put
  `Blocked by: #<n>` at the top of the child body.
- Frontier: choose the first open, unassigned child in map order without
  an open blocker.
- Claim: `gh issue edit <number> --repo XTLS/libXray --add-assignee @me`
- Resolve: comment with the answer, close the child, then add a context
  pointer to the map's Decisions-so-far.
