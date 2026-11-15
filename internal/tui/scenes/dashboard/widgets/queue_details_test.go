package widgets

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
	"time"

	"github.com/jadegopher/trabbit/internal/domain"
	"github.com/jadegopher/trabbit/internal/settings"
	"github.com/jadegopher/trabbit/internal/tui/core"
)

func TestChartRangesUseVisiblePoints(t *testing.T) {
	t.Parallel()

	q := NewQueueCharts(core.DefaultTheme())
	q.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	samples := []domain.Sample{{Timestamp: 70000, Value: 100}, {Timestamp: 65000, Value: 90}, {Timestamp: 10000, Value: 10}}
	stats := domain.QueueDetailedStats{MessagesTotal: samples, MessageStats: domain.MessageStats{PublishDetails: domain.RateDetails{Samples: samples}}}
	q.Update(MsgQueueDetailedStats{Stats: stats})
	require.Equal(t, float64(10), q.queuedMsgsChart.ViewMinX())
	require.Equal(t, float64(70), q.queuedMsgsChart.ViewMaxX())
	require.Equal(t, float64(65), q.msgRatesChart.ViewMinX())
	require.Equal(t, float64(70), q.msgRatesChart.ViewMaxX())
	view := q.View().Content
	require.Equal(t, 32, lipgloss.Height(view))
	for _, line := range strings.Split(view, "\n") {
		require.Equal(t, 102, lipgloss.Width(line))
	}
}

func TestQueueChartsPreserveRowsWhileScrolling(t *testing.T) {
	t.Parallel()

	const panelWidth, panelHeight = 100, 12
	q := NewQueueCharts(core.DefaultTheme())
	q.SetFocused(true)
	q.Update(tea.WindowSizeMsg{Width: panelWidth, Height: panelHeight})
	samples := []domain.Sample{{Timestamp: 70000, Value: 120}, {Timestamp: 40000, Value: 60}, {Timestamp: 10000, Value: 30}}
	q.Update(MsgQueueDetailedStats{Stats: domain.QueueDetailedStats{
		MessagesTotal: samples, MessagesReady: samples,
		MessageStats: domain.MessageStats{PublishDetails: domain.RateDetails{Samples: samples}},
	}})

	originalView := q.View().Content
	queuedChart, ratesChart := q.queuedMsgsChart.View(), q.msgRatesChart.View()
	content := strings.Split(ansi.Strip(q.content()), "\n")
	maxOffset := q.panel.ClampOffset(len(content), q.content())
	require.Positive(t, maxOffset)

	checkRows := func() {
		t.Helper()

		rows := strings.Split(ansi.Strip(q.View().Content), "\n")
		require.Len(t, rows, panelHeight+core.BorderOverhead)
		for row := range panelHeight {
			// Compare only content cells; borders, side gaps and the scrollbar
			// stay fixed while rows move through the viewport.
			visible := ansi.Cut(rows[row+1], 2, 2+q.panel.ContentWidth())
			expected := lipgloss.PlaceHorizontal(q.panel.ContentWidth(), lipgloss.Left, content[q.scrollOffset+row])
			require.Equal(t, expected, visible, "offset %d, row %d", q.scrollOffset, row)
			require.Equal(t, panelWidth+core.BorderOverhead, lipgloss.Width(rows[row+1]))
		}
		require.Equal(t, queuedChart, q.queuedMsgsChart.View())
		require.Equal(t, ratesChart, q.msgRatesChart.View())
	}
	checkRows()
	for range maxOffset {
		q.Update(tea.MouseWheelMsg{Button: tea.MouseWheelDown})
		checkRows()
	}
	require.Equal(t, maxOffset, q.scrollOffset)
	for range maxOffset {
		q.Update(tea.MouseWheelMsg{Button: tea.MouseWheelUp})
		checkRows()
	}
	require.Zero(t, q.scrollOffset)
	require.Equal(t, originalView, q.View().Content)
}

