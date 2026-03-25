package config

import "testing"

func TestMerge(t *testing.T) {
	trueVal := true
	global := Config{
		WorktreeDir: ".wt",
		OnCreate:    "npm install",
		Aliases:     map[string]string{"main": "trunk"},
	}
	local := Config{
		WorktreeDir:      ".worktrees",
		AutoDeleteBranch: &trueVal,
		Aliases:          map[string]string{"dev": "development"},
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
