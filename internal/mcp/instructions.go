package mcp

const instructions = `Synco manages parallel git worktrees with dedicated tmux sessions from a single repository root. The user orchestrates work from the main worktree, dispatching tasks to branch worktrees.

## Primary use case: dispatching a task to a new worktree

When the user gives you a task to work on in a worktree, follow this sequence:

1. **List worktrees** (synco_list_worktrees) — check what already exists to avoid duplicates.
2. **Create a worktree** (synco_create_worktree) — pick a descriptive branch name for the task (e.g. "feat/add-auth", "fix/login-crash"). This creates the git worktree and a tmux session.
3. **Bootstrap the worktree** (synco_send_keys) — send a command to the new worktree's tmux session to start work. The most common bootstrap is launching Claude Code with the task instructions:
   synco_send_keys(branch="feat/add-auth", keys='claude --dangerously-skip-permissions "Your task instructions here"')
   The instructions sent to Claude should be a clear, self-contained prompt describing what to do. Include relevant context the user provided.
4. **Verify it started** (synco_session_output) — check that the command is running in the session.

This is the most common pattern: the user describes a task, you create a worktree and send a claude command to it with the task as the prompt.

## Monitoring and managing worktrees

- **Check progress** (synco_session_output) — read terminal output to see how work is going in a worktree.
- **Send follow-up commands** (synco_send_keys) — run additional commands in a worktree's session (tests, builds, etc.).
- **Inspect task files** (synco_inspect_task) — read TICKET.md or other files from a worktree.
- **Switch sessions** (synco_switch_session) — move the user's tmux client to a worktree.
- **Read config** (synco_get_config) — check worktree directory, hooks, aliases, and other settings.
- **Clean up** (synco_delete_worktree) — remove a worktree when work is done and merged.

## Important conventions

- **Branch names are the primary identifier.** All tools take a "branch" parameter — use the short name (e.g. "feat/auth-refactor", not "refs/heads/feat/auth-refactor").
- **Always list before creating.** Call synco_list_worktrees first to avoid duplicates.
- **Check session output after sending commands.** Don't assume success — verify with synco_session_output.
- **The main worktree cannot be deleted.** This is enforced by the tool.
- **on_create hooks run automatically.** When a worktree is created, configured hooks (e.g. dependency install) run. Check output to confirm completion before sending further commands.
`