func TestQueueChartsSurviveHiddenAndNarrowPanels(t *testing.T) {
	t.Parallel()

	q := NewQueueCharts(core.DefaultTheme())
	samples := []domain.Sample{{Timestamp: 70000, Value: 60000}, {Timestamp: 40000, Value: 30000}, {Timestamp: 10000, Value: 10000}}
	stats := domain.QueueDetailedStats{
		MessagesTotal: samples, MessagesReady: samples, MessagesUnacked: samples,
		MessageStats: domain.MessageStats{PublishDetails: domain.RateDetails{Samples: samples}},
	}

	// Stats can arrive before the first size message, or while the panel is hidden.
	q.Update(MsgQueueDetailedStats{Stats: stats})
	for _, width := range []int{100, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 15, 100} {
		q.Update(tea.WindowSizeMsg{Width: width, Height: 20})
		q.Update(MsgQueueDetailedStats{Stats: stats})

		if width == 0 {
			require.Empty(t, q.View().Content)
			continue
		}

		for _, line := range strings.Split(q.View().Content, "\n") {
			require.Equal(t, width+2, lipgloss.Width(line), "panel width %d", width)
		}
	}

	// Restoring space must redraw the retained history, rather than clearing it.
	require.Contains(t, q.queuedMsgsChart.View(), "⣀")
	require.Equal(t, float64(10), q.queuedMsgsChart.ViewMinX())
	require.Equal(t, float64(70), q.queuedMsgsChart.ViewMaxX())
	require.Equal(t, float64(40), q.msgRatesChart.ViewMinX())
	require.Equal(t, float64(70), q.msgRatesChart.ViewMaxX())
}

func TestRateChartWithOneRatePoint(t *testing.T) {
	t.Parallel()

	q := NewQueueCharts(core.DefaultTheme())
	q.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	q.Update(MsgQueueDetailedStats{Stats: domain.QueueDetailedStats{MessageStats: domain.MessageStats{
		PublishDetails: domain.RateDetails{Samples: []domain.Sample{{Timestamp: 70000, Value: 100}, {Timestamp: 65000, Value: 0}}},
	}}})
	require.Less(t, q.msgRatesChart.ViewMinX(), q.msgRatesChart.ViewMaxX())
	require.Equal(t, float64(70), q.msgRatesChart.ViewMaxX())
}

func TestQueueOverviewStartsWithIdentity(t *testing.T) {
	t.Parallel()

	q := NewQueueCharts(core.DefaultTheme())
	q.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	q.Update(MsgQueueDetailedStats{Stats: domain.QueueDetailedStats{
		Name: "orders", VHost: "/", Type: "quorum", State: "running",
	}})
	content := ansi.Strip(q.content())
	require.True(t, strings.HasPrefix(content, "Queue: orders\nVhost: /  Type: quorum  State: running\n"))
	require.Contains(t, ansi.Strip(q.View().Content), "Queue: orders")

	q.SetLoading("next-vhost", "next-queue")
	content = ansi.Strip(q.content())
	require.Contains(t, content, "Queue: next-queue\nVhost: next-vhost")
	require.NotContains(t, content, "quorum")
}

func TestEmptyThroughputRetainsAxesAcrossRefreshAndResize(t *testing.T) {
	t.Parallel()

	for _, style := range []string{"dotted", "lines"} {
		t.Run(style, func(t *testing.T) {
			t.Parallel()

			q := NewQueueCharts(core.DefaultTheme())
			q.Update(MsgChartSettings{Settings: settings.ChartSettings{Style: style, Size: "medium", TimeRange: "1m"}})
			samples := []domain.Sample{{Timestamp: 70000, Value: 100}, {Timestamp: 65000, Value: 90}}
			populated := domain.QueueDetailedStats{MessagesTotal: samples, MessageStats: domain.MessageStats{
				PublishDetails: domain.RateDetails{Samples: samples},
			}}
			for _, width := range []int{100, 0, 5, 15, 100} {
				q.Update(tea.WindowSizeMsg{Width: width, Height: 30})
				q.Update(MsgQueueDetailedStats{Stats: populated})
				for _, empty := range []domain.QueueDetailedStats{{}, {MessagesTotal: samples}} {
					q.Update(MsgQueueDetailedStats{Stats: empty})
					chart := ansi.Strip(q.msgRatesChart.View())
					require.False(t, strings.ContainsFunc(chart, func(r rune) bool { return r >= '\u2801' && r <= '\u28ff' }))
					if width >= 15 {
						require.Contains(t, chart, "│")
						require.Contains(t, chart, "─")
						require.Equal(t, width-3, lipgloss.Width(chart))
						if len(empty.MessagesTotal) > 0 {
							require.Equal(t, float64(65), q.msgRatesChart.ViewMinX())
							require.Equal(t, float64(70), q.msgRatesChart.ViewMaxX())
						}
					}
					if width == 0 {
						require.Empty(t, q.View().Content)
						continue
					}
					view := ansi.Strip(q.View().Content)
					require.Equal(t, 32, lipgloss.Height(view))
					for row, line := range strings.Split(view, "\n") {
						require.Equal(t, width+core.BorderOverhead, lipgloss.Width(line))
						if row > 0 && row < 31 {
							require.True(t, strings.HasPrefix(line, "│"))
							require.True(t, strings.HasSuffix(line, "│"))
						}
					}
				}
			}
		})
	}
}

