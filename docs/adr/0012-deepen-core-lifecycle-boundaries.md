# Deepen core lifecycle boundaries

Syncopate is moving lifecycle behavior into deeper core modules so TUI and MCP code remain thin adapters over worktree/session operations. The current code works, but session identity, tmux command execution, snapshot gathering, and lifecycle rules are spread across several packages; concentrating those rules will make new features easier to test without changing user-facing flags or keybindings.
