package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// Dialogs cover a frozen copy of the scene. These helpers size the saved rows,
// splice in the centered dialog, and move its input cursor into screen coordinates.

// fitLine clips and pads to terminal columns, accounting for ANSI codes and wide characters.
func fitLine(line string, width int) string {
	line = ansi.Truncate(line, max(0, width), "")
	return line + strings.Repeat(" ", max(0, width-ansi.StringWidth(line)))
}

// frameLines snapshots the scene into exactly height rows of width columns.
// Blank rows and padding let the dialog fully cover any content beneath it.
func frameLines(content string, width, height int) []string {
	lines := strings.Split(content, "\n")
	result := make([]string, max(0, height))
	for i := range result {
		if i < len(lines) {
			result[i] = fitLine(lines[i], width)
		} else {
			result[i] = strings.Repeat(" ", max(0, width))
		}
	}
	return result
}

// overlay centers the dialog within the screen, clipping it when it is too large.
// Each covered row keeps the background on either side of the dialog.
func overlay(background []string, content string, width, height int) string {
	w, h := lipgloss.Size(content)
	w = min(w, width)
	h = min(h, height)
	x, y := max(0, (width-w)/2), max(0, (height-h)/2)
	lines := strings.Split(content, "\n")
	// Keep the saved backdrop intact for the next render or when the dialog closes.
	rows := append([]string(nil), background...)
	for i := 0; i < h && y+i < len(rows); i++ {
		left := fitLine(ansi.Cut(rows[y+i], 0, x), x)
		right := fitLine(ansi.Cut(rows[y+i], x+w, width), width-x-w)
		// Reset styles at the seams so background and dialog colors do not bleed.
		rows[y+i] = left + "\x1b[0m" + fitLine(lines[i], w) + "\x1b[0m" + right
	}
	return strings.Join(rows, "\n")
}

// placedCursor converts the dialog's local cursor to screen coordinates using
// the same centering as overlay. A cursor outside the clipped dialog is hidden.
func placedCursor(view tea.View, width, height int) *tea.Cursor {
	if view.Cursor == nil {
		return nil
	}
	w, h := lipgloss.Size(view.Content)
	w = min(w, width)
	h = min(h, height)
	c := *view.Cursor
	if c.X < 0 || c.Y < 0 || c.X >= w || c.Y >= h {
		return nil
	}
	c.X += max(0, (width-w)/2)
	c.Y += max(0, (height-h)/2)
	return &c
}
