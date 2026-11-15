package components

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/jadegopher/trabbit/internal/tui/core"
	"github.com/stretchr/testify/require"
)

func TestShortcutsModalScrollResizeAndClose(t *testing.T) {
	t.Parallel()
	for _, size := range [][2]int{{90, 25}, {44, 12}, {140, 35}} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			t.Parallel()
			theme := core.DefaultTheme()
			editor := NewTextArea(theme)
			// Exercise overflow even after the everyday editing reference is shortened.
			var bindings []core.Shortcut
			for range 3 {
				bindings = append(bindings, editor.Shortcuts()...)
			}
			modal := NewShortcutsModal(theme, "Editor shortcuts", bindings)
			modal.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
			before := modal.View().Content
			require.LessOrEqual(t, lipgloss.Width(before), size[0])
			require.LessOrEqual(t, lipgloss.Height(before), size[1], "rendered: %q", ansi.Strip(before))
			require.Contains(t, ansi.Strip(before), "undo")
			require.Contains(t, ansi.Strip(before), "▐", "overflow must expose the shared scrollbar")
			body, _, _ := modal.body()
			require.NotContains(t, body, "delete before cursor")
			require.NotContains(t, body, "uppercase word")
			panelLines := strings.Split(ansi.Strip(before), "\n")
			panelWidth := lipgloss.Width(panelLines[0])
			for _, line := range panelLines {
				if strings.Contains(line, "▐") {
					require.Equal(t, "▐│", ansi.Cut(line, panelWidth-2, panelWidth), "scrollbar must touch the interior of the right border")
				}
			}
			require.True(t, strings.HasPrefix(panelLines[len(panelLines)-1], "╰"), "bottom border must follow all hints")
			for _, line := range panelLines {
				if strings.Contains(line, "navigate") || strings.Contains(line, "cancel") {
					require.True(t, strings.HasPrefix(line, "│"), line)
					require.True(t, strings.HasSuffix(line, "│"), line)
				}
			}
			require.NotContains(t, body, "meta+")
			require.NotContains(t, body, "super+")
			for range 20 {
				modal.Update(tea.KeyPressMsg{Code: tea.KeyPgDown})
			}
			require.Positive(t, modal.offset)
			bottom := modal.View().Content
			require.NotEqual(t, before, bottom)
			require.Contains(t, ansi.Strip(bottom), "navigate")
			require.Contains(t, ansi.Strip(bottom), "cancel")
			offset := modal.offset
			modal.View()
			require.Equal(t, offset, modal.offset)
			modal.Update(tea.WindowSizeMsg{Width: 120, Height: 100})
			body, _, height := modal.body()
			require.LessOrEqual(t, modal.offset, max(0, len(strings.Split(body, "\n"))-height))
			_, close := modal.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
			require.Equal(t, core.DialogResult{Dialog: modal, Close: true}, close())
		})
	}
}
