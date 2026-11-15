package login

import (
	tea "charm.land/bubbletea/v2"
	"context"
	"errors"
	"github.com/charmbracelet/x/ansi"
	"github.com/jadegopher/trabbit/internal/app"
	"github.com/jadegopher/trabbit/internal/tui/components"
	"github.com/jadegopher/trabbit/internal/tui/core"
	"github.com/jadegopher/trabbit/internal/tui/scenes/login/widgets"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestBrandingSitsAboveLoginBorder(t *testing.T) {
	t.Parallel()
	s := NewLoginScene(core.DefaultTheme(), nil)
	defer s.Close()
	s.Update(tea.WindowSizeMsg{Width: core.MinWidth, Height: 24})
	v := s.View()
	plain := ansi.Strip(v.Content)
	require.Less(t, strings.Index(plain, "RabbitMq tui"), strings.Index(plain, "╭"))
	require.NotNil(t, v.Cursor)
	lines := strings.Split(plain, "\n")
	require.Contains(t, lines[v.Cursor.Y], "URL:")
	for i, line := range lines {
		if strings.Contains(line, "RabbitMq tui") {
			require.Contains(t, lines[i+1], "╭", "login border must directly follow branding")
		}
	}
}

func TestLoginFailureRetainsInputsAndAllowsRetry(t *testing.T) {
	t.Parallel()
	s := NewLoginScene(core.DefaultTheme(), func(context.Context, app.Connection) (*app.Service, error) { return nil, errors.New("denied") })
	defer s.Close()
	_, cmd := s.Update(widgets.ActionLoginRequested{URL: "http://localhost", Username: "user"})
	require.NotNil(t, cmd)
	require.True(t, s.busy)
	_, cmd = s.Update(loginResult{attempt: s.attempt, Err: errors.New("denied")})
	require.False(t, s.busy)
	dialog := cmd().(core.OpenDialog).Model
	require.IsType(t, &components.ErrorModal{}, dialog)
	require.Contains(t, dialog.View().Content, "denied")
	require.NotContains(t, s.View().Content, "denied")
	_, cmd = s.Update(loginResult{attempt: s.attempt - 1, Service: app.New(nil)})
	require.Nil(t, cmd)
	_, cmd = s.Update(widgets.ActionLoginRequested{URL: "http://localhost", Username: "user"})
	require.NotNil(t, cmd)
	s.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	require.Contains(t, s.View().Content, "RabbitMq tui")
}

func TestLoginValidationFailureOpensDialog(t *testing.T) {
	t.Parallel()
	s := NewLoginScene(core.DefaultTheme(), nil)
	defer s.Close()
	_, cmd := s.Update(widgets.LoginValidationFailed{Err: errors.New("invalid URL")})
	require.NotNil(t, cmd)
	dialog := cmd().(core.OpenDialog).Model
	require.IsType(t, &components.ErrorModal{}, dialog)
	require.Contains(t, dialog.View().Content, "invalid URL")
	require.NotContains(t, s.View().Content, "invalid URL")
}

func TestStartupConnectionFailureRetainsDraftForRetry(t *testing.T) {
	t.Parallel()

	connection := app.Connection{URL: "https://rabbit.example", Username: "alice", Password: "secret"}
	var requestContext context.Context
	s := NewLoginSceneWithConnection(core.DefaultTheme(), func(ctx context.Context, cfg app.Connection) (*app.Service, error) {
		require.Equal(t, connection, cfg)
		requestContext = ctx
		return nil, errors.New("denied")
	}, connection)
	defer s.Close()
	s.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	require.Contains(t, s.View().Content, connection.URL)
	require.Contains(t, s.View().Content, connection.Username)
	require.NotContains(t, s.View().Content, connection.Password)

	initial := s.Init()().(widgets.ActionLoginRequested)
	_, cmd := s.Update(initial)
	require.True(t, s.busy)
	_, duplicate := s.Update(initial)
	require.Nil(t, duplicate)

	// The batch also starts the spinner; deliver only the connection result.
	for _, command := range cmd().(tea.BatchMsg) {
		if result, ok := command().(loginResult); ok {
			_, cmd = s.Update(result)
		}
	}
	require.False(t, s.busy)
	require.IsType(t, core.OpenDialog{}, cmd())
	_, retry := s.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	require.Equal(t, initial, retry())
	s.Close()
	require.ErrorIs(t, requestContext.Err(), context.Canceled)
}
