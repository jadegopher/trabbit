package components

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/require"

	"github.com/jadegopher/trabbit/internal/tui/core"
)

func TestDataTableColumnGrowthUsesOnlySpareViewportSpace(t *testing.T) {
	t.Parallel()

	theme := core.DefaultTheme()
	table := NewDataTable(theme, []DataColumn{
		{Title: "VHost", Width: 5, Grow: 1},
		{Title: "Name", Width: 15, Grow: 2},
		{Title: "Type", Width: 10},
		{Title: "State", Width: 10},
	}, true)
	table.SetFocused(true)
	table.SetRows([]DataRow{{ID: "queue", Cells: []string{"/", "queue", "classic", "running"}, Selectable: true}})
	minimumWidth := table.MinimumWidth()
	for _, size := range []struct {
		width  int
		widths []int
	}{
		{75, []int{13, 33, 10, 10}},
		{76, []int{14, 33, 10, 10}},
		{35, []int{5, 15, 10, 10}},
		{0, []int{5, 15, 10, 10}},
		{75, []int{13, 33, 10, 10}},
	} {
		table.SetSize(size.width, 8)
		require.Equal(t, size.widths, table.viewportColumnWidths())
		require.Equal(t, minimumWidth, table.MinimumWidth(), "growth cannot change the parent layout minimum")
		if size.width == 0 {
			require.Empty(t, table.View().Content)
			continue
		}

		line := table.line(table.rows[0].Cells, table.viewportColumnWidths())
		require.Equal(t, table.contentWidth(), lipgloss.Width(line))
		require.Contains(t, table.View().Content, theme.SelectedStyle.Render(line))
		table.Update(tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModShift})
		if size.width > table.naturalWidth() {
			require.Zero(t, table.horizontal, "growth cannot introduce horizontal scrolling")
		}
	}

	// Styled wide text keeps its natural width and remains available by scrolling.
	name := theme.LabelStyle.Render(strings.Repeat("界", 40) + " END")
	table.SetRows([]DataRow{{ID: "queue", Cells: []string{"/", name, "classic", "running"}, Selectable: true}})
	require.Equal(t, lipgloss.Width(name), table.viewportColumnWidths()[1])
	for range 20 {
		table.Update(tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModShift})
	}
	require.Contains(t, ansi.Strip(table.View().Content), "END")
}

func TestDataTableReadOnlyScrollAndContainment(t *testing.T) {
	t.Parallel()

	theme := core.DefaultTheme()
	table := NewDataTable(theme, []DataColumn{{Title: "Tag", Width: 10}, {Title: "Arguments", Width: 15}}, false)
	rows := make([]DataRow, 40)
	for i := range rows {
		rows[i] = DataRow{Cells: []string{fmt.Sprintf("consumer-%02d", i), theme.LabelStyle.Render(strings.Repeat("界", 40) + " END")}}
	}
	table.SetRows(rows)
	table.SetFocused(true)
	table.SetFooter("page\nhelp")
	for _, width := range []int{100, 30, 0, 100} {
		table.SetSize(width, 8)
		if width == 0 {
			require.Empty(t, table.View().Content)
			continue
		}
		for step := 0; step < 50; step++ {
			table.Update(tea.KeyPressMsg{Code: tea.KeyDown})
			before := table.offset
			view := ansi.Strip(table.View().Content)
			require.Equal(t, before, table.offset, "View cannot change scroll state")
			require.Equal(t, -1, table.Cursor())
			require.Equal(t, width+core.BorderOverhead, lipgloss.Width(view))
			require.Equal(t, 8+core.BorderOverhead, lipgloss.Height(view))
			require.Contains(t, view, "▐│")
			lines := strings.Split(view, "\n")
			require.Contains(t, lines[1], "Tag", "header stays pinned")
			for _, line := range lines {
				require.Equal(t, width+core.BorderOverhead, lipgloss.Width(line))
			}
		}
	}
	for step := 0; step < 50; step++ {
		table.Update(tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModShift})
	}
	require.Contains(t, ansi.Strip(table.View().Content), "END", "horizontal scroll reveals the end of wide styled cells")
	table.SetRows(rows[:1])
	require.Zero(t, table.offset, "refresh clamps a shorter result")
}

