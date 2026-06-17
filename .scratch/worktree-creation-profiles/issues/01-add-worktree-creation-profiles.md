# Add Worktree Creation Profiles

Status: ready-for-agent

## Problem

Worktree creation currently assumes one default lifecycle: create the git worktree, create a tmux session, and run the configured `on_create` hook. That is useful for normal development, but too rigid for lighter workflows.

Sometimes users only want to inspect code in another branch and do not want dependency installation, environment setup, or other expensive setup hooks. Other times users want a repeatable task-oriented setup that can create a worktree and immediately bootstrap an agent command. Future batch workflows, such as fetching every open PR branch with a certain label, should be able to reuse the same creation behavior instead of hard-coding lifecycle choices.

## Goal

Introduce named worktree creation profiles that let users choose how much of the creation lifecycle should run for a given worktree.

Profiles should be usable from both interactive TUI creation and MCP/tool-driven creation so humans, agents, and future jobs all share the same lifecycle policy.

## Proposed Config Shape

```yaml
default_creation_profile: dev

creation_profiles:
  inspect:
    create_session: false
    run_on_create: false

  dev:
    create_session: true
    run_on_create: true

  agent:
    create_session: true
    run_on_create: true
    bootstrap: 'claude --dangerously-skip-permissions "{{task}}"'
```

The exact config names can change during implementation if a better fit emerges, but the behavior should remain explicit and profile-based.

## Acceptance Criteria

- Users can define named creation profiles in `.synco.yaml`.
- A profile can control whether Syncopate creates a tmux session for the worktree.
- A profile can control whether Syncopate runs the configured `on_create` hook.
- A profile can optionally define one bootstrap command to run after creation when a session exists.
- Existing behavior remains unchanged when no profiles are configured.
- If profiles are configured and no profile is selected, Syncopate uses `default_creation_profile` when set.
- If neither profiles nor `default_creation_profile` are set, creation behaves like today's default development flow.
- The TUI create flow lets users select a creation profile for both new branches and existing branches.
- The MCP `synco_create_worktree` tool accepts a profile name and uses the same lifecycle behavior as the TUI.
- Unknown profile names produce a clear validation error before creating a worktree.
- Bootstrap commands can interpolate at least `{{branch}}` and `{{worktree_path}}`.

## Out Of Scope

- Querying GitHub pull requests.
- Fetching all PR branches with a specific label.
- A full reusable command registry, command palette, command tags, or multi-target command execution.
- General-purpose scripting beyond the single optional bootstrap command on a creation profile.

## Future Workflow Example

A later job system could resolve all open PR branches with label `needs-review`, then create each branch with the `inspect` profile. That job should call the same creation lifecycle used by the TUI and MCP tools rather than implementing its own worktree/session/hook policy.

## Notes

This ticket intentionally builds on the existing lifecycle model where a work context is a git worktree plus an optional tmux session. It also keeps the broader Command Center work separate: profiles decide how a worktree is created, while a future command system can manage reusable commands, tags, palettes, and targeting.
