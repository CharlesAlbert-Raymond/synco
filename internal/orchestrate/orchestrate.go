package orchestrate

import (
	"fmt"
	"strings"

	"github.com/charles-albert-raymond/synco/internal/config"
	"github.com/charles-albert-raymond/synco/internal/state"
	"github.com/charles-albert-raymond/synco/internal/tmux"
	"github.com/charles-albert-raymond/synco/internal/worktree"
)

// CreateWorktree creates a git worktree, a tmux session, and runs the on_create hook.
func CreateWorktree(repoRoot string, cfg config.Config, branch, base string) (wtPath, sessName string, err error) {
	wtPath = cfg.WorktreePath(repoRoot, branch)

	if err := worktree.Add(repoRoot, wtPath, branch, true, base); err != nil {
		return "", "", fmt.Errorf("failed to create worktree: %w", err)
	}

	project := tmux.ResolveProjectName(repoRoot, cfg.ProjectName)
	sessName = tmux.SessionNameFor(project, branch)
	sessionExists := tmux.SessionExists(sessName)
	if !sessionExists {
		if err := tmux.NewSessionWithLayout(sessName, wtPath, cfg); err != nil {
			return wtPath, "", fmt.Errorf("worktree created at %s but tmux session failed: %w", wtPath, err)
		}

		if err := config.RunHookInTmux(sessName, cfg.OnCreate, branch, wtPath); err != nil {
			return wtPath, sessName, fmt.Errorf("worktree and session created but on_create hook failed: %w", err)
		}
	}

	return wtPath, sessName, nil
}

// BranchSourceFromName converts a user-provided existing branch name into a branch source.
func BranchSourceFromName(repoRoot, branch string) worktree.BranchSource {
	if strings.HasPrefix(branch, "origin/") {
		localBranch := strings.TrimPrefix(branch, "origin/")
		return worktree.BranchSource{
			Kind:        worktree.BranchSourceOrigin,
			Branch:      localBranch,
			RemoteRef:   branch,
			LocalExists: worktree.BranchExists(repoRoot, localBranch),
		}
	}
	return worktree.BranchSource{Kind: worktree.BranchSourceLocal, Branch: branch, LocalExists: true}
}

// CreateWorktreeFromExisting creates a worktree from an existing local or origin branch source.
// Origin sources create a local tracking branch when one does not already exist.
func CreateWorktreeFromExisting(repoRoot string, cfg config.Config, source worktree.BranchSource) (wtPath, sessName string, err error) {
	localBranch := source.Branch

	wtPath = cfg.WorktreePath(repoRoot, localBranch)

	switch source.Kind {
	case worktree.BranchSourceOrigin:
		if source.RemoteRef == "" {
			source.RemoteRef = "origin/" + localBranch
		}
		if worktree.BranchExists(repoRoot, localBranch) {
			err = worktree.Add(repoRoot, wtPath, localBranch, false, "")
		} else {
			err = worktree.AddTracking(repoRoot, wtPath, localBranch, source.RemoteRef)
		}
	case worktree.BranchSourceLocal:
		err = worktree.Add(repoRoot, wtPath, localBranch, false, "")
	default:
		return "", "", fmt.Errorf("unknown branch source %q", source.Kind)
	}
	if err != nil {
		return "", "", fmt.Errorf("failed to create worktree: %w", err)
	}

	project := tmux.ResolveProjectName(repoRoot, cfg.ProjectName)
	sessName = tmux.SessionNameFor(project, localBranch)
	sessionExists := tmux.SessionExists(sessName)
	if !sessionExists {
		if err := tmux.NewSessionWithLayout(sessName, wtPath, cfg); err != nil {
			return wtPath, "", fmt.Errorf("worktree created at %s but tmux session failed: %w", wtPath, err)
		}

		if err := config.RunHookInTmux(sessName, cfg.OnCreate, localBranch, wtPath); err != nil {
			return wtPath, sessName, fmt.Errorf("worktree and session created but on_create hook failed: %w", err)
		}
	}

	return wtPath, sessName, nil
}

// DeleteWorktreeOpts controls delete behavior.
type DeleteWorktreeOpts struct {
	DeleteBranch bool
}

// DeleteWorktree removes a worktree, optionally deletes the branch, and kills the tmux session.
// It does NOT handle "deleting self" tmux switching — TUI callers handle that separately.
func DeleteWorktree(repoRoot string, cfg config.Config, entry state.Entry, opts DeleteWorktreeOpts) error {
	// Run on_destroy hook
	if err := config.RunHook(cfg.OnDestroy, entry.BranchShort, entry.Worktree.Path); err != nil {
		return fmt.Errorf("on_destroy hook failed: %w", err)
	}

	// Kill tmux session first for instant UI feedback
	if entry.HasSession {
		_ = tmux.KillSession(entry.SessionName)
	}

	// Fast-remove worktree: rename to trash dir, prune git metadata,
	// then delete files in the background
	if err := worktree.RemoveFast(repoRoot, entry.Worktree.Path); err != nil {
		return fmt.Errorf("failed to remove worktree: %w", err)
	}

	// Optionally delete branch
	if opts.DeleteBranch {
		if err := worktree.DeleteBranch(repoRoot, entry.BranchShort); err != nil {
			return fmt.Errorf("worktree removed but branch delete failed: %w", err)
		}
	}

	return nil
}
