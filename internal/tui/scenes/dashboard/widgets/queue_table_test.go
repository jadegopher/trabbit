package widgets

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"fmt"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"

	"github.com/jadegopher/trabbit/internal/domain"
	"github.com/jadegopher/trabbit/internal/tui/components"
	"github.com/jadegopher/trabbit/internal/tui/core"
)

func TestTableWheelStopsAtBoundaries(t *testing.T) {
	t.Parallel()
	tbl := NewTable(core.DefaultTheme())
	tbl.setSize(80, 8)
	queues := make(domain.Queues, 20)
	for i := range queues {
		queues[i].Name = strings.Repeat("q", i+1)
	}
	tbl.Update(MsgQueues{Queues: queues})
	top := tbl.View()
	for range 3 {
		_, cmd := tbl.Update(tea.MouseWheelMsg{Button: tea.MouseWheelUp})
		require.Nil(t, cmd)
		require.Equal(t, top, tbl.View())
	}
	for range len(queues) - 1 {
		_, cmd := tbl.Update(tea.MouseWheelMsg{Button: tea.MouseWheelDown})
		require.NotNil(t, cmd, "moving to another queue must still emit selection changes")
	}
	require.Equal(t, len(queues)-1, tbl.table.Cursor())
	bottom := tbl.View()
	for range 3 {
		_, cmd := tbl.Update(tea.MouseWheelMsg{Button: tea.MouseWheelDown})
		require.Nil(t, cmd, "blocked scrolling must not request another selection refresh")
		require.Equal(t, bottom, tbl.View())
	}
}

func TestTableWrapsBothDirections(t *testing.T) {
	t.Parallel()

	tbl := NewTable(core.DefaultTheme())
	tbl.setSize(80, 20)
	tbl.Update(MsgQueues{Queues: domain.Queues{{Name: "first"}, {Name: "last"}}})
	tbl.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	require.Equal(t, "last", tbl.ActiveQueue().Name)
	tbl.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	require.Equal(t, "first", tbl.ActiveQueue().Name)
	tbl.SetLoading(1)
	require.False(t, tbl.loader.IsActive(), "fast refresh must retain rows")
}

func TestTableAlignment(t *testing.T) {
	t.Parallel()

	theme := core.DefaultTheme()
	tbl := NewTable(theme, WithPagination(1000))
	tbl.setSize(77, 20)

	queues := domain.Queues{
		{
			VHost: "/",
			Name:  "test-queue",
			Type:  "classic",
			State: "running",
			Messages: domain.Messages{
				Ready:   10,
				Unacked: 5,
				Total:   15,
			},
		},
	}

	_, _ = tbl.Update(
		MsgQueues{
			Queues: queues,
			Pagination: domain.Pagination{
				Page:          1,
				PageSize:      3,
				TotalCount:    1,
				FilteredCount: 1,
			},
		},
	)

	view := tbl.View()
	core.AssertGolden(t, "queue_table_alignment", view.Content)
}

func TestQueueTableColumnsFillPanelOnResize(t *testing.T) {
	t.Parallel()

	tbl := NewTable(core.DefaultTheme())
	tbl.Update(MsgQueues{Queues: domain.Queues{{VHost: "/", Name: "stu-queue", Type: "classic", State: "running"}}})
	for _, width := range []int{77, 100, 51, 0, 77} {
		tbl.setSize(width, 12)
		if width == 0 {
			require.Empty(t, tbl.View().Content)
			continue
		}

		lines := strings.Split(ansi.Strip(tbl.View().Content), "\n")
		// Only the fixed state column, scrollbar and right border follow State.
		stateOffset := width - core.BorderOverhead - core.ScrollbarColumns - colStateWidth
		require.Equal(t, stateOffset, lipgloss.Width(strings.Split(lines[1], colStateTitle)[0]))
		require.Equal(t, stateOffset, lipgloss.Width(strings.Split(lines[2], "running")[0]))
		for _, line := range lines {
			require.Equal(t, width, lipgloss.Width(line))
		}
	}
}

