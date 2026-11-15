package components

import (
	"fmt"
	"strings"
	"testing"

	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/jadegopher/trabbit/internal/tui/core"
	"github.com/stretchr/testify/require"
)

func TestTextAreaBlockedScrollPreservesEditingState(t *testing.T) {
	t.Parallel()
	for _, boundary := range []string{"top", "bottom", "fits"} {
		t.Run(boundary, func(t *testing.T) {
			t.Parallel()
			theme := core.DefaultTheme()
			editor := NewTextArea(theme)
			editor.EnableRowNumbers()
			editor.SetWidth(30)
			editor.SetHeight(8)
			editor.SetValue(strings.Repeat("payload line\n", 20))
			delta := -1
			switch boundary {
			case "top":
				editor.MoveToBegin()
			case "bottom":
				editor.MoveToEnd()
				delta = 1
			case "fits":
				editor.SetValue("payload")
				delta = 1
			}
			editor.Model, _ = editor.Model.Update(struct{}{})
			before, cursor := editor.ScrollView(theme), editor.Cursor()
			require.NotNil(t, cursor)
			position := *cursor
			for range 5 {
				editor.Scroll(delta)
				require.Equal(t, before, editor.ScrollView(theme))
				require.Equal(t, &position, editor.Cursor(), "blocked scrolling must not hide the editing cursor")
				require.Empty(t, editor.scrollContent, "blocked scrolling must not switch renderers")
			}

			// Check the same boundary after a mouse selection enters its own viewport.
			editor.BeginMouseSelection(editor.numberWidth, 0)
			editor.DragMouseSelection(editor.numberWidth+3, 0)
			editor.EndMouseSelection()
			selected := editor.SelectedText()
			require.NotEmpty(t, selected)
			before, cursor = editor.ScrollView(theme), editor.Cursor()
			editor.Scroll(delta)
			require.Equal(t, before, editor.ScrollView(theme))
			require.Equal(t, cursor, editor.Cursor())
			require.Equal(t, selected, editor.SelectedText())
		})
	}
}

func TestTextAreaMouseSelectionUsesWrappedCellCoordinates(t *testing.T) {
	t.Parallel()
	editor := NewTextArea(core.DefaultTheme())
	editor.MaxHeight = 0
	editor.EnableRowNumbers()
	editor.SetWidth(14)
	editor.SetHeight(10)
	editor.SetValue("界 hello wrapped text\nsecond line")
	editor.MoveToBegin()
	start := textarea.Position{Row: 0, Col: 1}
	end := textarea.Position{Row: 1, Col: 6}
	sx, sy := editor.selectionCoordinates(start)
	ex, ey := editor.selectionCoordinates(end)
	require.Greater(t, ey, 1, "the selection must cross a wrapped row")

	editor.BeginMouseSelection(sx, sy)
	editor.DragMouseSelection(ex, ey)
	editor.EndMouseSelection()
	require.Equal(t, " hello wrapped text\nsecond", editor.SelectedText())

	editor.DragMouseSelection(sx, sy)
	require.Equal(t, " hello wrapped text\nsecond", editor.SelectedText(), "released drags must ignore motion")
	editor.BeginMouseSelection(ex, ey)
	editor.DragMouseSelection(sx, sy)
	editor.EndMouseSelection()
	require.Equal(t, " hello wrapped text\nsecond", editor.SelectedText(), "backwards dragging must use the same bounds")
}