func TestChartsSpanPlotAndStayInsideScrollablePanel(t *testing.T) {
	t.Parallel()

	for _, width := range []int{40, 100, 220} {
		q := NewQueueCharts(core.DefaultTheme())
		q.Update(tea.WindowSizeMsg{Width: width, Height: 20})
		samples := []domain.Sample{{Timestamp: 70000, Value: 100}, {Timestamp: 40000, Value: 100}, {Timestamp: 10000, Value: 100}}
		counters := []domain.Sample{{Timestamp: 70000, Value: 6000}, {Timestamp: 40000, Value: 3000}, {Timestamp: 10000, Value: 0}}
		q.Update(MsgQueueDetailedStats{Stats: domain.QueueDetailedStats{
			MessagesTotal: samples,
			MessageStats: domain.MessageStats{
				PublishDetails: domain.RateDetails{Samples: counters},
				GetDetails:     domain.RateDetails{Samples: []domain.Sample{{Timestamp: 80000, Value: 10}, {Timestamp: 1000, Value: 0}}},
			},
		}})

		for _, chart := range []string{q.queuedMsgsChart.View(), q.msgRatesChart.View()} {
			left, right := false, false
			for _, line := range strings.Split(ansi.Strip(chart), "\n") {
				require.Equal(t, width-3, lipgloss.Width(line))
				runes := []rune(line)
				if axis := strings.IndexRune(line, '│'); axis >= 0 {
					left = left || (runes[axis+1] >= '\u2801' && runes[axis+1] <= '\u28ff')
				}
				right = right || (runes[len(runes)-1] >= '\u2801' && runes[len(runes)-1] <= '\u28ff')
			}
			require.True(t, left, "plot should start beside the Y axis at width %d", width)
			require.True(t, right, "plot should reach the final content column at width %d", width)
		}

		for _, offset := range []int{0, 10, 1000} {
			q.scrollOffset = offset
			lines := strings.Split(ansi.Strip(q.View().Content), "\n")
			require.Len(t, lines, 22)
			require.Contains(t, strings.Join(lines, "\n"), "▐")
			for i, line := range lines {
				require.Equal(t, width+2, lipgloss.Width(line))
				if i > 0 && i < len(lines)-1 {
					require.True(t, strings.HasPrefix(line, "│ "))
					require.True(t, strings.HasSuffix(line, "  │") || strings.HasSuffix(line, " ▐│"))
					require.True(t, strings.HasSuffix(line, "│"))
				}
			}
		}
	}
}

