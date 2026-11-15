package publish

import (
	"context"
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/jadegopher/trabbit/internal/app"
	"github.com/jadegopher/trabbit/internal/domain"
	"github.com/jadegopher/trabbit/internal/tui/core"
	"github.com/jadegopher/trabbit/internal/tui/scenes/publish/widgets"
	"github.com/stretchr/testify/require"
)

type publishBackend struct {
	app.Backend
	run  func(context.Context, string, string, app.PublishOptions) (app.PublishResult, error)
	list func(context.Context) (domain.VHosts, error)
}

func (b publishBackend) Publish(ctx context.Context, vhost, exchange string, p app.PublishOptions) (app.PublishResult, error) {
	return b.run(ctx, vhost, exchange, p)
}

func (b publishBackend) ListVHosts(ctx context.Context) (domain.VHosts, error) {
	if b.list == nil {
		return nil, nil
	}
	return b.list(ctx)
}

func newScene(t *testing.T, b publishBackend) *Scene {
	s := New(core.DefaultTheme(), app.New(b), widgets.PublishTarget{VHost: "/", Exchange: app.DefaultExchange, RoutingKey: "orders"})
	s.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	s.Init()
	t.Cleanup(s.Close)
	return s
}

func submit(s *Scene) (widgets.PublishRequest, tea.Cmd) {
	_, cmd := s.Update(tea.KeyPressMsg{Code: tea.KeyEnter, Mod: tea.ModCtrl})
	request := cmd().(widgets.PublishRequest)
	_, cmd = s.Update(request)
	return request, cmd
}

