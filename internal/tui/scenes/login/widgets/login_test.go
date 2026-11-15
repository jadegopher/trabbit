package widgets

import (
	"github.com/charmbracelet/x/ansi"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"

	"github.com/jadegopher/trabbit/internal/tui/core"
)

func TestLogin_FocusedCursorOnly(t *testing.T) {
	t.Parallel()

	theme := core.DefaultTheme()
	l := NewLogin(theme)

	// URL (index 0) is focused
	view := l.View()
	lines := strings.Split(ansi.Strip(view.Content), "\n")
	assert.NotContains(t, view.Content, "RabbitMq tui")
	assert.Contains(t, lines[view.Cursor.Y], "URL:")

	// Switch to User (index 1)
	l.Update(tea.KeyPressMsg{Text: core.KeyDown})
	assert.Equal(t, 1, l.focused)

	view = l.View()
	assert.Contains(t, lines[view.Cursor.Y], "User:")

	// Switch to Password (index 2)
	l.Update(tea.KeyPressMsg{Text: core.KeyDown})
	assert.Equal(t, 2, l.focused)

	view = l.View()
	assert.Contains(t, lines[view.Cursor.Y], "Password:")
}

func TestLogin_FocusAndResetBehavior(t *testing.T) {
	t.Parallel()

	theme := core.DefaultTheme()
	l := NewLogin(theme)

	// User (index 1) is "guest" (pos 5), blurred.
	assert.Equal(t, 5, l.inputs[1].Position())

	// Focus it
	l.inputs[1].Focus()
	// Position should still be 5
	assert.Equal(t, 5, l.inputs[1].Position())

	// Reset it
	l.inputs[1].Reset()
	assert.Equal(t, 0, l.inputs[1].Position(), "Position should be 0 after Reset")
}
