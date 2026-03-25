package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/charles-albert-raymond/syncopate/internal/config"
	"github.com/charles-albert-raymond/syncopate/internal/state"
	"github.com/charles-albert-raymond/syncopate/internal/tmux"
	"github.com/charles-albert-raymond/syncopate/internal/worktree"
)

type confirmModel struct {
	entry        state.Entry
	repoRoot     string
	config       config.Config
	deleteBranch bool
	err          string
}

type deleteDoneMsg struct{}

func newConfirmModel(entry state.Entry, repoRoot string, cfg config.Config) confirmModel {
	return confirmModel{
		entry:        entry,
		repoRoot:     repoRoot,
		config:       cfg,
		deleteBranch: cfg.ShouldDeleteBranch(),
	}
}

func (m confirmModel) Update(msg tea.Msg) (confirmModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "b":
			m.deleteBranch = !m.deleteBranch
			return m, nil

		case "y", "Y":
			// Run on_destroy hook before tearing down
			if err := config.RunHook(
				m.config.OnDestroy,
				m.entry.BranchShort,
				m.entry.Worktree.Path,
			); err != nil {
				m.err = fmt.Sprintf("on_destroy hook failed: %v", err)
				return m, nil
			}

			// Remove worktree first (before killing session)
			if err := worktree.Remove(m.repoRoot, m.entry.Worktree.Path); err != nil {
				m.err = fmt.Sprintf("Failed to remove worktree: %v", err)
				return m, nil
			}

			// Delete the git branch if toggled on
			if m.deleteBranch {
				if err := worktree.DeleteBranch(m.repoRoot, m.entry.BranchShort); err != nil {
					m.err = fmt.Sprintf("Worktree removed but branch delete failed: %v", err)
					return m, nil
				}
			}

			// Kill tmux session last — if we're inside it, switch away first
			if m.entry.HasSession {
				deletingSelf := false
				if current, err := tmux.CurrentSessionName(); err == nil && current == m.entry.SessionName {
					deletingSelf = true
					mainSession := tmux.SessionNameFor("main")
					// Ensure the main session exists with a sidebar
					if err := tmux.NewSession(mainSession, m.repoRoot); err != nil {
						// Session might already exist, that's fine
						_ = err
					}
					if layout := m.config.DefaultLayout(); layout != nil {
						_ = tmux.ApplyLayout(mainSession, layout)
					}
					_ = tmux.ApplyTheme(mainSession, m.config.Theme)
					_ = tmux.EnsureSidebar(mainSession, m.repoRoot)
					_ = tmux.SwitchClient(mainSession)
				}
				_ = tmux.KillSession(m.entry.SessionName)

				if deletingSelf {
					// We switched to main — this sidebar instance is gone.
					// Return quit so the process exits cleanly if still alive.
					return m, tea.Quit
				}
			}

			return m, func() tea.Msg { return deleteDoneMsg{} }

		case "n", "N", "esc":
			return m, nil
		}
	}
	return m, nil
}

func (m confirmModel) View() string {
	var b strings.Builder

	b.WriteString(errorStyle.Render("Delete Worktree"))
	b.WriteString("\n\n")

	b.WriteString(fmt.Sprintf("Branch:  %s\n", branchStyle.Render(m.entry.BranchShort)))
	b.WriteString(fmt.Sprintf("Path:    %s\n", pathStyle.Render(m.entry.Worktree.Path)))

	if m.entry.HasSession {
		b.WriteString(fmt.Sprintf("Session: %s\n", sessionActiveStyle.Render(m.entry.SessionName)))
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Foreground(colorWarning).Render("This will also kill the tmux session."))
		b.WriteString("\n")
	}

	if m.config.OnDestroy != "" {
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Foreground(colorSecondary).Render(
			fmt.Sprintf("Will run on_destroy: %s", m.config.OnDestroy)))
		b.WriteString("\n")
	}

	// Delete branch checkbox
	b.WriteString("\n")
	check := "○"
	checkStyle := lipgloss.NewStyle().Foreground(colorMuted)
	if m.deleteBranch {
		check = "●"
		checkStyle = lipgloss.NewStyle().Foreground(colorDanger)
	}
	b.WriteString(fmt.Sprintf("%s %s",
		checkStyle.Render(check),
		lipgloss.NewStyle().Foreground(colorText).Render("Also delete git branch"),
	))
	b.WriteString("\n")

	if m.err != "" {
		b.WriteString("\n")
		b.WriteString(errorStyle.Render(m.err))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  %s / %s / %s",
		lipgloss.NewStyle().Foreground(colorDanger).Bold(true).Render("[y]es"),
		lipgloss.NewStyle().Foreground(colorSuccess).Bold(true).Render("[n]o"),
		lipgloss.NewStyle().Foreground(colorSecondary).Render("[b] toggle branch delete"),
	))

	return dialogStyle.Render(b.String())
}
