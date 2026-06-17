# Make TUI And MCP Thin Adapters

Status: ready-for-agent
Type: Architecture
Priority: Medium

## Problem

TUI and MCP modules currently translate user input, render responses, and own command policy. This widens their interfaces and makes behavior harder to test through one seam.

Examples:

- `internal/tui/list.go` owns session ensure/switch/attach behavior.
- `internal/tui/confirm.go` owns delete-self lifecycle behavior.
- `internal/tui/app.go` owns popup launching and metadata title persistence.
- `internal/mcp/tools.go` owns lifecycle details, title metadata writes, and result shaping in the same handlers.

Once lifecycle and snapshot modules are deepened, TUI and MCP should become adapters: they should translate inputs into calls and translate results into UI/tool output.

## Files

- `internal/tui/app.go`
- `internal/tui/list.go`
- `internal/tui/create.go`
- `internal/tui/confirm.go`
- `internal/tui/project.go`
- `internal/mcp/tools.go`
- `internal/mcp/tools_test.go`
- Lifecycle/snapshot modules from earlier tickets

## Proposed Solution

Refactor TUI and MCP call sites to use the deep lifecycle and snapshot interfaces.

TUI should own:

- Bubble Tea state transitions
- key handling
- rendering
- status messages
- popup wiring
- filter/cursor behavior

MCP should own:

- tool definitions
- argument parsing
- mapping lifecycle/snapshot results to JSON-friendly results
- MCP error result formatting

Neither adapter should own core worktree/session lifecycle rules.

## Implementation Notes

- This ticket should be done after issues `03` and `04`; otherwise it will just move code sideways.
- Keep UI behavior unchanged, including messages where practical.
- Preserve MCP result fields for compatibility with existing users/agents.
- Prefer deleting duplicated helper code over adding compatibility wrappers.
- Keep tests at the adapter interface: TUI tests for interactions/render state, MCP tests for argument/result mapping.

## Acceptance Criteria

- TUI code delegates create/delete/switch/ensure/restore behavior to lifecycle modules.
- MCP handlers delegate create/delete/switch/ensure/title behavior to lifecycle modules where applicable.
- Existing MCP tool names, inputs, and result fields remain unchanged.
- Existing TUI keybindings and popup behavior remain unchanged.
- Adapter tests are narrower and lifecycle tests cover shared behavior.
- `go test ./...` passes.

## Suggested Tests

- TUI list enter delegates to lifecycle session activation and updates messages from result/error.
- TUI delete confirmation delegates delete behavior and handles quit when deleting current session.
- MCP create maps request arguments to lifecycle create call and preserves response fields.
- MCP delete rejects main worktree and delegates deletion for non-main entries.
- MCP switch rejects calls outside tmux and delegates switch behavior inside tmux.

## Comments

This ticket is deliberately last because it should mostly delete duplicated policy after the deeper modules exist.