func TestPublishSceneOwnsRequestsAndRetry(t *testing.T) {
	t.Parallel()
	calls := 0
	s := newScene(t, publishBackend{run: func(_ context.Context, vhost, exchange string, p app.PublishOptions) (app.PublishResult, error) {
		calls++
		require.Equal(t, "/", vhost)
		require.Equal(t, app.DefaultExchange, exchange)
		require.Equal(t, "ordersr1p", p.RoutingKey)
		require.Equal(t, "line one\nline two", p.Payload)
		if calls == 1 {
			return app.PublishResult{}, errors.New("offline")
		}
		return app.PublishResult{Routed: true}, nil
	}})
	s.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	s.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	s.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	for _, r := range "r1p[" {
		s.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	s.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	s.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	s.Update(tea.PasteMsg{Content: "line one\nline two"})
	request, cmd := submit(s)
	_, duplicate := s.Update(request)
	require.Nil(t, duplicate)
	s.Update(result{id: request.ID + 1})
	require.True(t, s.form.Busy())
	s.Update(cmd())
	require.False(t, s.form.Busy())
	require.Contains(t, s.View().Content, "offline")
	retry, cmd := submit(s)
	s.Update(result{id: request.ID})
	require.True(t, s.form.Pending(retry.ID))
	s.Update(cmd())
	require.Contains(t, s.View().Content, "Published • routed")
	require.Equal(t, 2, calls)
}

func TestPublishBackCancelsRequestAndRejectsLateMessages(t *testing.T) {
	t.Parallel()
	s := newScene(t, publishBackend{run: func(ctx context.Context, _ string, _ string, _ app.PublishOptions) (app.PublishResult, error) {
		require.ErrorIs(t, ctx.Err(), context.Canceled)
		return app.PublishResult{}, ctx.Err()
	}})
	_, cmd := submit(s)
	_, back := s.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	_, closeCmd := s.Update(back())
	dialog := closeCmd().(core.OpenDialog).Model
	_, accept := dialog.Update(tea.KeyPressMsg{Code: 'y', Text: "y"})
	_, confirmed := s.Update(accept())
	require.Equal(t, core.CloseScreen{Model: s}, confirmed())
	s.Close()
	_, late := s.Update(cmd())
	require.Nil(t, late)
	require.True(t, s.closed)
}

func TestPublishScreenHasTwoPanelsAndBottomHints(t *testing.T) {
	t.Parallel()
	s := newScene(t, publishBackend{})
	for _, width := range []int{90, 120, 180} {
		s.Update(tea.WindowSizeMsg{Width: width, Height: 24})
		view := ansi.Strip(s.View().Content)
		lines := strings.Split(view, "\n")
		require.Len(t, lines, 24)
		require.Equal(t, 2, strings.Count(lines[1], "╭"))
		require.Equal(t, 2, strings.Count(view, "╰"))
		require.Contains(t, lines[0], "Publish message")
		require.Contains(t, lines[len(lines)-1], "F5 publish")
		require.NotContains(t, view, "Overview")
		require.NotContains(t, view, "Connections")
		require.NotContains(t, view, "Refresh")
		for _, line := range lines {
			require.Equal(t, width, lipgloss.Width(line))
		}
	}
}

func TestPublishLoadsVHostsAndPreservesQueuePreset(t *testing.T) {
	t.Parallel()
	backend := publishBackend{list: func(ctx context.Context) (domain.VHosts, error) {
		require.NoError(t, ctx.Err())
		return domain.VHosts{{Name: "/"}, {Name: "/team"}, {Name: "/other"}}, nil
	}}
	target := widgets.PublishTarget{VHost: "/team", Exchange: "events", RoutingKey: "orders"}
	s := New(core.DefaultTheme(), app.New(backend), target)
	t.Cleanup(s.Close)
	s.Update(tea.WindowSizeMsg{Width: 140, Height: 30})
	init := s.Init()
	require.Equal(t, target, s.form.Target(), "preset must be available before the API returns")
	s.Update(init())
	require.Equal(t, target, s.form.Target())
	require.Nil(t, s.vhostsCancel)
	s.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	s.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	require.Equal(t, "/", s.form.Target().VHost)
	// Consumed results cannot replace the available choices.
	s.Update(vhostsResult{values: domain.VHosts{{Name: "/unexpected"}}})
	s.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	require.Equal(t, "/other", s.form.Target().VHost)
	_, cmd := s.Update(tea.KeyPressMsg{Code: tea.KeyEnter, Mod: tea.ModCtrl})
	request := cmd().(widgets.PublishRequest)
	require.Equal(t, "/other", request.Target.VHost)
	require.Equal(t, "events", request.Target.Exchange)
	require.Equal(t, "orders", request.Options.RoutingKey)
}

func TestPublishVHostLoadFailureKeepsPresetUsable(t *testing.T) {
	t.Parallel()
	backend := publishBackend{list: func(context.Context) (domain.VHosts, error) {
		return nil, errors.New("offline")
	}}
	s := New(core.DefaultTheme(), app.New(backend), widgets.PublishTarget{VHost: "/team", RoutingKey: "orders"})
	t.Cleanup(s.Close)
	s.Update(tea.WindowSizeMsg{Width: 140, Height: 30})
	s.Update(s.Init()())
	require.Equal(t, "/team", s.form.Target().VHost)
	require.Contains(t, s.View().Content, "offline")
	_, cmd := s.Update(tea.KeyPressMsg{Code: tea.KeyEnter, Mod: tea.ModCtrl})
	require.Equal(t, "/team", cmd().(widgets.PublishRequest).Target.VHost)
}

func TestPublishCloseCancelsVHostLoadAndIgnoresLateResult(t *testing.T) {
	t.Parallel()
	backend := publishBackend{list: func(ctx context.Context) (domain.VHosts, error) {
		require.ErrorIs(t, ctx.Err(), context.Canceled)
		return domain.VHosts{{Name: "/late"}}, ctx.Err()
	}}
	s := New(core.DefaultTheme(), app.New(backend), widgets.PublishTarget{VHost: "/team"})
	cmd := s.Init()
	s.Close()
	_, next := s.Update(cmd())
	require.Nil(t, next)
	require.Nil(t, s.vhostsCancel)
	require.Equal(t, "/team", s.form.Target().VHost)
}
