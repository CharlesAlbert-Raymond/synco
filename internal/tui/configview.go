package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/charles-albert-raymond/synco/internal/config"
	"github.com/charles-albert-raymond/synco/internal/notify"
)

type configViewModel struct {
	config   config.Config
	repoRoot string
}

func newConfigViewModel(cfg config.Config, repoRoot string) configViewModel {
	return configViewModel{config: cfg, repoRoot: repoRoot}
}

func (m configViewModel) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Configuration"))
	b.WriteString("\n\n")

	// Sources
	headerStyle := lipgloss.NewStyle().Foreground(colorSecondary).Bold(true)
	labelStyle := lipgloss.NewStyle().Foreground(colorMuted).Width(16)
	valueStyle := lipgloss.NewStyle().Foreground(colorText)
	emptyStyle := lipgloss.NewStyle().Foreground(colorMuted).Italic(true)

	b.WriteString(headerStyle.Render("Sources"))
	b.WriteString("\n")
	b.WriteString(labelStyle.Render("  Global:"))
	b.WriteString(valueStyle.Render(config.GlobalConfigPath()))
	b.WriteString("\n")
	b.WriteString(labelStyle.Render("  Local:"))
	b.WriteString(valueStyle.Render(filepath.Join(m.repoRoot, ".synco.yaml")))
	b.WriteString("\n\n")

	// Resolved config
	b.WriteString(headerStyle.Render("Resolved Config"))
	b.WriteString("\n")

	renderField := func(label, value string) {
		b.WriteString(labelStyle.Render("  " + label + ":"))
		if value == "" {
			b.WriteString(emptyStyle.Render("(not set)"))
		} else {
			b.WriteString(valueStyle.Render(value))
		}
		b.WriteString("\n")
	}

	renderField("worktree_dir", m.config.WorktreeDir)
	absDir := m.config.WorktreeDir
	if !filepath.IsAbs(absDir) {
		absDir = filepath.Join(m.repoRoot, absDir)
	}
	b.WriteString(labelStyle.Render(""))
	b.WriteString(pathStyle.Render(fmt.Sprintf("→ %s", absDir)))
	b.WriteString("\n")

	renderField("on_create", m.config.OnCreate)
	renderField("on_destroy", m.config.OnDestroy)
	renderField("default_creation_profile", m.config.DefaultCreationProfile)

	deleteBranchVal := "false"
	if m.config.ShouldDeleteBranch() {
		deleteBranchVal = "true"
	}
	renderField("auto_delete_branch", deleteBranchVal)

	if len(m.config.Aliases) > 0 {
		b.WriteString("\n")
		b.WriteString(headerStyle.Render("Aliases"))
		b.WriteString("\n")
		for branch, alias := range m.config.Aliases {
			b.WriteString(labelStyle.Render("  " + branch + ":"))
			b.WriteString(valueStyle.Render(alias))
			b.WriteString("\n")
		}
	}

	if len(m.config.Scripts) > 0 {
		b.WriteString("\n")
		b.WriteString(headerStyle.Render("Scripts"))
		b.WriteString("\n")
		for name, script := range m.config.Scripts {
			b.WriteString(labelStyle.Render("  " + name + ":"))
			b.WriteString(valueStyle.Render(script.Path))
			b.WriteString("\n")
		}
	}

	if len(m.config.CreationProfiles) > 0 {
		b.WriteString("\n")
		b.WriteString(headerStyle.Render("Creation Profiles"))
		b.WriteString("\n")
		for name, profile := range m.config.CreationProfiles {
			detail := fmt.Sprintf("session=%t, on_create=%t", profile.ShouldCreateSession(), profile.ShouldRunOnCreate())
			if profile.Bootstrap != "" {
				detail += ", bootstrap=true"
			}
			if scriptDetail := lifecycleScriptDetail(profile); scriptDetail != "" {
				detail += ", " + scriptDetail
			}
			b.WriteString(labelStyle.Render("  " + name + ":"))
			b.WriteString(valueStyle.Render(detail))
			b.WriteString("\n")
		}
	}

	// Theme
	if m.config.Theme != nil {
		b.WriteString("\n")
		b.WriteString(headerStyle.Render("Theme"))
		b.WriteString("\n")
		renderField("pane_border", m.config.Theme.PaneBorder)
		renderField("pane_border_active", m.config.Theme.PaneBorderActive)
		if m.config.Theme.PaneBorderLabels {
			renderField("pane_border_labels", "true")
		}
	}

	// Layouts
	if len(m.config.Layouts) > 0 {
		b.WriteString("\n")
		b.WriteString(headerStyle.Render("Layouts"))
		b.WriteString("\n")
		for name, layout := range m.config.Layouts {
			b.WriteString(labelStyle.Render("  " + name + ":"))
			b.WriteString(valueStyle.Render(fmt.Sprintf("%d pane(s)", len(layout.Panes))))
			b.WriteString("\n")
			for i, pane := range layout.Panes {
				split := pane.Split
				if split == "" {
					split = "horizontal"
				}
				size := pane.Size
				if size == "" {
					size = "auto"
				}
				detail := fmt.Sprintf("    [%d] %s (%s, %s)", i, pane.Command, split, size)
				b.WriteString(emptyStyle.Render(detail))
				b.WriteString("\n")
			}
		}
	}

	// Notifications
	b.WriteString("\n")
	b.WriteString(headerStyle.Render("Notifications"))
	b.WriteString("\n")
	renderField("enabled", fmt.Sprintf("%v", m.config.NotificationsEnabled()))
	if m.config.NotificationsEnabled() {
		renderField("bell", fmt.Sprintf("%v", m.config.BellEnabled()))
		renderField("system_notif", fmt.Sprintf("%v", m.config.SystemNotificationEnabled()))
		renderField("sound", m.config.NotificationSound())
		if m.config.Notifications != nil && m.config.Notifications.OnSilence != "" {
			renderField("on_done", m.config.Notifications.OnSilence)
		}
	}
	hookStatus := "not configured"
	if notify.IsHookConfigured() {
		hookStatus = "active"
	}
	renderField("claude hook", hookStatus)
	if hookStatus == "not configured" {
		b.WriteString(emptyStyle.Render("    run: synco setup-hooks"))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render(" ? close • esc close"))

	return borderStyle.Render(b.String())
}

func lifecycleScriptDetail(profile config.CreationProfile) string {
	var parts []string
	if len(profile.Scripts.BeforeCreate) > 0 {
		parts = append(parts, fmt.Sprintf("before_create=%d", len(profile.Scripts.BeforeCreate)))
	}
	if len(profile.Scripts.AfterCreate) > 0 {
		parts = append(parts, fmt.Sprintf("after_create=%d", len(profile.Scripts.AfterCreate)))
	}
	if len(profile.Scripts.BeforeDestroy) > 0 {
		parts = append(parts, fmt.Sprintf("before_destroy=%d", len(profile.Scripts.BeforeDestroy)))
	}
	if len(profile.Scripts.AfterDestroy) > 0 {
		parts = append(parts, fmt.Sprintf("after_destroy=%d", len(profile.Scripts.AfterDestroy)))
	}
	return strings.Join(parts, ", ")
}