func TestQueueTableTruncatesLongNamesWithoutHorizontalScrolling(t *testing.T) {
	t.Parallel()

	theme := core.DefaultTheme()
	tbl := NewTable(theme)
	for _, name := range []string{strings.Repeat("long-queue", 20), strings.Repeat("界", 100), theme.LabelStyle.Render(strings.Repeat("styled-name", 20))} {
		tbl.Update(MsgQueues{Queues: domain.Queues{{VHost: "/", Name: name, Type: "classic", State: "running"}, {Name: "test-queue"}}})
		for _, width := range []int{77, 100, 51, 35, 0, 77} {
			tbl.setSize(width, 12)
			view := tbl.View()
			for _, keyCode := range []rune{tea.KeyRight, tea.KeyLeft} {
				tbl.Update(tea.KeyPressMsg{Code: keyCode, Mod: tea.ModShift})
				require.Equal(t, view, tbl.View(), "Shift+arrows must not scroll the queue table")
			}
			require.Equal(t, name, tbl.ActiveQueue().Name, "truncation must preserve the queue identity")
			if width == 0 {
				require.Empty(t, view.Content)
				continue
			}

			lines := strings.Split(ansi.Strip(view.Content), "\n")
			require.Contains(t, lines[2], "…")
			if width >= tbl.TotalWidth() {
				require.Contains(t, lines[1], "Type")
				require.Contains(t, lines[1], "State")
				require.Contains(t, lines[2], "classic")
				require.Contains(t, lines[2], "running")
			}
			for _, line := range lines {
				require.Equal(t, width, lipgloss.Width(line))
			}
		}
	}
}

func TestTableOptionalPagination(t *testing.T) {
	t.Parallel()

	plain := NewTable(core.DefaultTheme())
	paged := NewTable(core.DefaultTheme(), WithPagination(25))
	for _, tbl := range []*Table{plain, paged} {
		tbl.setSize(77, 20)
		tbl.Update(MsgQueues{Pagination: domain.Pagination{Page: 1, PageSize: 3, FilteredCount: 10}})
	}
	require.NotContains(t, plain.View().Content, "GoTo")
	require.Contains(t, paged.View().Content, "GoTo")
	require.Equal(t, lipgloss.Height(plain.View().Content), lipgloss.Height(paged.View().Content))
	_, cmd := plain.Update(tea.KeyPressMsg{Code: 'g', Text: "g"})
	require.Nil(t, cmd)
	_, cmd = paged.Update(tea.KeyPressMsg{Code: 'g', Text: "g"})
	require.NotNil(t, cmd)
	require.IsType(t, &components.FormModal{}, cmd().(core.OpenDialog).Model)
}

func TestTableScrollbarStaysInsideBorder(t *testing.T) {
	t.Parallel()
	tbl := NewTable(core.DefaultTheme())
	queues := make(domain.Queues, 100)
	for i := range queues {
		queues[i].Name = strings.Repeat("界", 50)
	}
	tbl.Update(MsgQueues{Queues: queues})
	for _, width := range []int{100, 45, 100} {
		tbl.setSize(width, 12)
		for step := 0; step < 100; step++ {
			tbl.Update(tea.KeyPressMsg{Code: tea.KeyDown})
			view := ansi.Strip(tbl.View().Content)
			require.Contains(t, view, "▐│")
			lines := strings.Split(view, "\n")
			require.Len(t, lines, 12)
			for row, line := range lines {
				require.Equal(t, width, lipgloss.Width(line))
				if row > 0 && row < len(lines)-1 {
					require.True(t, strings.HasPrefix(line, "│"))
					require.True(t, strings.HasSuffix(line, "│"))
				}
			}
		}
	}
}

