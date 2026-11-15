package widgets

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/require"
)

func TestPublishUndoKeepsPayloadViewportAndCursor(t *testing.T) {
	t.Parallel()
	p := newTestPublisher()
	p.payload.SetValue(strings.Repeat("hello world\n", 80))
	p.payload.MoveToBegin()
	layout := p.geometry()
	y := layout.editor.Min.Y + 3
	p.Update(tea.MouseWheelMsg{X: layout.editor.Min.X, Y: y, Button: tea.MouseWheelDown})
	line := strings.Split(ansi.Strip(p.View().Content), "\n")[y]
	x := lipgloss.Width(strings.SplitN(line, "hello", 2)[0])
	p.Update(tea.MouseClickMsg{X: x, Y: y, Button: tea.MouseLeft})
	p.Update(tea.MouseReleaseMsg{X: x, Y: y, Button: tea.MouseLeft})
	before := ansi.Strip(p.View().Content)

	p.Update(tea.KeyPressMsg{Code: 'x', Text: "x"})
	p.Update(tea.KeyPressMsg{Code: 'z', Mod: tea.ModCtrl})

	require.Equal(t, before, ansi.Strip(p.View().Content))
	require.NotNil(t, p.View().Cursor)
	require.Equal(t, x, p.View().Cursor.X)
	require.Equal(t, y, p.View().Cursor.Y)
}
