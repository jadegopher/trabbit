package core

import (
	tea "charm.land/bubbletea/v2"
	"fmt"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/require"
)

func TestNoticeBoundsAndWrapping(t *testing.T) {
	t.Parallel()
	for _, kind := range []NoticeKind{NoticeInfo, NoticeSuccess, NoticeWarning, NoticeError} {
		for _, width := range []int{0, 1, 2, 29, 90} {
			t.Run(fmt.Sprintf("kind%d/width%d", kind, width), func(t *testing.T) {
				t.Parallel()
				theme := DefaultTheme()
				for _, message := range []string{"", "Published • routed", "Failure: 日本語 " + strings.Repeat("long detail ", 20)} {
					view := theme.RenderNotice(width, 3, message, kind)
					if width == 0 {
						require.Empty(t, view)
						continue
					}
					require.Equal(t, 3, lipgloss.Height(view))
					for _, line := range strings.Split(view, "\n") {
						require.Equal(t, width, lipgloss.Width(line))
					}
					if message == "Published • routed" && width >= 29 {
						require.Contains(t, ansi.Strip(view), message)
					}
				}
			})
		}
	}
}

func TestSingleRowNoticeHasSideBorders(t *testing.T) {
	t.Parallel()
	theme := DefaultTheme()
	view := theme.RenderNotice(29, 1, "Published • routed\nextra detail", NoticeSuccess)
	plain := ansi.Strip(view)
	require.Equal(t, 1, lipgloss.Height(view))
	require.Equal(t, 29, lipgloss.Width(view))
	require.True(t, strings.HasPrefix(plain, "│"))
	require.True(t, strings.HasSuffix(plain, "│"))
	require.Contains(t, plain, "Published • routed")
	require.Contains(t, plain, "…")
}

func TestTopNoticePreservesScreenGeometryAndCursor(t *testing.T) {
	t.Parallel()
	view := tea.NewView("header\nsecond line\nthird line")
	view.Cursor = tea.NewCursor(3, 2)
	result := DefaultTheme().RenderTopNotice(view, 80, 3, "Press Ctrl+C again to quit", NoticeWarning)
	lines := strings.Split(ansi.Strip(result.Content), "\n")
	require.Len(t, lines, 3)
	require.Contains(t, lines[0], "Press Ctrl+C again to quit")
	require.True(t, strings.HasPrefix(lines[1], "second line"))
	require.True(t, strings.HasPrefix(lines[2], "third line"))
	require.Equal(t, view.Cursor, result.Cursor)
	for _, line := range lines {
		require.Equal(t, 80, lipgloss.Width(line))
	}
	view.Cursor = tea.NewCursor(3, 0)
	result = DefaultTheme().RenderTopNotice(view, 80, 3, "Confirm quit", NoticeWarning)
	require.Nil(t, result.Cursor, "notice must hide a cursor on the replaced row")
}
