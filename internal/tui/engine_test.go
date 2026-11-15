package tui

import (
	"context"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/require"

	"github.com/jadegopher/trabbit/internal/app"
	"github.com/jadegopher/trabbit/internal/settings"
	"github.com/jadegopher/trabbit/internal/tui/components"
	"github.com/jadegopher/trabbit/internal/tui/core"
	"github.com/jadegopher/trabbit/internal/tui/scenes/dashboard"
	"github.com/jadegopher/trabbit/internal/tui/scenes/publish"
	"github.com/jadegopher/trabbit/internal/tui/scenes/publish/widgets"
)

func TestStartupConnectionReachesDashboard(t *testing.T) {
	t.Parallel()

	connection := app.Connection{URL: "http://localhost:15672", Username: "alice", Password: "secret"}
	service := app.New(nil)
	var calls int
	e := NewEngineWithConnection(func(_ context.Context, cfg app.Connection) (*app.Service, error) {
		calls++
		require.Equal(t, connection, cfg)
		return service, nil
	}, settings.NewMemory(), &connection)
	defer e.Close()

	_, cmd := e.Update(e.Init()())
	for _, command := range cmd().(tea.BatchMsg) {
		_, result := e.Update(command())
		if result != nil {
			e.Update(result())
		}
	}
	require.Equal(t, 1, calls)
	require.IsType(t, &dashboard.Scene{}, e.scene)
}

type savedScene struct {
	visible, closed bool
	size            tea.WindowSizeMsg
	messages        []tea.Msg
}

func (s *savedScene) Init() tea.Cmd { return nil }

func (s *savedScene) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	s.messages = append(s.messages, msg)
	switch m := msg.(type) {
	case core.VisibilityMsg:
		s.visible = m.Visible
	case tea.WindowSizeMsg:
		s.size = m
	}
	return s, nil
}

func (s *savedScene) View() tea.View { return tea.NewView("dashboard") }

func (s *savedScene) Close() { s.closed = true }

func TestFullScreenNavigationRestoresSceneAndRoutesResults(t *testing.T) {
	t.Parallel()
	e := NewEngine(settings.NewMemory())
	dashboard := &savedScene{visible: true}
	e.scene = dashboard
	e.width, e.height = 120, 30
	screen := publish.New(core.DefaultTheme(), app.New(nil), widgets.PublishTarget{VHost: "/", RoutingKey: "orders"})
	e.Update(core.OpenScreen{Model: screen})
	require.Same(t, screen, e.scene)
	require.Same(t, dashboard, e.returnScene)
	require.False(t, dashboard.visible)
	require.NotContains(t, e.View().Content, "dashboard")
	e.Update(scopedMsg{session: e.session, value: sceneMsg{owner: dashboard, value: "background result"}})
	require.Contains(t, dashboard.messages, "background result")
	// Background navigation commands cannot cover the active screen.
	e.Update(
		scopedMsg{
			session: e.session,
			value:   sceneMsg{owner: dashboard, value: core.OpenDialog{Model: &savedScene{}}},
		},
	)
	require.Nil(t, e.dialog)
	e.Update(tea.WindowSizeMsg{Width: 140, Height: 35})
	_, cmd := e.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	require.NotNil(t, cmd)
	_, cmd = e.Update(cmd())
	require.NotNil(t, cmd)
	e.Update(cmd())
	require.NotNil(t, e.dialog)
	require.Same(t, screen, e.scene)
	_, cmd = e.Update(tea.KeyPressMsg{Code: 'y', Text: "y"})
	_, cmd = e.Update(cmd())
	// Closing the dialog also schedules visibility commands; process the batch.
	var deliver func(tea.Cmd)
	deliver = func(command tea.Cmd) {
		if command == nil {
			return
		}
		msg := command()
		if batch, ok := msg.(tea.BatchMsg); ok {
			for _, child := range batch {
				deliver(child)
			}
			return
		}
		_, next := e.Update(msg)
		deliver(next)
	}
	deliver(cmd)
	require.Same(t, dashboard, e.scene)
	require.Nil(t, e.returnScene)
	require.True(t, dashboard.visible)
	require.Equal(t, tea.WindowSizeMsg{Width: 140, Height: 35}, dashboard.size)
	count := len(dashboard.messages)
	e.Update(scopedMsg{session: e.session, value: sceneMsg{owner: screen, value: "late publish result"}})
	require.Len(t, dashboard.messages, count)
}

