package orchestrate

import (
	"fmt"
	"strings"

	"github.com/charles-albert-raymond/synco/internal/config"
	"github.com/charles-albert-raymond/synco/internal/session"
	"github.com/charles-albert-raymond/synco/internal/state"
	"github.com/charles-albert-raymond/synco/internal/tmux"
	"github.com/charles-albert-raymond/synco/internal/worktree"
)

// CreateWorktreeOpts controls create behavior.
type CreateWorktreeOpts struct {
	CreationProfile string
}

// CreateWorktree creates a git worktree and applies the selected creation profile.
func CreateWorktree(repoRoot string, cfg config.Config, branch, base string, opts CreateWorktreeOpts) (wtPath, sessName string, err error) {
	profile, _, err := cfg.ResolveCreationProfile(opts.CreationProfile)
	if err != nil {
		return "", "", err
	}
	if err := validateCreationProfile(profile); err != nil {
		return "", "", err
	}

	wtPath = cfg.WorktreePath(repoRoot, branch)
	if err := config.RunProfileScripts(cfg, repoRoot, profile, config.LifecycleBeforeCreate, branch, wtPath, repoRoot); err != nil {
		return "", "", err
	}

	if err := worktree.Add(repoRoot, wtPath, branch, true, base); err != nil {
		return "", "", fmt.Errorf("failed to create worktree: %w", err)
	}

	if !profile.ShouldCreateSession() {
		if err := config.RunProfileScripts(cfg, repoRoot, profile, config.LifecycleAfterCreate, branch, wtPath, wtPath); err != nil {
			return wtPath, "", fmt.Errorf("worktree created but after_create scripts failed: %w", err)
		}
		return wtPath, "", nil
	}
	if sessName, err = createSessionForProfile(repoRoot, cfg, profile, branch, wtPath); err != nil {
		return wtPath, sessName, err
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
func CreateWorktreeFromExisting(repoRoot string, cfg config.Config, source worktree.BranchSource, opts CreateWorktreeOpts) (wtPath, sessName string, err error) {
	profile, _, err := cfg.ResolveCreationProfile(opts.CreationProfile)
	if err != nil {
		return "", "", err
	}
	if err := validateCreationProfile(profile); err != nil {
		return "", "", err
	}

	localBranch := source.Branch

	wtPath = cfg.WorktreePath(repoRoot, localBranch)
	if err := config.RunProfileScripts(cfg, repoRoot, profile, config.LifecycleBeforeCreate, localBranch, wtPath, repoRoot); err != nil {
		return "", "", err
	}

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

	if !profile.ShouldCreateSession() {
		if err := config.RunProfileScripts(cfg, repoRoot, profile, config.LifecycleAfterCreate, localBranch, wtPath, wtPath); err != nil {
			return wtPath, "", fmt.Errorf("worktree created but after_create scripts failed: %w", err)
		}
		return wtPath, "", nil
	}
	if sessName, err = createSessionForProfile(repoRoot, cfg, profile, localBranch, wtPath); err != nil {
		return wtPath, sessName, err
	}

	return wtPath, sessName, nil
}

func validateCreationProfile(profile config.CreationProfile) error {
	if !profile.ShouldCreateSession() && profile.Bootstrap != "" {
		return fmt.Errorf("creation profile bootstrap requires create_session=true")
	}
	return nil
}

func createSessionForProfile(repoRoot string, cfg config.Config, profile config.CreationProfile, branch, wtPath string) (string, error) {
	project := session.ResolveProjectName(repoRoot, cfg.ProjectName)
	sessName := session.SessionNameFor(project, branch)
	if tmux.SessionExists(sessName) {
		return sessName, nil
	}
	if err := tmux.NewSessionWithLayout(sessName, wtPath, cfg); err != nil {
		return "", fmt.Errorf("worktree created at %s but tmux session failed: %w", wtPath, err)
	}
	if profile.ShouldRunOnCreate() {
		if err := config.RunHookInTmux(sessName, cfg.OnCreate, branch, wtPath); err != nil {
			return sessName, fmt.Errorf("worktree and session created but on_create hook failed: %w", err)
		}
	}
	if err := config.RunProfileScriptsInTmux(cfg, repoRoot, profile, config.LifecycleAfterCreate, branch, wtPath, sessName); err != nil {
		return sessName, fmt.Errorf("worktree and session created but after_create scripts failed: %w", err)
	}
	if profile.Bootstrap != "" {
		if err := tmux.SendKeys(sessName, expandBootstrap(profile.Bootstrap, branch, wtPath)); err != nil {
			return sessName, fmt.Errorf("worktree and session created but bootstrap failed: %w", err)
		}
	}
	return sessName, nil
}

func expandBootstrap(command, branch, wtPath string) string {
	replacer := strings.NewReplacer(
		"{{branch}}", branch,
		"{{worktree_path}}", wtPath,
	)
	return replacer.Replace(command)
}

// DeleteWorktreeOpts controls delete behavior.
type DeleteWorktreeOpts struct {
	DeleteBranch    bool
	CreationProfile string
}

// DeleteWorktree removes a worktree, optionally deletes the branch, and kills the tmux session.
// It does NOT handle "deleting self" tmux switching — TUI callers handle that separately.
func DeleteWorktree(repoRoot string, cfg config.Config, entry state.Entry, opts DeleteWorktreeOpts) error {
	profile, _, err := cfg.ResolveCreationProfile(opts.CreationProfile)
	if err != nil {
		return err
	}

	if err := config.RunProfileScripts(cfg, repoRoot, profile, config.LifecycleBeforeDestroy, entry.BranchShort, entry.Worktree.Path, entry.Worktree.Path); err != nil {
		return err
	}

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

	if err := config.RunProfileScripts(cfg, repoRoot, profile, config.LifecycleAfterDestroy, entry.BranchShort, entry.Worktree.Path, repoRoot); err != nil {
		return fmt.Errorf("after_destroy scripts failed: %w", err)
	}

	return nil
}
