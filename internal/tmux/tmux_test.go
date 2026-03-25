package tmux

import "testing"

func TestSessionNameFor(t *testing.T) {
	project := "myproject"
	tests := []struct {
		branch string
		want   string
	}{
		{"main", "myproject-main"},
		{"feature-x", "myproject-feature-x"},
		{"feature/auth-refactor", "myproject-feature-auth-refactor"},
		{"feat/add-mcp-for-syncopate", "myproject-feat-add-mcp-for-syncopate"},
		{"my.branch.name", "myproject-my-branch-name"},
		{"a//b", "myproject-a-b"},
		{"---dashes---", "myproject-dashes"},
		{"", "myproject-"},
		{"simple", "myproject-simple"},
		{"UPPER_case", "myproject-UPPER_case"},
	}

	for _, tt := range tests {
		t.Run(tt.branch, func(t *testing.T) {
			got := SessionNameFor(project, tt.branch)
			if got != tt.want {
				t.Errorf("SessionNameFor(%q, %q) = %q, want %q", project, tt.branch, got, tt.want)
			}
		})
	}
}

func TestProjectName(t *testing.T) {
	tests := []struct {
		repoRoot string
		want     string
	}{
		{"/home/user/projects/my-app", "my-app"},
		{"/home/user/projects/My.Project", "My-Project"},
		{"/home/user/projects/simple", "simple"},
	}

	for _, tt := range tests {
		t.Run(tt.repoRoot, func(t *testing.T) {
			got := ProjectName(tt.repoRoot)
			if got != tt.want {
				t.Errorf("ProjectName(%q) = %q, want %q", tt.repoRoot, got, tt.want)
			}
		})
	}
}
