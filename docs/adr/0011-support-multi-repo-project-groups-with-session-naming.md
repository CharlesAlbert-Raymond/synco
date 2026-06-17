# Support multi-repo project groups with session naming

Multi-repo projects are configured as named repo lists in the global YAML config and rendered by a project TUI that gathers each repo independently. Syncopate relies on per-repo project labels and hierarchical tmux session names instead of introducing a central project database, which keeps grouping transparent and compatible with tmux's native session tree.
