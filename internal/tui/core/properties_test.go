package core

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/require"
)

func TestPropertiesAlignValuesAndWrapWithinWidth(t *testing.T) {
	t.Parallel()
	theme := DefaultTheme()
	properties := []Property{
		{Label: "Minimum heap (words)", Value: "233"},
		{Label: "Minor GCs", Value: "4"},
	}
	wide := strings.Split(ansi.Strip(theme.RenderProperties(properties, 80)), "\n")
	require.Equal(t, strings.Index(wide[0], "233"), strings.Index(wide[1], "4"))
	properties = append(properties, Property{Label: "Arguments", Value: theme.LabelStyle.Render(strings.Repeat("界", 12) + " END")})
	for _, width := range []int{80, 28, 8, 0, 80} {
		view := theme.RenderProperties(properties, width)
		for _, line := range strings.Split(view, "\n") {
			require.LessOrEqual(t, lipgloss.Width(line), width)
		}
		if width > 0 {
			require.Contains(t, ansi.Strip(view), "END")
		}
	}
}
