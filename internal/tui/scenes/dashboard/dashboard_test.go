package dashboard

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/require"

	"github.com/jadegopher/trabbit/internal/app"
	"github.com/jadegopher/trabbit/internal/settings"
	"github.com/jadegopher/trabbit/internal/tui/components"
	"github.com/jadegopher/trabbit/internal/tui/core"
)

func TestTabsRetainQueueStateAndPausePolling(t *testing.T) {
	t.Parallel()

	s := NewDashboardScene(core.DefaultTheme(), app.New(nil), time.Now, settings.NewMemory())
	defer s.Close()
	queues := s.queues
	for i, tab := range tabsConfig {
		s.Update(tea.KeyPressMsg{Text: string(rune('1' + i))})
		require.Equal(t, tab.state, s.activeTab)
		require.Same(t, queues, s.queues)
	}
}
func TestDashboardDialogs(t *testing.T) {
	t.Parallel()

	s := NewDashboardScene(core.DefaultTheme(), app.New(nil), time.Now, settings.NewMemory())
	defer s.Close()
	_, cmd := s.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	require.IsType(t, &components.ConfirmModel{}, cmd().(core.OpenDialog).Model)
	_, cmd = s.Update(tea.KeyPressMsg{Text: "r"})
	require.IsType(t, &components.SelectionModal{}, cmd().(core.OpenDialog).Model)
}

func TestDashboardLogoutUsesSharedCancel(t *testing.T) {
	t.Parallel()

	s := NewDashboardScene(core.DefaultTheme(), app.New(nil), time.Now, settings.NewMemory())
	defer s.Close()

	_, cmd := s.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	require.Nil(t, cmd, "backspace must not open the logout dialog")
	for _, msg := range []tea.KeyPressMsg{
		{Code: tea.KeyEsc},
		{Code: 'q', Mod: tea.ModCtrl},
	} {
		_, cmd := s.Update(msg)
		require.NotNil(t, cmd)
		require.IsType(t, &components.ConfirmModel{}, cmd().(core.OpenDialog).Model)
	}
}
