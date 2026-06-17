package config

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	LifecycleBeforeCreate  = "before_create"
	LifecycleAfterCreate   = "after_create"
	LifecycleBeforeDestroy = "before_destroy"
	LifecycleAfterDestroy  = "after_destroy"
)

// RunHook executes a lifecycle script with worktree context as env vars.
// Returns nil if the hook is empty (not configured).
func RunHook(script, branch, worktreePath string) error {
	if script == "" {
		return nil
	}

	cmd := exec.Command("sh", "-c", script)
	cmd.Dir = worktreePath
	cmd.Env = append(os.Environ(),
		"SYNCO_BRANCH="+branch,
		"SYNCO_WORKTREE_PATH="+worktreePath,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("hook failed: %w", err)
	}
	return nil
}

// RunHookInTmux sends the hook script to a tmux session instead of running inline.
// This is preferred so the user can see output in their session.
func RunHookInTmux(sessionName, script, branch, worktreePath string) error {
	if script == "" {
		return nil
	}

	// Wrap the script with env vars so it has context
	wrapped := fmt.Sprintf(
		"SYNCO_BRANCH=%q SYNCO_WORKTREE_PATH=%q %s",
		branch, worktreePath, script,
	)

	cmd := exec.Command("tmux", "send-keys", "-t", sessionName, wrapped, "Enter")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("send hook to tmux: %s: %w", string(out), err)
	}
	return nil
}

// ScriptNamesForEvent returns the ordered script names for a lifecycle event.
func (p CreationProfile) ScriptNamesForEvent(event string) []string {
	switch event {
	case LifecycleBeforeCreate:
		return p.Scripts.BeforeCreate
	case LifecycleAfterCreate:
		return p.Scripts.AfterCreate
	case LifecycleBeforeDestroy:
		return p.Scripts.BeforeDestroy
	case LifecycleAfterDestroy:
		return p.Scripts.AfterDestroy
	default:
		return nil
	}
}

// RunProfileScripts executes named profile scripts for a lifecycle event in order.
func RunProfileScripts(cfg Config, repoRoot string, profile CreationProfile, event, branch, worktreePath, workdir string) error {
	for _, name := range profile.ScriptNamesForEvent(event) {
		if err := RunProfileScript(cfg, repoRoot, name, event, branch, worktreePath, workdir); err != nil {
			return err
		}
	}
	return nil
}

// RunProfileScriptsInTmux sends named profile scripts to a tmux session in order.
func RunProfileScriptsInTmux(cfg Config, repoRoot string, profile CreationProfile, event, branch, worktreePath, sessionName string) error {
	for _, name := range profile.ScriptNamesForEvent(event) {
		path, err := ResolveScriptPath(cfg, repoRoot, name, event)
		if err != nil {
			return err
		}
		if err := runScriptInTmux(sessionName, path, branch, worktreePath); err != nil {
			return fmt.Errorf("script %q for %s failed: %w", name, event, err)
		}
	}
	return nil
}

func runScriptInTmux(sessionName, path, branch, worktreePath string) error {
	statusFile, err := os.CreateTemp("", "synco-script-status-*")
	if err != nil {
		return fmt.Errorf("create status file: %w", err)
	}
	statusPath := statusFile.Name()
	_ = statusFile.Close()
	_ = os.Remove(statusPath)
	defer os.Remove(statusPath)

	command := fmt.Sprintf(
		"SYNCO_BRANCH=%s SYNCO_WORKTREE_PATH=%s bash %s; __synco_status=$?; printf '%%s' \"$__synco_status\" > %s",
		shellQuote(branch), shellQuote(worktreePath), shellQuote(path), shellQuote(statusPath),
	)
	cmd := exec.Command("tmux", "send-keys", "-t", sessionName, command, "Enter")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("send script to tmux: %s: %w", string(out), err)
	}

	for {
		data, err := os.ReadFile(statusPath)
		if err == nil {
			status := strings.TrimSpace(string(data))
			if status == "0" {
				return nil
			}
			return fmt.Errorf("exit status %s", status)
		}
		if !os.IsNotExist(err) {
			return fmt.Errorf("read status file: %w", err)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// RunProfileScript executes one named profile script through bash.
func RunProfileScript(cfg Config, repoRoot, name, event, branch, worktreePath, workdir string) error {
	path, err := ResolveScriptPath(cfg, repoRoot, name, event)
	if err != nil {
		return err
	}
	cmd := exec.Command("bash", path)
	cmd.Dir = workdir
	cmd.Env = append(os.Environ(),
		"SYNCO_BRANCH="+branch,
		"SYNCO_WORKTREE_PATH="+worktreePath,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("script %q for %s failed: %w", name, event, err)
	}
	return nil
}

// ResolveScriptPath returns the absolute path for a named script.
func ResolveScriptPath(cfg Config, repoRoot, name, event string) (string, error) {
	def, ok := cfg.Scripts[name]
	if !ok {
		return "", fmt.Errorf("script %q for %s is not defined", name, event)
	}
	if def.Path == "" {
		return "", fmt.Errorf("script %q for %s has no path", name, event)
	}
	path := ExpandRepoPath(def.Path)
	if !filepath.IsAbs(path) {
		path = filepath.Join(repoRoot, path)
	}
	if info, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("script %q for %s not found at %s", name, event, path)
		}
		return "", fmt.Errorf("script %q for %s cannot be read at %s: %w", name, event, path, err)
	} else if info.IsDir() {
		return "", fmt.Errorf("script %q for %s is a directory at %s", name, event, path)
	}
	return path, nil
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}
