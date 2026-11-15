package widgets

import (
	"slices"

	tea "charm.land/bubbletea/v2"

	"github.com/jadegopher/trabbit/internal/settings"
	"github.com/jadegopher/trabbit/internal/tui/components"
	"github.com/jadegopher/trabbit/internal/tui/core"
)

type ActionChartSettings struct{ Settings settings.ChartSettings }

func NewModalChartSettings(theme core.Theme, initial settings.ChartSettings) *components.FormModal {
	styles := []string{"dotted", "lines"}
	sizes := []string{"small", "medium", "large"}
	ranges := []string{"1m", "10m", "30m", "1h"}
	style := components.NewSelect(theme, styles, max(0, slices.Index(styles, initial.Style)))
	size := components.NewSelect(theme, sizes, max(0, slices.Index(sizes, initial.Size)))
	span := components.NewSelect(theme, []string{"last 1m", "last 10m", "last 30m", "last 1h"}, max(0, slices.Index(ranges, initial.TimeRange)))

	fields := []components.FormField{
		{Section: "Appearance", Label: "Style: ", Item: style},
		{Label: "Size: ", Item: size},
		{Label: "Time range: ", Item: span},
	}
	messages := make(map[string]*components.Toggle)
	throughput := make(map[string]*components.Toggle)
	for i, name := range settings.MessageSeries() {
		messages[name] = components.NewToggle(theme, initial.MessageVisible(name))
		field := components.FormField{Label: name + ": ", Item: messages[name]}
		if i == 0 {
			field.Section = "Queued messages"
		}
		fields = append(fields, field)
	}
	for i, name := range settings.ThroughputSeries() {
		throughput[name] = components.NewToggle(theme, initial.ThroughputVisible(name))
		field := components.FormField{Label: name + ": ", Item: throughput[name]}
		if i == 0 {
			field.Section = "Message Throughput"
		}
		fields = append(fields, field)
	}

	m := components.NewFormModal(theme, "Queue View Settings", fields)
	m.OnSubmit = func() (tea.Msg, error) {
		chosen := initial.Clone()
		chosen.Style, chosen.Size, chosen.TimeRange = style.Value(), size.Value(), ranges[span.Index]
		for name, toggle := range messages {
			if toggle.Value != initial.MessageVisible(name) {
				if chosen.Messages == nil {
					chosen.Messages = make(map[string]bool)
				}
				chosen.Messages[name] = toggle.Value
			}
		}
		for name, toggle := range throughput {
			if toggle.Value != initial.ThroughputVisible(name) {
				if chosen.Throughput == nil {
					chosen.Throughput = make(map[string]bool)
				}
				chosen.Throughput[name] = toggle.Value
			}
		}
		return ActionChartSettings{Settings: chosen}, nil
	}
	return m
}
