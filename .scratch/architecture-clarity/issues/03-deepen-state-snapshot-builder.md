# Deepen State Into Snapshot Builder

Status: ready-for-agent
Type: Architecture
Priority: High

## Problem

`state.GatherWithOpts` joins worktrees, tmux sessions, metadata, current session, and port discovery in one path. Port discovery uses `tmux`, `ps`, and `lsof`, but every state gather pays for it and accepts its silent failure modes.

This makes the state module less deep than it could be. The core snapshot behavior is an in-process join, but it is coupled to local process adapters.

## Files

- `internal/state/state.go`
- `internal/state/state_test.go`
- `internal/tmux/ports.go`
- `internal/worktree/worktree.go`
- `internal/metadata/metadata.go`
- Callers in `internal/tui/list.go`, `internal/mcp/tools.go`, `internal/restore/restore.go`

## Proposed Solution

Deepen `internal/state` around a snapshot-building interface.

The deep module should own:

- joining worktrees to expected session names
- marking whether sessions exist
- annotating titles from metadata
- marking current session
- collecting other sessions with ports, if port enrichment is enabled

Separate the core join from local discovery adapters:

- worktree list adapter
- session list/current-session adapter
- metadata adapter
- optional port provider adapter

The external interface can stay close to today's `Gather` / `GatherWithOpts` while the implementation gains internal seams for testing.

## Implementation Notes

- Do not over-design exported interfaces. If only tests need fake adapters, keep seam types private or use an options struct with function fields.
- Preserve the current `Gather` and `GatherWithOpts` call sites if possible.
- Consider adding an option to disable port enrichment internally for tests and possibly future fast refresh paths.
- Preserve current behavior where port discovery failures do not fail the whole gather.
- Use the new session identity module from issue `01` if it exists.

## Acceptance Criteria

- The core snapshot join can be tested without real git, tmux, metadata files, `ps`, or `lsof`.
- `Gather` and `GatherWithOpts` continue to return the same shape of `GatherResult`.
- Port enrichment is isolated so failures remain non-fatal and do not obscure core worktree/session state.
- Multi-repo gather behavior remains unchanged.
- `go test ./...` passes.

## Suggested Tests

- Main worktree uses the stable root session name regardless of branch.
- Linked worktrees use sanitized branch session names.
- Missing sessions produce entries with `HasSession=false`.
- Metadata titles are applied by branch.
- Current session is marked exactly when session names match.
- Other session ports exclude synco-managed sessions.
- Port provider failure or nil result still returns core entries.

## Comments

This ticket improves read-path locality and makes future features such as git status indicators easier to add without bloating `state.GatherWithOpts`.
