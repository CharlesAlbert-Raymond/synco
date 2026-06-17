# Use reconciled state snapshots for views and tools

The app builds a state snapshot by joining git worktrees, tmux sessions, metadata, current-session detection, and listening ports. The TUI, popups, project view, and MCP tools consume that reconciled view so behavior is based on observed local state rather than a separate persisted registry that could drift from git or tmux.
