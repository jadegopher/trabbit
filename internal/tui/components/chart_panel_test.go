package components

import (
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/NimbleMarkets/ntcharts/v2/linechart/timeserieslinechart"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/require"

	"github.com/jadegopher/trabbit/internal/tui/core"
)

func TestChartPanelContainsStyledContentAndScrollbar(t *testing.T) {
	t.Parallel()

	p := NewChartPanel(core.DefaultTheme())
	styled := lipgloss.NewStyle().Foreground(lipgloss.Color("#ff6600")).Render(strings.Repeat("界⣀", 80))
	content := strings.Repeat(styled+"\n", 20)
	for _, width := range []int{80, 17, 3, 2, 1, 0, 80} {
		p.SetSize(width, 12)
		for _, offset := range []int{-1, 0, 5, 10000} {
			before := p
			view := ansi.Strip(p.Render(content, offset, true))
			require.Equal(t, before, p, "rendering must not mutate the panel")
			if width == 0 {
				require.Empty(t, view)
				continue
			}

			lines := strings.Split(view, "\n")
			require.Len(t, lines, 14)
			for row, line := range lines {
				require.Equal(t, width+2, lipgloss.Width(line))
				if row == 0 || row == len(lines)-1 {
					continue
				}
				require.True(t, strings.HasPrefix(line, "│"))
				require.True(t, strings.HasSuffix(line, "│"))
				if width > 3 {
					require.True(t, strings.HasPrefix(line, "│ "))
					require.True(t, strings.HasSuffix(line, "  │") || strings.HasSuffix(line, " ▐│"))
				}
			}
			if width > 3 {
				require.Contains(t, view, "▐│")
			}
		}
	}
}

func TestChartPanelClampsToWrappedContent(t *testing.T) {
	t.Parallel()

	p := NewChartPanel(core.DefaultTheme())
	p.SetSize(13, 3)
	content := strings.Repeat("x", 100)
	require.Equal(t, 7, p.ClampOffset(1000, content))
	require.Equal(t, 0, p.ClampOffset(-5, content))
	require.Contains(t, ansi.Strip(p.Render(content, 1000, false)), "▐│\n╰")

	p.SetSize(103, 3)
	require.Zero(t, p.ClampOffset(7, content))
	p.SetSize(0, 0)
	require.Zero(t, p.ClampOffset(7, content))
	require.Empty(t, p.Render(content, 7, false))
}

func TestChartPanelDrawsAcrossResizeAndHiddenStates(t *testing.T) {
	t.Parallel()

	p := NewChartPanel(core.DefaultTheme())
	p.SetSize(83, 20)
	chart := p.NewChart(timeserieslinechart.WithTimeRange(time.Unix(10, 0), time.Unix(70, 0)))
	chart.Push(timeserieslinechart.TimePoint{Time: time.Unix(10, 0), Value: 60000})
	chart.Push(timeserieslinechart.TimePoint{Time: time.Unix(70, 0), Value: 60000})
	for _, width := range []int{83, 0, 1, 3, 5, 10, 83} {
		p.SetSize(width, 20)
		require.NotPanics(t, func() { p.DrawChart(&chart) })
		if width > 3 {
			require.Equal(t, width-3, lipgloss.Width(chart.View()))
			require.Equal(t, 9, lipgloss.Height(chart.View()))
		}
	}

	require.True(t, strings.ContainsFunc(chart.View(), func(r rune) bool {
		return r >= '\u2801' && r <= '\u28ff'
	}), "restoring the panel must redraw retained samples")
}