func TestTextAreaMouseSelectionAfterWheelBrowsing(t *testing.T) {
	t.Parallel()
	editor := NewTextArea(core.DefaultTheme())
	editor.SetVirtualCursor(false)
	editor.MaxHeight = 0
	editor.EnableRowNumbers()
	editor.SetWidth(30)
	editor.SetHeight(8)
	editor.SetValue(strings.Repeat("hello world\n", 60))
	editor.Scroll(20)
	offset := editor.scrollOffset
	before := ansi.Strip(editor.ScrollView(core.DefaultTheme()))
	require.NotEqual(t, offset, editor.ScrollYOffset())
	editor.BeginMouseSelection(editor.numberWidth, 1)
	require.Equal(t, offset+1, editor.Line())
	require.Zero(t, editor.Column())
	require.Equal(t, before, ansi.Strip(editor.ScrollView(core.DefaultTheme())), "click must keep visible rows in place")
	editor.DragMouseSelection(editor.numberWidth, 1)
	editor.EndMouseSelection()
	require.False(t, editor.HasSelection(), "releasing a stationary click must not select extra rows")
	require.Equal(t, offset, editor.scrollOffset)
	require.NotNil(t, editor.Cursor())
	require.Equal(t, 1, editor.Cursor().Y)
	editor.BeginMouseSelection(editor.numberWidth, 1)
	x, y := editor.numberWidth+5, 1
	editor.DragMouseSelection(x, y)
	editor.EndMouseSelection()
	require.Equal(t, "hello", editor.SelectedText())
	require.Equal(t, before, ansi.Strip(editor.ScrollView(core.DefaultTheme())), "drag must keep visible rows in place")

	editor.BeginMouseSelection(x, y)
	editor.Blur()
	editor.DragMouseSelection(0, 0)
	require.False(t, editor.HasSelection(), "losing focus must end a pending drag")
	editor.Focus()
	editor.Click(editor.numberWidth, 1)
	editor.Update(tea.KeyPressMsg{Code: 'x', Text: "x"})
	require.Equal(t, "xhello world", strings.Split(editor.Value(), "\n")[offset+1])
	require.Equal(t, offset, editor.scrollOffset, "typing at the clicked position must preserve visible rows")
}

func TestTextAreaEditingKeepsVisibleRows(t *testing.T) {
	t.Parallel()
	for _, action := range []struct {
		name string
		edit func(*TextArea)
	}{
		{"undo", func(editor *TextArea) { editor.Update(tea.KeyPressMsg{Code: 'z', Mod: tea.ModCtrl}) }},
		{"redo", func(editor *TextArea) {
			editor.Update(tea.KeyPressMsg{Code: 'z', Mod: tea.ModCtrl})
			editor.Update(tea.KeyPressMsg{Code: 'y', Mod: tea.ModCtrl})
		}},
		{"paste", func(editor *TextArea) { editor.Update(tea.PasteMsg{Content: "pasted"}) }},
		{"delete", func(editor *TextArea) { editor.Update(tea.KeyPressMsg{Code: tea.KeyBackspace}) }},
		{"cut", func(editor *TextArea) {
			editor.Update(tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModShift})
			editor.Update(tea.KeyPressMsg{Code: 'x', Mod: tea.ModCtrl})
		}},
		{"direct cut", func(editor *TextArea) {
			editor.Update(tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModShift})
			editor.Cut()
		}},
		{"replace value", func(editor *TextArea) {
			editor.SetValue(strings.Replace(editor.Value(), "xline", "changed line", 1))
		}},
		{"empty undo", func(editor *TextArea) {
			editor.undo = nil
			editor.Update(tea.KeyPressMsg{Code: 'z', Mod: tea.ModCtrl})
		}},
		{"line start", func(editor *TextArea) { editor.Update(tea.KeyPressMsg{Code: tea.KeyHome}) }},
	} {
		t.Run(action.name, func(t *testing.T) {
			t.Parallel()
			editor := NewTextArea(core.DefaultTheme())
			editor.MaxHeight = 0
			editor.EnableRowNumbers()
			editor.SetWidth(30)
			editor.SetHeight(8)
			var lines []string
			for row := range 50 {
				lines = append(lines, fmt.Sprintf("line %02d", row))
			}
			editor.SetValue(strings.Join(lines, "\n"))
			editor.MoveToBegin()
			editor.Scroll(20)
			editor.Click(editor.numberWidth, 3)
			editor.Update(tea.KeyPressMsg{Code: 'x', Text: "x"})
			before := strings.Split(ansi.Strip(editor.ScrollView(core.DefaultTheme())), "\n")

			action.edit(editor)

			after := strings.Split(ansi.Strip(editor.ScrollView(core.DefaultTheme())), "\n")
			require.Equal(t, before[0], after[0], "editing in the middle must preserve the first visible row")
			require.Equal(t, before[len(before)-1], after[len(after)-1], "editing in the middle must preserve the last visible row")
			require.NotNil(t, editor.Cursor())
			require.Equal(t, 3, editor.Cursor().Y)
		})
	}
}

