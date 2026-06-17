# Use a Go CLI with Bubble Tea for the terminal UI

Syncopate is a local developer workflow tool, so the core app is a Go command-line binary with Bubble Tea models for interactive views. This keeps distribution simple, gives direct access to local `git` and `tmux` commands, and avoids introducing a separate daemon or web UI for workflows that must already happen inside the terminal.