func TestQueueTotalPreservesFilterAcrossRefreshAndClearing(t *testing.T) {
	t.Parallel()
	theme := core.DefaultTheme()
	tbl := NewTable(theme, WithPagination(1000))
	tbl.setSize(77, 20)
	filter := theme.LabelFocused.Render("◆ Filter active")
	tbl.SetFilter(filter)
	tbl.Update(MsgQueues{RequestID: 1, Pagination: domain.Pagination{Page: 1, PageCount: 1, TotalCount: 5, FilteredCount: 1}})
	require.Contains(t, ansi.Strip(tbl.View().Content), "1 of 1 ⚬ Total: 1 ⚬ ◆ Filter active")
	tbl.Update(MsgQueues{RequestID: 2, Pagination: domain.Pagination{Page: 2, PageCount: 2, TotalCount: 43, FilteredCount: 4}})
	require.Contains(t, ansi.Strip(tbl.View().Content), "2 of 2 ⚬ Total: 4 ⚬ ◆ Filter active")
	tbl.Update(MsgQueues{RequestID: 1, Pagination: domain.Pagination{TotalCount: 99, FilteredCount: 99}})
	require.Contains(t, ansi.Strip(tbl.View().Content), "Total: 4 ⚬ ◆ Filter active")
	tbl.Update(MsgQueues{RequestID: 3, Pagination: domain.Pagination{Page: 1, PageCount: 0, TotalCount: 43, FilteredCount: 0}})
	require.Contains(t, ansi.Strip(tbl.View().Content), "1 of 1 ⚬ Total: 0 ⚬ ◆ Filter active")
	tbl.SetFilter("")
	view := ansi.Strip(tbl.View().Content)
	require.Contains(t, view, "Total: 43")
	require.NotContains(t, view, "Filter active")
}

func TestQueueTablePinsPaginationThroughScrollingLoadingAndResizing(t *testing.T) {
	t.Parallel()

	tbl := NewTable(core.DefaultTheme(), WithPagination(1000))
	queues := make(domain.Queues, 100)
	for i := range queues {
		queues[i] = domain.Queue{VHost: "/", Name: fmt.Sprintf("queue-%03d", i), Type: "classic", State: "running"}
	}
	result := MsgQueues{RequestID: 1, Queues: queues, Pagination: domain.Pagination{Page: 1, PageCount: 2, TotalCount: 100, FilteredCount: 100}}
	tbl.Update(result)
	for _, width := range []int{77, 51, 35, 0, 77} {
		tbl.setSize(width, 12)
		if width == 0 {
			require.Empty(t, tbl.View().Content)
			continue
		}
		assertFooter := func() {
			t.Helper()
			view := ansi.Strip(tbl.View().Content)
			lines := strings.Split(view, "\n")
			require.Len(t, lines, 12)
			require.Contains(t, lines[len(lines)-3], "1 of 2 ⚬ Total: 100")
			require.Contains(t, lines[len(lines)-2], "g GoTo")
			require.NotContains(t, lines[len(lines)-3], "Loading")
			for _, line := range lines {
				require.Equal(t, width, lipgloss.Width(line))
			}
		}
		for range len(queues) {
			tbl.Update(tea.KeyPressMsg{Code: tea.KeyDown})
			assertFooter()
		}
		tbl.SetLoading(2)
		tbl.ShowLoading()
		assertFooter()
		tbl.FinishLoading()
		assertFooter()
		tbl.Update(result)
		assertFooter()
	}
}

func TestQueueTableRefreshPreservesSelectedIdentity(t *testing.T) {
	t.Parallel()

	tbl := NewTable(core.DefaultTheme())
	first := domain.Queue{VHost: "/", Name: "first"}
	second := domain.Queue{VHost: "/", Name: "second"}
	tbl.Update(MsgQueues{RequestID: 1, Queues: domain.Queues{first, second}})
	tbl.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	_, cmd := tbl.Update(MsgQueues{RequestID: 2, Queues: domain.Queues{second, first}})
	require.Equal(t, "second", tbl.ActiveQueue().Name)
	require.Nil(t, cmd, "reordering the same selection must not schedule another detail request")
	_, cmd = tbl.Update(MsgQueues{RequestID: 3, Queues: domain.Queues{first}})
	require.Equal(t, "first", tbl.ActiveQueue().Name)
	require.Equal(t, MsgQueueSelectionChanged{VHost: "/", Name: "first"}, cmd())
}
