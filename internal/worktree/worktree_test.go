package worktree

import "testing"

func TestParsePorcelain(t *testing.T) {
	input := []byte(`worktree /home/user/repo
HEAD abc123def456
branch refs/heads/main

worktree /home/user/repo/.worktrees/feature-x
HEAD def789abc012
branch refs/heads/feature/x

worktree /home/user/repo/.worktrees/detached
HEAD 111222333444
detached

`)
	wts := parsePorcelain(input, "/home/user/repo")

	if len(wts) != 3 {
		t.Fatalf("expected 3 worktrees, got %d", len(wts))
	}

	// First worktree is always main
	if !wts[0].IsMain {
		t.Error("first worktree should be main")
	}
	if wts[0].Path != "/home/user/repo" {
		t.Errorf("wt[0].Path = %q, want /home/user/repo", wts[0].Path)
	}
	if wts[0].Branch != "main" {
		t.Errorf("wt[0].Branch = %q, want main", wts[0].Branch)
	}
	if wts[0].HEAD != "abc123def456" {
		t.Errorf("wt[0].HEAD = %q, want abc123def456", wts[0].HEAD)
	}

	// Second worktree
	if wts[1].IsMain {
		t.Error("second worktree should not be main")
	}
	if wts[1].Branch != "feature/x" {
		t.Errorf("wt[1].Branch = %q, want feature/x", wts[1].Branch)
	}

	// Detached worktree
	if wts[2].Branch != "(detached)" {
		t.Errorf("wt[2].Branch = %q, want (detached)", wts[2].Branch)
	}
}

func TestParsePorcelainEmpty(t *testing.T) {
	wts := parsePorcelain([]byte{}, "/repo")
	if len(wts) != 0 {
		t.Fatalf("expected 0 worktrees, got %d", len(wts))
	}
}

func TestParsePorcelainNoTrailingNewline(t *testing.T) {
	input := []byte(`worktree /repo
HEAD abc123
branch refs/heads/main`)

	wts := parsePorcelain(input, "/repo")
	if len(wts) != 1 {
		t.Fatalf("expected 1 worktree, got %d", len(wts))
	}
	if wts[0].Branch != "main" {
		t.Errorf("Branch = %q, want main", wts[0].Branch)
	}
}
