package components

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/jadegopher/trabbit/internal/tui/core"
)

const (
	subtabAbbreviationColumns = 3
	subtabBorderColumns       = core.BorderOverhead
	subtabTitlePaddingColumns = 2
	subtabMinimumBorderWidth  = subtabBorderColumns + subtabTitlePaddingColumns
)

// Subtabs owns circular navigation and labels within an existing panel border.
// The host owns the child models and decides when keyboard input is navigation.
type Subtabs struct {
	Labels []string
	Active int
}

func (s *Subtabs) Move(delta int) {
	if len(s.Labels) > 0 {
		s.Active = ((s.Active+delta)%len(s.Labels) + len(s.Labels)) % len(s.Labels)
	}
}

func (s Subtabs) Decorate(view tea.View, theme core.Theme) tea.View {
	lines := strings.Split(view.Content, "\n")
	width := lipgloss.Width(lines[0])
	if width < subtabMinimumBorderWidth || len(s.Labels) == 0 {
		return view
	}
	active := max(0, min(s.Active, len(s.Labels)-1))
	renderTitle := func(abbreviate bool) string {
		labels := make([]string, len(s.Labels))
		for i, label := range s.Labels {
			style := theme.BlurredText
			if i == active {
				style = theme.LabelFocused
			} else if abbreviate {
				label = ansi.Truncate(label, subtabAbbreviationColumns, "")
			}
			labels[i] = style.Render(label)
		}
		separator := " - "
		if abbreviate {
			separator = " "
		}
		return " " + strings.Join(labels, separator) + " "
	}
	title := renderTitle(false)
	available := width - subtabBorderColumns
	if lipgloss.Width(title) > available {
		title = renderTitle(true)
	}
	if lipgloss.Width(title) > available {
		title = strings.TrimSpace(title)
	}
	if lipgloss.Width(title) > available {
		// At the smallest sizes, retain the selected label before clipping.
		title = " " + theme.LabelFocused.Render(s.Labels[active]) + " "
	}
	title = ansi.Truncate(title, available, "")
	// Preserve the original corners and border styling without changing geometry.
	lines[0] = ansi.Cut(lines[0], 0, 1) + title + ansi.Cut(lines[0], 1+lipgloss.Width(title), width)
	view.Content = strings.Join(lines, "\n")
	return view
}
