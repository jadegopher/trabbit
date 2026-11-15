package widgets

import (
	tea "charm.land/bubbletea/v2"
	"github.com/jadegopher/trabbit/internal/tui/core"
	"github.com/jadegopher/trabbit/internal/version"
)

func Branding(theme core.Theme, width int) tea.View {
	wordmark := theme.PlainStyle.Foreground(theme.Secondary).Render(trabbitBanner)
	rabbit := theme.PlainStyle.Foreground(theme.Primary).Render(rabbitSign)
	sign := core.NewLayout(0, 0, core.Horizontal).
		AddFixedView("wordmark", 0, tea.NewView(wordmark)).
		AddFixedView("gap", 3, tea.NewView("   ")).
		AddFixedView("rabbit", 0, tea.NewView(rabbit)).Render()
	area := core.NewLayout(width, 0, core.Vertical)
	return tea.NewView(area.Place(sign.Content) + "\n" + area.Place("RabbitMq tui "+version.Current))
}
