package components

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/require"

	"github.com/jadegopher/trabbit/internal/tui/core"
)

func TestScrollableControlsKeepBoundaryFrames(t *testing.T) {
	t.Parallel()
	for _, control := range []string{"form", "fields", "table", "records", "help"} {
		t.Run(control, func(t *testing.T) {
			t.Parallel()
			theme := core.DefaultTheme()
			var view func() tea.View
			var scroll func(tea.MouseButton) tea.Cmd
			switch control {
			case "form", "fields":
				fields := make([]FormField, 20)
				for i := range fields {
					fields[i] = FormField{Label: "Field", Item: NewTextInput(theme)}
				}
				if control == "form" {
					m := NewFormModal(theme, "Form", fields)
					m.Init()
					m.Update(tea.WindowSizeMsg{Width: 80, Height: 12})
					view = m.View
					scroll = func(button tea.MouseButton) tea.Cmd {
						_, cmd := m.Update(tea.MouseWheelMsg{Button: button})
						return cmd
					}
				} else {
					m := NewFormBody(theme, fields)
					m.SetSize(40, 5)
					m.Focus()
					view = m.View
					scroll = func(button tea.MouseButton) tea.Cmd {
						delta := 1
						if button == tea.MouseWheelUp {
							delta = -delta
						}
						m.Scroll(delta)
						return nil
					}
				}
			case "table", "records":
				m := NewDataTable(theme, []DataColumn{{Title: "Value", Width: 10}}, false)
				m.SetRecordLayout(control == "records")
				m.SetRows(make([]DataRow, 20))
				m.SetSize(40, 5)
				m.SetFocused(true)
				view = m.View
				scroll = func(button tea.MouseButton) tea.Cmd {
					return m.Update(tea.MouseWheelMsg{Button: button})
				}
			case "help":
				m := NewShortcutsModal(theme, "Shortcuts", NewTextArea(theme).Shortcuts())
				m.Update(tea.WindowSizeMsg{Width: 40, Height: 12})
				view = m.View
				scroll = func(button tea.MouseButton) tea.Cmd {
					_, cmd := m.Update(tea.MouseWheelMsg{Button: button})
					return cmd
				}
			}

			top := view()
			for range 3 {
				require.Nil(t, scroll(tea.MouseWheelUp))
				require.Equal(t, top, view())
			}
			for range 100 {
				scroll(tea.MouseWheelDown)
			}
			bottom := view()
			require.NotEqual(t, top.Content, bottom.Content)
			for range 3 {
				require.Nil(t, scroll(tea.MouseWheelDown))
				require.Equal(t, bottom, view())
			}
			for range 100 {
				scroll(tea.MouseWheelUp)
			}
			require.Equal(t, top, view())
		})
	}
}

func TestChartPanelScrollUsesWrappedBounds(t *testing.T) {
	t.Parallel()
	p := NewChartPanel(core.DefaultTheme())
	content := strings.Repeat("界", 50)
	for _, width := range []int{13, 103, 0, 13} {
		p.SetSize(width, 3)
		offset := 0
		p.HandleScroll(tea.MouseWheelMsg{Button: tea.MouseWheelUp}, &offset, content)
		require.Zero(t, offset)
		for range 100 {
			p.HandleScroll(tea.MouseWheelMsg{Button: tea.MouseWheelDown}, &offset, content)
		}
		require.Equal(t, p.ClampOffset(100, content), offset)
		before := p.Render(content, offset, true)
		for range 3 {
			p.HandleScroll(tea.KeyPressMsg{Code: tea.KeyDown}, &offset, content)
			require.Equal(t, before, p.Render(content, offset, true))
		}
	}
}
