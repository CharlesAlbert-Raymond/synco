# Merge global and repo YAML configuration

Syncopate reads global config from `~/.config/synco/config.yaml` and overlays repo-local `.synco.yaml` values. Global config captures user-wide defaults and project groups, while repo config can override worktree paths, hooks, themes, layouts, aliases, and project labels without requiring a database or hidden service state.