func TestDataTableSelectionSkipsProtectedRows(t *testing.T) {
	t.Parallel()

	table := NewDataTable(core.DefaultTheme(), []DataColumn{{Title: "From", Width: 12}}, true)
	table.SetSize(20, 8)
	table.SetFocused(true)
	rows := []DataRow{
		{ID: "default", Cells: []string{"Default"}},
		{ID: "first", Cells: []string{"first"}, Selectable: true},
		{ID: "protected", Cells: []string{"protected"}},
		{ID: "second", Cells: []string{"second"}, Selectable: true},
	}
	table.SetRows(rows)
	require.Equal(t, 1, table.Cursor())
	table.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	require.Equal(t, 1, table.Cursor())
	table.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	require.Equal(t, 3, table.Cursor())
	table.Update(tea.MouseClickMsg{Button: tea.MouseLeft, X: 3, Y: 2})
	require.Equal(t, 3, table.Cursor(), "the default row cannot be selected")
	table.SetRows([]DataRow{rows[3], rows[0], rows[1]})
	require.Zero(t, table.Cursor(), "refresh preserves the selected identity")
	table.SetRows(rows[:1])
	require.Equal(t, -1, table.Cursor())
}

func TestResponsiveDataTableShowsAllFieldsByVerticalScrolling(t *testing.T) {
	t.Parallel()
	theme := core.DefaultTheme()
	table := NewDataTable(theme, []DataColumn{{Title: "Channel", Width: 12}, {Title: "Arguments", Width: 15}}, false)
	table.SetResponsive(true)
	table.SetFocused(true)
	table.SetFooter("1 of 1\ncontrols")
	table.SetRows([]DataRow{{Cells: []string{"channel-1", theme.LabelStyle.Render(strings.Repeat("界", 30) + " END")}}})
	for _, width := range []int{150, 28, 0, 28, 150} {
		table.SetSize(width, 8)
		if width == 0 {
			require.Empty(t, table.View().Content)
			continue
		}
		seen := ""
		for step := 0; step < 40; step++ {
			offset := table.offset
			view := ansi.Strip(table.View().Content)
			require.Equal(t, offset, table.offset)
			require.Equal(t, width+core.BorderOverhead, lipgloss.Width(view))
			require.Equal(t, 8+core.BorderOverhead, lipgloss.Height(view))
			for _, line := range strings.Split(view, "\n") {
				require.Equal(t, width+core.BorderOverhead, lipgloss.Width(line))
				require.True(t, strings.HasSuffix(line, "│") || strings.HasSuffix(line, "╮") || strings.HasSuffix(line, "╯"))
			}
			seen += view
			table.Update(tea.KeyPressMsg{Code: tea.KeyDown})
		}
		require.Contains(t, seen, "END")
		require.Equal(t, -1, table.Cursor())
		table.Update(tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModShift})
		require.Zero(t, table.horizontal)
	}
	table.SetRows(nil)
	require.Zero(t, table.offset)
}

func TestDataTableCentersFooterInsideReservedScrollbarColumn(t *testing.T) {
	t.Parallel()
	table := NewDataTable(core.DefaultTheme(), []DataColumn{{Title: "Tag", Width: 8}}, false)
	table.SetFooter("1 of 2\ncontrols")
	for _, width := range []int{80, 40} {
		table.SetSize(width, 8)
		lines := strings.Split(ansi.Strip(table.View().Content), "\n")
		for _, text := range []string{"1 of 2", "controls"} {
			found := false
			for _, line := range lines {
				if strings.Contains(line, text) {
					left := strings.Index(line, text) - len("│")
					expected := (width - core.ScrollbarColumns - lipgloss.Width(text)) / 2
					require.Equal(t, expected, left)
					found = true
				}
			}
			require.True(t, found)
		}
	}
}