func TestTextAreaKeyboardOnlyUndoKeepsVisibleRows(t *testing.T) {
	t.Parallel()
	editor := NewTextArea(core.DefaultTheme())
	editor.MaxHeight = 0
	editor.SetWidth(30)
	editor.SetHeight(8)
	editor.SetValue(strings.Repeat("payload line\n", 50))
	editor.MoveToBegin()
	for range 25 {
		editor.CursorDown()
	}
	editor.Model, _ = editor.Model.Update(struct{}{})
	for range 4 {
		editor.CursorUp()
	}
	offset := editor.ScrollYOffset()
	row := editor.Cursor().Y
	editor.Update(tea.KeyPressMsg{Code: 'x', Text: "x"})
	editor.Update(tea.KeyPressMsg{Code: 'z', Mod: tea.ModCtrl})

	require.Equal(t, offset, editor.scrollOffset)
	require.NotNil(t, editor.Cursor())
	require.Equal(t, row, editor.Cursor().Y)
}

func TestTextAreaEditingScrollsOnlyAtViewportEdges(t *testing.T) {
	t.Parallel()
	editor := NewTextArea(core.DefaultTheme())
	editor.MaxHeight = 0
	editor.EnableRowNumbers()
	editor.SetWidth(30)
	editor.SetHeight(8)
	editor.SetValue(strings.Repeat("payload line\n", 50))
	editor.MoveToBegin()
	editor.Scroll(20)
	editor.Click(editor.numberWidth, 3)
	for range 4 {
		editor.Update(tea.KeyPressMsg{Code: tea.KeyDown})
		require.Equal(t, 20, editor.scrollOffset)
	}
	editor.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	require.Equal(t, 21, editor.scrollOffset)
	require.Equal(t, editor.Height()-1, editor.Cursor().Y)
	for range editor.Height() - 1 {
		editor.Update(tea.KeyPressMsg{Code: tea.KeyUp})
		require.Equal(t, 21, editor.scrollOffset)
	}
	editor.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	require.Equal(t, 20, editor.scrollOffset)
	require.Zero(t, editor.Cursor().Y)

	editor.Update(tea.KeyPressMsg{Code: 'r', Mod: tea.ModCtrl})
	require.Empty(t, editor.Value())
	require.Zero(t, editor.scrollOffset, "clearing must clamp the viewport to the remaining document")
	require.Zero(t, editor.Cursor().Y)
	editor.Update(tea.KeyPressMsg{Code: 'z', Mod: tea.ModCtrl})
	require.NotEmpty(t, editor.Value())
	require.NotNil(t, editor.Cursor(), "undo must reveal a restored cursor outside the cleared viewport")
}

