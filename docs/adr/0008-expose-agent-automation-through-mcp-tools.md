# Expose agent automation through MCP tools

Syncopate exposes worktree and session operations through an MCP server rather than a second bespoke automation API. This lets coding agents list contexts, create or delete worktrees, switch sessions, send commands, capture output, inspect task files, and edit titles using the same local primitives as the TUI.
