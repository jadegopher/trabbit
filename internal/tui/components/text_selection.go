package components

import (
	"sort"
	"strings"
	"unicode/utf8"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/jadegopher/trabbit/internal/tui/core"
)

// selectBoundary adds standard line/document selection to Bubbles' selection
// model. Existing anchors are retained when combining these keys with Shift+arrows.
func (t *TextArea) selectBoundary(k tea.KeyPressMsg) (bool, tea.Cmd) {
	if k.Mod&tea.ModShift == 0 || !t.Focused() {
		return false, nil
	}
	current := textarea.Position{Row: t.Line(), Col: t.Column()}
	target := current
	lines := strings.Split(t.Value(), "\n")
	switch {
	case key.Matches(k, core.SelectLineStart()):
		target.Col = 0
	case key.Matches(k, core.SelectLineEnd()):
		target.Col = utf8.RuneCountInString(lines[target.Row])
	case key.Matches(k, core.SelectDocumentStart()):
		target = textarea.Position{}
	case key.Matches(k, core.SelectDocumentEnd()):
		target.Row = len(lines) - 1
		target.Col = utf8.RuneCountInString(lines[target.Row])
	default:
		return false, nil
	}

	anchor := current
	if start, end, selected := t.Selection(); selected {
		anchor = start
		if current == start {
			anchor = end
		}
	}
	ax, ay := t.selectionCoordinates(anchor)
	tx, ty := t.selectionCoordinates(target)
	t.BeginSelection(ax, ay)
	t.ExtendSelection(tx, ty)
	t.EndSelection()
	var cmd tea.Cmd
	t.Model, cmd = t.Model.Update(struct{}{})
	return true, cmd
}

// Bubbles exposes selection endpoints through textarea-relative coordinates.
// Invert its own mapper so wrapping, gutters and wide characters stay native.
func (t *TextArea) selectionCoordinates(pos textarea.Position) (int, int) {
	rows := 2*utf8.RuneCountInString(t.Value()) + 2
	offset := t.ScrollYOffset()
	width := lipgloss.Width(t.Model.View())
	// Search segment ends: coordinates below the buffer clamp to its end,
	// so searching starts would mistake the padded rows for document content.
	y := sort.Search(rows, func(row int) bool {
		end := t.PositionAt(width, row-offset)
		return end.Row > pos.Row || (end.Row == pos.Row && end.Col >= pos.Col)
	}) - offset
	x := sort.Search(width+1, func(col int) bool {
		return t.PositionAt(col, y).Col >= pos.Col
	})
	return x, y
}
