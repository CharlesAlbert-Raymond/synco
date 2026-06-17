package tmux

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
)

func TestFixedSidebarWidth(t *testing.T) {
	tests := []struct {
		name   string
		width  string
		want   int
		wantOK bool
	}{
		{"fixed width", "28", 28, true},
		{"trims spaces", " 28 ", 28, true},
		{"ignores percentages", "30%", 0, false},
		{"ignores empty width", "", 0, false},
		{"ignores invalid width", "wide", 0, false},
		{"ignores non-positive width", "0", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := fixedSidebarWidth(tt.width)
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("fixedSidebarWidth(%q) = %d, %v; want %d, %v", tt.width, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

func TestInstallSidebarResizeHook(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not installed")
	}

	session := fmt.Sprintf("synco-hook-test-%d", os.Getpid())
	_ = exec.Command("tmux", "kill-session", "-t", session).Run()
	cmd := exec.Command("tmux", "new-session", "-d", "-s", session, "-x", "170", "-y", "50")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Skipf("tmux unavailable: %s: %v", string(out), err)
	}
	defer exec.Command("tmux", "kill-session", "-t", session).Run()

	cmd = exec.Command("tmux", "split-window", "-P", "-F", "#{pane_id}", "-t", session, "-fhb", "-l", "28", "sleep 60")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("split-window failed: %s: %v", string(out), err)
	}
	paneID := strings.TrimSpace(string(out))

	if err := installSidebarResizeHook(session, paneID, "28"); err != nil {
		t.Fatalf("installSidebarResizeHook failed: %v", err)
	}

	resizeWindow(t, session, 80)
	if got := paneWidth(t, paneID); got != 28 {
		t.Fatalf("pane width after shrink = %d, want 28", got)
	}

	resizeWindow(t, session, 170)
	if got := paneWidth(t, paneID); got != 28 {
		t.Fatalf("pane width after grow = %d, want 28", got)
	}

	if err := installSidebarResizeHook(session, paneID, "30%"); err != nil {
		t.Fatalf("clearing resize hook failed: %v", err)
	}
	cmd = exec.Command("tmux", "show-hooks", "-w", "-t", session, "window-resized")
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("show-hooks failed: %s: %v", string(out), err)
	}
	if strings.Contains(string(out), "window-resized[90]") {
		t.Fatalf("window-resized[90] hook was not cleared: %s", string(out))
	}
}

func resizeWindow(t *testing.T, session string, width int) {
	t.Helper()
	cmd := exec.Command("tmux", "resize-window", "-t", session, "-x", strconv.Itoa(width), "-y", "50")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("resize-window failed: %s: %v", string(out), err)
	}
}

func paneWidth(t *testing.T, paneID string) int {
	t.Helper()
	cmd := exec.Command("tmux", "display-message", "-p", "-t", paneID, "#{pane_width}")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("display pane width failed: %s: %v", string(out), err)
	}
	width, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		t.Fatalf("parse pane width failed: %q: %v", strings.TrimSpace(string(out)), err)
	}
	return width
}
