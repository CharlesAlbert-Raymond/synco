# Model work contexts as git worktrees plus tmux sessions

Each Syncopate work context is represented by a git worktree on disk and an optional tmux session for interactive execution. This keeps source isolation delegated to git and runtime isolation delegated to tmux, while Syncopate reconciles both into a single state snapshot for the TUI and MCP tools.
