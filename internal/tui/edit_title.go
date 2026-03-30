package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/charles-albert-raymond/synco/internal/config"
)

type editTitleDoneMsg struct {
	branch string
	title  string
}

type editTitleModel struct {
	branch   string
	input    textinput.Model
	repoRoot string
	config   config.Config
}

func newEditTitleModel(branch, currentTitle, repoRoot string, cfg config.Config) editTitleModel {
	ti := textinput.New()
	ti.Placeholder = "short description"
	ti.SetValue(currentTitle)
	ti.Focus()
	ti.CharLimit = 60
	ti.Width = 40
	ti.PromptStyle = inputLabelStyle
	ti.TextStyle = lipgloss.NewStyle().Foreground(colorText)

	return editTitleModel{
		branch:   branch,
		input:    ti,
		repoRoot: repoRoot,
		config:   cfg,
	}
}

func (m editTitleModel) Update(msg tea.Msg) (editTitleModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, nil // handled by parent
		case "enter":
			title := strings.TrimSpace(m.input.Value())
			branch := m.branch
			return m, func() tea.Msg {
				return editTitleDoneMsg{branch: branch, title: title}
			}
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m editTitleModel) View() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Edit Title"))
	b.WriteString("\n\n")
	b.WriteString(inputLabelStyle.Render("Branch: "))
	b.WriteString(branchStyle.Render(m.branch))
	b.WriteString("\n\n")
	b.WriteString(inputLabelStyle.Render("Title:"))
	b.WriteString("\n")
	b.WriteString(m.input.View())
	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render(" enter save • esc cancel • clear to remove title"))
	return borderStyle.Render(b.String())
}
