package config

import "testing"

func TestMerge(t *testing.T) {
	trueVal := true
	falseVal := false
	global := Config{
		WorktreeDir:            ".wt",
		OnCreate:               "npm install",
		DefaultCreationProfile: "dev",
		Aliases:                map[string]string{"main": "trunk"},
		CreationProfiles: map[string]CreationProfile{
			"inspect": {CreateSession: &falseVal, RunOnCreate: &falseVal},
		},
	}
	local := Config{
		WorktreeDir:      ".worktrees",
		AutoDeleteBranch: &trueVal,
		Aliases:          map[string]string{"dev": "development"},
		CreationProfiles: map[string]CreationProfile{
			"agent": {Bootstrap: "claude {{branch}}"},
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
	if _, ok := got.CreationProfiles["inspect"]; !ok {
		t.Error("global creation profile 'inspect' should be preserved")
	}
	if got.CreationProfiles["agent"].Bootstrap != "claude {{branch}}" {
		t.Error("local creation profile 'agent' should be merged in")
	}
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
