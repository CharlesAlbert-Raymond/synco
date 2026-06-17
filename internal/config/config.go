package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Pane defines a single pane in a layout.
type Pane struct {
	Command string `yaml:"command"`
	Split   string `yaml:"split,omitempty"` // "horizontal" or "vertical"
	Size    string `yaml:"size,omitempty"`  // e.g. "30%"
}

// Layout defines a named window layout with multiple panes.
type Layout struct {
	Panes []Pane `yaml:"panes"`
}

// Theme holds tmux border color configuration.
type Theme struct {
	PaneBorder       string `yaml:"pane_border,omitempty"`
	PaneBorderActive string `yaml:"pane_border_active,omitempty"`
	PaneBorderLabels bool   `yaml:"pane_border_labels,omitempty"`
}

// Notifications holds notification preferences.
type Notifications struct {
	Enabled            *bool  `yaml:"enabled,omitempty"`
	SilenceSeconds     int    `yaml:"silence_seconds,omitempty"`
	Bell               *bool  `yaml:"bell,omitempty"`
	SystemNotification *bool  `yaml:"system_notification,omitempty"`
	Sound              string `yaml:"sound,omitempty"`
	OnSilence          string `yaml:"on_silence,omitempty"`
}

// ProjectDef defines a multi-repo project group in the global config.
type ProjectDef struct {
	Repos []string `yaml:"repos"`
}

// ScriptDef defines a named bash script that profiles can reference.
type ScriptDef struct {
	Path string `yaml:"path"`
}

// LifecycleScripts defines ordered script names for profile lifecycle events.
type LifecycleScripts struct {
	BeforeCreate  []string `yaml:"before_create,omitempty"`
	AfterCreate   []string `yaml:"after_create,omitempty"`
	BeforeDestroy []string `yaml:"before_destroy,omitempty"`
	AfterDestroy  []string `yaml:"after_destroy,omitempty"`
}

// CreationProfile controls which lifecycle steps run when creating a worktree.
type CreationProfile struct {
	CreateSession *bool            `yaml:"create_session,omitempty"`
	RunOnCreate   *bool            `yaml:"run_on_create,omitempty"`
	Bootstrap     string           `yaml:"bootstrap,omitempty"`
	Scripts       LifecycleScripts `yaml:"scripts,omitempty"`
}

const BuiltInCreationProfileDev = "dev"

// Config holds the merged synco configuration.
type Config struct {
	WorktreeDir            string                     `yaml:"worktree_dir"`
	SidebarWidth           string                     `yaml:"sidebar_width,omitempty"`
	ProjectName            string                     `yaml:"project_name,omitempty"`
	OnCreate               string                     `yaml:"on_create"`
	OnDestroy              string                     `yaml:"on_destroy"`
	AutoDeleteBranch       *bool                      `yaml:"auto_delete_branch,omitempty"`
	Aliases                map[string]string          `yaml:"aliases,omitempty"`
	Scripts                map[string]ScriptDef       `yaml:"scripts,omitempty"`
	DefaultCreationProfile string                     `yaml:"default_creation_profile,omitempty"`
	CreationProfiles       map[string]CreationProfile `yaml:"creation_profiles,omitempty"`
	Theme                  *Theme                     `yaml:"theme,omitempty"`
	Layouts                map[string]Layout          `yaml:"layouts,omitempty"`
	Notifications          *Notifications             `yaml:"notifications,omitempty"`
	Projects               map[string]ProjectDef      `yaml:"projects,omitempty"`
}

func boolPtr(v bool) *bool {
	return &v
}

// DefaultCreationProfile returns the built-in development creation behavior.
func DefaultCreationProfile() CreationProfile {
	return CreationProfile{CreateSession: boolPtr(true), RunOnCreate: boolPtr(true)}
}

// ShouldCreateSession reports whether this profile creates a tmux session.
func (p CreationProfile) ShouldCreateSession() bool {
	if p.CreateSession == nil {
		return true
	}
	return *p.CreateSession
}

// ShouldRunOnCreate reports whether this profile runs the configured on_create hook.
func (p CreationProfile) ShouldRunOnCreate() bool {
	if p.RunOnCreate == nil {
		return true
	}
	return *p.RunOnCreate
}

