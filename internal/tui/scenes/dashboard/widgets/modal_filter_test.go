package widgets

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/require"

	"github.com/jadegopher/trabbit/internal/tui/core"
)

func TestFilterModal_UI(t *testing.T) {
	t.Parallel()

	theme := core.DefaultTheme()
	m := NewModalFilter(theme, []string{"/"}, ActionFilterQueues{})

	view := m.View()
	core.AssertGolden(t, "modal_filter_queues", view.Content)
}

func TestFilterModalResetFromEveryField(t *testing.T) {
	t.Parallel()

	for _, field := range []string{"filter", "vhost", "regex"} {
		t.Run(field, func(t *testing.T) {
			t.Parallel()

			m := NewModalFilter(core.DefaultTheme(), []string{"/"}, ActionFilterQueues{
				Filter: "orders", VHost: "/", UseRegex: true,
			})
			m.Init()
			if field != "filter" {
				m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
			}
			if field == "regex" {
				m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
			}
			require.Contains(t, ansi.Strip(m.View().Content), "ctrl+r reset filter")

			_, cmd := m.Update(tea.KeyPressMsg{Code: 'r', Mod: tea.ModCtrl})
			require.NotNil(t, cmd)
			result := cmd().(core.DialogResult)
			require.Same(t, m, result.Dialog)
			require.True(t, result.Close)
			require.Equal(t, ActionFilterQueues{}, result.Value)
		})
	}
}
