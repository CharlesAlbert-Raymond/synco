package metadata

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// WorktreeMeta holds per-worktree metadata not derivable from git.
type WorktreeMeta struct {
	Title string `yaml:"title,omitempty"`
}

// Store maps branch names to their metadata.
type Store struct {
	Worktrees map[string]WorktreeMeta `yaml:"worktrees"`
}

const metadataFile = ".synco-metadata.yaml"

// Load reads the metadata store from <repoRoot>/<worktreeDir>/.synco-metadata.yaml.
// Returns an empty store if the file doesn't exist.
func Load(repoRoot, worktreeDir string) (*Store, error) {
	path := filepath.Join(repoRoot, worktreeDir, metadataFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Store{Worktrees: make(map[string]WorktreeMeta)}, nil
		}
		return nil, err
	}
	var s Store
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	if s.Worktrees == nil {
		s.Worktrees = make(map[string]WorktreeMeta)
	}
	return &s, nil
}

// Save writes the metadata store back to disk.
func (s *Store) Save(repoRoot, worktreeDir string) error {
	dir := filepath.Join(repoRoot, worktreeDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := yaml.Marshal(s)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, metadataFile), data, 0o644)
}

// Title returns the title for a branch, or empty string if none.
func (s *Store) Title(branch string) string {
	if m, ok := s.Worktrees[branch]; ok {
		return m.Title
	}
	return ""
}

// SetTitle sets the title for a branch.
func (s *Store) SetTitle(branch, title string) {
	s.Worktrees[branch] = WorktreeMeta{Title: title}
}

// Delete removes metadata for a branch.
func (s *Store) Delete(branch string) {
	delete(s.Worktrees, branch)
}

// Prune removes entries for branches not in the provided set.
func (s *Store) Prune(activeBranches map[string]bool) {
	for branch := range s.Worktrees {
		if !activeBranches[branch] {
			delete(s.Worktrees, branch)
		}
	}
}
