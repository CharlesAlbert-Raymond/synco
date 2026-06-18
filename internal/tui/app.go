package tui

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/charles-albert-raymond/synco/internal/config"
	"github.com/charles-albert-raymond/synco/internal/metadata"
	"github.com/charles-albert-raymond/synco/internal/orchestrate"
	"github.com/charles-albert-raymond/synco/internal/session"
	"github.com/charles-albert-raymond/synco/internal/state"
	"github.com/charles-albert-raymond/synco/internal/tmux"
)

const refreshInterval = 2 * time.Second

type view int

const (
	viewList view = iota
	viewCreate
	viewConfirmDelete
	viewConfig
	viewEditTitle
)

type errMsg struct{ error }
type tickMsg time.Time
type popupDoneMsg struct {
	intent popupIntentEnvelope
	ok     bool
	err    error
}
type rebuildDoneMsg struct{ err error }

type jobKind string

const (
	jobCreating jobKind = "creating"
	jobDeleting jobKind = "deleting"
)

type jobDoneMsg struct {
	branch string
	kind   jobKind
	title  string
	err    error
}

type Model struct {
	currentView      view
	list             listModel
	create           createModel
	confirm          confirmModel
	configView       configViewModel
	editTitle        editTitleModel
	repoRoot         string
	config           config.Config
	width            int
	height           int
	err              error
	sidebarMode      bool
	sourceDir        string // synco source dir for rebuilding (set via ldflags)
	jobs             map[string]jobKind
	RestartRequested bool // signals main() to re-exec after quit
}

// projectNameFromConfig returns the config's project_name, or empty for auto-derive.
func (m Model) projectNameFromConfig() string {
	return m.config.ProjectName
}

func NewModel(repoRoot string, cfg config.Config, sourceDir string) Model {
	jobs := make(map[string]jobKind)
	lm := newListModel()
	lm.config = cfg
	lm.projectName = cfg.ProjectName
	lm.jobs = jobs
	return Model{
		currentView: viewList,
		list:        lm,
		repoRoot:    repoRoot,
		config:      cfg,
		sourceDir:   sourceDir,
		jobs:        jobs,
	}
}

func NewSidebarModel(repoRoot string, cfg config.Config, sourceDir string) Model {
	jobs := make(map[string]jobKind)
	lm := newListModel()
	lm.config = cfg
	lm.projectName = cfg.ProjectName
	lm.jobs = jobs
	return Model{
		currentView: viewList,
		list:        lm,
		repoRoot:    repoRoot,
		config:      cfg,
		sidebarMode: true,
		sourceDir:   sourceDir,
		jobs:        jobs,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(fetchEntries(m.repoRoot, m.projectNameFromConfig()), tickCmd())
}

func tickCmd() tea.Cmd {
	return tea.Tick(refreshInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.width = msg.Width
		m.list.height = msg.Height
		return m, nil

	case tea.FocusMsg:
		if m.sidebarMode {
			m.list.resetCursorOnNext = true
		}
		return m, fetchEntries(m.repoRoot, m.projectNameFromConfig())

	case tea.BlurMsg:
		if m.sidebarMode {
			// When sidebar loses focus, reset cursor to the current worktree
			m.list.resetCursorToCurrent()
			return m, unfocusSidebar()
		}
		return m, nil

	case popupDoneMsg:
		if msg.err != nil {
			m.list.message = fmt.Sprintf("Popup failed: %v", msg.err)
			m.list.msgStyle = errorStyle
			return m, fetchEntries(m.repoRoot, m.projectNameFromConfig())
		}
		if !msg.ok {
			return m, fetchEntries(m.repoRoot, m.projectNameFromConfig())
		}
		return m.handlePopupIntent(msg.intent)

	case jobDoneMsg:
		delete(m.jobs, msg.branch)
		m.list.jobs = m.jobs
		if msg.err != nil {
			m.list.message = fmt.Sprintf("%s %s failed: %v", msg.kind, msg.branch, msg.err)
			m.list.msgStyle = errorStyle
			return m, fetchEntries(m.repoRoot, m.projectNameFromConfig())
		}
		if msg.kind == jobCreating && msg.title != "" {
			m.saveTitle(msg.branch, msg.title)
		}
		if msg.kind == jobDeleting {
			m.deleteTitle(msg.branch)
		}
		m.list.message = fmt.Sprintf("%s complete: %s", msg.kind, msg.branch)
		m.list.msgStyle = successStyle
		return m, fetchEntries(m.repoRoot, m.projectNameFromConfig())

	case tickMsg:
		// Periodic refresh only on the list view (don't interrupt forms)
		if m.currentView == viewList {
			return m, tea.Batch(fetchEntries(m.repoRoot, m.projectNameFromConfig()), tickCmd())
		}
		return m, tickCmd()

	case rebuildDoneMsg:
		if msg.err != nil {
			m.list.message = fmt.Sprintf("Rebuild failed: %v", msg.err)
			m.list.msgStyle = errorStyle
			m.currentView = viewList
			return m, nil
		}
		m.RestartRequested = true
		return m, tea.Quit

	case errMsg:
		m.err = msg.error
		m.list.message = msg.Error()
		m.list.msgStyle = errorStyle
		m.currentView = viewList
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "ctrl+r":
			if !m.list.filtering {
				return m, m.rebuildCmd()
			}
		case "q":
			if m.currentView == viewList && !m.list.filtering {
				return m, tea.Quit
			}
		}
	}

	switch m.currentView {
	case viewList:
		return m.updateList(msg)
	case viewCreate:
		return m.updateCreate(msg)
	case viewConfirmDelete:
		return m.updateConfirm(msg)
	case viewConfig:
		return m.updateConfig(msg)
	case viewEditTitle:
		return m.updateEditTitle(msg)
	}

	return m, nil
}

