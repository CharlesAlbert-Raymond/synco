# Deepen Worktree And Session Lifecycle

Status: ready-for-agent
Type: Architecture
Priority: High

## Problem

Worktree/session lifecycle rules are spread across adapters and partial orchestration modules.

Examples:

- `internal/orchestrate/orchestrate.go` owns create/delete flows, but not all lifecycle policy.
- `internal/tui/list.go` owns ensuring sessions and switching/attaching behavior.
- `internal/tui/confirm.go` owns deleting-self handling before delete.
- `internal/mcp/tools.go` duplicates create/delete/switch/title cleanup behavior.
- `internal/restore/restore.go` owns restoring missing sessions.

Deleting Bubble Tea or MCP should not delete lifecycle behavior. Today, some behavior would disappear with those adapters.

## Files

- `internal/orchestrate/orchestrate.go`
- `internal/restore/restore.go`
- `internal/tui/list.go`
- `internal/tui/confirm.go`
- `internal/mcp/tools.go`
- `internal/metadata/metadata.go`
- `internal/tmux/*.go`
- `internal/worktree/worktree.go`

## Proposed Solution

Deepen `internal/orchestrate` or create a sibling lifecycle module that owns worktree/session command policy.

The deep module should provide high-leverage operations such as:

- create new worktree and session
- create from local/origin branch source
- ensure a session exists for an entry
- switch to a worktree session from inside tmux
- attach to a worktree session from outside tmux where appropriate
- restore sessions for entries missing sessions
- delete worktree, session, optional branch, metadata title
- handle deleting the currently active session by switching to the root session first

The exact interface should be designed before implementation, but the important seam is: TUI and MCP should call lifecycle behavior instead of owning it.

## Implementation Notes

- Do not try to solve all UI-specific concerns in the lifecycle module. Bubble Tea message handling and rendering stay in `internal/tui`.
- Keep command behavior observable through returned result structs, not printed messages.
- Use adapters or function fields for git/tmux/metadata where needed so lifecycle behavior can be tested without local process calls.
- Move metadata title cleanup for delete into the lifecycle module so MCP and TUI do not diverge.
- Keep `config.RunHook` and `config.RunHookInTmux` behavior unchanged.
- Preserve current deleting-self safety behavior from `confirm.go`.

## Acceptance Criteria

- TUI and MCP no longer duplicate lifecycle rules for create/delete/switch/ensure/restore.
- Deleting a worktree consistently removes title metadata across TUI and MCP flows.
- Existing create-from-existing-branch semantics remain unchanged.
- Existing delete behavior, including optional branch delete and deleting-self safety, remains unchanged.
- Lifecycle behavior has tests using fake adapters.
- `go test ./...` passes.

## Suggested Tests

- Create new worktree calls git add, creates session with layout, runs on_create hook.
- Create from origin uses tracking branch when local branch does not exist.
- Ensure session is a no-op when `HasSession=true`.
- Ensure session creates layout/theme session when `HasSession=false`.
- Delete runs on_destroy, kills session, removes worktree, optionally deletes branch, clears metadata.
- Deleting current session switches to root session before kill.
- Restore creates only missing sessions and records skipped/restored/errors.

## Comments

This is the top recommendation from the architecture review because it gives the most leverage across TUI, MCP, restore, and future features.
