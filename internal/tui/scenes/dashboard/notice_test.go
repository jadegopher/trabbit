package dashboard

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/jadegopher/trabbit/internal/app"
	"github.com/jadegopher/trabbit/internal/settings"
	"github.com/jadegopher/trabbit/internal/tui/core"
	"github.com/jadegopher/trabbit/internal/tui/scenes/dashboard/tabs"
	"github.com/stretchr/testify/require"
)

func TestRefreshFailureReplacesHeaderAndRecoveryRestoresIt(t *testing.T) {
	t.Parallel()
	s := NewDashboardScene(core.DefaultTheme(), app.New(nil), time.Now, settings.NewMemory())
	defer s.Close()
	s.Update(tea.WindowSizeMsg{Width: core.MinWidth, Height: 24})
	before := strings.Split(ansi.Strip(s.View().Content), "\n")
	err := errors.New("connection refused " + strings.Repeat("details ", core.MinWidth))
	s.Update(tabs.RefreshFinished{Err: err})
	failed := strings.Split(ansi.Strip(s.View().Content), "\n")
	require.Contains(t, failed[0], "Refresh failed: connection refused")
	require.Equal(t, core.MinWidth, lipgloss.Width(failed[0]))
	require.NotContains(t, failed[0], "Overview")
	require.NotContains(t, failed[len(failed)-1], "connection refused")
	require.Equal(t, before[dashboardHeaderRows:], failed[dashboardHeaderRows:])

	// Detail failures share the banner without changing list polling.
	_, cmd := s.Update(tabs.RefreshFailed{Err: errors.New("details unavailable")})
	require.Nil(t, cmd)
	require.Contains(t, s.View().Content, "details unavailable")
	s.Update(tabs.RefreshFinished{})
	restored := strings.Split(ansi.Strip(s.View().Content), "\n")
	require.Contains(t, restored[0], "Overview")
	require.Contains(t, restored[0], "[r]")
	require.NotContains(t, s.View().Content, "details unavailable")
}
