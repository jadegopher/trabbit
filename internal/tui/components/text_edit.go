package components

import (
	"fmt"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"github.com/jadegopher/trabbit/internal/tui/core"
)

// editorHistoryLimit bounds the number of undo snapshots retained per editor.
const editorHistoryLimit = 100

const rowNumberGapColumns = 1

const wrappedLineMarker = "↪"

const textAreaPayloadLimitBytes = 1024 * 1024

// EnableRowNumbers numbers logical lines and marks soft wraps in the gutter.
// The gutter grows with the document rather than limiting payload height.
func (t *TextArea) EnableRowNumbers() {
	t.rowNumbers = true
	t.ShowLineNumbers = false
	t.Prompt = ""
	t.syncRowNumbers()
}

func (t *TextArea) rowNumberPrompt(info textarea.PromptInfo) string {
	if info.LineNumber >= t.numberedRows {
		return strings.Repeat(" ", t.numberWidth)
	}
	// PromptInfo counts visual rows; the native mapper resolves the logical line
	// without duplicating the editor's word wrapping or wide-character rules.
	position := t.PositionAt(0, info.LineNumber-t.ScrollYOffset())
	label := strconv.Itoa(position.Row + 1)
	if position.Col > 0 {
		label = wrappedLineMarker
	}
	return fmt.Sprintf("%*s ", t.numberWidth-rowNumberGapColumns, label)
}

// Gutter width follows logical line count, so wrapping never changes its size.
func (t *TextArea) syncRowNumbers() {
	if !t.rowNumbers || t.outerWidth <= 0 {
		return
	}
	t.numberWidth = len(strconv.Itoa(t.LineCount())) + rowNumberGapColumns
	t.Model.SetPromptFunc(t.numberWidth, t.rowNumberPrompt)
	t.Model.SetWidth(t.outerWidth)
	t.numberedRows = t.documentHeight()
	t.scrollContent = ""
}

func (t *TextArea) snapshot() textSnapshot {
	return textSnapshot{value: t.Value(), row: t.Line(), col: t.Column()}
}

// History contains values, not model copies: Bubbles models share viewport and
// text storage. Keep a bounded, in-memory stack for the lifetime of this editor.
func (t *TextArea) remember(before textSnapshot) {
	t.undo = append(t.undo, before)
	if len(t.undo) > editorHistoryLimit {
		t.undo = t.undo[len(t.undo)-editorHistoryLimit:]
	}
	t.redo = nil
}

func (t *TextArea) restoreHistory(from, to *[]textSnapshot) {
	if len(*from) == 0 {
		return
	}
	current := t.snapshot()
	saved := (*from)[len(*from)-1]
	*from = (*from)[:len(*from)-1]
	*to = append(*to, current)
	t.Model.SetValue(saved.value)
	t.restoreCursor(saved)
	t.Model, _ = t.Model.Update(struct{}{})
	t.scrollContent = ""
	t.syncRowNumbers()
}

// Replacing the buffer resets its cursor. Restore a logical position, clamping
// the row when a replacement contains fewer lines than the previous value.
func (t *TextArea) restoreCursor(saved textSnapshot) {
	t.MoveToBegin()
	for t.Line() < min(saved.row, t.LineCount()-1) {
		t.CursorDown()
	}
	t.SetCursorColumn(saved.col)
}

func (t *TextArea) SetValue(value string) {
	preserve := t.cursorViewport && t.scrollContent != ""
	if preserve {
		defer t.preserveViewport()()
	}
	t.EndMouseSelection()
	before := t.snapshot()
	t.Model.SetValue(value)
	if preserve {
		t.restoreCursor(before)
	}
	t.scrollContent = ""
	if before.value != t.Value() {
		t.remember(before)
	}
	t.syncRowNumbers()
}

func (t *TextArea) SetWidth(width int) {
	t.EndMouseSelection()
	t.outerWidth = width
	t.Model.SetWidth(width)
	t.syncRowNumbers()
	t.scrollContent = ""
}

func (t *TextArea) SetHeight(height int) {
	t.EndMouseSelection()
	t.Model.SetHeight(height)
	t.scrollContent = ""
}

func (t *TextArea) Cut() tea.Cmd {
	if t.cursorViewport && t.scrollContent != "" {
		defer t.preserveViewport()()
	}
	cmd := t.CopySelection()
	before := t.snapshot()
	t.DeleteSelection()
	if before.value != t.Value() {
		t.remember(before)
	}
	t.scrollContent = ""
	t.syncRowNumbers()
	return cmd
}