func (m Model) updateList(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// When filtering, only handle esc here; let list model handle the rest
		if m.list.filtering {
			if msg.String() == "esc" {
				m.list.exitFilter()
				return m, nil
			}
			// Fall through to list Update/UpdateSidebar which handles filter input
			break
		}

		switch msg.String() {
		case "c":
			if m.sidebarMode {
				return m, launchCreatePopup(m.repoRoot)
			}
			m.currentView = viewCreate
			m.create = newCreateModel(m.repoRoot, m.config)
			return m, m.create.branchInput.Focus()
		case "e":
			entry, ok := m.list.selectedEntry()
			if ok {
				if m.isJobActive(entry.BranchShort) {
					m.blockedByJob(entry.BranchShort)
					return m, nil
				}
				if m.sidebarMode {
					return m, launchEditTitlePopup(m.repoRoot, entry.BranchShort, entry.Title)
				}
				m.currentView = viewEditTitle
				m.editTitle = newEditTitleModel(entry.BranchShort, entry.Title, m.repoRoot, m.config)
				return m, textinput.Blink
			}
		case "d":
			if len(m.list.entries) > 0 {
				entry, ok := m.list.selectedEntry()
				if !ok {
					return m, nil
				}
				if entry.Worktree.IsMain {
					m.list.message = "Cannot delete the main worktree"
					m.list.msgStyle = errorStyle
					return m, nil
				}
				if m.isJobActive(entry.BranchShort) {
					m.blockedByJob(entry.BranchShort)
					return m, nil
				}
				if m.sidebarMode {
					return m, launchDeletePopup(m.repoRoot, entry.BranchShort)
				}
				m.currentView = viewConfirmDelete
				m.confirm = newConfirmModel(entry, m.repoRoot, m.config)
				return m, nil
			}
		case "r":
			return m, fetchEntries(m.repoRoot, m.projectNameFromConfig())
		case "?":
			m.currentView = viewConfig
			m.configView = newConfigViewModel(m.config, m.repoRoot)
			return m, nil
		case "esc":
			if m.sidebarMode {
				return m, unfocusSidebar()
			}
		}
	}

	if m.sidebarMode {
		var cmd tea.Cmd
		m.list, cmd = m.list.UpdateSidebar(msg, m.repoRoot)
		return m, cmd
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg, m.repoRoot)
	return m, cmd
}

func (m Model) updateCreate(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "esc" {
			m.currentView = viewList
			return m, nil
		}
	case createIntentMsg:
		m.currentView = viewList
		return m.startCreateJob(msg)
	}

	var cmd tea.Cmd
	m.create, cmd = m.create.Update(msg)
	return m, cmd
}

func (m Model) updateConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "esc" || msg.String() == "n" || msg.String() == "N" {
			m.currentView = viewList
			return m, nil
		}
	case deleteIntentMsg:
		m.currentView = viewList
		return m.startDeleteJob(msg)
	}

	var cmd tea.Cmd
	m.confirm, cmd = m.confirm.Update(msg)
	return m, cmd
}

func (m Model) updateConfig(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		if msg.String() == "?" || msg.String() == "esc" {
			m.currentView = viewList
			return m, nil
		}
	}
	return m, nil
}

func (m Model) updateEditTitle(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "esc" {
			m.currentView = viewList
			return m, nil
		}
	case editTitleIntentMsg:
		m.currentView = viewList
		m.saveTitle(msg.Branch, msg.Title)
		m.list.message = "Title updated"
		m.list.msgStyle = successStyle
		return m, fetchEntries(m.repoRoot, m.projectNameFromConfig())
	}

	var cmd tea.Cmd
	m.editTitle, cmd = m.editTitle.Update(msg)
	return m, cmd
}

