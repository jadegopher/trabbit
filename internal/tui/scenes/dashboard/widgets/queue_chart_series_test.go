package widgets

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/require"

	"github.com/jadegopher/trabbit/internal/domain"
	"github.com/jadegopher/trabbit/internal/settings"
	"github.com/jadegopher/trabbit/internal/tui/core"
)

func TestSeriesVisibilityRebuildsChartsAcrossRefreshAndResize(t *testing.T) {
	t.Parallel()

	for _, style := range []string{"dotted", "lines"} {
		t.Run(style, func(t *testing.T) {
			t.Parallel()

			q := NewQueueCharts(core.DefaultTheme())
			q.SetFocused(true)
			stats := domain.QueueDetailedStats{
				MessagesReady: []domain.Sample{{Timestamp: 70000, Value: 5}, {Timestamp: 40000, Value: 5}},
				MessagesTotal: []domain.Sample{{Timestamp: 70000, Value: 100000}, {Timestamp: 10000, Value: 100000}},
				MessageStats: domain.MessageStats{
					PublishDetails: domain.RateDetails{Samples: []domain.Sample{{Timestamp: 70000, Value: 30}, {Timestamp: 40000, Value: 0}}},
					GetDetails:     domain.RateDetails{Samples: []domain.Sample{{Timestamp: 70000, Value: 300000}, {Timestamp: 10000, Value: 0}}},
				},
				Policy: strings.Repeat("界policy", 50),
			}
			chosen := settings.DefaultChartSettings()
			chosen.Style = style
			chosen.Messages = map[string]bool{"Total": false}
			chosen.Throughput = map[string]bool{"Get": true, "Publish": false}
			q.Update(MsgChartSettings{Settings: chosen})
			for _, width := range []int{100, 15, 0, 100} {
				q.Update(tea.WindowSizeMsg{Width: width, Height: 12})
				q.Update(MsgQueueDetailedStats{Stats: stats})
				q.Update(tea.KeyPressMsg{Code: tea.KeyEnd})
				if width == 0 {
					require.Empty(t, q.View().Content)
					continue
				}
				view := ansi.Strip(q.View().Content)
				require.Equal(t, 14, lipgloss.Height(view))
				require.Contains(t, view, "▐│")
				rows := strings.Split(view, "\n")
				for i, row := range rows {
					require.Equal(t, width+core.BorderOverhead, lipgloss.Width(row))
					if i > 0 && i < len(rows)-1 {
						require.True(t, strings.HasPrefix(row, "│ "))
						require.True(t, strings.HasSuffix(row, "│"))
					}
				}
			}
			require.Equal(t, float64(40), q.queuedMsgsChart.ViewMinX(), "hidden Total history must not set the time axis")
			require.Less(t, q.queuedMsgsChart.ViewMaxY(), float64(100000))
			require.GreaterOrEqual(t, q.msgRatesChart.ViewMaxY(), float64(5000), "newly enabled Get must be plotted")

			for _, name := range settings.MessageSeries() {
				chosen.Messages[name] = false
			}
			for _, name := range settings.ThroughputSeries() {
				chosen.Throughput[name] = false
			}
			q.Update(MsgChartSettings{Settings: chosen})
			for _, chart := range []string{q.queuedMsgsChart.View(), q.msgRatesChart.View()} {
				require.False(t, strings.ContainsFunc(chart, func(r rune) bool { return r >= '\u2801' && r <= '\u28ff' }))
				require.Contains(t, chart, "─")
			}
			q.Update(MsgChartSettings{Settings: settings.DefaultChartSettings()})
			require.GreaterOrEqual(t, q.queuedMsgsChart.ViewMaxY(), float64(100000))
			require.Less(t, q.msgRatesChart.ViewMaxY(), float64(5000))
		})
	}
}
