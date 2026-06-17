# Use tmux as the runtime control plane

Syncopate uses tmux sessions and panes as the durable runtime surface for worktree navigation, sidebar rendering, popups, command delivery, pane capture, layouts, and focus management. This accepts tmux lock-in in exchange for visible, inspectable processes that survive UI restarts and can be controlled both by users and by Syncopate without inventing a separate terminal/session manager.