func TestQueueDetails_Formatting(t *testing.T) {
	t.Parallel()

	q := &QueueDetails{}

	t.Run(
		"indent", func(t *testing.T) {
			t.Parallel()
			s := "line1\nline2"
			got := q.indent(s, 2)
			want := "  line1\n  line2"
			if got != want {
				t.Errorf("got %q, want %q", got, want)
			}
		},
	)

	t.Run(
		"formatMap", func(t *testing.T) {
			t.Parallel()
			m := map[string]interface{}{
				"key1": "value1",
				"key2": 123,
				"key3": []string{"a", "b"},
			}
			got := q.formatMap(m)
			// sort.Strings(lines) ensures deterministic output
			wantLines := []string{
				"key1: \"value1\"",
				"key2: 123",
				"key3: [\"a\",\"b\"]",
			}
			want := strings.Join(wantLines, "\n")
			if got != want {
				t.Errorf("got %q, want %q", got, want)
			}
		},
	)

	t.Run(
		"formatMap empty", func(t *testing.T) {
			t.Parallel()
			got := q.formatMap(nil)
			want := "none"
			if got != want {
				t.Errorf("got %q, want %q", got, want)
			}
		},
	)

	t.Run(
		"renderMessages", func(t *testing.T) {
			t.Parallel()
			q := &QueueDetails{
				theme: core.DefaultTheme(),
				lastStats: domain.QueueDetailedStats{
					Messages: domain.Messages{
						Total:           1234,
						TotalBytes:      1024 * 1024,
						Ready:           1000,
						ReadyBytes:      800 * 1024,
						Unacked:         234,
						UnackedBytes:    224 * 1024,
						RAM:             1234,
						RAMBytes:        1024 * 1024,
						Persistent:      0,
						PersistentBytes: 0,
					},
				},
			}
			got := q.renderMessages()
			// Check if it contains expected headers and some values
			if !strings.Contains(got, "Messages") {
				t.Errorf("expected title 'Messages', got %q", got)
			}
			if !strings.Contains(got, "Count") || !strings.Contains(got, "Size") {
				t.Errorf("expected headers 'Count' and 'Size', got %q", got)
			}
			if !strings.Contains(got, "Total") || !strings.Contains(got, "1.2k") {
				t.Errorf("expected 'Total' and formatted count '1.2k', got %q", got)
			}
			if !strings.Contains(got, "1.0 MB") {
				t.Errorf("expected formatted bytes '1.0 MB', got %q", got)
			}
		},
	)
}

func TestCalculateRatePoints(t *testing.T) {
	t.Parallel()

	q := &QueueDetails{}

	t.Run(
		"empty samples", func(t *testing.T) {
			t.Parallel()

			points := q.calculateRatePoints(nil)
			if len(points) != 0 {
				t.Errorf("expected 0 points, got %d", len(points))
			}
		},
	)

	t.Run(
		"single sample", func(t *testing.T) {
			t.Parallel()

			samples := []domain.Sample{{Timestamp: 1000, Value: 100}}
			points := q.calculateRatePoints(samples)
			if len(points) != 0 {
				t.Errorf("expected 0 points, got %d", len(points))
			}
		},
	)

	t.Run(
		"normal rates", func(t *testing.T) {
			t.Parallel()

			now := time.Now().UnixMilli()
			samples := []domain.Sample{
				{Timestamp: now, Value: 4000},         // rate 150 from previous
				{Timestamp: now - 10000, Value: 2500}, // rate 150 from previous
				{Timestamp: now - 20000, Value: 1000}, // rate 100 from previous
				{Timestamp: now - 30000, Value: 0},
			}
			points := q.calculateRatePoints(samples)
			if len(points) != 3 {
				t.Fatalf("expected 3 points, got %d", len(points))
			}

			// Points should be returned in chronological order (oldest first)
			// S4 (now-30000) -> S3 (now-20000): rate 100
			// S3 (now-20000) -> S2 (now-10000): rate 150
			// S2 (now-10000) -> S1 (now): rate 150
			if points[0].Value != 100 {
				t.Errorf("expected rate 100, got %f", points[0].Value)
			}
			if points[1].Value != 150 {
				t.Errorf("expected rate 150, got %f", points[1].Value)
			}
			if points[2].Value != 150 {
				t.Errorf("expected rate 150, got %f", points[2].Value)
			}
		},
	)

	t.Run(
		"all samples are included", func(t *testing.T) {
			t.Parallel()

			now := time.Now().UnixMilli()
			samples := []domain.Sample{
				{Timestamp: now, Value: 3000},
				{Timestamp: now - 10000, Value: 2000}, // rate 100 from previous
				{Timestamp: now - 20000, Value: 1000},
				{Timestamp: now - 30000, Value: 0},
			}
			points := q.calculateRatePoints(samples)
			if len(points) != 3 {
				t.Fatalf("expected 3 points, got %d", len(points))
			}

			if points[1].Value != 100 {
				t.Errorf("expected second point to be 100, got %f", points[1].Value)
			}
			if points[2].Value != 100 {
				t.Errorf("expected third point to be 100, got %f", points[2].Value)
			}
		},
	)

	t.Run(
		"counter reset", func(t *testing.T) {
			t.Parallel()

			now := time.Now().UnixMilli()
			samples := []domain.Sample{
				{Timestamp: now, Value: 500},
				{Timestamp: now - 10000, Value: 1000},
				{Timestamp: now - 20000, Value: 0},
				{Timestamp: now - 30000, Value: 0},
			}
			points := q.calculateRatePoints(samples)
			if len(points) != 3 {
				t.Fatalf("expected 3 points, got %d", len(points))
			}
			if points[0].Value != 0 {
				t.Errorf("expected rate 0, got %f", points[0].Value)
			}
			if points[1].Value != 100 {
				t.Errorf("expected rate 100, got %f", points[1].Value)
			}
			if points[2].Value != 0 {
				t.Errorf("expected rate 0 (reset), got %f", points[2].Value)
			}
		},
	)
}

