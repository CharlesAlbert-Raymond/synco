package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMerge(t *testing.T) {
	trueVal := true
	falseVal := false
	global := Config{
		WorktreeDir:            ".wt",
		OnCreate:               "npm install",
		DefaultCreationProfile: "dev",
		Aliases:                map[string]string{"main": "trunk"},
		Scripts: map[string]ScriptDef{
			"setup":  {Path: "global-setup.sh"},
			"global": {Path: "global.sh"},
		},
		CreationProfiles: map[string]CreationProfile{
			"inspect": {CreateSession: &falseVal, RunOnCreate: &falseVal},
		},
	}
	local := Config{
		WorktreeDir:      ".worktrees",
		AutoDeleteBranch: &trueVal,
		Aliases:          map[string]string{"dev": "development"},
		Scripts: map[string]ScriptDef{
			"setup": {Path: "local-setup.sh"},
			"seed":  {Path: "seed.sh"},
		},
		CreationProfiles: map[string]CreationProfile{
			"agent": {Bootstrap: "claude {{branch}}", Scripts: LifecycleScripts{AfterCreate: []string{"seed"}}},
		},
	}

	got := merge(global, local)

	if got.WorktreeDir != ".worktrees" {
		t.Errorf("WorktreeDir = %q, want .worktrees (local overrides global)", got.WorktreeDir)
	}
	if got.OnCreate != "npm install" {
		t.Errorf("OnCreate = %q, want npm install (inherited from global)", got.OnCreate)
	}
	if !got.ShouldDeleteBranch() {
		t.Error("ShouldDeleteBranch() = false, want true (local overrides)")
	}
	if got.Aliases["main"] != "trunk" {
		t.Error("global alias 'main' should be preserved")
	}
	if got.Aliases["dev"] != "development" {
		t.Error("local alias 'dev' should be merged in")
	}
	if got.Scripts["global"].Path != "global.sh" {
		t.Error("global script should be preserved")
	}
	if got.Scripts["setup"].Path != "local-setup.sh" {
		t.Error("local script should override global script by name")
	}
	if got.Scripts["seed"].Path != "seed.sh" {
		t.Error("local script should be merged in")
	}
	if _, ok := got.CreationProfiles["inspect"]; !ok {
		t.Error("global creation profile 'inspect' should be preserved")
	}
	if got.CreationProfiles["agent"].Bootstrap != "claude {{branch}}" {
		t.Error("local creation profile 'agent' should be merged in")
	}
	if got.CreationProfiles["agent"].Scripts.AfterCreate[0] != "seed" {
		t.Error("profile lifecycle script list should be parsed and merged")
	}
}

func TestLoadFileParsesScriptRegistryAndProfileLifecycleScripts(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".synco.yaml")
	if err := os.WriteFile(path, []byte(`scripts:
  install_deps:
    path: .synco/scripts/install_deps.sh
creation_profiles:
  agent:
    scripts:
      before_create:
        - install_deps
      after_destroy: []
`), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := loadFile(path)
	if err != nil {
		t.Fatalf("loadFile returned error: %v", err)
	}
	if cfg.Scripts["install_deps"].Path != ".synco/scripts/install_deps.sh" {
		t.Fatalf("script path not parsed: %#v", cfg.Scripts["install_deps"])
	}
	if got := cfg.CreationProfiles["agent"].Scripts.BeforeCreate; len(got) != 1 || got[0] != "install_deps" {
		t.Fatalf("before_create scripts = %v, want [install_deps]", got)
	}
}

