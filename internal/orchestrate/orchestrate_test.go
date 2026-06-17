package orchestrate

import (
	"strings"
	"testing"

	"github.com/charles-albert-raymond/synco/internal/config"
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
