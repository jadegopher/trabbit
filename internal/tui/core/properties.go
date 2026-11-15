package core

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

const (
	propertyColumnGap = 2
	// Reserve at least half the usable width for values.
	propertyLabelWidthDivisor = 2
)

// Property is a display label and its value, independent of application data.
type Property struct {
	Label, Value string
}

// RenderProperties aligns values in a second column and wraps both columns to
// their bounds. Long labels cannot push values outside the owning viewport.
func (t Theme) RenderProperties(properties []Property, width int) string {
	if width <= 0 || len(properties) == 0 {
		return ""
	}
	labelWidth := 0
	for _, property := range properties {
		labelWidth = max(labelWidth, lipgloss.Width(property.Label))
	}
	labelWidth = min(labelWidth, max(1, (width-propertyColumnGap)/propertyLabelWidthDivisor))
	valueWidth := width - labelWidth - propertyColumnGap
	var rows []string
	for _, property := range properties {
		if valueWidth <= 0 {
			rows = append(rows, ansi.Wrap(property.Label+": "+property.Value, width, " "))
			continue
		}
		label := t.LabelStyle.Render(ansi.Wrap(property.Label, labelWidth, " "))
		value := t.PlainStyle.Foreground(t.TextColor).Render(ansi.Wrap(property.Value, valueWidth, " "))
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top,
			lipgloss.NewStyle().Width(labelWidth).Render(label),
			strings.Repeat(" ", propertyColumnGap),
			lipgloss.NewStyle().Width(valueWidth).Render(value)))
	}
	return strings.Join(rows, "\n")
}
