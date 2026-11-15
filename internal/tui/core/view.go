package core

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"strings"
)

// CropView clips by terminal cells and translates the cursor once into the viewport.
func CropView(v tea.View, width, height, offset int) tea.View {
	if width <= 0 || height <= 0 {
		return tea.NewView("")
	}
	lines := strings.Split(v.Content, "\n")
	result := make([]string, height)
	for y := range result {
		line := ""
		if y+offset < len(lines) {
			line = ansi.Truncate(lines[y+offset], width, "")
		}
		result[y] = lipgloss.PlaceHorizontal(width, lipgloss.Left, line)
	}
	v.Content = strings.Join(result, "\n")
	if v.Cursor != nil {
		c := *v.Cursor
		c.Y -= offset
		v.Cursor = nil
		if c.X >= 0 && c.X < width && c.Y >= 0 && c.Y < height {
			v.Cursor = &c
		}
	}
	return v
}
