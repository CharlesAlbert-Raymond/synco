package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/charles-albert-raymond/synco/internal/config"
	"github.com/charles-albert-raymond/synco/internal/state"
)

// PopupCreateModel wraps createModel for standalone popup usage.
type PopupCreateModel struct {
	create     createModel
	repoRoot   string
	config     config.Config
	intentFile string
}

// NewPopupCreateModel creates a model for the create worktree popup.
func NewPopupCreateModel(repoRoot string, cfg config.Config, intentFile string) PopupCreateModel {
	return PopupCreateModel{
		create:     newCreateModel(repoRoot, cfg),
		repoRoot:   repoRoot,
		config:     cfg,
		intentFile: intentFile,
	}
}

func (m PopupCreateModel) Init() tea.Cmd {
	return m.create.branchInput.Focus()
}

func (m PopupCreateModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		}
	case createIntentMsg:
		_ = writePopupIntent(m.intentFile, popupIntentEnvelope{Kind: "create", Create: &msg})
		return m, tea.Quit
	}

	var cmd tea.Cmd
	m.create, cmd = m.create.Update(msg)
	return m, cmd
}

func (m PopupCreateModel) View() string {
	return m.create.View()
}

// PopupConfirmModel wraps confirmModel for standalone popup usage.
type PopupConfirmModel struct {
	confirm    confirmModel
	intentFile string
}

// NewPopupConfirmModel creates a model for the delete confirmation popup.
func NewPopupConfirmModel(entry state.Entry, repoRoot string, cfg config.Config, intentFile string) PopupConfirmModel {
	return PopupConfirmModel{
		confirm:    newConfirmModel(entry, repoRoot, cfg),
		intentFile: intentFile,
	}
}

func (m PopupConfirmModel) Init() tea.Cmd {
	return nil
}

func (m PopupConfirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc", "n", "N":
			return m, tea.Quit
		}
	case deleteIntentMsg:
		_ = writePopupIntent(m.intentFile, popupIntentEnvelope{Kind: "delete", Delete: &msg})
		return m, tea.Quit
	}

	var cmd tea.Cmd
	m.confirm, cmd = m.confirm.Update(msg)
	return m, cmd
}

func (m PopupConfirmModel) View() string {
	return m.confirm.View()
}

// PopupEditTitleModel wraps editTitleModel for standalone popup usage.
type PopupEditTitleModel struct {
	edit       editTitleModel
	repoRoot   string
	config     config.Config
	intentFile string
}

// NewPopupEditTitleModel creates a model for the edit title popup.
func NewPopupEditTitleModel(branch, currentTitle, repoRoot string, cfg config.Config, intentFile string) PopupEditTitleModel {
	return PopupEditTitleModel{
		edit:       newEditTitleModel(branch, currentTitle, repoRoot, cfg),
		repoRoot:   repoRoot,
		config:     cfg,
		intentFile: intentFile,
	}
}

func (m PopupEditTitleModel) Init() tea.Cmd {
	return m.edit.input.Focus()
}

func (m PopupEditTitleModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		}
	case editTitleIntentMsg:
		_ = writePopupIntent(m.intentFile, popupIntentEnvelope{Kind: "edit_title", EditTitle: &msg})
		return m, tea.Quit
	}

	var cmd tea.Cmd
	m.edit, cmd = m.edit.Update(msg)
	return m, cmd
}

func (m PopupEditTitleModel) View() string {
	return m.edit.View()
}
