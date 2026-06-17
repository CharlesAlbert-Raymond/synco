# Add profile lifecycle scripts

Status: ready-for-agent
Priority: High
Type: Feature

## Problem

Creation profiles can currently toggle session creation, choose whether the global/local `on_create` hook runs, and send one `bootstrap` command into the created tmux session. That is not expressive enough for project-specific setup and cleanup workflows that need multiple ordered bash scripts tied to a profile.

Users need profiles to orchestrate lifecycle scripts, with reusable script definitions available from both global user config and repo-owned project config.

## Goal

Add profile-scoped lifecycle script execution for the full worktree lifecycle:

- `before_create`
- `after_create`
- `before_destroy`
- `after_destroy`

Scripts are bash scripts, referenced by name from a profile event list, and executed in the order declared for that lifecycle event.

## Proposed Shape

Support a merged script registry from global config and repo config, plus ordered event lists on creation profiles.

Example:

```yaml
scripts:
  install_deps:
    path: .synco/scripts/install_deps.sh
  seed_db:
    path: .synco/scripts/seed_db.sh
  cleanup_services:
    path: .synco/scripts/cleanup_services.sh

creation_profiles:
  agent:
    create_session: true
    run_on_create: true
    scripts:
      before_create:
        - install_deps
      after_create:
        - seed_db
      before_destroy:
        - cleanup_services
      after_destroy: []
    bootstrap: claude {{branch}}
```

Global config can define user-wide scripts. Repo config can define project scripts. When global and repo script names conflict, repo config wins, matching the existing local-over-global merge model.

Repo-owned scripts should commonly live under `.synco/scripts/`, but the config should be the source of truth rather than relying on directory scanning.

## Execution Semantics

- Execute script files through `bash <script-path>` rather than requiring executable bits.
- Run scripts with the same context environment used by existing hooks:
  - `SYNCO_BRANCH`
  - `SYNCO_WORKTREE_PATH`
- Run scripts from the target worktree path when that path exists.
- `before_create` runs before the worktree exists, so it must run from the repo root while still receiving the planned `SYNCO_WORKTREE_PATH`.
- `after_create` runs after the worktree and session are created.
- `before_destroy` runs before the tmux session/worktree are removed.
- `after_destroy` runs after removal has been attempted/completed; it should not assume the worktree path exists.
- For create flows with a tmux session, `after_create` scripts should run in the target session when practical so users can see output, consistent with ADR 0009.
- Existing `on_create`, `on_destroy`, and `bootstrap` behavior should remain compatible.

Recommended ordering:

- Create: `before_create` scripts, create worktree/session, existing `on_create` if enabled, `after_create` scripts, `bootstrap`.
- Destroy: `before_destroy` scripts, existing `on_destroy`, remove session/worktree, `after_destroy` scripts.

## Failure Semantics

- `before_create` failures stop creation before the worktree is created.
- `after_create` failures return an error after the worktree/session have been created, matching current partial-success behavior for `on_create` failures.
- `before_destroy` failures stop deletion so cleanup/preflight scripts can protect data.
- `after_destroy` failures are reported, but cannot roll back deletion.

## Acceptance Criteria

- Config supports a top-level `scripts` registry with named bash script paths.
- Config merge preserves global scripts and lets repo-local scripts override by name.
- `CreationProfile` supports ordered lifecycle event lists under `scripts`.
- Worktree creation runs profile scripts for `before_create` and `after_create` in the defined order.
- Worktree deletion runs profile scripts for `before_destroy` and `after_destroy` in the defined order.
- Scripts are invoked via `bash` and receive `SYNCO_BRANCH` and `SYNCO_WORKTREE_PATH`.
- Missing script references, missing files, or failed scripts produce clear errors naming the script and lifecycle event.
- Existing `on_create`, `on_destroy`, `bootstrap`, and `run_on_create` behavior continues to work.
- Unit tests cover config parsing, merge precedence, profile resolution, ordering, failure behavior, and path/env handling.
- The config view surfaces configured scripts and profile lifecycle script lists at a minimal level.

## Out Of Scope

- Manual `synco run-script` CLI commands.
- TUI command palette or script-running actions.
- MCP tools for manually triggering scripts.
- Directory auto-discovery of `.synco/scripts/*`.
- Non-bash script runtimes.

## Notes

This overlaps conceptually with the Command Center backlog, but this ticket is intentionally narrower: lifecycle automation owned by creation profiles, not a general command/action system.

## Comments
