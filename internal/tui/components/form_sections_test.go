package components

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/require"

	"github.com/jadegopher/trabbit/internal/tui/core"
)

func TestFormSectionsAndWrappingKeepFocusedFieldsVisible(t *testing.T) {
	t.Parallel()

	for _, textFields := range []bool{false, true} {
		t.Run(fmt.Sprint(textFields), func(t *testing.T) {
			t.Parallel()

			theme := core.DefaultTheme()
			var fields []FormField
			for i := range 12 {
				var item FormItem = NewToggle(theme, true)
				if textFields {
					input := NewTextInput(theme)
					input.SetValue("draft")
					item = input
				}
				field := FormField{Label: fmt.Sprintf("Field %02d: ", i), Item: item}
				if i%3 == 0 {
					field.Section = fmt.Sprintf("Section %d", i/3)
				}
				fields = append(fields, field)
			}
			modal := NewFormModal(theme, "Settings", fields)
			modal.Init()
			modal.Update(tea.WindowSizeMsg{Width: 100, Height: 14})
			content := ansi.Strip(modal.renderContent().Content)
			for i := range 4 {
				require.Equal(t, 1, strings.Count(content, fmt.Sprintf("Section %d", i)))
			}
			for i := range fields {
				require.Equal(t, i, modal.cursor)
				require.Contains(t, ansi.Strip(modal.View().Content), strings.TrimSpace(fields[i].Label))
				if textFields {
					require.NotNil(t, modal.View().Cursor)
				}
				modal.Update(tea.KeyPressMsg{Code: tea.KeyDown})
			}
			require.Zero(t, modal.cursor, "down from the final field wraps past section headings")
			require.Zero(t, modal.scrollOffset)
			modal.Update(tea.KeyPressMsg{Code: tea.KeyUp})
			require.Equal(t, len(fields)-1, modal.cursor)
			require.Contains(t, ansi.Strip(modal.View().Content), "Field 11:")
			modal.Update(tea.KeyPressMsg{Code: tea.KeyTab})
			require.Zero(t, modal.cursor)
			modal.Update(tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
			require.Equal(t, len(fields)-1, modal.cursor)
		})
	}
}

func TestFormToggleUsesRealSpaceKeyAndSharedHint(t *testing.T) {
	t.Parallel()

	toggle := NewToggle(core.DefaultTheme(), false)
	modal := NewFormModal(core.DefaultTheme(), "Settings", []FormField{{Label: "Enabled", Item: toggle}})
	modal.Init()
	modal.Update(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	require.True(t, toggle.Value)
	require.Contains(t, modal.theme.RenderHelp(modal), "toggle")
}
