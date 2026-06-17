# Run lifecycle hooks in the target session when possible

Creation hooks are sent into the target tmux session so users can see setup output in the same place they will work. Destruction hooks still run inline before deletion so cleanup can fail safely before the tmux session and worktree are removed.
