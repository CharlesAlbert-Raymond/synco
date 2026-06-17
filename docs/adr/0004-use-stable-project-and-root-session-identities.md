# Use stable project and root session identities

Tmux sessions are named by project and branch, with the root worktree addressed by a stable `root` key and named as the project itself. This avoids breaking navigation when the main worktree changes branches, and the `project/branch` naming convention gives tmux's session list a natural hierarchy for single-repo and multi-repo workflows.
