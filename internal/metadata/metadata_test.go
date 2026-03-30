package metadata

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMissingFile(t *testing.T) {
	store, err := Load(t.TempDir(), ".worktrees")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(store.Worktrees) != 0 {
		t.Fatalf("expected empty store, got %d entries", len(store.Worktrees))
	}
}

func TestSetTitleAndSave(t *testing.T) {
	dir := t.TempDir()
	wtDir := ".worktrees"

	store, _ := Load(dir, wtDir)
	store.SetTitle("feat/auth", "Auth refactor")
	store.SetTitle("fix/bug-42", "Fix login crash")

	if err := store.Save(dir, wtDir); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	// Reload and verify
	store2, err := Load(dir, wtDir)
	if err != nil {
		t.Fatalf("reload failed: %v", err)
	}
	if store2.Title("feat/auth") != "Auth refactor" {
		t.Errorf("expected 'Auth refactor', got %q", store2.Title("feat/auth"))
	}
	if store2.Title("fix/bug-42") != "Fix login crash" {
		t.Errorf("expected 'Fix login crash', got %q", store2.Title("fix/bug-42"))
	}
	if store2.Title("nonexistent") != "" {
		t.Errorf("expected empty title for nonexistent branch")
	}
}

func TestDelete(t *testing.T) {
	dir := t.TempDir()
	wtDir := ".wt"

	store, _ := Load(dir, wtDir)
	store.SetTitle("branch-a", "Title A")
	store.SetTitle("branch-b", "Title B")
	store.Delete("branch-a")

	if err := store.Save(dir, wtDir); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	store2, _ := Load(dir, wtDir)
	if store2.Title("branch-a") != "" {
		t.Errorf("expected deleted title to be empty")
	}
	if store2.Title("branch-b") != "Title B" {
		t.Errorf("expected 'Title B', got %q", store2.Title("branch-b"))
	}
}

func TestPrune(t *testing.T) {
	store := &Store{Worktrees: map[string]WorktreeMeta{
		"active":   {Title: "Active"},
		"orphaned": {Title: "Orphaned"},
	}}

	store.Prune(map[string]bool{"active": true})

	if store.Title("active") != "Active" {
		t.Errorf("active branch should survive prune")
	}
	if store.Title("orphaned") != "" {
		t.Errorf("orphaned branch should be pruned")
	}
}

func TestSaveCreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	wtDir := "deep/nested/worktrees"

	store, _ := Load(dir, wtDir)
	store.SetTitle("test", "Test Title")

	if err := store.Save(dir, wtDir); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	path := filepath.Join(dir, wtDir, metadataFile)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("expected metadata file to exist at %s", path)
	}
}