func TestChartPresetsRebuildAndClampPopulatedPanels(t *testing.T) {
	t.Parallel()

	for _, style := range []string{"dotted", "lines"} {
		for size, height := range map[string]int{"small": 6, "medium": 9, "large": 15} {
			t.Run(style+"/"+size, func(t *testing.T) {
				t.Parallel()

				q := NewQueueCharts(core.DefaultTheme())
				q.SetFocused(true)
				samples := []domain.Sample{{Timestamp: 3600000, Value: 120}, {Timestamp: 3570000, Value: 60}, {Timestamp: 3540000, Value: 30}, {Timestamp: 1000, Value: 0}}
				stats := domain.QueueDetailedStats{MessagesTotal: samples, MessageStats: domain.MessageStats{PublishDetails: domain.RateDetails{Samples: samples}}, Policy: strings.Repeat("界policy", 80)}
				for _, span := range []string{"1m", "10m", "30m", "1h"} {
					chosen := settings.ChartSettings{Style: style, Size: size, TimeRange: span}
					q.Update(MsgChartSettings{Settings: chosen})
					for _, width := range []int{100, 15, 5, 0, 100} {
						q.Update(tea.WindowSizeMsg{Width: width, Height: 20})
						q.Update(MsgQueueDetailedStats{Stats: stats})
						q.Update(tea.KeyPressMsg{Code: tea.KeyEnd})
						if width == 0 {
							require.Empty(t, q.View().Content)
							require.Zero(t, q.scrollOffset)
							continue
						}
						view := ansi.Strip(q.View().Content)
						require.Equal(t, 22, lipgloss.Height(view))
						for i, line := range strings.Split(view, "\n") {
							require.Equal(t, width+2, lipgloss.Width(line))
							if i > 0 && i < 21 {
								require.True(t, strings.HasPrefix(line, "│ "))
								require.True(t, strings.HasSuffix(line, "  │") || strings.HasSuffix(line, " ▐│"))
							}
						}
						require.Contains(t, view, "▐│")
						require.Equal(t, height, lipgloss.Height(q.queuedMsgsChart.View()))
					}
					require.Equal(t, max(float64(1), 3600-chosen.Duration().Seconds()), q.queuedMsgsChart.ViewMinX())
					require.Equal(t, float64(3600), q.queuedMsgsChart.ViewMaxX())
					dots := strings.ContainsFunc(q.queuedMsgsChart.View(), func(r rune) bool { return r >= '\u2801' && r <= '\u28ff' })
					require.Equal(t, style == "dotted", dots)
					require.Equal(t, q.panel.ClampOffset(q.scrollOffset, q.content()), q.scrollOffset)
				}
			})
		}
	}
}
