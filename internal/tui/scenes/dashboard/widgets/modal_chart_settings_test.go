package widgets

import (
	"github.com/charmbracelet/x/ansi"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/require"

	"github.com/jadegopher/trabbit/internal/settings"
	"github.com/jadegopher/trabbit/internal/tui/core"
)

func TestChartSettingsSelectorsRestoreAndCyclePresets(t *testing.T) {
	t.Parallel()

	initial := settings.ChartSettings{Style: "lines", Size: "large", TimeRange: "1h"}
	modal := NewModalChartSettings(core.DefaultTheme(), initial)
	modal.Init()
	modal.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	content := ansi.Strip(modal.View().Content)
	for _, section := range []string{"Appearance", "Queued messages", "Message Throughput"} {
		require.Contains(t, content, section)
	}
	require.NotContains(t, content, "Messages /")
	require.NotContains(t, content, "Throughput /")
	value, err := modal.OnSubmit()
	require.NoError(t, err)
	require.Equal(t, ActionChartSettings{Settings: initial}, value)

	modal.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	modal.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	modal.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	modal.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	modal.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	value, err = modal.OnSubmit()
	require.NoError(t, err)
	require.Equal(t, ActionChartSettings{Settings: settings.ChartSettings{Style: "dotted", Size: "small", TimeRange: "1m"}}, value)
	require.False(t, modal.CloseOnSubmit, "save failures must remain editable in the modal")
}

func TestTimeRangeSelectorKeepsItsFootprintAcrossCycles(t *testing.T) {
	t.Parallel()

	modal := NewModalChartSettings(core.DefaultTheme(), settings.DefaultChartSettings())
	modal.Init()
	modal.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	modal.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	modal.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	ranges := []string{"1m", "10m", "30m", "1h"}
	markerColumn := -1
	for step := 0; step < 12; step++ {
		content := ansi.Strip(modal.View().Content)
		for _, row := range strings.Split(content, "\n") {
			if !strings.Contains(row, "Time range:") {
				continue
			}
			require.Equal(t, 1, strings.Count(row, "last "))
			require.Contains(t, row, "last "+ranges[step%4])
			column := strings.Index(row, ">")
			if markerColumn < 0 {
				markerColumn = column
			}
			require.Equal(t, markerColumn, column, "the closing marker must stay in one column")
		}
		value, err := modal.OnSubmit()
		require.NoError(t, err)
		require.Equal(t, ranges[step%4], value.(ActionChartSettings).Settings.TimeRange)
		modal.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	}
}

func TestChartSeriesTogglesRestoreAndKeepDraftIsolated(t *testing.T) {
	t.Parallel()

	initial := settings.DefaultChartSettings()
	initial.Messages = map[string]bool{"Ready": false}
	initial.Throughput = map[string]bool{"GetNoAck": true}
	modal := NewModalChartSettings(core.DefaultTheme(), initial)
	modal.Init()
	modal.Update(tea.WindowSizeMsg{Width: 100, Height: 12})
	for range 3 {
		modal.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	}
	for _, name := range settings.MessageSeries() {
		require.Contains(t, ansi.Strip(modal.View().Content), name+":")
		modal.Update(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
		modal.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	}
	for _, name := range settings.ThroughputSeries() {
		require.Contains(t, ansi.Strip(modal.View().Content), name+":")
		modal.Update(tea.KeyPressMsg{Code: tea.KeyRight})
		modal.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	}
	value, err := modal.OnSubmit()
	require.NoError(t, err)
	chosen := value.(ActionChartSettings).Settings
	for _, name := range settings.MessageSeries() {
		require.Equal(t, !initial.MessageVisible(name), chosen.MessageVisible(name), name)
	}
	for _, name := range settings.ThroughputSeries() {
		require.Equal(t, !initial.ThroughputVisible(name), chosen.ThroughputVisible(name), name)
	}
	require.Equal(t, map[string]bool{"Ready": false}, initial.Messages)
	require.Equal(t, map[string]bool{"GetNoAck": true}, initial.Throughput)
	reopened := NewModalChartSettings(core.DefaultTheme(), chosen)
	value, err = reopened.OnSubmit()
	require.NoError(t, err)
	require.Equal(t, ActionChartSettings{Settings: chosen}, value)
}