func TestTextAreaWrappedUndoKeepsVisibleRows(t *testing.T) {
	t.Parallel()
	editor := NewTextArea(core.DefaultTheme())
	editor.MaxHeight = 0
	editor.EnableRowNumbers()
	editor.SetWidth(18)
	editor.SetHeight(8)
	editor.SetValue(strings.Repeat("日本語 wrapped payload ", 40))
	editor.MoveToBegin()
	editor.Scroll(20)
	editor.Click(editor.numberWidth, 3)
	before := ansi.Strip(editor.ScrollView(core.DefaultTheme()))
	position := editor.snapshot()
	editor.Update(tea.KeyPressMsg{Code: 'x', Text: "x"})
	editor.Update(tea.KeyPressMsg{Code: 'z', Mod: tea.ModCtrl})

	require.Equal(t, position, editor.snapshot())
	require.Equal(t, before, ansi.Strip(editor.ScrollView(core.DefaultTheme())))
	require.NotNil(t, editor.Cursor())
	require.Equal(t, 3, editor.Cursor().Y)
}

func TestTextAreaDocumentCoordinatesExcludePadding(t *testing.T) {
	t.Parallel()
	editor := NewTextArea(core.DefaultTheme())
	editor.MaxHeight = 0
	editor.ShowLineNumbers = false
	editor.SetWidth(55)
	editor.SetHeight(19)
	editor.SetValue("one\ntwo\nlast")
	_, y := editor.selectionCoordinates(textarea.Position{Row: 2, Col: 4})
	require.Equal(t, 2, y+editor.ScrollYOffset())
	content, total := editor.documentView()
	require.Equal(t, 3, total)
	require.Contains(t, ansi.Strip(content), "one")
	require.Contains(t, ansi.Strip(content), "two")
	require.Contains(t, ansi.Strip(content), "last")
}

func TestTextAreaNumberedRowsThroughResizeAndScrolling(t *testing.T) {
	t.Parallel()
	editor := NewTextArea(core.DefaultTheme())
	editor.SetVirtualCursor(false)
	editor.MaxHeight = 0
	editor.CharLimit = 0
	editor.EnableRowNumbers()
	editor.SetHeight(8)
	editor.SetWidth(30)
	editor.SetValue(strings.Repeat("日本語 text\n", 120) + "last")
	editor.MoveToBegin()

	for _, width := range []int{30, 12, 40} {
		editor.SetWidth(width)
		editor.MoveToBegin()
		view := ansi.Strip(editor.ScrollView(core.DefaultTheme()))
		for row, line := range strings.Split(view, "\n") {
			require.Equal(t, width+1, lipgloss.Width(line))
			position := editor.PositionAt(0, row-editor.ScrollYOffset())
			label := fmt.Sprintf("%3d ", position.Row+1)
			if position.Col > 0 {
				label = "  ↪ "
			}
			require.True(t, strings.HasPrefix(line, label), line)
		}
		editor.Scroll(95)
		view = ansi.Strip(editor.ScrollView(core.DefaultTheme()))
		position := editor.PositionAt(0, 95-editor.ScrollYOffset())
		label := fmt.Sprintf("%3d ", position.Row+1)
		if position.Col > 0 {
			label = "  ↪ "
		}
		require.True(t, strings.HasPrefix(view, label))
		// Editing clears wheel browsing and retains number/cursor geometry.
		editor.Update(tea.KeyPressMsg{Code: 'x', Text: "x"})
		require.Empty(t, editor.scrollContent)
		require.NotNil(t, editor.Cursor())
	}

	editor.SetValue("one\ntwo\nthree")
	editor.MoveToBegin()
	view := ansi.Strip(editor.ScrollView(core.DefaultTheme()))
	require.True(t, strings.HasPrefix(view, "1 one"))
	require.Contains(t, view, "2 two")
	require.Contains(t, view, "3 three")
	require.NotContains(t, view, "4 ")
	editor.Click(editor.numberWidth, 1)
	require.Equal(t, 1, editor.Line())
	require.Zero(t, editor.Column())
}

