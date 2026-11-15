package login

import (
	"context"

	tea "charm.land/bubbletea/v2"

	"github.com/jadegopher/trabbit/internal/app"
	"github.com/jadegopher/trabbit/internal/tui/components"
	"github.com/jadegopher/trabbit/internal/tui/core"
	"github.com/jadegopher/trabbit/internal/tui/scenes/login/widgets"
)

const loginWidth = core.MinWidth

type MsgLoginSuccess struct{ Service *app.Service }

type loginResult struct {
	attempt int
	Service *app.Service
	Err     error
}
type Scene struct {
	theme         core.Theme
	input         *widgets.Login
	initialLogin  *widgets.ActionLoginRequested
	connect       app.Connector
	ctx           context.Context
	cancel        context.CancelFunc
	attempt       int
	busy          bool
	loader        components.Spinner
	width, height int
}

func NewLoginScene(theme core.Theme, connect app.Connector) *Scene {
	ctx, cancel := context.WithCancel(context.Background())
	return &Scene{
		theme:   theme,
		input:   widgets.NewLogin(theme),
		connect: connect,
		ctx:     ctx,
		cancel:  cancel,
		loader:  components.NewSpinner(theme, "Logging in..."),
	}
}

func NewLoginSceneWithConnection(theme core.Theme, connect app.Connector, connection app.Connection) *Scene {
	s := NewLoginScene(theme, connect)
	s.input.SetConnection(connection)
	s.initialLogin = &widgets.ActionLoginRequested{
		URL: connection.URL, Username: connection.Username, Password: connection.Password,
	}
	return s
}

func (s *Scene) Close() { s.cancel(); s.loader.Stop() }

func (s *Scene) Init() tea.Cmd {
	if s.initialLogin != nil {
		return core.Cmd(*s.initialLogin)
	}
	return s.input.Init()
}

func (s *Scene) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case tea.WindowSizeMsg:
		s.width = m.Width
		s.height = m.Height
	case core.VisibilityMsg:
		if !m.Visible {
			s.loader.Stop()
			return s, nil
		}
		if s.busy {
			return s, s.loader.Start()
		}
	case widgets.LoginValidationFailed:
		return s, core.Cmd(core.OpenDialog{Model: components.NewError(m.Err, s.theme)})
	case widgets.ActionLoginRequested:
		if s.busy {
			return s, nil
		}
		s.busy = true
		s.attempt++
		id := s.attempt

		cfg := app.Connection{URL: m.URL, Username: m.Username, Password: m.Password}
		ctx, connect := s.ctx, s.connect

		return s, tea.Batch(
			s.loader.Start(),
			func() tea.Msg {
				service, err := connect(ctx, cfg)
				return loginResult{id, service, err}
			},
		)
	case loginResult:
		if m.attempt != s.attempt {
			return s, nil
		}
		s.busy = false
		s.loader.Stop()
		if m.Err == nil {
			return s, core.Cmd(MsgLoginSuccess{Service: m.Service})
		}
		return s, core.Cmd(core.OpenDialog{Model: components.NewError(m.Err, s.theme)})
	}
	if s.busy {
		return s, s.loader.Update(msg)
	}
	_, cmd := s.input.Update(msg)
	return s, cmd
}

func (s *Scene) View() tea.View {
	v := s.input.View()
	if s.busy {
		v.Content += "\n" + s.loader.View()
		v.Cursor = nil
	}
	panel := s.theme.PlaceInModal(v, loginWidth, s.theme.Secondary)
	content := core.NewLayout(loginWidth, 0, core.Vertical).
		AddFixedView("branding", 0, widgets.Branding(s.theme, loginWidth)).
		AddFixedView("login", 0, panel).Render()
	return core.NewLayout(s.width, s.height, core.Vertical).PlaceView(content)
}