func TestResponsiveSelectionUsesRecordBoundsAndProtectsRows(t *testing.T) {
	t.Parallel()
	table := NewDataTable(core.DefaultTheme(), []DataColumn{{Title: "From", Width: 20}, {Title: "Routing key", Width: 20}, {Title: "Arguments", Width: 30}}, true)
	table.SetResponsive(true)
	table.SetFocused(true)
	table.SetSize(50, 14)
	rows := []DataRow{
		{ID: "default", Cells: []string{"Default", "orders", ""}},
		{ID: "first", Cells: []string{"first", "one", "{}"}, Selectable: true},
		{ID: "second", Cells: []string{"second", "two", "{}"}, Selectable: true},
	}
	table.SetRows(rows)
	require.Equal(t, 1, table.Cursor())
	view := ansi.Strip(table.View().Content)
	require.Contains(t, view, "│╭ From")
	require.Contains(t, view, "││ Routing key")
	require.Contains(t, view, "│╰ Arguments")
	// Click a field inside the third record, then inside the protected first one.
	table.Update(tea.MouseClickMsg{Button: tea.MouseLeft, X: 5, Y: table.recordBounds[2].start + 2})
	require.Equal(t, 2, table.Cursor())
	table.Update(tea.MouseClickMsg{Button: tea.MouseLeft, X: 5, Y: 2})
	require.Equal(t, 2, table.Cursor())
	table.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	require.Equal(t, 1, table.Cursor())
	table.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	require.Equal(t, 1, table.Cursor())
	table.SetRows([]DataRow{rows[2], rows[0], rows[1]})
	require.Equal(t, "first", table.rows[table.Cursor()].ID)
	for _, width := range []int{35, 0, 150, 35} {
		table.SetSize(width, 8)
		if width == 0 {
			require.Empty(t, table.View().Content)
			continue
		}
		rendered := ansi.Strip(table.View().Content)
		require.Equal(t, width+core.BorderOverhead, lipgloss.Width(rendered))
		require.Equal(t, 8+core.BorderOverhead, lipgloss.Height(rendered))
		for _, line := range strings.Split(rendered, "\n") {
			require.Equal(t, width+core.BorderOverhead, lipgloss.Width(line))
		}
	}
}

func TestResponsiveSelectionScrollsLongArgumentsAndPreservesRefreshPosition(t *testing.T) {
	t.Parallel()
	table := NewDataTable(core.DefaultTheme(), []DataColumn{{Title: "From", Width: 20}, {Title: "Arguments", Width: 40}}, true)
	table.SetResponsive(true)
	table.SetFocused(true)
	table.SetSize(32, 6)
	rows := []DataRow{{ID: "one", Cells: []string{"events", strings.Repeat("界", 60) + " END"}, Selectable: true}}
	table.SetRows(rows)
	seen := ""
	for step := 0; step < 15; step++ {
		table.Update(tea.KeyPressMsg{Code: tea.KeyPgDown})
		offset := table.offset
		seen += ansi.Strip(table.View().Content)
		table.SetRows(rows)
		require.Equal(t, offset, table.offset)
		require.Equal(t, 0, table.Cursor())
	}
	require.Contains(t, seen, "END")
	require.Zero(t, table.horizontal)
}

func TestDataTableWrappedSelectionSkipsProtectedRowsAndStopsWheel(t *testing.T) {
	t.Parallel()

	table := NewDataTable(core.DefaultTheme(), []DataColumn{{Title: "Name", Width: 10}}, true)
	table.SetWrapNavigation(true)
	table.SetFocused(true)
	table.SetSize(20, 8)
	table.SetRows([]DataRow{
		{ID: "protected"},
		{ID: "first", Cells: []string{"first"}, Selectable: true},
		{ID: "last", Cells: []string{"last"}, Selectable: true},
		{ID: "protected-end"},
	})
	table.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	require.Equal(t, 2, table.Cursor())
	table.Update(tea.MouseWheelMsg{Button: tea.MouseWheelDown})
	require.Equal(t, 2, table.Cursor())
	table.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	require.Equal(t, 1, table.Cursor())
	table.Update(tea.MouseWheelMsg{Button: tea.MouseWheelUp})
	require.Equal(t, 1, table.Cursor())
	table.SetRows([]DataRow{{ID: "protected"}})
	table.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	require.Equal(t, -1, table.Cursor())
	table.SetRows(nil)
	table.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	require.Equal(t, -1, table.Cursor())
}
