package dashboard

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/jadegopher/trabbit/internal/app"
	"github.com/jadegopher/trabbit/internal/domain"
	"github.com/jadegopher/trabbit/internal/settings"
	"github.com/jadegopher/trabbit/internal/tui/core"
	"github.com/jadegopher/trabbit/internal/tui/scenes/dashboard/refresher"
	"github.com/jadegopher/trabbit/internal/tui/scenes/dashboard/widgets"
	"github.com/stretchr/testify/require"
)

func TestDashboardRoutesPublishScreenRequest(t *testing.T) {
	t.Parallel()
	s := NewDashboardScene(core.DefaultTheme(), app.New(nil), time.Now, settings.NewMemory())
	defer s.Close()
	s.Update(tea.WindowSizeMsg{Width: 180, Height: 32})
	s.queues.Update(widgets.MsgQueues{Queues: domain.Queues{{Name: "orders", VHost: "/"}}})
	_, cmd := s.Update(tea.KeyPressMsg{Code: 'p', Text: "p"})
	require.IsType(t, core.OpenScreen{}, cmd())
	require.Equal(t, stateQueues, s.activeTab)
	s.Update(core.VisibilityMsg{Visible: false})
	require.False(t, s.visible)
}

func TestRefreshHintFitsMinimumScreen(t *testing.T) {
	t.Parallel()
	s := NewDashboardScene(core.DefaultTheme(), app.New(nil), time.Now, settings.NewMemory())
	defer s.Close()
	s.Update(tea.WindowSizeMsg{Width: core.MinWidth, Height: 24})
	for _, result := range []refresher.Result{{}, {Err: fmt.Errorf("offline")}} {
		s.refreshController.Update(result)
		header := strings.Split(ansi.Strip(s.View().Content), "\n")[0]
		require.LessOrEqual(t, lipgloss.Width(header), core.MinWidth)
		require.Contains(t, header, "↻ 0s ago • Every 5s [r]")
	}
}
