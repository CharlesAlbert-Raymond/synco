# Architecture Clarity Refactor

Status: ready-for-agent
Type: Architecture
Priority: High

## Summary

Improve Syncopate's architecture by deepening shallow modules around session identity, snapshot gathering, tmux command execution, and worktree/session lifecycle behavior.

The current code works, but several important rules are spread across `main.go`, `internal/tui`, `internal/mcp`, `internal/restore`, `internal/orchestrate`, `internal/state`, and `internal/tmux`. This makes behavior harder to reason about, harder to test without tmux/git/process dependencies, and easier to accidentally regress when adding new features.

## Goals

- Concentrate worktree/session lifecycle behavior behind one deep module interface.
- Keep TUI and MCP modules as adapters that translate user/tool input into core behavior.
- Split pure session identity rules away from tmux command execution.
- Make state gathering testable without requiring live `tmux`, `ps`, or `lsof` calls.
- Create seams only where there are real adapters: production local process calls and fake/in-memory adapters for tests.

## Non-Goals

- Do not change user-facing CLI flags or TUI keybindings.
- Do not rewrite the Bubble Tea UI.
- Do not add broad abstraction layers without immediate production/test adapters.
- Do not change persisted config or metadata formats unless required by a specific ticket.

## Current Architecture Friction

- `main.go` owns launch routing, config fallback, repo root resolution, project mode, popup routing, and tmux launch behavior.
- `internal/tmux` mixes pure naming rules, git root detection, tmux command execution, sidebar policy, layout/theme application, process tree inspection, and migration logic.
- `internal/state.GatherWithOpts` joins worktrees, tmux sessions, metadata, current session, and port discovery in one path.
- TUI and MCP both contain command behavior such as session creation, session switching, title cleanup, and delete rules.
- `internal/orchestrate` is already the right direction, but it does not yet own enough lifecycle behavior to give callers a small, high-leverage interface.

## Suggested Implementation Order

1. `01-split-session-identity-from-tmux.md`
2. `02-extract-tmux-command-runner.md`
3. `03-deepen-state-snapshot-builder.md`
4. `04-deepen-worktree-session-lifecycle.md`
5. `05-make-tui-and-mcp-thin-adapters.md`

## Validation

- `go test ./...` passes after each issue.
- Existing behavior remains unchanged for common flows:
  - start synco outside tmux
  - start synco inside tmux
  - create a new worktree
  - create from existing local/origin branch
  - switch sessions from classic and sidebar modes
  - delete a worktree
  - MCP list/create/delete/switch/send/capture/title tools

## Comments

Created from the architecture review report generated at `/var/folders/y5/cvpnk9x500zc29q54c35nnzc0000gn/T/architecture-review-20260617-0001.html`.
