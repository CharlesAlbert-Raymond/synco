# Split Session Identity From Tmux Implementation

Status: ready-for-agent
Type: Architecture
Priority: High

## Problem

Pure session identity rules live in `internal/tmux`, so non-tmux modules import `tmux` for concepts that are not tmux command execution.

Examples:

- `tmux.RootSessionKey`
- `tmux.ProjectName`
- `tmux.ResolveProjectName`
- `tmux.SessionNameFor`
- `tmux.IsProjectSession`
- `tmux.MainWorktreeRoot`

This makes the `tmux` module shallow: its interface includes both identity rules and local process execution. Deleting tmux command execution would also delete naming rules that should remain usable and testable in-process.

## Files

- `internal/tmux/tmux.go`
- `internal/tmux/tmux_test.go`
- `internal/state/state.go`
- `internal/orchestrate/orchestrate.go`
- `internal/tui/confirm.go`
- `internal/mcp/tools.go`
- `main.go`

## Proposed Solution

Create a small deep module for session identity, for example `internal/session` or `internal/identity`.

Move these rules behind that module's interface:

- stable root session key
- project name derivation and sanitization
- config project name override resolution
- session name construction
- project-session matching
- main worktree root resolution, if keeping git root normalization with identity is still clearer than leaving it in `tmux`

Then update callers to import the new module for identity concepts and keep `internal/tmux` focused on tmux behavior.

## Implementation Notes

- Prefer the smallest module name that reads naturally at call sites. `session.SessionNameFor(...)` and `session.RootKey` are likely clearer than `tmux.SessionNameFor(...)` for pure identity.
- If `MainWorktreeRoot` stays in the new module, document that it is identity-adjacent but uses local git as an implementation detail.
- Do not introduce an interface type unless there are two adapters. This ticket is mostly in-process refactoring.
- Keep public behavior unchanged: existing session names must not change.

## Acceptance Criteria

- Pure identity functions/constants no longer live in `internal/tmux`.
- Non-tmux modules do not import `internal/tmux` solely for naming or root-key constants.
- Existing tests for project/session naming are moved or duplicated into the new module.
- Existing tmux tests still cover tmux-specific behavior.
- `go test ./...` passes.

## Suggested Tests

- `SessionNameFor(project, RootKey)` returns `project`.
- `SessionNameFor(project, "feature/foo")` returns `project/feature-foo`.
- `ResolveProjectName(repoRoot, configLabel)` prefers sanitized config labels.
- `IsProjectSession` matches root and branch sessions but rejects sibling prefixes.

## Comments

This is the best first ticket because it reduces coupling before larger lifecycle work.
