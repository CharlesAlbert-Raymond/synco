package tui

import "github.com/charmbracelet/lipgloss"

func renderFrame(content string, width, height int) string {
	style := lipgloss.NewStyle()
	if width > 0 {
		style = style.MaxWidth(width).Width(width)
	}
	if height > 0 {
		style = style.MaxHeight(height).Height(height)
	}
	return style.Render(content)
}