// ResolveCreationProfile resolves a requested profile name to concrete lifecycle behavior.
func (c Config) ResolveCreationProfile(name string) (CreationProfile, string, error) {
	profileName := strings.TrimSpace(name)
	if profileName == "" {
		profileName = strings.TrimSpace(c.DefaultCreationProfile)
	}
	if profileName == "" {
		return DefaultCreationProfile(), BuiltInCreationProfileDev, nil
	}
	if profile, ok := c.CreationProfiles[profileName]; ok {
		return profile, profileName, nil
	}
	if profileName == BuiltInCreationProfileDev {
		return DefaultCreationProfile(), profileName, nil
	}
	return CreationProfile{}, "", fmt.Errorf("unknown creation profile %q", profileName)
}

// CreationProfileNames returns selectable profile names, including the built-in dev profile.
func (c Config) CreationProfileNames() []string {
	names := make([]string, 0, len(c.CreationProfiles)+1)
	seen := map[string]bool{BuiltInCreationProfileDev: true}
	names = append(names, BuiltInCreationProfileDev)
	for name := range c.CreationProfiles {
		if seen[name] {
			continue
		}
		seen[name] = true
		names = append(names, name)
	}
	defaultName := strings.TrimSpace(c.DefaultCreationProfile)
	if defaultName != "" && !seen[defaultName] {
		names = append(names, defaultName)
	}
	sort.Strings(names)
	return names
}

// DefaultLayout returns the "default" layout, or nil if none is configured.
func (c Config) DefaultLayout() *Layout {
	if c.Layouts == nil {
		return nil
	}
	l, ok := c.Layouts["default"]
	if !ok {
		return nil
	}
	return &l
}

// NotificationsEnabled returns true when notifications are configured and not explicitly disabled.
func (c Config) NotificationsEnabled() bool {
	if c.Notifications == nil {
		return false
	}
	if c.Notifications.Enabled != nil {
		return *c.Notifications.Enabled
	}
	return true // enabled by default when section exists
}

// SilenceThreshold returns the configured silence seconds, or 10 as default.
func (c Config) SilenceThreshold() int {
	if c.Notifications != nil && c.Notifications.SilenceSeconds > 0 {
		return c.Notifications.SilenceSeconds
	}
	return 10
}

// BellEnabled returns whether the terminal bell should fire on silence.
func (c Config) BellEnabled() bool {
	if c.Notifications != nil && c.Notifications.Bell != nil {
		return *c.Notifications.Bell
	}
	return true
}

// SystemNotificationEnabled returns whether macOS system notifications should fire.
func (c Config) SystemNotificationEnabled() bool {
	if c.Notifications != nil && c.Notifications.SystemNotification != nil {
		return *c.Notifications.SystemNotification
	}
	return true
}

// NotificationSound returns the macOS notification sound name, or "Glass" as default.
func (c Config) NotificationSound() string {
	if c.Notifications != nil && c.Notifications.Sound != "" {
		return c.Notifications.Sound
	}
	return "Glass"
}

// AliasFor returns the alias for a branch, or empty string if none.
func (c Config) AliasFor(branch string) string {
	if c.Aliases == nil {
		return ""
	}
	return c.Aliases[branch]
}

// ShouldDeleteBranch returns the resolved value of auto_delete_branch (default: false).
func (c Config) ShouldDeleteBranch() bool {
	if c.AutoDeleteBranch != nil {
		return *c.AutoDeleteBranch
	}
	return false
}

// Load reads the global config then the local config, merging them.
// Local fields override global when set.
func Load(repoRoot string) (Config, error) {
	global, err := loadFile(globalConfigPath())
	if err != nil && !os.IsNotExist(err) {
		return Config{}, fmt.Errorf("global config: %w", err)
	}

	local, err := loadFile(filepath.Join(repoRoot, ".synco.yaml"))
	if err != nil && !os.IsNotExist(err) {
		return Config{}, fmt.Errorf("local config: %w", err)
	}

	merged := merge(global, local)

	if merged.WorktreeDir == "" {
		merged.WorktreeDir = ".worktrees"
	}

	return merged, nil
}

