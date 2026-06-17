# Intent-only popups with background jobs

Status: ready-for-agent
Priority: High
Type: Improvement

## Problem

Create and delete tmux popups currently perform the requested work inside the modal flow. Long-running lifecycle scripts, worktree removal, branch deletion, or session setup block the sidebar/TUI after the user has already made a decision.

This makes Syncopate feel frozen while scripts execute and prevents the user from continuing work in other sessions.

## Goal

Keep tmux popups as modal input surfaces, but make them intent-only. Popups should collect create, delete, and edit-title decisions, emit a typed intent to the parent TUI, and close immediately.

The parent TUI owns all side effects. Long-running create and delete operations run as parent-owned background jobs with in-memory status.

## Design Decisions

- Keep tmux `display-popup` for transient UI.
- Popups remain modal for input only.
- Popups emit intents, not side effects.
- The parent TUI/sidebar process owns all project state mutation.
- Create, delete, and edit-title should follow the same intent-only popup pattern.
- Create and delete operations run as background jobs owned by the parent TUI.
- Background job state is in-memory only for this version.
- Create `on_create` hooks are sent to the new tmux session; the create job completes once setup is dispatched.
- Delete jobs run `on_destroy` in the background and wait for it before killing the tmux session or removing the worktree.
- Active jobs prevent duplicate/conflicting actions for the same branch.
- Jobs are not cancellable in this version.
- No durable job runner or dedicated job log view in this version.

## Acceptance Criteria

- Create, delete, and edit-title popups do not mutate project state directly.
- Popup submission writes/returns an intent to the parent TUI and closes immediately after the intent is accepted.
- Cancelling a popup produces no side effects.
- Classic embedded forms and popup forms share the same intent-emitting model behavior.
- The parent TUI handles edit-title intents synchronously by saving metadata.
- The parent TUI launches create and delete intents as background jobs.
- The TUI remains responsive while create/delete jobs are running.
- Active jobs are displayed in the list/sidebar, for example `creating...` or `deleting...`.
- Duplicate or conflicting jobs for the same branch are blocked with a visible status message.
- A worktree marked `deleting...` cannot be opened or deleted again while the job is active.
- Create jobs do not automatically switch the user to the new session when they complete.
- Create title metadata is saved only after creation succeeds.
- Delete title metadata is removed only after deletion succeeds.
- Delete order is: run `on_destroy`, kill tmux session if present, fast-remove worktree, optionally delete branch, then remove title metadata.
- Background create/delete failures are surfaced as TUI status messages.
- The list refreshes after job completion so git/tmux state remains the source of truth.

## Out Of Scope

- Durable or persisted background jobs across Syncopate restarts.
- Cancelling a job after submit.
- A dedicated job log or output view.
- Replacing tmux popups with panes or full-screen forms.
- Running delete hooks fire-and-forget while deletion proceeds.

## Comments
