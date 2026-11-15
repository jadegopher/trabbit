package components

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/jadegopher/trabbit/internal/tui/core"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestSubtabsCircularBorder(t *testing.T) {
	t.Parallel()
	tabs := Subtabs{Labels: []string{"Overview", "Publish", "Consumers"}}
	tabs.Move(-1)
	require.Equal(t, 2, tabs.Active)
	tabs.Move(1)
	require.Equal(t, 0, tabs.Active)
	for _, width := range []int{60, 90, 180} {
		offset := 0
		before := tea.NewView(core.DefaultTheme().RenderScrollContainer(width-2, 10, false, &offset, "", 0))
		after := tabs.Decorate(before, core.DefaultTheme())
		require.Equal(t, lipgloss.Width(before.Content), lipgloss.Width(after.Content))
		require.Equal(t, lipgloss.Height(before.Content), lipgloss.Height(after.Content))
		require.Contains(t, ansi.Strip(after.Content), "Overview")
	}
}

func TestSubtabsAbbreviateInactiveLabelsOnNarrowPanels(t *testing.T) {
	t.Parallel()
	theme := core.DefaultTheme()
	tabs := Subtabs{Labels: []string{"Overview", "Consumers", "Bindings", "Runtime metrics"}}
	for active := range tabs.Labels {
		tabs.Active = active
		for _, width := range []int{90, 40, 30, 90} {
			offset := 0
			before := tea.NewView(theme.RenderInsetScrollContainer(width-core.BorderOverhead, 3, false, &offset, "", 0))
			after := tabs.Decorate(before, theme)
			header := strings.Split(ansi.Strip(after.Content), "\n")[0]
			require.Equal(t, width, lipgloss.Width(header))
			require.True(t, strings.HasPrefix(header, "╭"))
			require.True(t, strings.HasSuffix(header, "╮"))
			require.Contains(t, header, tabs.Labels[active])
			for i, label := range tabs.Labels {
				if i == active {
					continue
				}
				if width < 90 {
					require.Contains(t, header, label[:3])
					require.NotContains(t, header, label)
				} else {
					require.Contains(t, header, label)
				}
			}
		}
	}
}
