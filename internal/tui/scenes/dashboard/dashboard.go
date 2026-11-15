package dashboard

import (
	"context"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/jadegopher/trabbit/internal/app"
	"github.com/jadegopher/trabbit/internal/log"
	"github.com/jadegopher/trabbit/internal/settings"
	"github.com/jadegopher/trabbit/internal/tui/components"
	"github.com/jadegopher/trabbit/internal/tui/core"
	"github.com/jadegopher/trabbit/internal/tui/scenes/dashboard/refresher"
	"github.com/jadegopher/trabbit/internal/tui/scenes/dashboard/tabs"
)

const (
	stateOverview       = "overview"
	stateConnections    = "connections"
	stateChannels       = "channels"
	stateExchanges      = "exchanges"
	stateQueues         = "queues"
	stateAdmin          = "admin"
	defaultState        = stateQueues
	dashboardHeaderRows = 1
)

type tabInfo struct{ label, state string }

var tabsConfig = []tabInfo{
	{"Overview", stateOverview},
	{"Connections", stateConnections},
	{"Channels", stateChannels},
	{"Exchanges", stateExchanges},
	{"Queues", stateQueues},
	{"Admin", stateAdmin},
}

type Scene struct {
	refreshErr        error
	theme             core.Theme
	queues            *tabs.QueueTab
	activeTab         string
	refreshController *refresher.Controller
	width, height     int
	visible           bool
	cancel            context.CancelFunc
}

func NewDashboardScene(
	theme core.Theme,
	service *app.Service,
	now func() time.Time,
	preferences *settings.Store,
) *Scene {
	ctx, cancel := context.WithCancel(context.Background())
	return &Scene{
		theme: theme, queues: tabs.NewQueueTab(theme, service, ctx, preferences), activeTab: defaultState,
		refreshController: refresher.NewController(theme, now, 5*time.Second), cancel: cancel, visible: true,
	}
}
func (s *Scene) Init() tea.Cmd { return s.refreshController.Init() }
func (s *Scene) Close() {
	s.cancel()
	s.queues.SetVisible(false)
	s.refreshController.SetVisible(false)
}
func (s *Scene) setVisible() tea.Cmd {
	visible := s.visible && s.activeTab == stateQueues
	return tea.Batch(s.queues.SetVisible(visible), s.refreshController.SetVisible(visible))
}
func (s *Scene) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case tea.WindowSizeMsg:
		s.width = m.Width
		s.height = m.Height
		layout := core.NewLayout(m.Width, m.Height, core.Vertical).AddFixed("header", dashboardHeaderRows).AddRelative("content", 1)
		_, cmd := s.queues.Update(layout.GetSize("content"))
		return s, cmd
	case core.VisibilityMsg:
		s.visible = m.Visible
		return s, s.setVisible()
	case refresher.MsgRefreshRequested:
		if !s.refreshController.Accepts(m) {
			return s, nil
		}
		return s, tea.Batch(s.refreshController.Update(m), s.queues.Refresh())
	case tabs.SelectionChanged:
		return s, s.refreshController.ResetDeadline()
	case tabs.RefreshFailed:
		s.refreshErr = m.Err
		return s, nil
	case tabs.RefreshFinished:
		s.refreshErr = m.Err
		return s, s.refreshController.Update(refresher.Result{Err: m.Err})
	case core.DialogResult:
		if interval, ok := m.Value.(refresher.IntervalSelected); ok {
			return s, s.refreshController.SetInterval(interval.Duration)
		}
		_, cmd := s.queues.Update(m)
		return s, cmd
	case tea.KeyPressMsg:
		if key.Matches(m, core.Cancel()) {
			dialog := components.NewConfirm(
				"Are you sure you want to log out?",
				s.theme,
			).WithResult(ActionLogoutRequested{})
			return s, core.Cmd(core.OpenDialog{Model: dialog})
		}
		if key.Matches(m, core.RefreshSettings()) && s.activeTab == stateQueues {
			return s, core.Cmd(core.OpenDialog{Model: refresher.NewSetRefreshModal(s.theme)})
		}
		if index, ok := s.tabBar().Match(m); ok {
			log.Debug("Dashboard tab selected", log.Component, "dashboard", "from", s.activeTab, "to", tabsConfig[index].state)
			s.activeTab = tabsConfig[index].state
			return s, s.setVisible()
		}
		if s.activeTab != stateQueues {
			return s, nil
		}
	case tea.MouseMsg:
		if s.activeTab != stateQueues {
			return s, nil
		}
		// The content starts below the dashboard header.
		switch v := m.(type) {
		case tea.MouseClickMsg:
			if v.Y < 1 {
				return s, nil
			}
			v.Y--
			msg = v
		case tea.MouseWheelMsg:
			if v.Y < 1 {
				return s, nil
			}
			v.Y--
			msg = v
		}
	}
	_, cmd := s.queues.Update(msg)
	return s, tea.Batch(cmd, s.refreshController.Update(msg))
}
func (s *Scene) View() tea.View {
	selector := s.tabBar().View(s.theme)
	header := core.NewLayout(s.width, dashboardHeaderRows, core.Horizontal).AddFixed(
		"tabs",
		lipgloss.Width(selector),
	).AddRelative("refresh", 1)
	refresh := ""
	if s.activeTab == stateQueues {
		refresh = lipgloss.PlaceHorizontal(
			header.GetWidth("refresh"),
			lipgloss.Right,
			s.refreshController.View().Content,
		)
	}
	top := header.JoinViews(map[string]tea.View{"tabs": tea.NewView(selector), "refresh": tea.NewView(refresh)})
	if s.activeTab == stateQueues && s.refreshErr != nil {
		top = tea.NewView(s.theme.RenderNotice(s.width, dashboardHeaderRows, "Refresh failed: "+s.refreshErr.Error(), core.NoticeError))
	}
	content := tea.NewView(s.theme.HeaderStyle.Render(s.activeTab + " (Coming soon...)"))
	if s.activeTab == stateQueues {
		content = s.queues.View()
	}
	return core.NewLayout(s.width, s.height, core.Vertical).AddFixedView("header", dashboardHeaderRows, top).AddRelativeView(
		"content",
		1,
		content,
	).Render()
}

// tabBar supplies navigation bindings; the scene retains workflow selection.
func (s *Scene) tabBar() components.TabBar {
	bar := components.TabBar{}
	for i, tab := range tabsConfig {
		bar.Tabs = append(bar.Tabs, core.TabNumber(i+components.FirstTabNumber, tab.label))
		if s.activeTab == tab.state {
			bar.Active = i
		}
	}
	return bar
}
