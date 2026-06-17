# Extract Tmux Command Runner Seam

Status: ready-for-agent
Type: Architecture
Priority: High

## Problem

`internal/tmux` directly calls `exec.Command` and `syscall.Exec` across multiple files. Error formatting, output parsing, command construction, and process replacement are spread throughout the implementation.

Examples:

- `internal/tmux/tmux.go` for sessions, send-keys, attach, switch, migration, capture
- `internal/tmux/sidebar.go` for sidebar panes, hooks, focus, popups, attach
- `internal/tmux/layout.go` for layouts and theme commands
- `internal/tmux/ports.go` for `tmux`, `ps`, and `lsof`

This makes tests either rely on pure helper extraction or skip the real command behavior. A runner seam would provide locality for command execution and leverage for fake-adapter tests.

## Files

- `internal/tmux/tmux.go`
- `internal/tmux/sidebar.go`
- `internal/tmux/layout.go`
- `internal/tmux/ports.go`
- `internal/tmux/tmux_test.go`

## Proposed Solution

Introduce an internal tmux command runner seam with two adapters:

- production adapter: executes local commands using `exec.Command` / `syscall.Exec`
- fake adapter: records commands and returns configured output/errors in tests

Keep the seam internal to the `tmux` module unless another module genuinely needs to provide an adapter. The goal is not to expose a broad cross-package interface; the goal is to concentrate local process execution and make tmux behavior testable.

## Implementation Notes

- Start small. A package-level runner variable or unexported struct can be enough if tests can replace it safely.
- Include operations needed by current code:
  - run command with output
  - run command with combined output
  - run command without output
  - look up executable
  - exec-replace process for attach paths
- Ensure tests restore the production runner after replacing it.
- Do not change command arguments unless needed to preserve behavior.
- Keep parsing functions pure where they already are pure, such as `parseLsofLine`.

## Acceptance Criteria

- Direct `exec.Command` usage inside `internal/tmux` is concentrated behind one runner implementation, except where a direct call is clearly simpler and documented.
- Tests can fake tmux command output without invoking the real `tmux` binary.
- Existing behavior for session create/switch/attach/sidebar/popup/layout/ports remains unchanged.
- `go test ./...` passes.

## Suggested Tests

- `ListSessions` parses fake `tmux list-sessions` output and filters by project.
- `NewSession` treats "session already exists" as success only when the fake `has-session` path says it exists.
- `EnsureSidebar` does not split when fake pane output already contains `--sidebar`.
- `PortsBySession` maps fake `tmux list-panes`, `ps`, and `lsof` output into sorted ports.

## Comments

This ticket creates the local-substitutable seam needed by later lifecycle and snapshot tests.
