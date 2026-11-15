package components

import (
	"github.com/jadegopher/trabbit/internal/tui/core"
)

func RenderSelector(theme core.Theme, text string, isSelected bool) string {
	if !isSelected {
		return text
	}

	return theme.SelectedStyle.Render(text)
}
