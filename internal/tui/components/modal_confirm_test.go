package components

import (
	"testing"

	"github.com/jadegopher/trabbit/internal/tui/core"
)

func TestUIConfirmModalLong(t *testing.T) {
	t.Parallel()

	theme := core.DefaultTheme()
	longMsg := "Are you sure you want to perform this action? It might have some unintended consequences if you are not careful."
	m := NewConfirm(longMsg, theme)

	view := m.View()
	core.AssertGolden(t, "confirm_modal_long", view.Content)
}

func TestUIConfirmModalShort(t *testing.T) {
	t.Parallel()

	theme := core.DefaultTheme()
	longMsg := "short"
	m := NewConfirm(longMsg, theme)

	view := m.View()
	core.AssertGolden(t, "confirm_modal_short", view.Content)
}
