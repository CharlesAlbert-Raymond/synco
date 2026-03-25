package tmux

import "testing"

func TestSessionNameFor(t *testing.T) {
	tests := []struct {
		branch string
		want   string
	}{
		{"main", "syncopate-main"},
		{"feature-x", "syncopate-feature-x"},
		{"feature/auth-refactor", "syncopate-feature-auth-refactor"},
		{"feat/add-mcp-for-syncopate", "syncopate-feat-add-mcp-for-syncopate"},
		{"my.branch.name", "syncopate-my-branch-name"},
		{"a//b", "syncopate-a-b"},
		{"---dashes---", "syncopate-dashes"},
		{"", "syncopate-"},
		{"simple", "syncopate-simple"},
		{"UPPER_case", "syncopate-UPPER_case"},
	}

	for _, tt := range tests {
		t.Run(tt.branch, func(t *testing.T) {
			got := SessionNameFor(tt.branch)
			if got != tt.want {
				t.Errorf("SessionNameFor(%q) = %q, want %q", tt.branch, got, tt.want)
			}
		})
	}
}
