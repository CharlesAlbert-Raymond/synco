package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/charles-albert-raymond/synco/internal/config"
	"github.com/charles-albert-raymond/synco/internal/state"
)

type confirmModel struct {
	entry        state.Entry
	repoRoot     string
	config       config.Config
	deleteBranch bool
	err          string
}

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
			intent := deleteIntentMsg{Entry: m.entry, DeleteBranch: m.deleteBranch}
			return m, func() tea.Msg { return intent }

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
