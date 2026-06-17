package worktree

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

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

func TestAddTrackingCreatesLocalTrackingBranch(t *testing.T) {
	remote := filepath.Join(t.TempDir(), "remote.git")
	runGit(t, "", "init", "--bare", remote)

	repo := filepath.Join(t.TempDir(), "repo")
	runGit(t, "", "clone", remote, repo)
	runGit(t, repo, "config", "user.email", "test@example.com")
	runGit(t, repo, "config", "user.name", "Test User")
	runGit(t, repo, "checkout", "-b", "main")
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, repo, "add", "README.md")
	runGit(t, repo, "commit", "-m", "initial")
	runGit(t, repo, "push", "origin", "HEAD:main")
	runGit(t, repo, "checkout", "-b", "feature/remote")
	if err := os.WriteFile(filepath.Join(repo, "feature.txt"), []byte("feature\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, repo, "add", "feature.txt")
	runGit(t, repo, "commit", "-m", "feature")
	runGit(t, repo, "push", "origin", "feature/remote")
	runGit(t, repo, "checkout", "main")
	runGit(t, repo, "branch", "-D", "feature/remote")

	if !RemoteBranchExists(repo, "origin/feature/remote") {
		t.Fatal("expected origin/feature/remote to exist")
	}

	wtPath := filepath.Join(repo, ".worktrees", "feature-remote")
	if err := AddTracking(repo, wtPath, "feature/remote", "origin/feature/remote"); err != nil {
		t.Fatalf("AddTracking failed: %v", err)
	}

	branch := strings.TrimSpace(runGitOutput(t, wtPath, "branch", "--show-current"))
	if branch != "feature/remote" {
		t.Fatalf("branch = %q, want feature/remote", branch)
	}

	upstream := strings.TrimSpace(runGitOutput(t, wtPath, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}"))
	if upstream != "origin/feature/remote" {
		t.Fatalf("upstream = %q, want origin/feature/remote", upstream)
	}
}

func TestBranchExists(t *testing.T) {
	repo := filepath.Join(t.TempDir(), "repo")
	runGit(t, "", "init", repo)
	runGit(t, repo, "config", "user.email", "test@example.com")
	runGit(t, repo, "config", "user.name", "Test User")
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, repo, "add", "README.md")
	runGit(t, repo, "commit", "-m", "initial")
	runGit(t, repo, "checkout", "-b", "feature/local")

	if !BranchExists(repo, "feature/local") {
		t.Fatal("expected feature/local to exist")
	}
	if BranchExists(repo, "feature/missing") {
		t.Fatal("expected feature/missing not to exist")
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	_ = runGitOutput(t, dir, args...)
}

func runGitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %s: %v", strings.Join(args, " "), string(out), err)
	}
	return string(out)
}