func TestSelectedPayloadCopyAndQuit(t *testing.T) {
	t.Parallel()
	e := NewEngine(settings.NewMemory())
	e.scene = &savedScene{visible: true}
	e.width, e.height = 120, 30
	screen := publish.New(core.DefaultTheme(), app.New(nil), widgets.PublishTarget{VHost: "/", RoutingKey: "orders"})
	e.Update(core.OpenScreen{Model: screen})
	e.Update(tea.PasteMsg{Content: "selected payload"})
	e.Update(tea.KeyPressMsg{Code: 'a', Mod: tea.ModCtrl})

	_, cmd := e.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModSuper})
	require.NotNil(t, cmd)
	require.Same(t, screen, e.scene)
	// A distinct Shift event is a copy binding; only the collapsed Ctrl+C event quits.
	_, shiftedCopy := e.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl | tea.ModShift})
	require.NotNil(t, shiftedCopy)
	require.Same(t, screen, e.scene)
	// Do not execute copy commands: tests must not change the system clipboard.
	_, first := e.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	require.Nil(t, first)
	require.False(t, e.returnScene.(*savedScene).closed)
	require.Contains(t, ansi.Strip(e.View().Content), "Press Ctrl+C again to quit")
	_, quit := e.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	require.IsType(t, tea.QuitMsg{}, quit())
	require.True(t, e.returnScene.(*savedScene).closed)
}

func TestOpeningScreenBelowMinimumSizeKeepsItHidden(t *testing.T) {
	t.Parallel()
	e := NewEngine(settings.NewMemory())
	e.scene = &savedScene{visible: true}
	screen := &savedScene{visible: true}
	e.Update(core.OpenScreen{Model: screen})
	require.False(t, screen.visible)
	e.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	require.True(t, screen.visible)
}

func TestQuitConfirmationAcrossScreensAndDialogs(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"login", "scene", "publish", "dialog", "small terminal"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			e := NewEngine(settings.NewMemory())
			e.width, e.height = 120, 30
			switch name {
			case "scene", "dialog", "small terminal":
				e.scene = &savedScene{visible: true}
			case "publish":
				e.scene = publish.New(core.DefaultTheme(), app.New(nil), widgets.PublishTarget{VHost: "/"})
			}
			e.Update(tea.WindowSizeMsg{Width: e.width, Height: e.height})
			if name == "dialog" {
				e.Update(core.OpenDialog{Model: components.NewShortcutsModal(e.theme, "Help", []core.Shortcut{core.Undo()})})
			}
			if name == "small terminal" {
				e.Update(tea.WindowSizeMsg{Width: 75, Height: 12})
			}
			_, first := e.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
			require.Nil(t, first)
			require.True(t, e.quitPending)
			view := ansi.Strip(e.View().Content)
			require.Contains(t, strings.Split(view, "\n")[0], "Press Ctrl+C again to quit")
			_, quit := e.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
			require.NotNil(t, quit)
			require.IsType(t, tea.QuitMsg{}, quit())
		})
	}
}

func TestQuitConfirmationCancelsOnOtherInputAndIgnoresRepeat(t *testing.T) {
	t.Parallel()
	for _, input := range []tea.Msg{
		tea.KeyPressMsg{Code: 'x', Text: "x"},
		tea.PasteMsg{Content: "payload"},
		tea.MouseClickMsg{Button: tea.MouseLeft},
	} {
		e := NewEngine(settings.NewMemory())
		e.scene = &savedScene{visible: true}
		e.width, e.height = 120, 30
		e.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
		_, repeated := e.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl, IsRepeat: true})
		require.Nil(t, repeated)
		require.True(t, e.quitPending)
		e.Update(input)
		require.False(t, e.quitPending)
		require.NotContains(t, ansi.Strip(e.View().Content), "again to quit")
		_, firstAgain := e.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
		require.Nil(t, firstAgain)
		require.True(t, e.quitPending)
		require.False(t, e.scene.(*savedScene).closed)
	}
}
