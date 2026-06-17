package session

import (
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var unsafeChars = regexp.MustCompile(`[^a-zA-Z0-9_-]`)

// RootKey is the stable identifier used for the root worktree's tmux session.
// Using a constant instead of the branch name ensures navigation keeps working
// when the user switches branches on the root worktree.
const RootKey = "root"

// ProjectName derives a sanitized project identifier from a repo root path.
// It resolves to the main working tree so that worktrees share the same
// project name as the root repo.
func ProjectName(repoRoot string) string {
	name := filepath.Base(MainWorktreeRoot(repoRoot))
	return sanitize(name)
}

// ResolveProjectName returns the project name for a repo, preferring the
// user-defined label from config over the directory-derived name.
func ResolveProjectName(repoRoot, configLabel string) string {
	if configLabel != "" {
		return sanitize(configLabel)
	}
	return ProjectName(repoRoot)
}

// MainWorktreeRoot returns the path of the main working tree for a repo.
// If repoRoot is already the main worktree (or detection fails), it returns repoRoot unchanged.
// This is identity-adjacent because session names need to be stable across linked worktrees.
func MainWorktreeRoot(repoRoot string) string {
	cmd := exec.Command("git", "-C", repoRoot, "rev-parse", "--path-format=absolute", "--git-common-dir")
	out, err := cmd.Output()
	if err != nil {
		return repoRoot
	}
	gitCommonDir := strings.TrimSpace(string(out))
	// git-common-dir points to the .git directory of the main worktree.
	// The main worktree root is its parent.
	root := filepath.Dir(gitCommonDir)
	if root == "" || root == "." {
		return repoRoot
	}
	return root
}

// sanitize cleans a string for use in tmux session names.
func sanitize(s string) string {
	safe := unsafeChars.ReplaceAllString(s, "-")
	for strings.Contains(safe, "--") {
		safe = strings.ReplaceAll(safe, "--", "-")
	}
	return strings.Trim(safe, "-")
}

// SessionNameFor derives a tmux session name from a project name and branch.
// The root session is named just "{project}" so it sorts first in choose-tree.
// Branch sessions are "{project}/{branch}" so they group underneath.
//
// This produces a natural hierarchy in tmux's session list:
//
//	synco              root
//	synco/feat-auth    branch worktree
//	synco/fix-bug      branch worktree
func SessionNameFor(project, branch string) string {
	if branch == RootKey {
		return project
	}
	return project + "/" + sanitize(branch)
}

// IsProjectSession returns true if the session name belongs to the given project.
func IsProjectSession(name, project string) bool {
	return name == project || strings.HasPrefix(name, project+"/")
}