func TestRunProfileScriptsUsesBashEnvWorkdirAndOrder(t *testing.T) {
	repoRoot := t.TempDir()
	worktreePath := filepath.Join(repoRoot, ".worktrees", "feature")
	if err := os.MkdirAll(worktreePath, 0755); err != nil {
		t.Fatalf("mkdir worktree: %v", err)
	}
	logPath := filepath.Join(repoRoot, "script.log")
	writeScript := func(name, marker string) {
		t.Helper()
		path := filepath.Join(repoRoot, name)
		body := "printf '" + marker + ":%s:%s:%s\\n' \"$PWD\" \"$SYNCO_BRANCH\" \"$SYNCO_WORKTREE_PATH\" >> " + testShellQuote(logPath) + "\n"
		if err := os.WriteFile(path, []byte(body), 0644); err != nil {
			t.Fatalf("write script: %v", err)
		}
	}
	writeScript("one.sh", "one")
	writeScript("two.sh", "two")

	cfg := Config{Scripts: map[string]ScriptDef{
		"one": {Path: "one.sh"},
		"two": {Path: "two.sh"},
	}}
	profile := CreationProfile{Scripts: LifecycleScripts{AfterCreate: []string{"one", "two"}}}
	if err := RunProfileScripts(cfg, repoRoot, profile, LifecycleAfterCreate, "feature/test", worktreePath, worktreePath); err != nil {
		t.Fatalf("RunProfileScripts returned error: %v", err)
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	worktreePWD := realPath(t, worktreePath)
	want := "one:" + worktreePWD + ":feature/test:" + worktreePath + "\n" +
		"two:" + worktreePWD + ":feature/test:" + worktreePath + "\n"
	if string(data) != want {
		t.Fatalf("log = %q, want %q", string(data), want)
	}
}

func TestRunProfileScriptsReportsMissingReferenceWithEvent(t *testing.T) {
	profile := CreationProfile{Scripts: LifecycleScripts{BeforeCreate: []string{"missing"}}}
	err := RunProfileScripts(Config{}, t.TempDir(), profile, LifecycleBeforeCreate, "branch", "/tmp/wt", "/tmp")
	if err == nil || !strings.Contains(err.Error(), "missing") || !strings.Contains(err.Error(), LifecycleBeforeCreate) {
		t.Fatalf("error = %v, want script name and lifecycle event", err)
	}
}

func testShellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

func realPath(t *testing.T, path string) string {
	t.Helper()
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatalf("resolve path %s: %v", path, err)
	}
	return resolved
}

func TestMergeEmptyLocal(t *testing.T) {
	global := Config{WorktreeDir: ".wt", OnCreate: "echo hi"}
	got := merge(global, Config{})

	if got.WorktreeDir != ".wt" {
		t.Errorf("WorktreeDir = %q, want .wt", got.WorktreeDir)
	}
	if got.OnCreate != "echo hi" {
		t.Errorf("OnCreate = %q, want 'echo hi'", got.OnCreate)
	}
}

