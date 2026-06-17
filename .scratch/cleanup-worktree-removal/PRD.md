# Cleanup worktree removal

## Problem

Cleanup can fail after the `on_destroy` hook removes the worktree path before Syncopate performs its own worktree removal.

## Goal

Deleting a worktree should be idempotent when cleanup hooks have already removed the worktree from disk or Git metadata.
