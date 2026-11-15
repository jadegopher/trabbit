package core

import (
	"charm.land/bubbles/v2/textarea"
	"charm.land/lipgloss/v2"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
)

func TestCalculateScrollbar(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		containerHeight int
		totalLines      int
		scrollOffset    int
		expected        Scrollbar
	}{
		{
			name:            "no overflow",
			containerHeight: 10,
			totalLines:      5,
			scrollOffset:    0,
			expected:        Scrollbar{HandleStart: -1, HandleEnd: -1},
		},
		{
			name:            "equal size",
			containerHeight: 10,
			totalLines:      10,
			scrollOffset:    0,
			expected:        Scrollbar{HandleStart: -1, HandleEnd: -1},
		},
		{
			name:            "small overflow - start",
			containerHeight: 10,
			totalLines:      11,
			scrollOffset:    0,
			expected:        Scrollbar{HandleStart: 0, HandleEnd: 9},
		},
		{
			name:            "small overflow - end",
			containerHeight: 10,
			totalLines:      11,
			scrollOffset:    1,
			expected:        Scrollbar{HandleStart: 1, HandleEnd: 10},
		},
		{
			name:            "large overflow - start",
			containerHeight: 10,
			totalLines:      100,
			scrollOffset:    0,
			expected:        Scrollbar{HandleStart: 0, HandleEnd: 1},
		},
		{
			name:            "large overflow - middle",
			containerHeight: 10,
			totalLines:      100,
			scrollOffset:    45,
			expected:        Scrollbar{HandleStart: 4, HandleEnd: 5},
		},
		{
			name:            "large overflow - end",
			containerHeight: 10,
			totalLines:      100,
			scrollOffset:    90,
			expected:        Scrollbar{HandleStart: 9, HandleEnd: 10},
		},
		{
			name:            "invalid container height",
			containerHeight: 0,
			totalLines:      10,
			scrollOffset:    0,
			expected:        Scrollbar{HandleStart: -1, HandleEnd: -1},
		},
		{
			name:            "offset beyond max",
			containerHeight: 10,
			totalLines:      20,
			scrollOffset:    50,
			expected: Scrollbar{
				HandleStart: 5,
				HandleEnd:   10,
			}, // maxOffset is 10, percentage 1.0, start = 1.0 * (10-5) = 5
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				t.Parallel()

				actual := CalculateScrollbar(tt.containerHeight, tt.totalLines, tt.scrollOffset)
				assert.Equal(t, tt.expected, actual)
			},
		)
	}
}

func TestFormatValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    float64
		expected string
	}{
		{0, "0"},
		{500, "500"},
		{999, "999"},
		{1000, "1k"},
		{1200, "1.2k"},
		{112000, "112k"},
		{1000000, "1m"},
		{1200000, "1.2m"},
		{1000000000, "1b"},
		{1200000000, "1.2b"},
	}

	for _, tt := range tests {
		t.Run(
			tt.expected, func(t *testing.T) {
				t.Parallel()

				actual := FormatValue(tt.input)
				assert.Equal(t, tt.expected, actual)
			},
		)
	}
}

func TestFormatRateValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    float64
		expected string
	}{
		{0, "0/s"},
		{0.2, "0.20/s"},
		{1.2, "1.2/s"},
		{1000, "1k/s"},
		{1000000, "1m/s"},
	}

	for _, tt := range tests {
		t.Run(
			tt.expected, func(t *testing.T) {
				t.Parallel()

				actual := FormatRateValue(tt.input)
				assert.Equal(t, tt.expected, actual)
			},
		)
	}
}

func TestFormatBytes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    int64
		expected string
	}{
		{0, "0 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
	}

	for _, tt := range tests {
		t.Run(
			tt.expected, func(t *testing.T) {
				t.Parallel()

				actual := FormatBytes(tt.input)
				assert.Equal(t, tt.expected, actual)
			},
		)
	}
}

func TestFormatTime(t *testing.T) {
	t.Parallel()

	t.Run(
		"zero time", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "", FormatTime(time.Time{}))
		},
	)

	t.Run(
		"non-zero time", func(t *testing.T) {
			t.Parallel()
			ts := time.Date(2026, 5, 17, 10, 0, 0, 0, time.UTC)
			assert.Equal(t, "2026-05-17T10:00:00Z", FormatTime(ts))
		},
	)
}

func TestRenderKeyOnlyHelp(t *testing.T) {
	t.Parallel()
	theme := DefaultTheme()
	binding := RefreshSettings()
	disabled := NewShortcut("disabled", KeyQ)
	disabled.SetEnabled(false)
	hints := theme.RenderHelp(Shortcuts{binding, disabled}, HelpKeysOnly)
	assert.Equal(t, "[r]", ansi.Strip(hints))
	assert.Equal(t, "refresh", binding.Help().Desc)
	assert.Contains(t, ansi.Strip(theme.RenderHelp(Shortcuts{binding})), "refresh")
	assert.Empty(t, theme.RenderHelp(nil, HelpKeysOnly))
}

func TestOptionalFieldsAndEditorUseSemanticTextColors(t *testing.T) {
	t.Parallel()
	theme := DefaultTheme()
	assert.Equal(t, theme.Inactive, theme.OptionalFieldStyle(false).GetForeground())
	assert.Equal(t, theme.TextColor, theme.OptionalFieldStyle(true).GetForeground())
	styles := theme.TextAreaStyles()
	for _, state := range []textarea.StyleState{styles.Focused, styles.Blurred} {
		assert.Equal(t, theme.TextColor, state.Text.GetForeground())
		assert.Equal(t, theme.TextColor, state.CursorLine.GetForeground())
		assert.Equal(t, theme.PlaceholderStyle.GetForeground(), state.Placeholder.GetForeground())
	}
}

func TestInsetScrollbarKeepsFooterInsideBorder(t *testing.T) {
	t.Parallel()
	theme := DefaultTheme()
	for _, content := range []string{"short", strings.Repeat("body row\n", 30)} {
		offset := 100
		view := ansi.Strip(theme.RenderInsetScrollContainer(24, 4, true, &offset, content, 0, "\nfixed hints"))
		lines := strings.Split(view, "\n")
		assert.Len(t, lines, 8, "four body rows, two fixed footer rows and two borders")
		assert.Equal(t, "│fixed hints             │", lines[len(lines)-2])
		assert.True(t, strings.HasPrefix(lines[len(lines)-1], "╰"))
		for _, line := range lines {
			assert.Equal(t, 26, lipgloss.Width(line))
		}
		assert.NotContains(t, lines[len(lines)-2], "▐", "the body scrollbar must stop before the footer")
	}
}
