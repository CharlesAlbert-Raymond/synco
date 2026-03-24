#!/usr/bin/env bash
# Syncopate on_create hook: bootstrap a worktree for developing syncopate itself.
# Copies the syncopate binary from the main worktree and launches it.

set -euo pipefail

# Find the main worktree (first entry in git worktree list)
MAIN_WORKTREE="$(git worktree list --porcelain | head -1 | cut -d' ' -f2)"

# Copy the binary from the main worktree into this one
cp "$MAIN_WORKTREE/syncopate" ./syncopate

# Launch syncopate
./syncopate
