package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestRenderFramePadsToExactSize(t *testing.T) {
	got := renderFrame("x", 6, 3)
	lines := strings.Split(got, "\n")
	if len(lines) != 3 {
		t.Fatalf("frame height = %d, want 3: %q", len(lines), got)
	}
	for i, line := range lines {
		if width := lipgloss.Width(line); width != 6 {
			t.Fatalf("line %d width = %d, want 6: %q", i, width, line)
		}
	}
}

func TestCompactVisibleRangeKeepsCursorVisible(t *testing.T) {
	start, end := compactVisibleRange(10, 8, 16)
	if start > 8 || end <= 8 {
		t.Fatalf("range [%d:%d] does not include cursor 8", start, end)
	}
	if end-start > 2 {
		t.Fatalf("range [%d:%d] too large for compact height", start, end)
	}
}

func TestTruncateHandlesZeroWidth(t *testing.T) {
	if got := truncate("abc", 0); got != "" {
		t.Fatalf("truncate zero width = %q, want empty", got)
	}
}