func (m Model) handlePopupIntent(intent popupIntentEnvelope) (tea.Model, tea.Cmd) {
	switch intent.Kind {
	case "create":
		if intent.Create == nil {
			return m, nil
		}
		return m.startCreateJob(*intent.Create)
	case "delete":
		if intent.Delete == nil {
			return m, nil
		}
		return m.startDeleteJob(*intent.Delete)
	case "edit_title":
		if intent.EditTitle == nil {
			return m, nil
		}
		m.saveTitle(intent.EditTitle.Branch, intent.EditTitle.Title)
		m.list.message = "Title updated"
		m.list.msgStyle = successStyle
		return m, fetchEntries(m.repoRoot, m.projectNameFromConfig())
	default:
		return m, fetchEntries(m.repoRoot, m.projectNameFromConfig())
	}
}

func (m Model) startCreateJob(intent createIntentMsg) (tea.Model, tea.Cmd) {
	if m.isJobActive(intent.Branch) {
		m.blockedByJob(intent.Branch)
		return m, nil
	}
	m.jobs[intent.Branch] = jobCreating
	m.list.jobs = m.jobs
	m.list.message = fmt.Sprintf("Creating %s...", intent.Branch)
	m.list.msgStyle = subtitleStyle
	return m, tea.Batch(fetchEntries(m.repoRoot, m.projectNameFromConfig()), m.createJobCmd(intent))
}

func (m Model) createJobCmd(intent createIntentMsg) tea.Cmd {
	return func() tea.Msg {
		opts := orchestrate.CreateWorktreeOpts{CreationProfile: intent.CreationProfile}
		var err error
		if intent.UseSource {
			_, _, err = orchestrate.CreateWorktreeFromExisting(m.repoRoot, m.config, intent.Source, opts)
		} else {
			_, _, err = orchestrate.CreateWorktree(m.repoRoot, m.config, intent.Branch, intent.Base, opts)
		}
		return jobDoneMsg{branch: intent.Branch, kind: jobCreating, title: intent.Title, err: err}
	}
}

func (m Model) startDeleteJob(intent deleteIntentMsg) (tea.Model, tea.Cmd) {
	branch := intent.Entry.BranchShort
	if m.isJobActive(branch) {
		m.blockedByJob(branch)
		return m, nil
	}
	if err := m.switchAwayIfDeletingSelf(intent.Entry); err != nil {
		m.list.message = fmt.Sprintf("Cannot switch away before delete: %v", err)
		m.list.msgStyle = errorStyle
		return m, nil
	}
	m.jobs[branch] = jobDeleting
	m.list.jobs = m.jobs
	m.list.message = fmt.Sprintf("Deleting %s...", branch)
	m.list.msgStyle = subtitleStyle
	return m, tea.Batch(fetchEntries(m.repoRoot, m.projectNameFromConfig()), m.deleteJobCmd(intent))
}

func (m Model) deleteJobCmd(intent deleteIntentMsg) tea.Cmd {
	return func() tea.Msg {
		opts := orchestrate.DeleteWorktreeOpts{DeleteBranch: intent.DeleteBranch}
		err := orchestrate.DeleteWorktree(m.repoRoot, m.config, intent.Entry, opts)
		return jobDoneMsg{branch: intent.Entry.BranchShort, kind: jobDeleting, err: err}
	}
}

func (m Model) switchAwayIfDeletingSelf(entry state.Entry) error {
	if !entry.HasSession {
		return nil
	}
	current, err := tmux.CurrentSessionName()
	if err != nil || current != entry.SessionName {
		return nil
	}
	project := session.ResolveProjectName(m.repoRoot, m.config.ProjectName)
	mainSession := session.SessionNameFor(project, session.RootKey)
	if err := tmux.NewSession(mainSession, m.repoRoot); err != nil {
		return err
	}
	if err := tmux.EnsureSidebar(mainSession, m.repoRoot, m.config.SidebarWidth); err != nil {
		return err
	}
	return tmux.SwitchClient(mainSession)
}

func (m Model) isJobActive(branch string) bool {
	_, ok := m.jobs[branch]
	return ok
}

func (m *Model) blockedByJob(branch string) {
	kind := m.jobs[branch]
	m.list.message = fmt.Sprintf("%s already %s...", branch, kind)
	m.list.msgStyle = errorStyle
}

func (m Model) saveTitle(branch, title string) {
	store, err := metadata.Load(m.repoRoot, m.config.WorktreeDir)
	if err != nil {
		return
	}
	if title == "" {
		store.Delete(branch)
	} else {
		store.SetTitle(branch, title)
	}
	_ = store.Save(m.repoRoot, m.config.WorktreeDir)
}

