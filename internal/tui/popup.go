package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/charles-albert-raymond/synco/internal/config"
	"github.com/charles-albert-raymond/synco/internal/metadata"
	"github.com/charles-albert-raymond/synco/internal/state"
)

// PopupCreateModel wraps createModel for standalone popup usage.
type PopupCreateModel struct {
	create   createModel
	repoRoot string
	config   config.Config
}

// NewPopupCreateModel creates a model for the create worktree popup.
func NewPopupCreateModel(repoRoot string, cfg config.Config) PopupCreateModel {
	return PopupCreateModel{
		create:   newCreateModel(repoRoot, cfg),
		repoRoot: repoRoot,
		config:   cfg,
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
	case createDoneMsg:
		if msg.title != "" {
			if store, err := metadata.Load(m.repoRoot, m.config.WorktreeDir); err == nil {
				store.SetTitle(msg.branch, msg.title)
				_ = store.Save(m.repoRoot, m.config.WorktreeDir)
			}
		}
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
	confirm confirmModel
}

// NewPopupConfirmModel creates a model for the delete confirmation popup.
func NewPopupConfirmModel(entry state.Entry, repoRoot string, cfg config.Config) PopupConfirmModel {
	return PopupConfirmModel{
		confirm: newConfirmModel(entry, repoRoot, cfg),
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
	case deleteDoneMsg:
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
	edit     editTitleModel
	repoRoot string
	config   config.Config
}

// NewPopupEditTitleModel creates a model for the edit title popup.
func NewPopupEditTitleModel(branch, currentTitle, repoRoot string, cfg config.Config) PopupEditTitleModel {
	return PopupEditTitleModel{
		edit:     newEditTitleModel(branch, currentTitle, repoRoot, cfg),
		repoRoot: repoRoot,
		config:   cfg,
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
	case editTitleDoneMsg:
		if store, err := metadata.Load(m.repoRoot, m.config.WorktreeDir); err == nil {
			if msg.title == "" {
				store.Delete(msg.branch)
			} else {
				store.SetTitle(msg.branch, msg.title)
			}
			_ = store.Save(m.repoRoot, m.config.WorktreeDir)
		}
		return m, tea.Quit
	}

	var cmd tea.Cmd
	m.edit, cmd = m.edit.Update(msg)
	return m, cmd
}

func (m PopupEditTitleModel) View() string {
	return m.edit.View()
}
