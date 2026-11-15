package components

import (
	"fmt"
	"testing"

	"github.com/jadegopher/trabbit/internal/tui/core"
)

func TestUIErrorModalLong(t *testing.T) {
	t.Parallel()

	theme := core.DefaultTheme()
	longMsg := "This is a very long error message that should definitely be longer than forty characters to see how it behaves in the modal window."
	m := NewError(fmt.Errorf("%s", longMsg), theme)

	view := m.View()
	core.AssertGolden(t, "error_modal_long", view.Content)
}

func TestUIErrorModalShort(t *testing.T) {
	t.Parallel()

	theme := core.DefaultTheme()
	longMsg := "short"
	m := NewError(fmt.Errorf("%s", longMsg), theme)

	view := m.View()
	core.AssertGolden(t, "error_modal_short", view.Content)
}
