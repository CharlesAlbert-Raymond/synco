# Cleanup errors when hook removes worktree

Status: ready-for-human

## Bug

When deleting a worktree, Syncopate runs `on_destroy` and then removes the worktree. Some cleanup hooks already call the repository cleanup script for the current worktree, which can remove the worktree before Syncopate reaches its own removal step.

## Evidence

The delete confirmation reports:

```text
failed to remove worktree: git worktree remove: fatal: '<worktree path>' is not a working tree: exit status 128
```

The UI still reports the feature destroyed afterward, making the error noisy and confusing.

## Expected Behavior

If the worktree has already been removed by `on_destroy`, deletion should continue without showing a fatal worktree-removal error. Optional branch deletion should still run when requested.

## Acceptance Criteria

- Worktree deletion tolerates a missing or already-unregistered worktree path after `on_destroy`.
- Other `git worktree remove` failures remain visible.
- Regression coverage exercises the already-removed cleanup path.

## Comments
