package tui

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charles-albert-raymond/synco/internal/worktree"
)

func TestFetchBranchesKeepsLocalAndOriginWhenLocalExists(t *testing.T) {
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
	runGit(t, repo, "push", "origin", "HEAD:feature/remote")
	runGit(t, repo, "checkout", "main")

	msg := fetchBranchesCmd(repo)()
	branches, ok := msg.(branchesMsg)
	if !ok {
		t.Fatalf("message type = %T, want branchesMsg", msg)
	}
	if branches.err != nil {
		t.Fatalf("branches error: %v", branches.err)
	}

	if !containsSource(branches.sources, "local", "feature/remote") {
		t.Fatalf("branches missing local feature/remote: %v", branches.sources)
	}
	if !containsSource(branches.sources, "origin", "feature/remote") {
		t.Fatalf("branches missing origin feature/remote: %v", branches.sources)
	}
}

func TestFetchBranchesHidesCheckedOutBranches(t *testing.T) {
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
	runGit(t, repo, "checkout", "-b", "feature/checked-out")
	runGit(t, repo, "push", "origin", "HEAD:feature/checked-out")

	msg := fetchBranchesCmd(repo)()
	branches, ok := msg.(branchesMsg)
	if !ok {
		t.Fatalf("message type = %T, want branchesMsg", msg)
	}
	if branches.err != nil {
		t.Fatalf("branches error: %v", branches.err)
	}

	if containsSource(branches.sources, "local", "feature/checked-out") {
		t.Fatalf("checked-out local branch should be hidden: %v", branches.sources)
	}
	if containsSource(branches.sources, "origin", "feature/checked-out") {
		t.Fatalf("origin branch with checked-out local branch should be hidden: %v", branches.sources)
	}
}

func TestFetchBranchesExcludesNonOriginRemotes(t *testing.T) {
	remote := filepath.Join(t.TempDir(), "remote.git")
	runGit(t, "", "init", "--bare", remote)
	upstream := filepath.Join(t.TempDir(), "upstream.git")
	runGit(t, "", "init", "--bare", upstream)

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
	runGit(t, repo, "push", upstream, "HEAD:feature/upstream-only")
	runGit(t, repo, "remote", "add", "upstream", upstream)
	runGit(t, repo, "fetch", "upstream")

	msg := fetchBranchesCmd(repo)()
	branches, ok := msg.(branchesMsg)
	if !ok {
		t.Fatalf("message type = %T, want branchesMsg", msg)
	}
	if branches.err != nil {
		t.Fatalf("branches error: %v", branches.err)
	}

	if containsBranch(branches.sources, "feature/upstream-only") {
		t.Fatalf("non-origin remote branch should be excluded: %v", branches.sources)
	}
}

func TestFetchBranchesFallsBackToLocalBranchesWhenFetchFails(t *testing.T) {
	repo := filepath.Join(t.TempDir(), "repo")
	runGit(t, "", "init", repo)
	runGit(t, repo, "config", "user.email", "test@example.com")
	runGit(t, repo, "config", "user.name", "Test User")
	runGit(t, repo, "checkout", "-b", "main")
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, repo, "add", "README.md")
	runGit(t, repo, "commit", "-m", "initial")
	runGit(t, repo, "checkout", "-b", "feature/local")
	runGit(t, repo, "checkout", "main")
	runGit(t, repo, "remote", "add", "origin", filepath.Join(t.TempDir(), "missing.git"))

	msg := fetchBranchesCmd(repo)()
	branches, ok := msg.(branchesMsg)
	if !ok {
		t.Fatalf("message type = %T, want branchesMsg", msg)
	}
	if branches.err != nil {
		t.Fatalf("branches error: %v", branches.err)
	}
	if branches.warning == "" {
		t.Fatal("expected fetch failure warning")
	}
	if !containsSource(branches.sources, "local", "feature/local") {
		t.Fatalf("local branch should still be listed after fetch failure: %v", branches.sources)
	}
}

func containsSource(sources []worktree.BranchSource, kind, branch string) bool {
	for _, source := range sources {
		if string(source.Kind) == kind && source.Branch == branch {
			return true
		}
	}
	return false
}

func containsBranch(sources []worktree.BranchSource, branch string) bool {
	for _, source := range sources {
		if source.Branch == branch {
			return true
		}
	}
	return false
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %s: %v", strings.Join(args, " "), string(out), err)
	}
}