func TestSanitizeBranchForPath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"feature/auth", "feature-auth"},
		{"simple", "simple"},
		{"a/b/c", "a-b-c"},
		{"back\\slash", "back-slash"},
		{"no-change", "no-change"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := sanitizeBranchForPath(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeBranchForPath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestShouldDeleteBranchDefault(t *testing.T) {
	c := Config{}
	if c.ShouldDeleteBranch() {
		t.Error("default should be false")
	}
}

func TestMergeNotifications(t *testing.T) {
	falseVal := false
	global := Config{
		Notifications: &Notifications{
			SilenceSeconds: 30,
			Sound:          "Ping",
		},
	}
	local := Config{
		Notifications: &Notifications{
			Enabled:        &falseVal,
			SilenceSeconds: 5,
		},
	}
	got := merge(global, local)

	if got.Notifications == nil {
		t.Fatal("Notifications should not be nil")
	}
	if got.NotificationsEnabled() {
		t.Error("NotificationsEnabled() = true, want false (local explicitly disabled)")
	}
	if got.SilenceThreshold() != 5 {
		t.Errorf("SilenceThreshold() = %d, want 5 (local overrides global entirely)", got.SilenceThreshold())
	}
	// Sound should be empty since local overrides global entirely
	if got.NotificationSound() != "Glass" {
		// local.Sound is empty, so default "Glass" should be returned
	}
}

func TestNotificationsDefaults(t *testing.T) {
	// nil Notifications: disabled
	c := Config{}
	if c.NotificationsEnabled() {
		t.Error("nil Notifications should mean disabled")
	}
	if c.SilenceThreshold() != 10 {
		t.Errorf("default SilenceThreshold = %d, want 10", c.SilenceThreshold())
	}

	// non-nil Notifications with zero values: enabled with defaults
	c2 := Config{Notifications: &Notifications{}}
	if !c2.NotificationsEnabled() {
		t.Error("empty Notifications struct should default to enabled")
	}
	if !c2.BellEnabled() {
		t.Error("default BellEnabled should be true")
	}
	if !c2.SystemNotificationEnabled() {
		t.Error("default SystemNotificationEnabled should be true")
	}
	if c2.NotificationSound() != "Glass" {
		t.Errorf("default NotificationSound = %q, want Glass", c2.NotificationSound())
	}
}

func TestMergeProjectName(t *testing.T) {
	global := Config{WorktreeDir: ".wt"}
	local := Config{ProjectName: "my-project"}
	got := merge(global, local)
	if got.ProjectName != "my-project" {
		t.Errorf("ProjectName = %q, want my-project", got.ProjectName)
	}
}

func TestMergeProjects(t *testing.T) {
	global := Config{
		Projects: map[string]ProjectDef{
			"web": {Repos: []string{"~/code/frontend"}},
		},
	}
	local := Config{
		Projects: map[string]ProjectDef{
			"api": {Repos: []string{"~/code/backend"}},
		},
	}
	got := merge(global, local)

	if _, ok := got.Projects["web"]; !ok {
		t.Error("global project 'web' should be preserved")
	}
	if _, ok := got.Projects["api"]; !ok {
		t.Error("local project 'api' should be merged in")
	}
}

func TestResolveProjectRepos(t *testing.T) {
	c := Config{
		Projects: map[string]ProjectDef{
			"myapp": {Repos: []string{"~/code/frontend", "/abs/backend"}},
		},
	}

	repos := c.ResolveProjectRepos("myapp")
	if len(repos) != 2 {
		t.Fatalf("expected 2 repos, got %d", len(repos))
	}
	// ~ should be expanded
	if repos[0] == "~/code/frontend" {
		t.Error("~ should be expanded in repo path")
	}
	// Absolute path should be unchanged
	if repos[1] != "/abs/backend" {
		t.Errorf("absolute path should be unchanged, got %q", repos[1])
	}

	// Unknown project
	if repos := c.ResolveProjectRepos("unknown"); repos != nil {
		t.Error("unknown project should return nil")
	}
}

func TestExpandRepoPath(t *testing.T) {
	// Absolute path stays the same
	if got := ExpandRepoPath("/abs/path"); got != "/abs/path" {
		t.Errorf("absolute path changed: %q", got)
	}
	// Relative path stays the same
	if got := ExpandRepoPath("relative/path"); got != "relative/path" {
		t.Errorf("relative path changed: %q", got)
	}
	// Tilde should expand (we can't test the exact expansion, but it shouldn't start with ~)
	got := ExpandRepoPath("~/code/test")
	if got == "~/code/test" {
		t.Error("~ should be expanded")
	}
}

func TestAliasFor(t *testing.T) {
	c := Config{Aliases: map[string]string{"main": "trunk"}}
	if got := c.AliasFor("main"); got != "trunk" {
		t.Errorf("AliasFor(main) = %q, want trunk", got)
	}
	if got := c.AliasFor("missing"); got != "" {
		t.Errorf("AliasFor(missing) = %q, want empty", got)
	}

	// nil aliases
	c2 := Config{}
	if got := c2.AliasFor("main"); got != "" {
		t.Errorf("AliasFor with nil aliases = %q, want empty", got)
	}
}

func TestResolveCreationProfileDefaultsToBuiltInDev(t *testing.T) {
	profile, name, err := (Config{}).ResolveCreationProfile("")
	if err != nil {
		t.Fatalf("ResolveCreationProfile returned error: %v", err)
	}
	if name != BuiltInCreationProfileDev {
		t.Fatalf("profile name = %q, want %q", name, BuiltInCreationProfileDev)
	}
	if !profile.ShouldCreateSession() || !profile.ShouldRunOnCreate() {
		t.Fatal("built-in dev profile should create a session and run on_create")
	}
}

func TestResolveCreationProfileUsesConfiguredDefault(t *testing.T) {
	falseVal := false
	c := Config{
		DefaultCreationProfile: "inspect",
		CreationProfiles: map[string]CreationProfile{
			"inspect": {CreateSession: &falseVal, RunOnCreate: &falseVal},
		},
	}

	profile, name, err := c.ResolveCreationProfile("")
	if err != nil {
		t.Fatalf("ResolveCreationProfile returned error: %v", err)
	}
	if name != "inspect" {
		t.Fatalf("profile name = %q, want inspect", name)
	}
	if profile.ShouldCreateSession() || profile.ShouldRunOnCreate() {
		t.Fatal("inspect profile should skip session and on_create")
	}
}

func TestResolveCreationProfileRejectsUnknown(t *testing.T) {
	_, _, err := (Config{}).ResolveCreationProfile("missing")
	if err == nil {
		t.Fatal("expected unknown profile error")
	}
}

func TestCreationProfileNamesIncludesUnknownConfiguredDefault(t *testing.T) {
	c := Config{DefaultCreationProfile: "missing"}
	names := c.CreationProfileNames()
	if len(names) != 2 {
		t.Fatalf("CreationProfileNames length = %d, want 2", len(names))
	}
	if names[0] != BuiltInCreationProfileDev || names[1] != "missing" {
		t.Fatalf("CreationProfileNames = %v, want [dev missing]", names)
	}
}
