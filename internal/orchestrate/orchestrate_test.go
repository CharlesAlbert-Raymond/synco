package orchestrate

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charles-albert-raymond/synco/internal/config"
	"github.com/charles-albert-raymond/synco/internal/state"
	"github.com/charles-albert-raymond/synco/internal/worktree"
)

func TestExpandBootstrap(t *testing.T) {
	got := expandBootstrap("run {{branch}} at {{worktree_path}}", "feature/profile", "/tmp/wt")
	want := "run feature/profile at /tmp/wt"
	if got != want {
		t.Fatalf("expandBootstrap() = %q, want %q", got, want)
	}
}

func TestValidateCreationProfileRejectsBootstrapWithoutSession(t *testing.T) {
	falseVal := false
	err := validateCreationProfile(config.CreationProfile{CreateSession: &falseVal, Bootstrap: "claude"})
	if err == nil || !strings.Contains(err.Error(), "create_session=true") {
		t.Fatalf("validateCreationProfile() error = %v, want create_session=true error", err)
	}
}

func TestCreateWorktreeRunsBeforeAndAfterCreateScriptsInOrder(t *testing.T) {
	repo := initGitRepo(t)
	logPath := filepath.Join(repo, "scripts.log")
	writeScript(t, repo, "before.sh", "printf 'before:%s:%s:%s\n' \"$PWD\" \"$SYNCO_BRANCH\" \"$SYNCO_WORKTREE_PATH\" >> "+shellQuote(logPath)+"\n")
	writeScript(t, repo, "after.sh", "printf 'after:%s:%s:%s\n' \"$PWD\" \"$SYNCO_BRANCH\" \"$SYNCO_WORKTREE_PATH\" >> "+shellQuote(logPath)+"\n")
	falseVal := false
	cfg := config.Config{
		WorktreeDir: ".worktrees",
		Scripts: map[string]config.ScriptDef{
			"before": {Path: "before.sh"},
			"after":  {Path: "after.sh"},
		},
		CreationProfiles: map[string]config.CreationProfile{
			"nosession": {
				CreateSession: &falseVal,
				Scripts: config.LifecycleScripts{
					BeforeCreate: []string{"before"},
					AfterCreate:  []string{"after"},
				},
			},
		},
	}

	wtPath, sessName, err := CreateWorktree(repo, cfg, "feature/scripts", "", CreateWorktreeOpts{CreationProfile: "nosession"})
	if err != nil {
		t.Fatalf("CreateWorktree returned error: %v", err)
	}
	if sessName != "" {
		t.Fatalf("session name = %q, want empty", sessName)
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	want := "before:" + realPath(t, repo) + ":feature/scripts:" + wtPath + "\n" +
		"after:" + realPath(t, wtPath) + ":feature/scripts:" + wtPath + "\n"
	if string(data) != want {
		t.Fatalf("log = %q, want %q", string(data), want)
	}
}

func TestCreateWorktreeAfterCreateScriptCopiesIgnoredFiles(t *testing.T) {
	repo := initGitRepo(t)
	ignoredName := ".env.local"
	if err := os.WriteFile(filepath.Join(repo, ".gitignore"), []byte(ignoredName+"\n"), 0644); err != nil {
		t.Fatalf("write gitignore: %v", err)
	}
	runGit(t, repo, "add", ".gitignore")
	runGit(t, repo, "commit", "-m", "ignore local env")
	ignoredContent := "API_TOKEN=test-token\n"
	if err := os.WriteFile(filepath.Join(repo, ignoredName), []byte(ignoredContent), 0644); err != nil {
		t.Fatalf("write ignored file: %v", err)
	}
	fileList := filepath.Join(repo, "copy-files.txt")
	if err := os.WriteFile(fileList, []byte(ignoredName+"\n"), 0644); err != nil {
		t.Fatalf("write file list: %v", err)
	}
	writeScript(t, repo, "copy_ignored.sh", "while IFS= read -r file; do [ -z \"$file\" ] && continue; cp "+shellQuote(repo)+"/\"$file\" \"$SYNCO_WORKTREE_PATH/$file\"; done < "+shellQuote(fileList)+"\n")
	falseVal := false
	cfg := config.Config{
		WorktreeDir: ".worktrees",
		Scripts:     map[string]config.ScriptDef{"copy_ignored": {Path: "copy_ignored.sh"}},
		CreationProfiles: map[string]config.CreationProfile{
			"nosession": {CreateSession: &falseVal, Scripts: config.LifecycleScripts{AfterCreate: []string{"copy_ignored"}}},
		},
	}

	wtPath, _, err := CreateWorktree(repo, cfg, "feature/ignored-copy", "", CreateWorktreeOpts{CreationProfile: "nosession"})
	if err != nil {
		t.Fatalf("CreateWorktree returned error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(wtPath, ignoredName))
	if err != nil {
		t.Fatalf("read copied ignored file: %v", err)
	}
	if string(data) != ignoredContent {
		t.Fatalf("copied ignored file = %q, want %q", string(data), ignoredContent)
	}
}

func TestCreateWorktreeBeforeCreateFailureStopsBeforeWorktreeAdd(t *testing.T) {
	repo := initGitRepo(t)
	writeScript(t, repo, "fail.sh", "exit 7\n")
	falseVal := false
	cfg := config.Config{
		WorktreeDir: ".worktrees",
		Scripts:     map[string]config.ScriptDef{"fail": {Path: "fail.sh"}},
		CreationProfiles: map[string]config.CreationProfile{
			"nosession": {CreateSession: &falseVal, Scripts: config.LifecycleScripts{BeforeCreate: []string{"fail"}}},
		},
	}

	wtPath := cfg.WorktreePath(repo, "feature/fail")
	_, _, err := CreateWorktree(repo, cfg, "feature/fail", "", CreateWorktreeOpts{CreationProfile: "nosession"})
	if err == nil || !strings.Contains(err.Error(), "fail") || !strings.Contains(err.Error(), config.LifecycleBeforeCreate) {
		t.Fatalf("error = %v, want script name and lifecycle event", err)
	}
	if _, statErr := os.Stat(wtPath); !os.IsNotExist(statErr) {
		t.Fatalf("worktree path exists or stat failed unexpectedly: %v", statErr)
	}
}

func TestDeleteWorktreeRunsBeforeAndAfterDestroyScriptsInOrder(t *testing.T) {
	repo := initGitRepo(t)
	wtPath := filepath.Join(repo, ".worktrees", "feature-destroy")
	runGit(t, repo, "worktree", "add", "-b", "feature/destroy", wtPath)
	logPath := filepath.Join(repo, "destroy.log")
	writeScript(t, repo, "before_destroy.sh", "printf 'before:%s:%s:%s\n' \"$PWD\" \"$SYNCO_BRANCH\" \"$SYNCO_WORKTREE_PATH\" >> "+shellQuote(logPath)+"\n")
	writeScript(t, repo, "after_destroy.sh", "printf 'after:%s:%s:%s\n' \"$PWD\" \"$SYNCO_BRANCH\" \"$SYNCO_WORKTREE_PATH\" >> "+shellQuote(logPath)+"\n")
	cfg := config.Config{
		Scripts: map[string]config.ScriptDef{
			"before_destroy": {Path: "before_destroy.sh"},
			"after_destroy":  {Path: "after_destroy.sh"},
		},
		CreationProfiles: map[string]config.CreationProfile{
			"cleanup": {Scripts: config.LifecycleScripts{BeforeDestroy: []string{"before_destroy"}, AfterDestroy: []string{"after_destroy"}}},
		},
	}
	entry := state.Entry{Worktree: worktree.Worktree{Path: wtPath, Branch: "feature/destroy"}, BranchShort: "feature/destroy"}
	wtPWD := realPath(t, wtPath)

	if err := DeleteWorktree(repo, cfg, entry, DeleteWorktreeOpts{CreationProfile: "cleanup"}); err != nil {
		t.Fatalf("DeleteWorktree returned error: %v", err)
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	want := "before:" + wtPWD + ":feature/destroy:" + wtPath + "\n" +
		"after:" + realPath(t, repo) + ":feature/destroy:" + wtPath + "\n"
	if string(data) != want {
		t.Fatalf("log = %q, want %q", string(data), want)
	}
}

func initGitRepo(t *testing.T) string {
	t.Helper()
	repo := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(repo, 0755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}
	runGit(t, repo, "init")
	runGit(t, repo, "config", "user.email", "test@example.com")
	runGit(t, repo, "config", "user.name", "Test User")
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("test\n"), 0644); err != nil {
		t.Fatalf("write readme: %v", err)
	}
	runGit(t, repo, "add", "README.md")
	runGit(t, repo, "commit", "-m", "initial")
	return repo
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %s: %v", args, string(out), err)
	}
}

func writeScript(t *testing.T, repo, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(repo, name), []byte(body), 0644); err != nil {
		t.Fatalf("write script %s: %v", name, err)
	}
}

func shellQuote(s string) string {
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
