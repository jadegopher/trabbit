package components

import (
	"image/color"

	"charm.land/lipgloss/v2"

	"github.com/jadegopher/trabbit/internal/tui/core"
)

const defaultWidth = 40

// Base template for all modals
type modalTemplate struct {
	modalWidth   int
	screenWidth  int
	screenHeight int
	theme        core.Theme
	style        lipgloss.Style
}

func newTemplate(theme core.Theme, width int, borderColor color.Color) modalTemplate {
	return modalTemplate{
		modalWidth: width,
		theme:      theme,
		style:      theme.ModalStyle(width, borderColor).Align(lipgloss.Center),
	}
}

func (t modalTemplate) render(content string, helpText string) string {
	if helpText != "" {
		helper := t.theme.BlurredText.PaddingTop(1).Render(helpText)
		content = content + "\n" + helper
	}
	return t.style.Render(content)
}
