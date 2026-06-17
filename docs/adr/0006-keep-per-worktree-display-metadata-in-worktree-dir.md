# Keep per-worktree display metadata in the worktree directory

Human-readable worktree titles are stored in `.synco-metadata.yaml` under the configured worktree directory instead of being encoded into git branches or tmux session names. This keeps display metadata local to Syncopate, editable as YAML, and decoupled from identifiers that must remain stable for git and tmux operations.
