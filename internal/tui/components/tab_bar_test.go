package components

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/jadegopher/trabbit/internal/tui/core"
	"github.com/stretchr/testify/require"
)

func TestTabBarBindingsAndLabels(t *testing.T) {
	t.Parallel()
	bar := TabBar{Tabs: []core.Shortcut{
		core.TabNumber(1, "Overview"),
		core.TabNumber(5, "Queues"),
	}, Active: 1}
	require.Equal(t, "[1]Overview [5]Queues", ansi.Strip(bar.View(core.DefaultTheme())))
	index, matched := bar.Match(tea.KeyPressMsg{Code: '5', Text: "5"})
	require.True(t, matched)
	require.Equal(t, 1, index)
	_, matched = bar.Match(tea.KeyPressMsg{Code: 'x', Text: "x"})
	require.False(t, matched)
}
