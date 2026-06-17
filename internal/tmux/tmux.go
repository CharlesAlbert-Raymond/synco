package tmux

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/charles-albert-raymond/synco/internal/config"
	"github.com/charles-albert-raymond/synco/internal/session"
)

// Session represents a tmux session managed by synco.
type Session struct {
	Name     string
	Attached bool
}

// ListSessions returns tmux sessions belonging to the given project.
// This includes the root session (named exactly "project") and all branch
// sessions (named "project/branch").
func ListSessions(project string) ([]Session, error) {
	cmd := exec.Command("tmux", "list-sessions", "-F", "#{session_name}\t#{session_attached}")
	out, err := cmd.Output()
	if err != nil {
		// tmux returns error when no server is running — that's fine
		if strings.Contains(err.Error(), "exit status") {
			return nil, nil
		}
		return nil, fmt.Errorf("tmux list-sessions: %w", err)
	}

	var sessions []Session
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		parts := strings.SplitN(scanner.Text(), "\t", 2)
		if len(parts) < 2 {
			continue
		}
		name := parts[0]
		if !session.IsProjectSession(name, project) {
			continue
		}
		sessions = append(sessions, Session{
			Name:     name,
			Attached: parts[1] != "0",
		})
	}
	return sessions, nil
}

// NewSession creates a detached tmux session with the given name and working directory.
func NewSession(name, workdir string) error {
	cmd := exec.Command("tmux", "new-session", "-d", "-s", name, "-c", workdir)
	if out, err := cmd.CombinedOutput(); err != nil {
		if SessionExists(name) {
			return nil
		}
		return fmt.Errorf("tmux new-session: %s: %w", string(out), err)
	}
	return nil
}

// NewSessionWithLayout creates a detached tmux session and applies the
// configured layout and theme. This is the standard entry point for session
// creation so that all callers get consistent behavior.
func NewSessionWithLayout(name, workdir string, cfg config.Config) error {
	if err := NewSession(name, workdir); err != nil {
		return err
	}
	if layout := cfg.DefaultLayout(); layout != nil {
		_ = ApplyLayout(name, layout)
	}
	_ = ApplyTheme(name, cfg.Theme)
	return nil
}

// SendKeys sends a command string to a tmux session.
func SendKeys(session, keys string) error {
	cmd := exec.Command("tmux", "send-keys", "-t", session, keys, "Enter")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("tmux send-keys: %s: %w", string(out), err)
	}
	return nil
}

// KillSession kills a tmux session.
func KillSession(name string) error {
	cmd := exec.Command("tmux", "kill-session", "-t", name)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("tmux kill-session: %s: %w", string(out), err)
	}
	return nil
}

// AttachSession replaces the current process with tmux attach.
func AttachSession(name string) error {
	tmuxPath, err := exec.LookPath("tmux")
	if err != nil {
		return fmt.Errorf("tmux not found: %w", err)
	}
	return syscall.Exec(tmuxPath, []string{"tmux", "attach-session", "-t", name}, os.Environ())
}

// IsInsideTmux returns true if we are currently inside a tmux session.
func IsInsideTmux() bool {
	return os.Getenv("TMUX") != ""
}

// CurrentSessionName returns the name of the tmux session we're currently in.
func CurrentSessionName() (string, error) {
	cmd := exec.Command("tmux", "display-message", "-p", "#{session_name}")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("tmux display-message: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// SwitchClient switches the current tmux client to the given session.
// This is used when already inside tmux instead of attach.
func SwitchClient(name string) error {
	cmd := exec.Command("tmux", "switch-client", "-t", name)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("tmux switch-client: %s: %w", string(out), err)
	}
	return nil
}

// SessionExists returns true if a tmux session with the given name exists.
func SessionExists(name string) bool {
	return exec.Command("tmux", "has-session", "-t", name).Run() == nil
}

// MigrateSessionNames renames old-format sessions to the new hierarchical format.
// Old format: "project-root", "project-branch" (dash separator)
// New format: "project" (root), "project/branch" (slash separator)
// Also migrates the intermediate format "project/root" → "project".
// Returns the number of sessions migrated.
func MigrateSessionNames(project string) int {
	cmd := exec.Command("tmux", "list-sessions", "-F", "#{session_name}")
	out, err := cmd.Output()
	if err != nil {
		return 0
	}

	oldPrefix := project + "-"
	migrated := 0

	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		name := scanner.Text()

		// Migrate intermediate format: "project/root" → "project"
		if name == project+"/"+session.RootKey {
			newName := project
			if !SessionExists(newName) {
				if exec.Command("tmux", "rename-session", "-t", name, newName).Run() == nil {
					migrated++
				}
			}
			continue
		}

		// Migrate old dash format: "project-*"
		if !strings.HasPrefix(name, oldPrefix) {
			continue
		}
		// Skip if it already has the new format (starts with project/)
		if strings.HasPrefix(name, project+"/") {
			continue
		}

		suffix := strings.TrimPrefix(name, oldPrefix)
		var newName string
		if suffix == session.RootKey {
			// "project-root" → "project"
			newName = project
		} else {
			// "project-branch" → "project/branch"
			newName = project + "/" + suffix
		}

		if !SessionExists(newName) {
			if exec.Command("tmux", "rename-session", "-t", name, newName).Run() == nil {
				migrated++
			}
		}
	}
	return migrated
}

// CapturePaneOutput captures the last N lines of terminal output from a session's pane.
func CapturePaneOutput(session string, lines int) (string, error) {
	if lines <= 0 {
		lines = 50
	}
	cmd := exec.Command("tmux", "capture-pane", "-t", session, "-p", "-S", fmt.Sprintf("-%d", lines))
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("tmux capture-pane: %w", err)
	}
	return strings.TrimRight(string(out), "\n"), nil
}