func TestTextAreaCommandHistoryAndPortableRedo(t *testing.T) {
	t.Parallel()
	for _, mod := range []tea.KeyMod{tea.ModSuper, tea.ModMeta, tea.ModCtrl} {
		t.Run(fmt.Sprint(mod), func(t *testing.T) {
			t.Parallel()
			editor := NewTextArea(core.DefaultTheme())
			editor.SetWidth(30)
			editor.SetHeight(5)
			editor.Update(tea.PasteMsg{Content: "first"})
			editor.Update(tea.PasteMsg{Content: " second"})
			editor.Update(tea.KeyPressMsg{Code: 'z', Mod: mod})
			require.Equal(t, "first", editor.Value())
			editor.Update(tea.KeyPressMsg{Code: 'z', Mod: mod | tea.ModShift})
			require.Equal(t, "first second", editor.Value())
			editor.Update(tea.KeyPressMsg{Code: 'z', Mod: mod})
			editor.Update(tea.KeyPressMsg{Code: 'y', Mod: tea.ModCtrl})
			require.Equal(t, "first second", editor.Value())
		})
	}
}

func TestTextAreaLogicalLinesKeepNumbersAcrossSoftWraps(t *testing.T) {
	t.Parallel()
	for _, longLine := range []string{strings.Repeat("x", 160), strings.Repeat("界", 80)} {
		t.Run(longLine[:1], func(t *testing.T) {
			t.Parallel()
			editor := NewTextArea(core.DefaultTheme())
			editor.MaxHeight = 0
			editor.EnableRowNumbers()
			editor.SetWidth(18)
			editor.SetHeight(5)
			value := longLine + "\nnext"
			editor.SetValue(value)
			editor.MoveToBegin()
			full, total := editor.documentView()
			lines := strings.Split(ansi.Strip(full), "\n")
			require.Greater(t, total, 10)
			require.Equal(t, 2, editor.numberWidth, "visual rows must not widen the logical gutter")
			require.True(t, strings.HasPrefix(lines[0], "1 "))
			for _, continuation := range lines[1 : total-1] {
				require.True(t, strings.HasPrefix(continuation, "↪ "), continuation)
			}
			require.True(t, strings.HasPrefix(lines[total-1], "2 next"))
			require.Equal(t, value, editor.Value(), "soft wrapping must not insert newlines")
			editor.Scroll(3)
			require.True(t, strings.HasPrefix(ansi.Strip(editor.ScrollView(core.DefaultTheme())), "↪ "))
			editor.Click(editor.numberWidth, 1)
			require.Equal(t, 0, editor.Line(), "clicking a continuation must stay on the logical first line")
			require.Positive(t, editor.Column())
			editor.SetWidth(12)
			_, narrowerTotal := editor.documentView()
			require.Greater(t, narrowerTotal, total)
			require.Equal(t, value, editor.Value())
		})
	}
}

func TestTextAreaSelectAllUsesCtrlOnly(t *testing.T) {
	t.Parallel()
	editor := NewTextArea(core.DefaultTheme())
	editor.SetWidth(30)
	editor.SetHeight(5)
	editor.SetValue("payload")
	for _, mod := range []tea.KeyMod{tea.ModSuper, tea.ModMeta} {
		editor.Update(tea.KeyPressMsg{Code: 'a', Mod: mod})
		require.False(t, editor.HasSelection())
		require.Equal(t, "payload", editor.Value())
	}
	editor.Update(tea.KeyPressMsg{Code: 'a', Mod: tea.ModCtrl})
	require.Equal(t, "payload", editor.SelectedText())
	editor.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	require.Empty(t, editor.Value())
	editor.Update(tea.KeyPressMsg{Code: 'z', Mod: tea.ModCtrl})
	require.Equal(t, "payload", editor.Value())
}

func TestTextAreaHelpKeepsEverydayActions(t *testing.T) {
	t.Parallel()
	editor := NewTextArea(core.DefaultTheme())
	var actions []string
	for _, shortcut := range editor.Shortcuts() {
		actions = append(actions, shortcut.Help().Desc)
	}
	require.Equal(t, []string{"clear", "undo", "redo", "cut", "copy", "paste", "select all", "select text", "line start", "line end"}, actions)
}