// A separate renderer lets the wheel browse without moving the editing cursor.
// Do not copy textarea.Model: its viewport is a pointer shared by such copies.
func (t *TextArea) documentView() (string, int) {
	model := textarea.New()
	model.SetVirtualCursor(false)
	model.MaxHeight, model.MaxWidth, model.CharLimit = 0, 0, 0
	model.ShowLineNumbers = t.ShowLineNumbers
	model.Prompt = t.Prompt
	if t.rowNumbers {
		model.SetPromptFunc(t.numberWidth, t.rowNumberPrompt)
	}
	model.SetStyles(t.Styles())
	model.SetWidth(max(1, t.outerWidth))
	total := t.documentHeight()
	model.SetHeight(total)
	model.SetValue(t.Value())
	model.MoveToBegin()
	if t.Focused() {
		model.Focus()
	}
	// Mirror public selection endpoints into a separate full-document renderer.
	// Its viewport stays independent of the editing model's cursor-follow scroll.
	renderer := TextArea{Model: model, outerWidth: t.outerWidth}
	current := textarea.Position{Row: t.Line(), Col: t.Column()}
	anchor := current
	if start, end, selected := t.Selection(); selected {
		anchor = start
		if current == start {
			anchor = end
		}
	}
	ax, ay := renderer.selectionCoordinates(anchor)
	cx, cy := renderer.selectionCoordinates(current)
	model.BeginSelection(ax, ay)
	model.ExtendSelection(cx, cy)
	model.EndSelection()
	return model.View(), total
}

func (t *TextArea) Scroll(delta int) {
	offset, total := t.ScrollYOffset(), t.scrollTotal
	if t.scrollContent == "" {
		total = t.documentHeight()
	} else {
		offset = t.scrollOffset
	}
	next := max(0, min(offset+delta, total-t.Height()))
	if next == offset {
		return
	}

	// Enter browsing mode only when the viewport moves; a blocked wheel must
	// preserve the editing cursor, selection and its current renderer.
	t.EndMouseSelection()
	t.cursorViewport = false
	if t.scrollContent == "" {
		t.scrollContent, t.scrollTotal = t.documentView()
	}
	t.scrollOffset = next
}

func (t *TextArea) ScrollView(theme core.Theme) string {
	offset, total := t.ScrollYOffset(), 0
	content := t.Model.View()
	if t.scrollContent != "" {
		offset, total = t.scrollOffset, t.scrollTotal
		content = core.CropView(tea.NewView(t.scrollContent), t.outerWidth, t.Height(), offset).Content
	} else {
		total = t.documentHeight()
	}
	view := core.CropView(tea.NewView(content), t.outerWidth, t.Height(), 0)
	return theme.RenderScrollbarViewport(t.outerWidth+core.ScrollbarColumns, t.Height(), view.Content, total, offset, t.Focused())
}

func (t *TextArea) Click(x, y int) {
	t.BeginMouseSelection(x, y)
	t.EndMouseSelection()
}

// BeginMouseSelection maps a click in the visible viewport to the native editor,
// including when the wheel has browsed away from the editing cursor.
func (t *TextArea) BeginMouseSelection(x, y int) {
	if t.scrollContent == "" {
		t.scrollOffset = t.ScrollYOffset()
	}
	if t.scrollContent != "" {
		y += t.scrollOffset - t.ScrollYOffset()
	}
	t.cursorViewport = true
	t.BeginSelection(x, y)
	t.dragging = true
	t.Model, _ = t.Model.Update(struct{}{})
	t.scrollContent, t.scrollTotal = t.documentView()
}

// DragMouseSelection confines the pointer to the text viewport so dragging
// across neighboring panels cannot select invisible rows or scrollbar cells.
func (t *TextArea) DragMouseSelection(x, y int) {
	if !t.dragging {
		return
	}
	x = max(0, min(x, t.outerWidth-1))
	y = max(0, min(y, t.Height()-1))
	if t.scrollContent != "" {
		y += t.scrollOffset - t.ScrollYOffset()
	}
	t.ExtendSelection(x, y)
	t.Model, _ = t.Model.Update(struct{}{})
	if t.cursorViewport {
		t.scrollContent, t.scrollTotal = t.documentView()
	}
}

func (t *TextArea) EndMouseSelection() {
	if !t.dragging {
		return
	}
	t.EndSelection()
	t.dragging = false
}

// Capture screen scroll before an action can reset the native model's viewport.
// The returned function runs after text, gutter and cursor updates have finished.
func (t *TextArea) preserveViewport() func() {
	offset := t.ScrollYOffset()
	if t.scrollContent != "" {
		offset = t.scrollOffset
	}
	return func() {
		t.scrollOffset = offset
		t.cursorViewport = true
		t.keepViewport()
	}
}

// Keep visible rows steady until the cursor crosses the viewport edge. Native
// cursor-follow offsets never become screen offsets, including after undo.
func (t *TextArea) keepViewport() {
	_, row := t.selectionCoordinates(textarea.Position{Row: t.Line(), Col: t.Column()})
	row += t.ScrollYOffset()
	if row < t.scrollOffset {
		t.scrollOffset = row
	} else if row >= t.scrollOffset+t.Height() {
		t.scrollOffset = row - t.Height() + 1
	}
	t.scrollContent, t.scrollTotal = t.documentView()
	t.scrollOffset = max(0, min(t.scrollOffset, t.scrollTotal-t.Height()))
}

// PositionAt uses the editor's own wrapping and terminal-cell coordinate mapper.
func (t *TextArea) documentHeight() int {
	lines := strings.Split(t.Value(), "\n")
	last := textarea.Position{Row: len(lines) - 1, Col: len([]rune(lines[len(lines)-1]))}
	_, y := t.selectionCoordinates(last)
	return y + t.ScrollYOffset() + 1
}