func (m Model) deleteTitle(branch string) {
	store, err := metadata.Load(m.repoRoot, m.config.WorktreeDir)
	if err != nil {
		return
	}
	store.Delete(branch)
	_ = store.Save(m.repoRoot, m.config.WorktreeDir)
}

func (m Model) View() string {
	var content string

	if m.sidebarMode {
		switch m.currentView {
		case viewList:
			content = m.list.ViewCompact(m.width, m.height)
		case viewCreate:
			content = m.list.ViewCompact(m.width, m.height) + "\n" + m.create.View()
		case viewConfirmDelete:
			content = m.list.ViewCompact(m.width, m.height) + "\n" + m.confirm.View()
		case viewEditTitle:
			content = m.list.ViewCompact(m.width, m.height) + "\n" + m.editTitle.View()
		case viewConfig:
			content = m.configView.View()
		}
	} else {
		switch m.currentView {
		case viewList:
			content = m.list.View()
		case viewCreate:
			content = m.list.View() + "\n\n" + m.create.View()
		case viewConfirmDelete:
			content = m.list.View() + "\n\n" + m.confirm.View()
		case viewEditTitle:
			content = m.list.View() + "\n\n" + m.editTitle.View()
		case viewConfig:
			content = m.configView.View()
		}
	}

	return renderFrame(content, m.width, m.height)
}

func launchCreatePopup(repoRoot string) tea.Cmd {
	return func() tea.Msg {
		intentFile, cleanup, err := popupIntentFile()
		if err != nil {
			return popupDoneMsg{err: err}
		}
		defer cleanup()
		if err := tmux.LaunchPopup(
			[]string{"--popup-create", "--root", repoRoot, "--intent-file", intentFile},
			70, 28, "Create Worktree",
		); err != nil {
			return popupDoneMsg{err: err}
		}
		intent, ok, err := readPopupIntent(intentFile)
		return popupDoneMsg{intent: intent, ok: ok, err: err}
	}
}

func launchEditTitlePopup(repoRoot string, branch, currentTitle string) tea.Cmd {
	return func() tea.Msg {
		intentFile, cleanup, err := popupIntentFile()
		if err != nil {
			return popupDoneMsg{err: err}
		}
		defer cleanup()
		args := []string{"--popup-edit-title", "--root", repoRoot, "--branch", branch, "--intent-file", intentFile}
		if currentTitle != "" {
			args = append(args, "--title", currentTitle)
		}
		if err := tmux.LaunchPopup(args, 60, 14, "Edit Title"); err != nil {
			return popupDoneMsg{err: err}
		}
		intent, ok, err := readPopupIntent(intentFile)
		return popupDoneMsg{intent: intent, ok: ok, err: err}
	}
}

func launchDeletePopup(repoRoot string, branch string) tea.Cmd {
	return func() tea.Msg {
		intentFile, cleanup, err := popupIntentFile()
		if err != nil {
			return popupDoneMsg{err: err}
		}
		defer cleanup()
		if err := tmux.LaunchPopup(
			[]string{"--popup-delete", "--root", repoRoot, "--branch", branch, "--intent-file", intentFile},
			60, 20, "Delete Worktree",
		); err != nil {
			return popupDoneMsg{err: err}
		}
		intent, ok, err := readPopupIntent(intentFile)
		return popupDoneMsg{intent: intent, ok: ok, err: err}
	}
}

func popupIntentFile() (string, func(), error) {
	f, err := os.CreateTemp("", "synco-popup-intent-*.json")
	if err != nil {
		return "", func() {}, err
	}
	path := f.Name()
	if err := f.Close(); err != nil {
		_ = os.Remove(path)
		return "", func() {}, err
	}
	cleanup := func() { _ = os.Remove(path) }
	return path, cleanup, nil
}

// rebuildCmd rebuilds the synco binary from source and signals a restart.
// If no source directory is available, it reloads config only and re-execs.
func (m Model) rebuildCmd() tea.Cmd {
	return func() tea.Msg {
		if m.sourceDir != "" {
			exe, err := os.Executable()
			if err != nil {
				return rebuildDoneMsg{err: fmt.Errorf("find executable: %w", err)}
			}
			cmd := exec.Command("go", "build", "-o", exe, ".")
			cmd.Dir = m.sourceDir
			if out, err := cmd.CombinedOutput(); err != nil {
				return rebuildDoneMsg{err: fmt.Errorf("%s: %w", string(out), err)}
			}
		}
		// Even without sourceDir, signal restart to reload config + pick up
		// any externally updated binary.
		return rebuildDoneMsg{}
	}
}

func unfocusSidebar() tea.Cmd {
	return func() tea.Msg {
		session, err := tmux.CurrentSessionName()
		if err == nil {
			tmux.FocusMainPane(session)
		}
		return nil
	}
}
