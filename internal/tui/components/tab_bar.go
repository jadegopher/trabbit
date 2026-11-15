package components

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/jadegopher/trabbit/internal/tui/core"
)

// FirstTabNumber is the human-facing start of numbered tab navigation.
const FirstTabNumber = 1

// TabBar shares bindings between navigation and rendering. Its host owns selection.
type TabBar struct {
	Tabs   []core.Shortcut
	Active int
}

func (b TabBar) Match(msg tea.KeyPressMsg) (int, bool) {
	for i, binding := range b.Tabs {
		if key.Matches(msg, binding) {
			return i, true
		}
	}
	return 0, false
}

func (b TabBar) View(theme core.Theme) string {
	tabs := make([]string, len(b.Tabs))
	for i, binding := range b.Tabs {
		tabTheme := theme
		tabTheme.KeyStyle = theme.TabKeyStyle
		labelStyle := theme.TabLabelStyle
		if i == b.Active {
			background := theme.TabSelectedStyle.GetBackground()
			tabTheme.KeyStyle = theme.TabLabelStyle.Background(background)
			labelStyle = labelStyle.Background(background)
		}
		tabs[i] = tabTheme.RenderHelp(core.Shortcuts{binding}, core.HelpKeysOnly) +
			labelStyle.Render(binding.Help().Desc)
	}
	return strings.Join(tabs, " ")
}