// LoadGlobal reads only the global config file.
func LoadGlobal() (Config, error) {
	cfg, err := loadFile(globalConfigPath())
	if err != nil && !os.IsNotExist(err) {
		return Config{}, fmt.Errorf("global config: %w", err)
	}
	if cfg.WorktreeDir == "" {
		cfg.WorktreeDir = ".worktrees"
	}
	return cfg, nil
}

// GlobalConfigPath returns the path to the global config file.
func GlobalConfigPath() string {
	return globalConfigPath()
}

func globalConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "synco", "config.yaml")
}

func loadFile(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// merge returns a Config where local fields override global when non-empty.
func merge(global, local Config) Config {
	out := global

	if local.WorktreeDir != "" {
		out.WorktreeDir = local.WorktreeDir
	}
	if local.SidebarWidth != "" {
		out.SidebarWidth = local.SidebarWidth
	}
	if local.OnCreate != "" {
		out.OnCreate = local.OnCreate
	}
	if local.OnDestroy != "" {
		out.OnDestroy = local.OnDestroy
	}
	if local.ProjectName != "" {
		out.ProjectName = local.ProjectName
	}
	if local.AutoDeleteBranch != nil {
		out.AutoDeleteBranch = local.AutoDeleteBranch
	}
	if local.DefaultCreationProfile != "" {
		out.DefaultCreationProfile = local.DefaultCreationProfile
	}

	// Merge aliases: local overrides global per-key
	if len(local.Aliases) > 0 {
		if out.Aliases == nil {
			out.Aliases = make(map[string]string)
		}
		for k, v := range local.Aliases {
			out.Aliases[k] = v
		}
	}

	// Scripts: local overrides global per-key
	if len(local.Scripts) > 0 {
		if out.Scripts == nil {
			out.Scripts = make(map[string]ScriptDef)
		}
		for k, v := range local.Scripts {
			out.Scripts[k] = v
		}
	}

	// Theme: local overrides global entirely if set
	if local.Theme != nil {
		out.Theme = local.Theme
	}

	// Notifications: local overrides global entirely if set
	if local.Notifications != nil {
		out.Notifications = local.Notifications
	}

	// Projects: local overrides global per-key
	if len(local.Projects) > 0 {
		if out.Projects == nil {
			out.Projects = make(map[string]ProjectDef)
		}
		for k, v := range local.Projects {
			out.Projects[k] = v
		}
	}

	// Creation profiles: local overrides global per-key
	if len(local.CreationProfiles) > 0 {
		if out.CreationProfiles == nil {
			out.CreationProfiles = make(map[string]CreationProfile)
		}
		for k, v := range local.CreationProfiles {
			out.CreationProfiles[k] = v
		}
	}

	// Layouts: local overrides global per-key
	if len(local.Layouts) > 0 {
		if out.Layouts == nil {
			out.Layouts = make(map[string]Layout)
		}
		for k, v := range local.Layouts {
			out.Layouts[k] = v
		}
	}

	return out
}

// WorktreePath computes the absolute worktree path for a branch.
func (c Config) WorktreePath(repoRoot, branch string) string {
	safeName := sanitizeBranchForPath(branch)
	return filepath.Join(repoRoot, c.WorktreeDir, safeName)
}

// ExpandRepoPath expands ~ to the user's home directory in a repo path.
func ExpandRepoPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

// ResolveProjectRepos returns the expanded repo root paths for a named project.
// Returns nil if the project is not defined.
func (c Config) ResolveProjectRepos(name string) []string {
	proj, ok := c.Projects[name]
	if !ok {
		return nil
	}
	repos := make([]string, len(proj.Repos))
	for i, r := range proj.Repos {
		repos[i] = ExpandRepoPath(r)
	}
	return repos
}

func sanitizeBranchForPath(branch string) string {
	// Replace path separators so feature/foo becomes feature-foo
	result := make([]byte, 0, len(branch))
	for i := 0; i < len(branch); i++ {
		ch := branch[i]
		if ch == '/' || ch == '\\' {
			result = append(result, '-')
		} else {
			result = append(result, ch)
		}
	}
	return string(result)
}
